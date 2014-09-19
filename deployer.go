package main

import (
	"encoding/json"
	"fmt"
	"github.com/armon/consul-api"
	//spew "github.com/davecgh/go-spew/spew"
	dockerapi "github.com/fsouza/go-dockerclient"
	"os/exec"
	"strings"
	//"log"
	//	"net/url"
	"archive/tar"
	//"bufio"
	"bytes"
	//"io"
	"io/ioutil"
	"net/url"
	"os"
	"time"
	//"github.com/armon/consul-api"
	//"errors"
	"path/filepath"
)

type Deployer struct {
	docker *dockerapi.Client
	consul *ConsulStore
}

func NewDeployer(url *url.URL) *Deployer {
	docker, err := dockerapi.NewClient(getopt("DOCKER_HOST", "unix:///var/run/docker.sock"))
	assert(err)

	//url, urlError := url.Parse("http://10.0.5.3:8500")
	//assert(urlError)

	consul, consulErr := NewConsulStore(url)
	assert(consulErr)

	//go consul.WatchAndHandle("test")

	deployer := &Deployer{
		docker: docker,
		consul: consul,
	}
	return deployer
}

func (d *Deployer) ListenForDeployEvent(prefix string) (int, error) {
	errCh := make(chan error, 1)
	pairCh := make(chan consulapi.KVPairs)
	quitCh := make(chan struct{})
	defer close(quitCh)
	go d.consul.Watch(prefix, pairCh, errCh, quitCh, true, true)

	var exitCh chan int
	for {
		var pairs consulapi.KVPairs

		// Wait for new pairs to come on our channel or an error to occur.
		select {
		case exit := <-exitCh:
			return exit, nil
		case pairs = <-pairCh:
		case err := <-errCh:
			return 0, err
		}
		for _, pair := range pairs {
			var deploy DeployRequest
			err := json.Unmarshal(pair.Value, &deploy)
			assert(err)
			fmt.Println("Deploying:", pair.Key, string(pair.Value))
			d.Deploy(deploy)
		}
	}
}

func (d *Deployer) NotifyDeploy(deploy DeployRequest) {
	json, err := json.Marshal(deploy)
	assert(err)
	d.consul.Put("deployer", json)
}

func (d *Deployer) build(basePath string, appName string) {

	inputbuf, outputbuf := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	tr := tar.NewWriter(inputbuf)
	err := filepath.Walk(basePath, func(path string, f os.FileInfo, err error) error {
		if strings.Contains(path, ".git") || f.IsDir() {
			return nil
		}
		relPath := path[len(basePath):len(path)]
		fmt.Printf("including: %s\n", relPath)

		t := time.Now()
		tr.WriteHeader(&tar.Header{Name: relPath, Size: f.Size(), ModTime: t, AccessTime: t, ChangeTime: t})
		b, fileErr := ioutil.ReadFile(path)
		assert(fileErr)
		tr.Write(b)
		return nil
	})
	tr.Close()
	assert(err)

	opts := dockerapi.BuildImageOptions{
		Name:           appName,
		SuppressOutput: false,
		InputStream:    inputbuf,
		OutputStream:   outputbuf,
	}
	buildErr := d.docker.BuildImage(opts)
	assert(buildErr)
	fmt.Printf("built: %s\n", appName)
}

func (d *Deployer) runDeplyScript(deploy DeployRequest) {
	cmd := exec.Command("bash", "-c", "/go/src/deployer/default-deploy.sh")
	cmd.Env = []string{"APP=" + deploy.ApplicationName, "BRANCH=" + deploy.Branch}
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.Run()
}

func (d *Deployer) shouldContainerDeploy(container *dockerapi.Container, deploy DeployRequest) bool {
	for _, kv := range container.Config.Env {
		kvp := strings.SplitN(kv, "=", 2)
		if kvp[0] == "DEPLOYER_NAME" {
			fmt.Println("Application: ", container.Config.Image, "runs: ", deploy.ApplicationName)
			return kvp[1] == deploy.ApplicationName
		}
	}
	fmt.Println("Container: ", container.Config.Image, "did is not running app:", deploy.ApplicationName)
	return false
}

func (d *Deployer) Deploy(deploy DeployRequest) {
	//fmt.Println("Restarting: ", name)
	containers, err := d.docker.ListContainers(dockerapi.ListContainersOptions{})
	assert(err)
	deployed := 0
	for _, container := range containers {
		//spew.Dump(container)
		inspectedContainer, err := d.docker.InspectContainer(container.ID)
		assert(err)
		if inspectedContainer.State.Running {
			//spew.Dump(inspectedContainer)
			//spew.Dump(inspectedContainer.Volumes)
			//for key := range inspectedContainer.Volumes {
			//if key == "/data" && container.Image == deploy.ApplicationName {
			if d.shouldContainerDeploy(inspectedContainer, deploy) {
				appDir := "/data/" + deploy.ApplicationName + "-" + deploy.Branch

				//1: run deploy script
				d.runDeplyScript(deploy)

				fmt.Println("Building: ", appDir+"/Dockerfile")
				//1.5: build image
				d.build(appDir, deploy.ApplicationName)

				//2: Create and start new container
				createdContainer, createErr := d.docker.CreateContainer(dockerapi.CreateContainerOptions{
					Config: inspectedContainer.Config,
				})
				assert(createErr)
				fmt.Println("Created: ", createdContainer.ID)

				fmt.Println("Starting: ", createdContainer.ID)
				err := d.docker.StartContainer(createdContainer.ID, inspectedContainer.HostConfig)
				assert(err)

				//3: TODO disable old in loadbalancer

				//4: Stop old container
				fmt.Println("Stopping: ", container.ID)
				d.docker.StopContainer(container.ID, 5)

				//d.applications[app.ID] = app
				//fmt.Println("App:", inspectedContainer.ID[:12], "is up and mounts:", key, " nr of apps:", len(service.applications))
				//service.registry.Add(Application{Image: inspectedContainer.Config.Image})
				deployed++
			}

			//}
		}
	}
	if deployed == 0 {
		fmt.Println("No matching containers to deploy to:", deploy.ApplicationName)
	}
}

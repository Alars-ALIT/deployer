package main

import (
	//"fmt"
	//spew "github.com/davecgh/go-spew/spew"
	dockerapi "github.com/fsouza/go-dockerclient"
	"log"
	"net/url"
	"os"
	//"github.com/armon/consul-api"
)

func getopt(name, def string) string {
	if env := os.Getenv(name); env != "" {
		return env
	}
	return def
}

func assert(err error) {
	if err != nil {
		log.Fatal("registrator: ", err)
	}
}

func main() {
	docker, err := dockerapi.NewClient(getopt("DOCKER_HOST", "unix:///var/run/docker.sock"))
	assert(err)

	url, urlError := url.Parse("http://10.0.5.3:8500")
	assert(urlError)

	consul, consulErr := NewConsulStore(url)
	assert(consulErr)

	//go consul.WatchAndHandle("test")

	deployer := &Deployer{
		docker: docker,
		consul: consul,
	}

	go deployer.ListenForDeployEvent("deployer")

	server := &WebServer{
		deployer: deployer,
	}
	server.start()

}

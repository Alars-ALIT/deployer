package main

import (
	//"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"time"
)

type WebServer struct {
	deployer *Deployer
}

type DeployRequest struct {
	ApplicationName string
	Branch          string
	Time            time.Time
}

func (server *WebServer) handler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploy := DeployRequest{
		ApplicationName: vars["app"],
		Branch:          vars["branch"],
		Time:            time.Now(),
	}

	server.deployer.NotifyDeploy(deploy)
	//fmt.Fprintf(w, "%s!", name)
}

func (server *WebServer) start() {
	log.Println("Starting server..")
	r := mux.NewRouter()
	//r.HandleFunc("/products/{key}", ProductHandler)
	r.HandleFunc("/deployer/{app}:{branch}", server.handler)
	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}

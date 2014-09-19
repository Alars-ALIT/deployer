package main

import (
	"fmt"
	"net/url"
	"testing"
)

func TestBuild(t *testing.T) {
	url, err := url.Parse("")
	assert(err)
	d := NewDeployer(url)
	d.build("/data/busybox-go-webapp-master/")
	fmt.Println("goodbye")
}

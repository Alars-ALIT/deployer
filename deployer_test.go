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
	d.build("/go/src/busybox-go-webapp/", "busybox-go-webapp")
	fmt.Println("goodbye")
}

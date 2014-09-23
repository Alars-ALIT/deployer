package main

import ()

type Type int

const (
	Local Type = iota
	Git
	Download
)

type Fetcher struct {
	appType Type
}

func (fetcher *Fetcher) fetch() {

}

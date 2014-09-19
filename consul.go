package main

import (
	//	"errors"
	//"log"
	//"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/armon/consul-api"
)

type ConsulStore struct {
	sync.Mutex
	client      *consulapi.Client
	prefix      string
	configIndex uint64
	watching    map[string]struct{}
}

func NewConsulStore(uri *url.URL) (*ConsulStore, error) {
	client, err := consulapi.NewClient(&consulapi.Config{
		Address:    "10.0.5.3:8500",
		HttpClient: http.DefaultClient,
	})
	if err != nil {
		return nil, err
	}
	return &ConsulStore{
		client:   client,
		prefix:   uri.Path,
		watching: make(map[string]struct{}),
	}, nil
}

func (s *ConsulStore) Put(key string, value []byte) (*consulapi.WriteMeta, error) {
	pair := &consulapi.KVPair{
		Key:   key,
		Value: value,
	}
	return s.client.KV().Put(pair, &consulapi.WriteOptions{})
}

/*
func (s *ConsulStore) WatchAndHandle(prefix string) (int, error) {
	errCh := make(chan error, 1)
	pairCh := make(chan consulapi.KVPairs)
	quitCh := make(chan struct{})
	defer close(quitCh)
	go s.Watch(prefix, pairCh, errCh, quitCh, true, true)

	var exitCh chan int
	for {
		var pairs consulapi.KVPairs

		// Wait for new pairs to come on our channel or an error
		// to occur.
		select {
		case exit := <-exitCh:
			return exit, nil
		case pairs = <-pairCh:
		case err := <-errCh:
			return 0, err
		}
		for _, pair := range pairs {
			fmt.Println("Pair:", pair.Key, string(pair.Value))
		}
	}
}
*/

func (s *ConsulStore) Watch(
	prefix string,
	pairCh chan<- consulapi.KVPairs,
	errCh chan<- error,
	quitCh <-chan struct{},
	errExit bool,
	watch bool) {
	// Get the initial list of k/v pairs. We don't do a retryableList
	// here because we want a fast fail if the initial request fails.

	pairs, meta, err := s.client.KV().List(prefix, nil)
	if err != nil {
		errCh <- err
		return
	}

	// Send the initial list out right away
	//pairCh <- pairs

	// If we're not watching, just return right away
	if !watch {
		return
	}

	// Loop forever (or until quitCh is closed) and watch the keys
	// for changes.
	curIndex := meta.LastIndex
	for {
		select {
		case <-quitCh:
			return
		default:
		}

		pairs, meta, err = retryableList(
			func() (consulapi.KVPairs, *consulapi.QueryMeta, error) {
				opts := &consulapi.QueryOptions{WaitIndex: curIndex}
				return s.client.KV().List(prefix, opts)
			})
		if err != nil {
			if errExit {
				errCh <- err
				return
			}
		}

		pairCh <- pairs
		curIndex = meta.LastIndex
	}
}

// This function is able to call KV listing functions and retry them.
// We want to retry if there are errors because it is safe (GET request),
// and erroring early is MUCH more costly than retrying over time and
// delaying the configuration propagation.
func retryableList(f func() (consulapi.KVPairs, *consulapi.QueryMeta, error)) (consulapi.KVPairs, *consulapi.QueryMeta, error) {
	i := 0
	for {
		p, m, e := f()
		if e != nil {
			if i >= 3 {
				return nil, nil, e
			}

			i++

			// Reasonably arbitrary sleep to just try again... It is
			// a GET request so this is safe.
			time.Sleep(time.Duration(i*2) * time.Second)
		}

		return p, m, e
	}
}

package balancer

import (
	"errors"
	"net/url"
	"sync"
	"sync/atomic"
)

var (
	ErrNoAvailableBackends = errors.New("no available backends")
	ErrAllBackendDown      = errors.New("all backends are down")
)

type RoundRobin struct {
	backends    []*url.URL
	current     uint32
	failedHosts map[string]bool
	failedMutex sync.RWMutex
}

func NewRoundRobin(backends []*url.URL) *RoundRobin {
	return &RoundRobin{
		backends:    backends,
		current:     0,
		failedHosts: make(map[string]bool),
	}
}

func (rr *RoundRobin) Next() (*url.URL, error) {
	if len(rr.backends) == 0 {
		return nil, ErrNoAvailableBackends
	}

	rr.failedMutex.RLock()
	if len(rr.failedHosts) == len(rr.backends) {
		rr.failedMutex.RUnlock()
		return nil, ErrAllBackendDown
	}
	rr.failedMutex.RUnlock()

	for i := 0; i < len(rr.backends); i++ {
		current := atomic.AddUint32(&rr.current, 1) % uint32(len(rr.backends))
		backend := rr.backends[current]

		rr.failedMutex.RLock()
		isFailed := rr.failedHosts[backend.String()]
		rr.failedMutex.RUnlock()

		if !isFailed {
			return backend, nil
		}
	}

	return nil, ErrNoAvailableBackends
}

func (rr *RoundRobin) MarkFailed(backendURL *url.URL) {
	if backendURL != nil {
		rr.failedMutex.Lock()
		rr.failedHosts[backendURL.String()] = true
		rr.failedMutex.Unlock()
	}
}

func (rr *RoundRobin) GetAllBackends() []*url.URL {
	return rr.backends
}

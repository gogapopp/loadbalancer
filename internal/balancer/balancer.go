package balancer

import (
	"loadbalancer/internal/config"
	"net/url"
)

type Balancer interface {
	Next() (*url.URL, error)
	MarkFailed(backendURL *url.URL)
	GetAllBackends() []*url.URL
}

func Factory(cfg *config.BalancerConfig) (Balancer, error) {
	var backends []*url.URL

	for _, backend := range cfg.Backends {
		u, err := url.Parse(backend.URL)
		if err != nil {
			return nil, err
		}
		backends = append(backends, u)
	}

	switch cfg.Algorithm {
	case "round_robin":
		return NewRoundRobin(backends), nil
	default:
		return NewRoundRobin(backends), nil
	}
}

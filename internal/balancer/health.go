package balancer

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	"go.uber.org/zap"
)

type HealthChecker struct {
	balancer    Balancer
	client      *http.Client
	logger      *zap.SugaredLogger
	stopCh      chan struct{}
	wg          sync.WaitGroup
	checkPeriod time.Duration
}

func NewHealthChecker(balancer Balancer, checkPeriod time.Duration, logger *zap.SugaredLogger) *HealthChecker {
	return &HealthChecker{
		balancer: balancer,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger:      logger,
		stopCh:      make(chan struct{}),
		checkPeriod: checkPeriod,
	}
}

func (hc *HealthChecker) Start() {
	hc.wg.Add(1)
	go func() {
		defer hc.wg.Done()

		ticker := time.NewTicker(hc.checkPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				hc.checkAll()
			case <-hc.stopCh:
				return
			}
		}
	}()
}

func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
	hc.wg.Wait()
}

func (hc *HealthChecker) checkAll() {
	backends := hc.balancer.GetAllBackends()
	for _, backend := range backends {
		go hc.checkBackend(backend)
	}
}

func (hc *HealthChecker) checkBackend(backend *url.URL) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", backend.String(), nil)
	if err != nil {
		hc.logger.Error("failed health check request", "backend", backend.String(), "error", err)
		return
	}

	resp, err := hc.client.Do(req)
	if err != nil {
		hc.logger.Warn("health check failed", "backend", backend.String(), "error", err)

		hc.balancer.MarkFailed(backend)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		hc.logger.Warn("health check failed", "backend", backend.String(), "status", resp.StatusCode)

		hc.balancer.MarkFailed(backend)
	}
}

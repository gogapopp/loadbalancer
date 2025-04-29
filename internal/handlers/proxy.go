package handlers

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"time"

	"loadbalancer/internal/balancer"
	"loadbalancer/internal/limiter"

	"go.uber.org/zap"
)

var (
	ErrRateLimitExceeded   = errors.New("rate limit exceeded")
	ErrNoAvailableBackends = errors.New("no available backends")
)

type ProxyHandler struct {
	balancer    balancer.Balancer
	rateLimiter *limiter.TokenBucket
	logger      *zap.SugaredLogger
}

func NewProxyHandler(balancer balancer.Balancer, rateLimiter *limiter.TokenBucket, logger *zap.SugaredLogger) *ProxyHandler {
	return &ProxyHandler{
		balancer:    balancer,
		rateLimiter: rateLimiter,
		logger:      logger,
	}
}

func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	clientIP := r.RemoteAddr

	if p.rateLimiter != nil {
		if !p.rateLimiter.Allow(clientIP) {
			p.logger.Warn(ErrRateLimitExceeded, "client", clientIP)
			http.Error(w, ErrRateLimitExceeded.Error(), http.StatusTooManyRequests)
			return
		}
	}

	backend, err := p.balancer.Next()
	if err != nil {
		p.logger.Error(ErrNoAvailableBackends, "error", err)
		http.Error(w, ErrNoAvailableBackends.Error(), http.StatusServiceUnavailable)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(backend)

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		p.logger.Error("proxy error for backend", backend.String(), "error", err)

		p.balancer.MarkFailed(backend)
		http.Error(w, "service Unavailable", http.StatusServiceUnavailable)
	}

	p.logger.Info("proxying request",
		"method", r.Method,
		"path", r.URL.Path,
		"backend", backend.String(),
		"client", clientIP,
	)

	proxy.ServeHTTP(w, r)

	p.logger.Info("request completed",
		"method", r.Method,
		"path", r.URL.Path,
		"backend", backend.String(),
		"duration", time.Since(startTime),
	)
}

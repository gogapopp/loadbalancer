package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"loadbalancer/internal/balancer"
	"loadbalancer/internal/config"
	"loadbalancer/internal/handlers"
	"loadbalancer/internal/limiter"

	"go.uber.org/zap"
)

const (
	readTimeout  = 10 * time.Second
	writeTimeout = 10 * time.Second
	idleTimeout  = 120 * time.Second
)

type Server struct {
	server        *http.Server
	healthChecker *balancer.HealthChecker
	logger        *zap.SugaredLogger
}

func New(cfg *config.Config, logger *zap.SugaredLogger) (*Server, error) {
	bal, err := balancer.Factory(&cfg.Balancer)
	if err != nil {
		return nil, fmt.Errorf("failed to create balancer: %w", err)
	}

	var rateLimiter *limiter.TokenBucket
	if cfg.RateLimiter.Enabled {
		rateLimiter = limiter.NewTokenBucket(&cfg.RateLimiter)
	}

	healthChecker := balancer.NewHealthChecker(bal, 30*time.Second, logger)

	proxy := handlers.NewProxyHandler(bal, rateLimiter, logger)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      proxy,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	return &Server{
		server:        srv,
		healthChecker: healthChecker,
		logger:        logger,
	}, nil
}

func (s *Server) Start() error {
	s.healthChecker.Start()

	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.healthChecker.Stop()

	return s.server.Shutdown(ctx)
}

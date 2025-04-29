package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"loadbalancer/internal/config"
	"loadbalancer/internal/logger"
	"loadbalancer/internal/server"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to configuration file")
	flag.Parse()

	logger, err := logger.New()
	if err != nil {
		logger.Fatal(err)
	}
	logger.Info("starting load balancer")

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Fatal(err)
	}

	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Fatal(err)
	}

	go func() {
		logger.Info("server started on: ", cfg.Server.Port)
		if err := srv.Start(); err != nil {
			logger.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error(err)
	}
}

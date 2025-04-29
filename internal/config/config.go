package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Backend struct {
	URL string `yaml:"url" env:"BACKEND_URL"`
}

type ServerConfig struct {
	Port int `yaml:"port" env:"SERVER_PORT" env-default:"8080"`
}

type BalancerConfig struct {
	Algorithm string    `yaml:"algorithm" env:"BALANCER_ALGORITHM" env-default:"round_robin"`
	Backends  []Backend `yaml:"backends"`
}

type RateLimiterConfig struct {
	Enabled      bool `yaml:"enabled" env:"RATE_LIMITER_ENABLED" env-default:"true"`
	DefaultLimit struct {
		Capacity   int `yaml:"capacity" env:"RATE_LIMITER_CAPACITY" env-default:"100"`
		RatePerSec int `yaml:"rate_per_sec" env:"RATE_LIMITER_RATE" env-default:"10"`
	} `yaml:"default_limit"`
}

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Balancer    BalancerConfig    `yaml:"balancer"`
	RateLimiter RateLimiterConfig `yaml:"rate_limiter"`
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config

	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %w", err)
	}

	return &cfg, nil
}

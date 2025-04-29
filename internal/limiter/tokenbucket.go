package limiter

import (
	"sync"
	"time"

	"loadbalancer/internal/config"
)

type Client struct {
	tokens     int
	capacity   int
	ratePerSec int
	lastRefill time.Time
	mu         sync.Mutex
}

type TokenBucket struct {
	clients     map[string]*Client
	defaultCap  int
	defaultRate int
	mu          sync.RWMutex
}

func NewTokenBucket(cfg *config.RateLimiterConfig) *TokenBucket {
	tb := &TokenBucket{
		clients:     make(map[string]*Client),
		defaultCap:  cfg.DefaultLimit.Capacity,
		defaultRate: cfg.DefaultLimit.RatePerSec,
	}

	go tb.cleanupInactive()

	return tb
}

func (tb *TokenBucket) Allow(clientID string) bool {
	client := tb.getClient(clientID)

	client.mu.Lock()
	defer client.mu.Unlock()

	tb.refillTokens(client)

	if client.tokens > 0 {
		client.tokens--
		return true
	}

	return false
}

func (tb *TokenBucket) getClient(clientID string) *Client {
	tb.mu.RLock()
	client, exists := tb.clients[clientID]
	tb.mu.RUnlock()

	if !exists {
		tb.mu.Lock()

		client = &Client{
			tokens:     tb.defaultCap,
			capacity:   tb.defaultCap,
			ratePerSec: tb.defaultRate,
			lastRefill: time.Now(),
		}

		tb.clients[clientID] = client

		tb.mu.Unlock()
	}

	return client
}

func (tb *TokenBucket) refillTokens(client *Client) {
	now := time.Now()
	elapsed := now.Sub(client.lastRefill).Seconds()
	tokensToAdd := int(elapsed * float64(client.ratePerSec))

	if tokensToAdd > 0 {
		client.tokens = min(client.tokens+tokensToAdd, client.capacity)
		client.lastRefill = now
	}
}

func (tb *TokenBucket) SetClientLimit(clientID string, capacity, ratePerSec int) {
	client := tb.getClient(clientID)

	client.mu.Lock()

	client.capacity = capacity
	client.ratePerSec = ratePerSec
	client.tokens = min(client.tokens, capacity)

	client.mu.Unlock()
}

func (tb *TokenBucket) cleanupInactive() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		var toDelete []string

		tb.mu.RLock()
		for id, client := range tb.clients {
			client.mu.Lock()
			if now.Sub(client.lastRefill) > time.Hour {
				toDelete = append(toDelete, id)
			}
			client.mu.Unlock()
		}
		tb.mu.RUnlock()

		if len(toDelete) > 0 {
			tb.mu.Lock()
			for _, id := range toDelete {
				delete(tb.clients, id)
			}
			tb.mu.Unlock()
		}
	}
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

package cache

import (
	"sync"
	"time"
)

type entry struct {
	value   []byte
	expires time.Time
}

// TTL is a small concurrency-safe in-memory cache with per-entry expiry. It is
// the first line of defense for GitHub rate limits: identical card requests are
// served from memory until their TTL lapses. A CDN in front (Cloudflare) backs
// this up for cross-instance caching.
type TTL struct {
	mu   sync.RWMutex
	data map[string]entry
}

func New() *TTL {
	c := &TTL{data: make(map[string]entry)}
	go c.janitor()
	return c
}

func (c *TTL) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expires) {
		return nil, false
	}
	return e.value, true
}

func (c *TTL) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	c.data[key] = entry{value: value, expires: time.Now().Add(ttl)}
	c.mu.Unlock()
}

func (c *TTL) janitor() {
	t := time.NewTicker(10 * time.Minute)
	for range t.C {
		now := time.Now()
		c.mu.Lock()
		for k, e := range c.data {
			if now.After(e.expires) {
				delete(c.data, k)
			}
		}
		c.mu.Unlock()
	}
}

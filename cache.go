// Package cache provides an in-memory cache with TTL support.
package cache

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

const DefaultTTL = 5 * time.Minute

// Entry holds a cached HTTP response.
type Entry struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	ExpiresAt  time.Time
}

// Cache is a thread-safe in-memory store.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*Entry
	ttl     time.Duration
}

// New creates a new Cache with the given TTL.
func New(ttl time.Duration) *Cache {
	c := &Cache{
		entries: make(map[string]*Entry),
		ttl:     ttl,
	}
	// Background goroutine to evict expired entries every minute
	go c.evictLoop()
	return c
}

// makeKey generates a cache key from HTTP method and URL.
func makeKey(method, url string) string {
	return fmt.Sprintf("%s:%s", method, url)
}

// Get retrieves a cached entry. Returns nil if not found or expired.
func (c *Cache) Get(method, url string) *Entry {
	key := makeKey(method, url)

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return nil
	}

	if time.Now().After(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil
	}

	return entry
}

// Set stores a response in the cache.
func (c *Cache) Set(method, url string, entry *Entry) {
	key := makeKey(method, url)
	entry.ExpiresAt = time.Now().Add(c.ttl)

	c.mu.Lock()
	c.entries[key] = entry
	c.mu.Unlock()
}

// Clear removes all cache entries. Returns the number of entries removed.
func (c *Cache) Clear() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := len(c.entries)
	c.entries = make(map[string]*Entry)
	return count
}

// Size returns the current number of cached entries.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Keys returns all current cache keys (for debugging).
func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.entries))
	for k := range c.entries {
		keys = append(keys, k)
	}
	return keys
}

// evictLoop runs in the background and removes expired entries periodically.
func (c *Cache) evictLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for key, entry := range c.entries {
			if now.After(entry.ExpiresAt) {
				delete(c.entries, key)
			}
		}
		c.mu.Unlock()
	}
}

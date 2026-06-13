package cache_test

import (
	"net/http"
	"testing"
	"time"

	"caching-proxy/internal/cache"
)

func TestCacheSetAndGet(t *testing.T) {
	c := cache.New(5 * time.Minute)

	entry := &cache.Entry{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       []byte(`{"id":1}`),
	}

	c.Set("GET", "https://example.com/products/1", entry)

	got := c.Get("GET", "https://example.com/products/1")
	if got == nil {
		t.Fatal("expected cache hit, got nil")
	}
	if got.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", got.StatusCode)
	}
	if string(got.Body) != `{"id":1}` {
		t.Errorf("unexpected body: %s", got.Body)
	}
}

func TestCacheMiss(t *testing.T) {
	c := cache.New(5 * time.Minute)

	got := c.Get("GET", "https://example.com/nonexistent")
	if got != nil {
		t.Fatal("expected cache miss, got an entry")
	}
}

func TestCacheTTLExpiry(t *testing.T) {
	// Use a very short TTL
	c := cache.New(50 * time.Millisecond)

	c.Set("GET", "https://example.com/test", &cache.Entry{
		StatusCode: 200,
		Body:       []byte("hello"),
	})

	// Should be a hit immediately
	if c.Get("GET", "https://example.com/test") == nil {
		t.Fatal("expected cache hit before TTL expiry")
	}

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Should be a miss now
	if c.Get("GET", "https://example.com/test") != nil {
		t.Fatal("expected cache miss after TTL expiry")
	}
}

func TestCacheClear(t *testing.T) {
	c := cache.New(5 * time.Minute)

	c.Set("GET", "https://example.com/a", &cache.Entry{StatusCode: 200})
	c.Set("GET", "https://example.com/b", &cache.Entry{StatusCode: 200})

	removed := c.Clear()
	if removed != 2 {
		t.Errorf("expected 2 entries cleared, got %d", removed)
	}
	if c.Size() != 0 {
		t.Errorf("expected cache size 0 after clear, got %d", c.Size())
	}
}

func TestCacheMethodAwareness(t *testing.T) {
	c := cache.New(5 * time.Minute)

	c.Set("GET", "https://example.com/resource", &cache.Entry{StatusCode: 200})

	// GET should hit
	if c.Get("GET", "https://example.com/resource") == nil {
		t.Fatal("expected GET to hit cache")
	}
	// POST to same URL should miss (different key)
	if c.Get("POST", "https://example.com/resource") != nil {
		t.Fatal("expected POST to miss cache")
	}
}

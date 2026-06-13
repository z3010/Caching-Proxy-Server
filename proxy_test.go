package proxy_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"caching-proxy/internal/cache"
	"caching-proxy/internal/proxy"
)

// newTestOrigin creates a fake origin server for testing.
func newTestOrigin(statusCode int, body string) *httptest.Server {
	callCount := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}))
}

func TestCacheMissOnFirstRequest(t *testing.T) {
	origin := newTestOrigin(200, `{"id":1}`)
	defer origin.Close()

	c := cache.New(5 * time.Minute)
	p := proxy.New(origin.URL, c)

	req := httptest.NewRequest(http.MethodGet, "/products/1", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Cache") != "MISS" {
		t.Errorf("expected X-Cache: MISS, got %s", rec.Header().Get("X-Cache"))
	}
}

func TestCacheHitOnSecondRequest(t *testing.T) {
	origin := newTestOrigin(200, `{"id":1}`)
	defer origin.Close()

	c := cache.New(5 * time.Minute)
	p := proxy.New(origin.URL, c)

	// First request — MISS
	req1 := httptest.NewRequest(http.MethodGet, "/products/1", nil)
	rec1 := httptest.NewRecorder()
	p.ServeHTTP(rec1, req1)

	if rec1.Header().Get("X-Cache") != "MISS" {
		t.Errorf("first request: expected MISS, got %s", rec1.Header().Get("X-Cache"))
	}

	// Second request — HIT
	req2 := httptest.NewRequest(http.MethodGet, "/products/1", nil)
	rec2 := httptest.NewRecorder()
	p.ServeHTTP(rec2, req2)

	if rec2.Header().Get("X-Cache") != "HIT" {
		t.Errorf("second request: expected HIT, got %s", rec2.Header().Get("X-Cache"))
	}
	if rec2.Code != 200 {
		t.Errorf("expected status 200 from cache, got %d", rec2.Code)
	}
}

func TestPostRequestNotCached(t *testing.T) {
	origin := newTestOrigin(201, `{"created":true}`)
	defer origin.Close()

	c := cache.New(5 * time.Minute)
	p := proxy.New(origin.URL, c)

	// Two POST requests — both should be MISS
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/products", nil)
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)

		if rec.Header().Get("X-Cache") != "MISS" {
			t.Errorf("POST request %d: expected MISS, got %s", i+1, rec.Header().Get("X-Cache"))
		}
	}
}

func TestOriginStatusCodePreserved(t *testing.T) {
	origin := newTestOrigin(404, `{"error":"not found"}`)
	defer origin.Close()

	c := cache.New(5 * time.Minute)
	p := proxy.New(origin.URL, c)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("expected status 404 preserved from origin, got %d", rec.Code)
	}
}

// Package proxy implements the HTTP caching proxy server.
package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"caching-proxy/internal/cache"
)

// Server is the caching proxy server.
type Server struct {
	origin     string
	cache      *cache.Cache
	httpClient *http.Client
}

// New creates a new proxy Server for the given origin URL.
func New(origin string, c *cache.Cache) *Server {
	return &Server{
		origin: origin,
		cache:  c,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ServeHTTP handles all incoming requests — implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Build the full target URL
	targetURL := s.origin + r.URL.RequestURI()

	isCacheable := r.Method == http.MethodGet

	// ── CACHE HIT ────────────────────────────────────────────────
	if isCacheable {
		if entry := s.cache.Get(r.Method, targetURL); entry != nil {
			log.Printf("  [CACHE HIT]  %s %s", r.Method, r.URL.RequestURI())
			copyHeaders(w.Header(), entry.Headers)
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(entry.StatusCode)
			w.Write(entry.Body)
			return
		}
	}

	// ── CACHE MISS — forward to origin ───────────────────────────
	log.Printf("  [CACHE MISS] %s %s  →  %s", r.Method, r.URL.RequestURI(), targetURL)

	// Read request body (needed for POST/PUT)
	var bodyBytes []byte
	if r.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
	}

	// Build the outgoing request
	proxyReq, err := http.NewRequest(r.Method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to build request: %v", err), http.StatusInternalServerError)
		return
	}

	// Forward headers, skipping hop-by-hop headers
	copyHeaders(proxyReq.Header, r.Header)
	proxyReq.Header.Del("Host")
	proxyReq.Header.Del("Connection")
	proxyReq.Header.Del("Transfer-Encoding")

	// Make the request to origin
	resp, err := s.httpClient.Do(proxyReq)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
			log.Printf("  [ERROR] Request timed out: %s", targetURL)
			http.Error(w, `{"error":"Gateway Timeout","message":"Origin server took too long to respond."}`, http.StatusGatewayTimeout)
			return
		}
		log.Printf("  [ERROR] Could not reach origin: %v", err)
		http.Error(w, fmt.Sprintf(`{"error":"Bad Gateway","message":"Could not reach origin: %v"}`, err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read origin response", http.StatusInternalServerError)
		return
	}

	// Store in cache if cacheable
	if isCacheable {
		s.cache.Set(r.Method, targetURL, &cache.Entry{
			StatusCode: resp.StatusCode,
			Headers:    resp.Header.Clone(),
			Body:       respBody,
		})
	}

	// Write response back to client
	copyHeaders(w.Header(), resp.Header)
	w.Header().Del("Transfer-Encoding") // let Go handle this
	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

// copyHeaders copies headers from src to dst.
func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, v := range values {
			dst.Set(key, v)
		}
	}
}

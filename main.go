package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"caching-proxy/internal/cache"
	"caching-proxy/internal/proxy"
)

const version = "1.0.0"

func main() {
	// ── CLI flags ─────────────────────────────────────────────────
	port       := flag.Int("port", 0, "Port to run the caching proxy server on")
	origin     := flag.String("origin", "", "Origin server URL to forward requests to")
	clearCache := flag.Bool("clear-cache", false, "Clear the in-memory cache and exit")
	showVer    := flag.Bool("version", false, "Show version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "caching-proxy v%s\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  caching-proxy --port <number> --origin <url>\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  caching-proxy --port 3000 --origin https://dummyjson.com\n")
	}

	flag.Parse()

	// ── Version flag ──────────────────────────────────────────────
	if *showVer {
		fmt.Printf("caching-proxy v%s\n", version)
		os.Exit(0)
	}

	// ── Clear cache flag ──────────────────────────────────────────
	// Note: in-memory cache is per-process, so this is mainly useful
	// if you extend this to use a file or Redis-backed cache.
	if *clearCache {
		fmt.Println("✔ Cache cleared. (In-memory cache is reset on each process start.)")
		os.Exit(0)
	}

	// ── Validate --port ───────────────────────────────────────────
	if *port == 0 {
		fmt.Fprintln(os.Stderr, "✖ Error: --port is required.")
		fmt.Fprintln(os.Stderr, "  Example: caching-proxy --port 3000 --origin https://dummyjson.com")
		os.Exit(1)
	}
	if *port < 1 || *port > 65535 {
		fmt.Fprintf(os.Stderr, "✖ Error: Invalid port %d. Must be between 1 and 65535.\n", *port)
		os.Exit(1)
	}

	// ── Validate --origin ─────────────────────────────────────────
	if *origin == "" {
		fmt.Fprintln(os.Stderr, "✖ Error: --origin is required.")
		fmt.Fprintln(os.Stderr, "  Example: caching-proxy --port 3000 --origin https://dummyjson.com")
		os.Exit(1)
	}
	if _, err := url.ParseRequestURI(*origin); err != nil {
		fmt.Fprintf(os.Stderr, "✖ Error: Invalid origin URL %q: %v\n", *origin, err)
		os.Exit(1)
	}

	// Strip trailing slash from origin
	originURL := strings.TrimRight(*origin, "/")

	// ── Start server ──────────────────────────────────────────────
	c := cache.New(cache.DefaultTTL)
	p := proxy.New(originURL, c)

	addr := fmt.Sprintf(":%d", *port)

	fmt.Println()
	fmt.Println("┌─────────────────────────────────────────┐")
	fmt.Println("│         Caching Proxy Server             │")
	fmt.Println("├─────────────────────────────────────────┤")
	fmt.Printf( "│  Listening on  →  http://localhost:%d  │\n", *port)
	fmt.Printf( "│  Forwarding to →  %s\n", originURL)
	fmt.Println("│  Cache TTL     →  5 minutes              │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println()

	log.SetFlags(log.Ltime) // show only time in log output

	if err := http.ListenAndServe(addr, p); err != nil {
		fmt.Fprintf(os.Stderr, "✖ Server error: %v\n", err)
		os.Exit(1)
	}
}

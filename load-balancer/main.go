package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	mu           sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

type LoadBalancer struct {
	backends []*Backend
	current  uint64
	mu       sync.RWMutex
}

// check if backend is alive
func (b *Backend) IsAlive() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Alive
}

// mark backend up/down
func (b *Backend) SetAlive(alive bool) {
	b.mu.Lock()
	b.Alive = alive
	b.mu.Unlock()
}

// add a backend server
func (lb *LoadBalancer) AddBackend(backendURL string) error {
	parsedURL, err := url.Parse(backendURL)
	if err != nil {
		return fmt.Errorf("invalid backend url %q: %w", backendURL, err)
	}

	backend := &Backend{
		URL:   parsedURL,
		Alive: true,
	}

	proxy := httputil.NewSingleHostReverseProxy(parsedURL)

	// if request fails, mark backend as down
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("backend %s is unavailable: %v", backend.URL, err)
		backend.SetAlive(false)
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}

	backend.ReverseProxy = proxy

	lb.mu.Lock()
	lb.backends = append(lb.backends, backend)
	lb.mu.Unlock()

	return nil
}

// get safe copy of backends
func (lb *LoadBalancer) snapshotBackends() []*Backend {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	out := make([]*Backend, len(lb.backends))
	copy(out, lb.backends)
	return out
}

// simple round robin
func (lb *LoadBalancer) GetNextBackend() *Backend {
	backends := lb.snapshotBackends()
	if len(backends) == 0 {
		return nil
	}

	start := (atomic.AddUint64(&lb.current, 1) - 1) % uint64(len(backends))

	for i := 0; i < len(backends); i++ {
		idx := (int(start) + i) % len(backends)
		b := backends[idx]
		if b.IsAlive() {
			return b
		}
	}

	return nil
}

// background health check
func (lb *LoadBalancer) HealthCheck() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	client := &http.Client{Timeout: 5 * time.Second}

	for range ticker.C {
		backends := lb.snapshotBackends()

		for _, backend := range backends {
			go func(b *Backend) {
				resp, err := client.Get(b.URL.String())

				if resp != nil {
					defer resp.Body.Close()
				}

				healthy := err == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300

				if !healthy {
					if b.IsAlive() {
						log.Printf("backend %s marked down", b.URL)
						b.SetAlive(false)
					}
					return
				}

				if !b.IsAlive() {
					log.Printf("backend %s marked up", b.URL)
					b.SetAlive(true)
				}
			}(backend)
		}
	}
}

// main handler
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend := lb.GetNextBackend()
	if backend == nil {
		http.Error(w, "no available backends", http.StatusServiceUnavailable)
		return
	}

	log.Printf("%s %s -> %s", r.Method, r.RequestURI, backend.URL)
	backend.ReverseProxy.ServeHTTP(w, r)
}

func main() {
	listenAddr := flag.String("listen", ":3000", "address to listen on")
	flag.Parse()

	lb := &LoadBalancer{}

	// sample backends
	backends := []string{
		"http://localhost:8001",
		"http://localhost:8002",
		"http://localhost:8003",
	}

	for _, backendURL := range backends {
		if err := lb.AddBackend(backendURL); err != nil {
			log.Printf("failed to add backend %s: %v", backendURL, err)
		}
	}

	if len(lb.snapshotBackends()) == 0 {
		log.Fatal("no backends configured")
	}

	log.Printf("starting load balancer on %s", *listenAddr)

	go lb.HealthCheck()

	server := &http.Server{
		Addr:              *listenAddr,
		Handler:           lb,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

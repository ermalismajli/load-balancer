package balancer

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// LoadBalancer represents the load balancer structure
type LoadBalancer struct {
	backends       []*Backend
	mutex          sync.RWMutex
	roundRobinCount uint64
	logger         *log.Logger
}

// Backend represents an individual backend server
type Backend struct {
	URL          *url.URL
	Proxy        *httputil.ReverseProxy
	IsAdmin      bool
	IsAlive      bool
	mutex        sync.RWMutex
	failCount    int
	RequestCount uint64
}

// NewLoadBalancer creates a new load balancer instance
func NewLoadBalancer(backendURLs []string, logger *log.Logger) *LoadBalancer {
	backends := make([]*Backend, len(backendURLs))
	for i, backendURL := range backendURLs {
		parsedURL, err := url.Parse(backendURL)
		if err != nil {
			logger.Fatal(err)
		}
		
		proxy := httputil.NewSingleHostReverseProxy(parsedURL)

		// Create logging transport for each backend
		originalDirector := proxy.Director
		backendIndex := i // Capture the backend index
		
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			logger.Printf("Request directed to backend %d: %s %s\n", 
				backendIndex+1, req.Method, req.Host)
		}
		
		// Set up custom error handling
		proxy.ErrorHandler = func(resp http.ResponseWriter, req *http.Request, err error) {
			logger.Printf("Backend %d error: %v\n", backendIndex+1, err)
			resp.WriteHeader(http.StatusBadGateway)
			resp.Write([]byte(fmt.Sprintf("Backend server %d is not available", backendIndex+1)))
		}
		
		// First server is the only one that can handle admin requests
		isAdmin := i == 0
		
		backends[i] = &Backend{
			URL:      parsedURL,
			Proxy:    proxy,
			IsAdmin:  isAdmin,
			IsAlive:  true,
		}
	}
	
	return &LoadBalancer{
		backends: backends,
		logger:   logger,
	}
}

// ServeHTTP handles the http requests
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract and validate JWT token
	role, err := ValidateJWT(r.Header.Get("Authorization"))
	if err != nil {
		lb.logger.Printf("JWT Validation error: %v\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Invalid or missing JWT token"))
		return
	}

	// Get appropriate backend based on role and round-robin
	backend := lb.getBackendForRequest(role)
	if backend == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("No available backend servers"))
		return
	}

	// Track the request count
	atomic.AddUint64(&backend.RequestCount, 1)
	
	// Forward the request
	backend.Proxy.ServeHTTP(w, r)
}

// getBackendForRequest returns the backend server based on the role and round-robin algorithm
func (lb *LoadBalancer) getBackendForRequest(role string) *Backend {
	// For Admin roles, always route to the first backend if it's available
	if role == "Admin" {
		lb.mutex.RLock()
		adminBackend := lb.backends[0]
		lb.mutex.RUnlock()
		
		if adminBackend.IsAlive {
			lb.logger.Printf("Admin request routed to dedicated admin backend (Backend 1)")
			return adminBackend
		}
		// If admin backend is down, we could fail the request or try other backends
		// For this implementation, we'll fail the request
		lb.logger.Printf("Admin request failed - admin backend is down")
		return nil
	}
	
	// For User and Client roles, use round-robin
	// Get the next backend index in a thread-safe manner
	nextIndex := int(atomic.AddUint64(&lb.roundRobinCount, 1) % uint64(len(lb.backends)))
	
	// Try the selected backend and then others in sequence if it's not available
	lb.mutex.RLock()
	backends := lb.backends
	lb.mutex.RUnlock()
	
	// Try up to the number of backends we have
	for i := 0; i < len(backends); i++ {
		idx := (nextIndex + i) % len(backends)
		backend := backends[idx]
		
		backend.mutex.RLock()
		isAlive := backend.IsAlive
		backend.mutex.RUnlock()
		
		if isAlive {
			lb.logger.Printf("%s request routed to Backend %d via round-robin", 
				role, idx+1)
			return backend
		}
	}
	
	// No available backends
	return nil
}

// HealthCheck periodically checks if backends are alive
func (lb *LoadBalancer) HealthCheck(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for range ticker.C {
		for i, backend := range lb.backends {
			status := "up"
			client := &http.Client{
				Timeout: 5 * time.Second,
			}
			
			resp, err := client.Get(backend.URL.String() + "/health")
			if err != nil || resp.StatusCode != http.StatusOK {
				// Mark backend as down if it fails health check
				backend.mutex.Lock()
				backend.IsAlive = false
				backend.failCount++
				backend.mutex.Unlock()
				status = "down"
			} else {
				// Mark backend as up
				backend.mutex.Lock()
				backend.IsAlive = true
				backend.failCount = 0
				backend.mutex.Unlock()
			}
			lb.logger.Printf("Backend %d health check: %s", i+1, status)
		}
	}
}

// GetStats returns statistics about the backends
func (lb *LoadBalancer) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	backends := make([]map[string]interface{}, len(lb.backends))
	
	lb.mutex.RLock()
	for i, backend := range lb.backends {
		backend.mutex.RLock()
		backends[i] = map[string]interface{}{
			"url":          backend.URL.String(),
			"isAdmin":      backend.IsAdmin,
			"isAlive":      backend.IsAlive,
			"failCount":    backend.failCount,
			"requestCount": backend.RequestCount,
		}
		backend.mutex.RUnlock()
	}
	lb.mutex.RUnlock()
	
	stats["backends"] = backends
	stats["totalRequests"] = lb.roundRobinCount
	
	return stats
}


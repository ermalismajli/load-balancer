package test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"log"

	"loadBalancer/balancer"
)

func TestLoadBalancer(t *testing.T) {
	// Create log files for each component
	lbLogFile, err := os.Create("loadbalancer.log")
	if err != nil {
		t.Fatalf("Failed to create load balancer log file: %v", err)
	}
	defer lbLogFile.Close()

	backend1LogFile, err := os.Create("backend1.log")
	if err != nil {
		t.Fatalf("Failed to create backend1 log file: %v", err)
	}
	defer backend1LogFile.Close()

	backend2LogFile, err := os.Create("backend2.log")
	if err != nil {
		t.Fatalf("Failed to create backend2 log file: %v", err)
	}
	defer backend2LogFile.Close()

	backend3LogFile, err := os.Create("backend3.log")
	if err != nil {
		t.Fatalf("Failed to create backend3 log file: %v", err)
	}
	defer backend3LogFile.Close()

	// Create loggers for each component
	lbLogger := log.New(lbLogFile, "LOADBALANCER: ", log.LstdFlags)
	backend1Logger := log.New(backend1LogFile, "BACKEND-1: ", log.LstdFlags)
	backend2Logger := log.New(backend2LogFile, "BACKEND-2: ", log.LstdFlags)
	backend3Logger := log.New(backend3LogFile, "BACKEND-3: ", log.LstdFlags)

	// Setup test backends with their own loggers
	backend1 := httptest.NewServer(createBackendHandler(1, backend1Logger))
	defer backend1.Close()
	backend2 := httptest.NewServer(createBackendHandler(2, backend2Logger))
	defer backend2.Close()
	backend3 := httptest.NewServer(createBackendHandler(3, backend3Logger))
	defer backend3.Close()

	// Create the load balancer with its own logger
	lb := balancer.NewLoadBalancer(
		[]string{backend1.URL, backend2.URL, backend3.URL},
		lbLogger,
	)

	// Start the load balancer server
	lbServer := httptest.NewServer(lb)
	defer lbServer.Close()

	// Variables to track requests per backend
	var adminToBackend1, userToBackend1, userToBackend2, userToBackend3, clientToBackend1, clientToBackend2, clientToBackend3 int
	var mu sync.Mutex // Mutex to protect counters during concurrent access
	
	// Log test start
	lbLogger.Printf("Starting load balancer test with 200 requests")
	
	// Run 200 concurrent requests
	var wg sync.WaitGroup
	wg.Add(200)

	// Make 200 requests with different token types
	for i := 0; i < 200; i++ {
		go func(i int) {
			defer wg.Done()
			
			// Determine role - mix of User, Client, Admin
			var role string
			switch i % 10 {
			case 0, 1:
				role = "Admin" // 20% Admin
			case 2, 3, 4, 5:
				role = "User" // 40% User
			default:
				role = "Client" // 40% Client
			}
			
			// Generate token for this request
			token, err := balancer.GenerateJWT(role)
			if err != nil {
				t.Errorf("Error generating token: %v", err)
				return
			}
			
			// Create an HTTP client with a timeout
			client := &http.Client{
				Timeout: 5 * time.Second,
			}
			
			// Create a real HTTP request to the load balancer
			req, err := http.NewRequest("GET", lbServer.URL, nil)
			if err != nil {
				t.Errorf("Error creating request: %v", err)
				return
			}
			req.Header.Add("Authorization", "Bearer "+token)
			
			// Log the outgoing request
			lbLogger.Printf("Client sending request #%d with role %s", i, role)
			
			// Send the request to the load balancer
			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("Error sending request: %v", err)
				return
			}
			defer resp.Body.Close()
			
			// Check response status
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Request %d failed with status %d", i, resp.StatusCode)
				return
			}
			
			// Read the response body to determine which backend handled it
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("Error reading response: %v", err)
				return
			}
			
			// The response body contains "Response from Backend X"
			responseStr := string(body)
			var backendID int
			
			if responseStr == "Response from Backend 1" {
				backendID = 1
			} else if responseStr == "Response from Backend 2" {
				backendID = 2
			} else if responseStr == "Response from Backend 3" {
				backendID = 3
			} else {
				t.Errorf("Unexpected response body: %s", responseStr)
				return
			}
			
			// Log the response
			lbLogger.Printf("Client received response for request #%d from Backend %d", i, backendID)
			
			// Update counters with mutex protection
			mu.Lock()
			defer mu.Unlock()
			
			if role == "Admin" {
				if backendID == 1 {
					adminToBackend1++
				} else {
					t.Errorf("Admin request went to backend %d instead of backend 1", backendID)
				}
			} else if role == "User" {
				switch backendID {
				case 1:
					userToBackend1++
				case 2:
					userToBackend2++
				case 3:
					userToBackend3++
				}
			} else if role == "Client" {
				switch backendID {
				case 1:
					clientToBackend1++
				case 2:
					clientToBackend2++
				case 3:
					clientToBackend3++
				}
			}
		}(i)
	}

	// Wait for all requests to complete
	wg.Wait()

	// Log the final distribution
	lbLogger.Printf("Test completed. Request distribution:")
	lbLogger.Printf("Admin requests to Backend 1: %d", adminToBackend1)
	lbLogger.Printf("User requests: Backend 1=%d, Backend 2=%d, Backend 3=%d", 
		userToBackend1, userToBackend2, userToBackend3)
	lbLogger.Printf("Client requests: Backend 1=%d, Backend 2=%d, Backend 3=%d",
		clientToBackend1, clientToBackend2, clientToBackend3)
	
	// Print distribution of requests to console as well
	fmt.Printf("Admin requests to Backend 1: %d\n", adminToBackend1)
	fmt.Printf("User requests: Backend 1=%d, Backend 2=%d, Backend 3=%d\n", 
		userToBackend1, userToBackend2, userToBackend3)
	fmt.Printf("Client requests: Backend 1=%d, Backend 2=%d, Backend 3=%d\n",
		clientToBackend1, clientToBackend2, clientToBackend3)
	
	// Verify admin requests only went to Backend 1
	if adminToBackend1 == 0 {
		t.Error("No admin requests were routed to Backend 1")
	}
	
	// Verify round-robin is working by checking if each backend received requests
	if userToBackend1 == 0 || userToBackend2 == 0 || userToBackend3 == 0 {
		t.Error("Round-robin failed for User requests")
	}
	
	if clientToBackend1 == 0 || clientToBackend2 == 0 || clientToBackend3 == 0 {
		t.Error("Round-robin failed for Client requests")
	}
}

// Creates a test backend handler that reports which backend it is
func createBackendHandler(backendID int, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Log the request details
		logger.Printf("Received request: %s %s", r.Method, r.URL.Path)
		logger.Printf("Headers: %v", r.Header)
		
		// Log JWT token if present
		if auth := r.Header.Get("Authorization"); auth != "" {
			logger.Printf("Authorization header: %s", auth)
		}
		
		// Send a simple response that identifies which backend handled the request
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Response from Backend %d", backendID)
		
		// Log the response
		logger.Printf("Sent response: 'Response from Backend %d'", backendID)
	}
}


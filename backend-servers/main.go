package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	// Command line flags for configuration
	port := flag.String("port", "8081", "Port to run the backend server on")
	backendID := flag.Int("id", 1, "Backend server ID (1, 2, or 3)")
	logFile := flag.String("log", "", "Path to log file (empty for stdout)")
	flag.Parse()
	
	// Setup logger
	var logger *log.Logger
	if *logFile != "" {
		file, err := os.Create(*logFile)
		if err != nil {
			log.Fatalf("Failed to create log file: %v", err)
		}
		defer file.Close()
		logger = log.New(file, fmt.Sprintf("backend%d: ", *backendID), log.LstdFlags)
	} else {
		logger = log.New(os.Stdout, fmt.Sprintf("backend%d: ", *backendID), log.LstdFlags)
	}
	
	// Create a simple router
	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("Health check received")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Backend %d is healthy", *backendID)
	})
	
	// Default handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("Received request: %s %s", r.Method, r.URL.Path)
		logger.Printf("Headers: %v", r.Header)
		
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Response from Backend %d\n", *backendID)
		fmt.Fprintf(w, "Path: %s\n", r.URL.Path)
		fmt.Fprintf(w, "Method: %s\n", r.Method)
		fmt.Fprintf(w, "Headers:\n")
		
		// Print request headers
		for name, values := range r.Header {
			for _, value := range values {
				fmt.Fprintf(w, "%s: %s\n", name, value)
			}
		}
		
		logger.Printf("Response sent for %s %s", r.Method, r.URL.Path)
	})
	
	// Admin endpoints only available on backend 1
	if *backendID == 1 {
		mux.HandleFunc("/admin/", func(w http.ResponseWriter, r *http.Request) {
			logger.Printf("Received admin request: %s %s", r.Method, r.URL.Path)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Admin endpoint on Backend 1\n")
			fmt.Fprintf(w, "Path: %s\n", r.URL.Path)
			logger.Printf("Admin response sent for %s %s", r.Method, r.URL.Path)
		})
	}
	
	// Create and start the server
	server := &http.Server{
		Addr:    ":" + *port,
		Handler: mux,
	}
	
	logger.Printf("Starting backend %d on port %s\n", *backendID, *port)
	if err := server.ListenAndServe(); err != nil {
		logger.Fatalf("Could not start server: %v\n", err)
	}
}


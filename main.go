package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"loadBalancer/balancer"
)

func main() {
	// Command line flags for configuration
	port := flag.String("port", "8080", "Port to run the load balancer on")
	backend1 := flag.String("backend1", "http://localhost:8081", "URL of backend server 1")
	backend2 := flag.String("backend2", "http://localhost:8082", "URL of backend server 2")
	backend3 := flag.String("backend3", "http://localhost:8083", "URL of backend server 3")
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
		logger = log.New(file, "loadbalancer: ", log.LstdFlags)
	} else {
		logger = log.New(os.Stdout, "loadbalancer: ", log.LstdFlags)
	}

	// Create load balancer
	lb := balancer.NewLoadBalancer([]string{*backend1, *backend2, *backend3}, logger)

	// Start health check in a goroutine
	go lb.HealthCheck(10 * time.Second)

	// Setup server
	server := &http.Server{
		Addr:    ":" + *port,
		Handler: lb,
	}

	// Start server in a goroutine
	go func() {
		logger.Printf("Starting load balancer on port %s\n", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Could not start server: %v\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	logger.Println("Shutting down server...")
	
	logger.Println("Server stopped")
}


# Custom Load Balancer with JWT-Based Routing

This project implements a custom HTTP load balancer in Go that distributes incoming requests across multiple backend servers using the round-robin algorithm, with special handling based on JWT role claims.

### Features

- Round-robin load balancing across 3 backend servers
- JWT validation and role-based routing
- Special handling for admin requests (always routed to backend 1)
- Health check monitoring of backend servers
- Detailed request logging
- Fallback handling when backends are down

### Building the Application

# Build the load balancer
go build -o loadbalancer ./main.go

# Build the backend servers
go build -o backend ./backend-servers/main.go

### Running the Load Balancer

# Start the backend servers

./backend -port 8081 -id 1 -log backend1.log & 
./backend -port 8082 -id 2 -log backend2.log & 
./backend -port 8083 -id 3 -log backend3.log &

# Start the load balancer
./loadbalancer -port 8080 \
  -backend1 http://localhost:8081 \
  -backend2 http://localhost:8082 \
  -backend3 http://localhost:8083 \
  -log loadbalancer.log

### Testing

The load balancer includes a comprehensive test suite that verifies the JWT-based routing logic and round-robin distribution.

## Running the Tests

### Use the provided script to run tests with log capture

chmod +x run_test.sh
./run_test.sh

### Or run tests directly
go test -v ./test/

The test will:

1. Create separate log files for the load balancer and each backend
2. Send 200 requests with different JWT roles (Admin, User, Client)
3. Verify correct distribution according to JWT-based routing rules
4. Output distribution statistics and log details

### Sample Log Output

Admin requests to Backend 1: 40
User requests: Backend 1=27, Backend 2=26, Backend 3=27
Client requests: Backend 1=28, Backend 2=26, Backend 3=26

## Monitoring

Each component logs detailed information about requests and responses:

- **Load Balancer**: Logs incoming requests, JWT validation, routing decisions
- **Backends**: Log received requests, headers, and response details


These logs are written to separate files for easy monitoring and debugging.

### Prerequisites

- Go 1.18 or higher


#!/bin/bash

# Clean up any existing log files
rm -f loadbalancer.log backend1.log backend2.log backend3.log

# Run the test
echo "Running load balancer test..."
go test -v ./test/

# Display log file statistics
echo ""
echo "Log file statistics:"
echo "-------------------"
echo "Load balancer log: $(wc -l loadbalancer.log)"
echo "Backend 1 log: $(wc -l backend1.log)"
echo "Backend 2 log: $(wc -l backend2.log)"
echo "Backend 3 log: $(wc -l backend3.log)"

echo ""
echo "You can view the detailed logs in the following files:"
echo "- loadbalancer.log"
echo "- backend1.log"
echo "- backend2.log"
echo "- backend3.log"


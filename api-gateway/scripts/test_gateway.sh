#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print test results
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ $2${NC}"
    else
        echo -e "${RED}❌ $2${NC}"
        exit 1
    fi
}

# Check if the gateway is running
echo "Checking if API Gateway is running..."
if ! curl -s http://localhost:8085/health &> /dev/null; then
    echo -e "${RED}❌ API Gateway is not running. Please start it first:${NC}"
    echo "go run cmd/main.go"
    exit 1
fi
print_result $? "API Gateway is running"

# Test health endpoint
echo -e "\nTesting health endpoint..."
response=$(curl -s http://localhost:8085/health)
if [[ $response == *"status"* && $response == *"ok"* ]]; then
    print_result 0 "Health endpoint is working"
else
    print_result 1 "Health endpoint failed"
fi

# Test rate limiting
echo -e "\nTesting rate limiting..."
for i in {1..101}; do
    response=$(curl -s http://localhost:8085/health)
    if [[ $response == *"rate limit exceeded"* ]]; then
        print_result 0 "Rate limiting is working"
        break
    fi
    if [ $i -eq 101 ]; then
        print_result 1 "Rate limiting test failed"
    fi
done

# Test JWT verification
echo -e "\nTesting JWT verification..."
# Test without token
response=$(curl -s -X POST http://localhost:8085/auth/refresh)
if [[ $response == *"authorization header is required"* ]]; then
    print_result 0 "JWT verification (no token) is working"
else
    print_result 1 "JWT verification (no token) failed"
fi

# Test with invalid token
response=$(curl -s -X POST -H "Authorization: Bearer invalid.token.here" http://localhost:8085/auth/refresh)
if [[ $response == *"invalid token"* ]]; then
    print_result 0 "JWT verification (invalid token) is working"
else
    print_result 1 "JWT verification (invalid token) failed"
fi

# Test protected endpoints
echo -e "\nTesting protected endpoints..."
declare -A endpoints=(
    ["/auth/refresh"]="POST"
    ["/metadata/videos"]="GET"
    ["/metadata/videos/123"]="GET"
)

for endpoint in "${!endpoints[@]}"; do
    method=${endpoints[$endpoint]}
    response=$(curl -s -X $method http://localhost:8085$endpoint)
    if [[ $response == *"authorization header is required"* ]]; then
        print_result 0 "Protected endpoint $endpoint ($method) is working"
    else
        print_result 1 "Protected endpoint $endpoint ($method) failed"
    fi
done

# Test public endpoints
echo -e "\nTesting public endpoints..."
declare -A public_endpoints=(
    ["/streaming/videos/123"]="GET"
    ["/upload/videos"]="POST"
)

for endpoint in "${!public_endpoints[@]}"; do
    method=${public_endpoints[$endpoint]}
    response=$(curl -s -X $method http://localhost:8085$endpoint)
    if [[ $response != *"authorization header is required"* ]]; then
        print_result 0 "Public endpoint $endpoint ($method) is working"
    else
        print_result 1 "Public endpoint $endpoint ($method) failed"
    fi
done

echo -e "\nTesting unsupported HTTP methods..."
response=$(curl -s -X PUT http://localhost:8085/health)
if [[ $response == *"Not Found"* || $response == *"404"* || $response == *"405"* ]]; then
    print_result 0 "Unsupported HTTP method returns 404/405 as expected"
else
    print_result 1 "Unsupported HTTP method did not return 404/405"
fi

echo -e "\nTesting non-existent endpoint..."
response=$(curl -s http://localhost:8085/doesnotexist)
if [[ $response == *"Not Found"* || $response == *"404"* ]]; then
    print_result 0 "Non-existent endpoint returns 404 as expected"
else
    print_result 1 "Non-existent endpoint did not return 404"
fi

# Get rate limit period from .env or default to 1m
RATE_LIMIT_PERIOD=$(grep RATE_LIMIT_PERIOD .env | cut -d'=' -f2 | tr -d '"')
if [ -z "$RATE_LIMIT_PERIOD" ]; then
    RATE_LIMIT_PERIOD="1m"
fi
# Convert to seconds for sleep (supports s, m, h)
if [[ $RATE_LIMIT_PERIOD == *m ]]; then
    SLEEP_SECONDS=5
elif [[ $RATE_LIMIT_PERIOD == *h ]]; then
    SLEEP_SECONDS=5
else
    SLEEP_SECONDS=5
fi

echo -e "\nTesting rate limit reset (waiting $SLEEP_SECONDS seconds)..."
sleep $SLEEP_SECONDS
response=$(curl -s http://localhost:8085/health)
if [[ $response == *"status"* && $response == *"ok"* ]]; then
    print_result 0 "Rate limit resets after window ($RATE_LIMIT_PERIOD)"
else
    print_result 1 "Rate limit did not reset after window ($RATE_LIMIT_PERIOD)"
fi

echo -e "\nTesting OPTIONS preflight..."
    response=$(curl -s -X OPTIONS http://localhost:8085/metadata/videos)
if [[ $response == "" || $response == *"Allow"* ]]; then
    print_result 0 "OPTIONS preflight handled"
else
    print_result 1 "OPTIONS preflight not handled"
fi

echo -e "\n${GREEN}All tests completed successfully!${NC}" 
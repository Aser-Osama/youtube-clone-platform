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

# Check if token is provided
if [ -z "$1" ]; then
    echo -e "${RED}❌ Please provide a JWT token as an argument${NC}"
    echo "Usage: $0 <jwt_token>"
    exit 1
fi

TOKEN=$1

# Check if the gateway is running
echo "Checking if API Gateway is running..."
if ! curl -s http://localhost:8085/health &> /dev/null; then
    echo -e "${RED}❌ API Gateway is not running. Please start it first:${NC}"
    echo "go run cmd/main.go"
    exit 1
fi
print_result $? "API Gateway is running"

# Test auth refresh endpoint
echo -e "\nTesting auth refresh endpoint..."
response=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8085/auth/refresh)
if [[ $response == *"proxy not implemented"* ]]; then
    print_result 0 "Auth refresh endpoint is working (proxy not implemented)"
else
    print_result 1 "Auth refresh endpoint failed"
fi

# Test metadata endpoints
echo -e "\nTesting metadata endpoints..."
response=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8085/metadata/videos)
if [[ $response == *"proxy not implemented"* ]]; then
    print_result 0 "Metadata list endpoint is working (proxy not implemented)"
else
    print_result 1 "Metadata list endpoint failed"
fi

response=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8085/metadata/videos/123)
if [[ $response == *"proxy not implemented"* ]]; then
    print_result 0 "Metadata get endpoint is working (proxy not implemented)"
else
    print_result 1 "Metadata get endpoint failed"
fi

# Test streaming endpoint
echo -e "\nTesting streaming endpoint..."
response=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8085/streaming/videos/123)
if [[ $response == *"proxy not implemented"* ]]; then
    print_result 0 "Streaming endpoint is working (proxy not implemented)"
else
    print_result 1 "Streaming endpoint failed"
fi

# Test upload endpoint
echo -e "\nTesting upload endpoint..."
response=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8085/upload/videos)
if [[ $response == *"proxy not implemented"* ]]; then
    print_result 0 "Upload endpoint is working (proxy not implemented)"
else
    print_result 1 "Upload endpoint failed"
fi

echo -e "\nTesting POST to upload endpoint..."
response=$(curl -s -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8085/upload/videos)
if [[ $response == *"proxy not implemented"* ]]; then
    print_result 0 "POST /upload/videos is protected and routed"
else
    print_result 1 "POST /upload/videos failed"
fi

echo -e "\nTesting GET with extra headers..."
response=$(curl -s -H "Authorization: Bearer $TOKEN" -H "X-Test-Header: test" http://localhost:8085/metadata/videos)
if [[ $response == *"proxy not implemented"* ]]; then
    print_result 0 "GET with extra headers works"
else
    print_result 1 "GET with extra headers failed"
fi

echo -e "\nTesting malformed Authorization header..."
response=$(curl -s -H "Authorization: Bearer" http://localhost:8085/auth/refresh)
if [[ $response == *"invalid authorization header format"* ]]; then
    print_result 0 "Malformed Authorization header is handled"
else
    print_result 1 "Malformed Authorization header not handled"
fi

echo -e "\nTesting OPTIONS preflight..."
response=$(curl -s -X OPTIONS -H "Authorization: Bearer $TOKEN" http://localhost:8085/metadata/videos)
if [[ $response == "" || $response == *"Allow"* ]]; then
    print_result 0 "OPTIONS preflight handled"
else
    print_result 1 "OPTIONS preflight not handled"
fi

echo -e "\n${GREEN}All tests completed successfully!${NC}"
echo -e "\nNote: All endpoints return 'proxy not implemented' as the proxy functionality is not yet implemented." 
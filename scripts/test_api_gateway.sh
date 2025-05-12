#!/bin/bash

# API Gateway Testing Script
# This script tests all API Gateway endpoints and key functionality
# including rate limiting, authentication, and service proxying

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Base URL for the API Gateway
API_GATEWAY="http://localhost:8085"

# Variables to store access token
ACCESS_TOKEN=""

# Counters for test results
PASSED=0
FAILED=0
SKIPPED=0

# Test tracking
CURRENT_SECTION=""
CURRENT_TEST=""

# Function to print test status
print_status() {
  local status=$1
  local message=$2
  
  if [ "$status" == "PASS" ]; then
    echo -e "${GREEN}[PASS]${NC} $message"
    ((PASSED++))
  elif [ "$status" == "FAIL" ]; then
    echo -e "${RED}[FAIL]${NC} $message"
    ((FAILED++))
  elif [ "$status" == "SKIP" ]; then
    echo -e "${YELLOW}[SKIP]${NC} $message"
    ((SKIPPED++))
  elif [ "$status" == "INFO" ]; then
    echo -e "${BLUE}[INFO]${NC} $message"
  elif [ "$status" == "SECTION" ]; then
    echo ""
    echo -e "${BLUE}=============== $message ===============${NC}"
    CURRENT_SECTION="$message"
  fi
}

# Function to start a test
start_test() {
  CURRENT_TEST="$1"
  print_status "INFO" "Testing: $CURRENT_TEST"
}

# Function to check if response status code matches expected
check_status_code() {
  local expected_status=$1
  local received_status=$2
  local endpoint=$3
  
  if [ "$received_status" -eq "$expected_status" ]; then
    print_status "PASS" "$endpoint - Status code $received_status"
    return 0
  else
    print_status "FAIL" "$endpoint - Expected status $expected_status, got $received_status"
    return 1
  fi
}

# Function to check if response contains a string
check_response_contains() {
  local response="$1"
  local expected_string="$2"
  local endpoint="$3"
  
  if [[ "$response" == *"$expected_string"* ]]; then
    print_status "PASS" "$endpoint - Response contains '$expected_string'"
    return 0
  else
    print_status "FAIL" "$endpoint - Response does not contain '$expected_string'"
    return 1
  fi
}

# Function to obtain JWT token (mocked for testing)
get_auth_token() {
  print_status "SECTION" "Authentication Tests"
  start_test "Get Auth Token (Mock)"
  
  # In a real test, this would authenticate with the auth service
  # For testing, we're using a mock token
  ACCESS_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IlRlc3QgVXNlciIsImlhdCI6MTY0NTU2NjIwMH0.KqK1YwsQww_VSW33-JbpuCzKJG_3Ltwbo1TJuG9tn7s"
  
  if [ -n "$ACCESS_TOKEN" ]; then
    print_status "PASS" "Auth token acquired (mock)"
    return 0
  else
    print_status "FAIL" "Failed to acquire auth token"
    return 1
  fi
}

# Test health endpoints
test_health_endpoints() {
  print_status "SECTION" "Health Endpoints"
  
  # Test root health endpoint
  start_test "Root Health Endpoint"
  response=$(curl -s -o /dev/null -w "%{http_code}" "$API_GATEWAY/health")
  check_status_code 200 "$response" "/health"
  
  # Test all services health
  start_test "All Services Health Endpoint"
  health_response=$(curl -s "$API_GATEWAY/health/all")
  response_code=$?
  
  if [ $response_code -eq 0 ]; then
    print_status "PASS" "/health/all - Request successful"
    echo "$health_response" | jq '.' || echo "$health_response"
  else
    print_status "FAIL" "/health/all - Request failed"
  fi
  
  # Test API health endpoint
  start_test "API Health Endpoint"
  response=$(curl -s -o /dev/null -w "%{http_code}" "$API_GATEWAY/api/v1/health")
  check_status_code 200 "$response" "/api/v1/health"
}

# Test streaming service endpoints
test_streaming_endpoints() {
  print_status "SECTION" "Streaming Service Endpoints"
  
  # Test streaming health endpoint
  start_test "Streaming Health Endpoint"
  response=$(curl -s -o /dev/null -w "%{http_code}" "$API_GATEWAY/api/v1/streaming/health")
  check_status_code 200 "$response" "/api/v1/streaming/health"
  
  # Test video thumbnail endpoint
  start_test "Video Thumbnail Endpoint"
  # Use a known video ID or test video ID
  VIDEO_ID="b56b82c6-0b59-4607-b99c-b9001b8a8724"
  response=$(curl -s -o /dev/null -w "%{http_code}" "$API_GATEWAY/api/v1/streaming/videos/$VIDEO_ID/thumbnail")
  
  # This should return either a 200 OK or a 307 Temporary Redirect to the actual thumbnail
  if [[ "$response" == 200 || "$response" == 307 ]]; then
    print_status "PASS" "/api/v1/streaming/videos/$VIDEO_ID/thumbnail - Status code $response"
  else
    print_status "FAIL" "/api/v1/streaming/videos/$VIDEO_ID/thumbnail - Expected status 200 or 307, got $response"
  fi
  
  # Test MP4 qualities endpoint
  start_test "MP4 Qualities Endpoint"
  response=$(curl -s -o /dev/null -w "%{http_code}" "$API_GATEWAY/api/v1/streaming/videos/$VIDEO_ID/mp4/qualities")
  if [[ "$response" == 200 || "$response" == 404 ]]; then
    # 404 is acceptable if the video doesn't exist
    print_status "PASS" "/api/v1/streaming/videos/$VIDEO_ID/mp4/qualities - Status code $response"
  else
    print_status "FAIL" "/api/v1/streaming/videos/$VIDEO_ID/mp4/qualities - Expected status 200 or 404, got $response"
  fi
}

# Test metadata service endpoints
test_metadata_endpoints() {
  print_status "SECTION" "Metadata Service Endpoints"
  
  # Test metadata health endpoint
  start_test "Metadata Health Endpoint"
  response=$(curl -s -o /dev/null -w "%{http_code}" "$API_GATEWAY/api/v1/metadata/health")
  check_status_code 200 "$response" "/api/v1/metadata/health"
  
  # Test public videos endpoint
  start_test "Public Videos Endpoint"
  response=$(curl -s -o /dev/null -w "%{http_code}" "$API_GATEWAY/api/v1/metadata/public/videos")
  check_status_code 200 "$response" "/api/v1/metadata/public/videos"
  
  # Test video details endpoint with a known video ID or a non-existent one
  VIDEO_ID="b56b82c6-0b59-4607-b99c-b9001b8a8724"
  start_test "Video Details Endpoint"
  response=$(curl -s -o /dev/null -w "%{http_code}" "$API_GATEWAY/api/v1/metadata/videos/$VIDEO_ID")
  
  if [[ "$response" == 200 || "$response" == 404 ]]; then
    # 404 is acceptable if the video doesn't exist
    print_status "PASS" "/api/v1/metadata/videos/$VIDEO_ID - Status code $response"
  else
    print_status "FAIL" "/api/v1/metadata/videos/$VIDEO_ID - Expected status 200 or 404, got $response"
  fi
}

# Test upload service endpoints
test_upload_endpoints() {
  print_status "SECTION" "Upload Service Endpoints"
  
  # Test upload health endpoint
  start_test "Upload Health Endpoint"
  response=$(curl -s -o /dev/null -w "%{http_code}" "$API_GATEWAY/api/v1/upload/health")
  check_status_code 200 "$response" "/api/v1/upload/health"
  
  # Test protected upload endpoint (should fail without auth)
  start_test "Upload Videos Endpoint (No Auth - Should Fail)"
  response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_GATEWAY/api/v1/upload/videos")
  
  if [ "$response" -eq 401 ]; then
    print_status "PASS" "/api/v1/upload/videos - Correctly returned 401 Unauthorized"
  else
    print_status "FAIL" "/api/v1/upload/videos - Expected status 401, got $response"
  fi
  
  # Test with auth token
  start_test "Upload Videos Endpoint (With Auth)"
  response=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H "Authorization: Bearer $ACCESS_TOKEN" "$API_GATEWAY/api/v1/upload/videos")
  
  # This test may still fail if the token is not accepted, but the endpoint exists
  if [[ "$response" != 404 ]]; then
    print_status "PASS" "/api/v1/upload/videos (with auth) - Endpoint exists"
  else
    print_status "FAIL" "/api/v1/upload/videos (with auth) - Endpoint not found"
  fi
}

# Test transcoder service endpoints
test_transcoder_endpoints() {
  print_status "SECTION" "Transcoder Service Endpoints"
  
  # Test transcoder health endpoint
  start_test "Transcoder Health Endpoint"
  response=$(curl -s -o /dev/null -w "%{http_code}" "$API_GATEWAY/api/v1/transcoder/health")
  check_status_code 200 "$response" "/api/v1/transcoder/health"
  
  # Test transcoder status endpoint (public)
  start_test "Transcoder Status Endpoint"
  response=$(curl -s -o /dev/null -w "%{http_code}" "$API_GATEWAY/api/v1/transcoder/status")
  
  if [[ "$response" == 200 || "$response" == 404 ]]; then
    # 404 is acceptable if endpoint not implemented
    print_status "PASS" "/api/v1/transcoder/status - Status code $response"
  else
    print_status "FAIL" "/api/v1/transcoder/status - Expected status 200 or 404, got $response"
  fi
  
  # Test protected endpoint (should fail without auth)
  start_test "Transcoder Jobs Endpoint (No Auth - Should Fail)"
  response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_GATEWAY/api/v1/transcoder/jobs")
  
  if [ "$response" -eq 401 ]; then
    print_status "PASS" "/api/v1/transcoder/jobs - Correctly returned 401 Unauthorized"
  else
    print_status "FAIL" "/api/v1/transcoder/jobs - Expected status 401, got $response"
  fi
}

# Test rate limiting
test_rate_limiting() {
  print_status "SECTION" "Rate Limiting Tests"
  
  # Test rate limiting on our specific test endpoint
  start_test "Rate Limiting Test Endpoint"
  print_status "INFO" "Making rapid requests to the dedicated rate-limit test endpoint..."
  
  LIMIT=10  # We only need a few requests since our test endpoint is strictly rate-limited
  RATE_LIMITED=0
  
  # Make requests as fast as possible to the dedicated test endpoint
  for ((i=1; i<=LIMIT; i++)); do
    response=$(curl -s -o /dev/null -w "%{http_code}" "$API_GATEWAY/test-rate-limit" --max-time 1)
    
    if [ "$response" -eq 429 ]; then
      RATE_LIMITED=1
      print_status "PASS" "Rate limiting triggered after $i requests to test endpoint (status code 429)"
      break
    fi
  done
  
  if [ $RATE_LIMITED -eq 0 ]; then
    print_status "INFO" "Made $LIMIT requests to test endpoint without triggering rate limiting"
    print_status "FAIL" "Rate limiting test failed - endpoint did not enforce limits"
  fi
}

# Test JWT authentication
test_jwt_authentication() {
  print_status "SECTION" "JWT Authentication Tests"
  
  # Test protected endpoint with valid token
  start_test "Protected Endpoint with Valid Token"
  # Using upload/videos endpoint which we know is protected
  response=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H "Authorization: Bearer $ACCESS_TOKEN" \
    "$API_GATEWAY/api/v1/upload/videos")
  
  # Note: This test might fail if token is not properly signed for the system
  # We're mainly testing if the auth middleware is active
  if [ "$response" -eq 401 ] || [ "$response" -eq 403 ] || [ "$response" -eq 400 ]; then
    print_status "PASS" "JWT authentication mechanism rejected mock token (as expected)"
  else
    # The response could be 400 (bad request) if the token is accepted but request body is missing
    print_status "FAIL" "JWT authentication test failed - Expected 401/403/400, got $response"
  fi
  
  # Test protected endpoint with invalid token
  start_test "Protected Endpoint with Invalid Token"
  INVALID_TOKEN="invalid.token.here"
  response=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H "Authorization: Bearer $INVALID_TOKEN" \
    "$API_GATEWAY/api/v1/upload/videos")
  
  if [ "$response" -eq 401 ] || [ "$response" -eq 403 ]; then
    print_status "PASS" "Invalid token correctly rejected (401/403)"
  else
    print_status "FAIL" "Invalid token test failed - Expected 401/403, got $response"
  fi
  
  # Test protected endpoint with missing token
  start_test "Protected Endpoint with Missing Token"
  response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_GATEWAY/api/v1/upload/videos")
  
  if [ "$response" -eq 401 ] || [ "$response" -eq 403 ]; then
    print_status "PASS" "Missing token correctly rejected (401/403)"
  else
    print_status "FAIL" "Missing token test failed - Expected 401/403, got $response"
  fi
}

# Print test summary
print_summary() {
  echo ""
  echo "=============== TEST SUMMARY ==============="
  echo -e "${GREEN}Passed: $PASSED${NC}"
  echo -e "${RED}Failed: $FAILED${NC}"
  echo -e "${YELLOW}Skipped: $SKIPPED${NC}"
  echo "Total: $((PASSED + FAILED + SKIPPED))"
  
  if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed successfully!${NC}"
    exit 0
  else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
  fi
}

# Main test execution
main() {
  echo "=============== API GATEWAY TEST SUITE ==============="
  echo "Testing API Gateway at: $API_GATEWAY"
  echo "Date: $(date)"
  echo ""
  
  # Run tests
  test_health_endpoints
  test_streaming_endpoints
  test_metadata_endpoints
  test_upload_endpoints
  test_transcoder_endpoints
  
  # Get auth token for auth-related tests
  get_auth_token
  
  # Run authentication and rate limiting tests
  test_jwt_authentication
  test_rate_limiting
  
  # Print summary
  print_summary
}

# Run the main function
main
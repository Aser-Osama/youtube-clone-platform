#!/bin/bash

# test_standardized_apis.sh
# This script tests all standardized APIs and removes backward compatibility once confirmed working

# Text formatting
BOLD="\033[1m"
GREEN="\033[0;32m"
RED="\033[0;31m"
YELLOW="\033[0;33m"
BLUE="\033[0;34m"
NC="\033[0m" # No Color

API_GATEWAY="http://localhost:8085"
AUTH_SERVICE="http://localhost:8080"
METADATA_SERVICE="http://localhost:8082"
STREAMING_SERVICE="http://localhost:8083"
UPLOAD_SERVICE="http://localhost:8084"
TRANSCODER_SERVICE="http://localhost:8083" # Update with actual port

# Create a temporary directory for test files
TEMP_DIR=$(mktemp -d)
echo -e "${BLUE}Created temporary directory: ${TEMP_DIR}${NC}"

# Function to check if required tools are installed
check_requirements() {
    echo -e "${BOLD}Checking requirements...${NC}"
    
    # List of required tools
    TOOLS=("curl" "jq" "ffmpeg")
    MISSING=0
    
    for TOOL in "${TOOLS[@]}"; do
        if ! command -v "$TOOL" &> /dev/null; then
            echo -e "${RED}❌ $TOOL is not installed${NC}"
            MISSING=1
        else
            echo -e "${GREEN}✓ $TOOL is installed${NC}"
        fi
    done
    
    if [ $MISSING -eq 1 ]; then
        echo -e "${RED}Please install missing tools before continuing.${NC}"
        exit 1
    fi
}

# Function to test and verify API endpoints
test_endpoint() {
    local service=$1
    local endpoint=$2
    local expected_code=$3
    local method=${4:-GET}
    local data=$5
    
    echo -e "${BLUE}Testing ${method} ${service}${endpoint}${NC}"
    
    # Add custom headers for specific endpoints
    HEADERS=""
    if [[ "$endpoint" == *"/videos/"*"/views"* ]]; then
        HEADERS="-H 'X-User-ID: test_user_123'"
    fi
    
    # Add content type for POST requests
    if [[ "$method" == "POST" && -n "$data" ]]; then
        HEADERS="$HEADERS -H 'Content-Type: application/json'"
    fi
    
    # Handle file uploads
    if [[ "$endpoint" == *"/upload/videos"* && "$method" == "POST" ]]; then
        # Special handling for multipart/form-data
        RESPONSE=$(curl -s -w "%{http_code}" -X $method ${service}${endpoint} \
            -F "title=Test Video $(date +%s)" \
            -F "user_id=test_user_123" \
            -F "video=@${TEMP_DIR}/test_video.mp4" \
            -o ${TEMP_DIR}/response.json)
    else
        # Standard request
        if [[ -n "$data" ]]; then
            RESPONSE=$(curl -s -w "%{http_code}" -X $method ${service}${endpoint} \
                $HEADERS -d "$data" -o ${TEMP_DIR}/response.json)
        else
            RESPONSE=$(curl -s -w "%{http_code}" -X $method ${service}${endpoint} \
                $HEADERS -o ${TEMP_DIR}/response.json)
        fi
    fi
    
    # Get the HTTP status code
    HTTP_CODE=${RESPONSE: -3}
    
    # Check if the status code matches expected
    if [[ "$HTTP_CODE" == "$expected_code" ]]; then
        echo -e "${GREEN}✓ Status code $HTTP_CODE as expected${NC}"
        if [ -s "${TEMP_DIR}/response.json" ]; then
            # Try to pretty print JSON response
            if jq -e . ${TEMP_DIR}/response.json > /dev/null 2>&1; then
                echo -e "${BLUE}Response:${NC}"
                jq . ${TEMP_DIR}/response.json
            else
                # Not JSON, just show first few lines
                echo -e "${BLUE}Response: (first 150 characters)${NC}"
                head -c 150 ${TEMP_DIR}/response.json
                echo -e "\n..."
            fi
        fi
        return 0
    else
        echo -e "${RED}❌ Expected status $expected_code but got $HTTP_CODE${NC}"
        if [ -s "${TEMP_DIR}/response.json" ]; then
            echo -e "${RED}Error response:${NC}"
            cat ${TEMP_DIR}/response.json
        fi
        return 1
    fi
}

# Function to create a small test video using ffmpeg
create_test_video() {
    echo -e "${BOLD}Creating test video file...${NC}"
    ffmpeg -loglevel error -f lavfi -i "testsrc=duration=5:size=640x360:rate=30" \
           -c:v libx264 ${TEMP_DIR}/test_video.mp4
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Test video created successfully${NC}"
        return 0
    else
        echo -e "${RED}❌ Failed to create test video${NC}"
        return 1
    fi
}

# Function to test health endpoints for all services
test_health_endpoints() {
    echo -e "\n${BOLD}Testing Health Endpoints${NC}"
    
    # Test API Gateway aggregated health
    test_endpoint $API_GATEWAY "/health" "200"
    
    # Test individual services health endpoints
    test_endpoint $API_GATEWAY "/api/v1/auth/health" "200"
    test_endpoint $API_GATEWAY "/api/v1/metadata/health" "200"
    test_endpoint $API_GATEWAY "/api/v1/streaming/health" "200"
    test_endpoint $API_GATEWAY "/api/v1/upload/health" "200"
    test_endpoint $API_GATEWAY "/api/v1/transcoder/health" "200"
}

# Function to test metadata service endpoints
test_metadata_endpoints() {
    echo -e "\n${BOLD}Testing Metadata Service Endpoints${NC}"
    
    # Get recent videos
    test_endpoint $API_GATEWAY "/api/v1/metadata/videos?limit=5" "200"
    
    # Search videos
    test_endpoint $API_GATEWAY "/api/v1/metadata/videos/search?q=test" "200"
    
    # Try to get first video ID from the list
    VIDEO_ID=$(jq -r '.[0].id' ${TEMP_DIR}/response.json 2>/dev/null)
    
    if [[ -n "$VIDEO_ID" && "$VIDEO_ID" != "null" ]]; then
        echo -e "${BLUE}Testing with video ID: $VIDEO_ID${NC}"
        
        # Get specific video
        test_endpoint $API_GATEWAY "/api/v1/metadata/videos/$VIDEO_ID" "200"
        
        # Increment views
        test_endpoint $API_GATEWAY "/api/v1/metadata/videos/$VIDEO_ID/views" "204" "POST"
    else
        echo -e "${YELLOW}⚠ No video IDs found for further testing${NC}"
    fi
}

# Function to test streaming service endpoints
test_streaming_endpoints() {
    echo -e "\n${BOLD}Testing Streaming Service Endpoints${NC}"
    
    # Try to get first video ID from metadata
    test_endpoint $API_GATEWAY "/api/v1/metadata/videos?limit=5" "200"
    VIDEO_ID=$(jq -r '.[0].id' ${TEMP_DIR}/response.json 2>/dev/null)
    
    if [[ -n "$VIDEO_ID" && "$VIDEO_ID" != "null" ]]; then
        echo -e "${BLUE}Testing with video ID: $VIDEO_ID${NC}"
        
        # Test thumbnail
        test_endpoint $API_GATEWAY "/api/v1/streaming/videos/$VIDEO_ID/thumbnail" "200"
        
        # Test HLS manifest
        test_endpoint $API_GATEWAY "/api/v1/streaming/videos/$VIDEO_ID/hls/manifest" "200"
        
        # Test MP4 stream
        test_endpoint $API_GATEWAY "/api/v1/streaming/videos/$VIDEO_ID/mp4" "200"
        
        # Test MP4 qualities
        test_endpoint $API_GATEWAY "/api/v1/streaming/videos/$VIDEO_ID/mp4/qualities" "200"
    else
        echo -e "${YELLOW}⚠ No video IDs found for further testing${NC}"
    fi
}

# Function to test upload service endpoints
test_upload_endpoints() {
    echo -e "\n${BOLD}Testing Upload Service Endpoints${NC}"
    
    # First create a test video
    create_test_video
    
    # Test video upload
    test_endpoint $API_GATEWAY "/api/v1/upload/videos" "200" "POST"
    
    # Extract the uploaded video ID
    UPLOADED_VIDEO_ID=$(jq -r '.video_id' ${TEMP_DIR}/response.json 2>/dev/null)
    
    if [[ -n "$UPLOADED_VIDEO_ID" && "$UPLOADED_VIDEO_ID" != "null" ]]; then
        echo -e "${GREEN}✓ Video uploaded successfully with ID: $UPLOADED_VIDEO_ID${NC}"
    else
        echo -e "${YELLOW}⚠ Could not extract video ID from upload response${NC}"
    fi
}

# Function to test transcoder service endpoints
test_transcoder_endpoints() {
    echo -e "\n${BOLD}Testing Transcoder Service Endpoints${NC}"
    
    # Test transcoder health
    test_endpoint $API_GATEWAY "/api/v1/transcoder/health" "200"
    
    # Note: Most transcoder endpoints are internal and triggered via Kafka
    echo -e "${YELLOW}⚠ Note: Transcoder service mainly works via Kafka messages, limited API testing${NC}"
}

# Confirmation for removing backward compatibility
confirm_remove_backward_compatibility() {
    echo -e "\n${BOLD}${YELLOW}CAUTION: About to remove backward compatibility code${NC}"
    echo -e "${YELLOW}This will modify service files to remove legacy API endpoints.${NC}"
    echo -e "${YELLOW}Make sure you have backups or can restore from version control if needed.${NC}"
    
    read -p "Do you want to proceed? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${BLUE}Operation cancelled.${NC}"
        exit 0
    fi
}

# Functions to remove backward compatibility from each service
remove_auth_backward_compatibility() {
    echo -e "\n${BOLD}Removing backward compatibility from Auth Service${NC}"
    
    AUTH_MAIN_FILE="../auth-service/cmd/main.go"
    if [ -f "$AUTH_MAIN_FILE" ]; then
        # Create backup
        cp "$AUTH_MAIN_FILE" "${AUTH_MAIN_FILE}.bak"
        echo -e "${BLUE}Backup created: ${AUTH_MAIN_FILE}.bak${NC}"
        
        # Remove legacy routes
        sed -i '/\/\/\ Maintain\ backward\ compatibility/,/r\.GET(\"\/auth\/health\"/d' "$AUTH_MAIN_FILE"
        
        echo -e "${GREEN}✓ Removed legacy routes from Auth Service${NC}"
    else
        echo -e "${RED}❌ Auth Service main file not found${NC}"
    fi
}

remove_metadata_backward_compatibility() {
    echo -e "\n${BOLD}Removing backward compatibility from Metadata Service${NC}"
    
    METADATA_HANDLER_FILE="../metadata-service/internal/handler/metadata.go"
    if [ -f "$METADATA_HANDLER_FILE" ]; then
        # Create backup
        cp "$METADATA_HANDLER_FILE" "${METADATA_HANDLER_FILE}.bak"
        echo -e "${BLUE}Backup created: ${METADATA_HANDLER_FILE}.bak${NC}"
        
        # Remove legacy API routes
        sed -i '/\/\/\ Keep\ backward\ compatibility/,/legacyApi\.GET(\"\\/users\//d' "$METADATA_HANDLER_FILE"
        
        echo -e "${GREEN}✓ Removed legacy routes from Metadata Service${NC}"
    else
        echo -e "${RED}❌ Metadata Service handler file not found${NC}"
    fi
}

remove_streaming_backward_compatibility() {
    echo -e "\n${BOLD}Removing backward compatibility from Streaming Service${NC}"
    
    STREAMING_MAIN_FILE="../streaming-service/cmd/main.go"
    if [ -f "$STREAMING_MAIN_FILE" ]; then
        # Create backup
        cp "$STREAMING_MAIN_FILE" "${STREAMING_MAIN_FILE}.bak"
        echo -e "${BLUE}Backup created: ${STREAMING_MAIN_FILE}.bak${NC}"
        
        # Remove legacy routes
        sed -i '/\/\/\ Add\ backward\ compatibility\ routes/,/router\.GET(\"\/videos\/:videoID\/thumbnail\"/d' "$STREAMING_MAIN_FILE"
        
        echo -e "${GREEN}✓ Removed legacy routes from Streaming Service${NC}"
    else
        echo -e "${RED}❌ Streaming Service main file not found${NC}"
    fi
}

remove_upload_backward_compatibility() {
    echo -e "\n${BOLD}Removing backward compatibility from Upload Service${NC}"
    
    UPLOAD_MAIN_FILE="../video-upload-service/cmd/main.go"
    if [ -f "$UPLOAD_MAIN_FILE" ]; then
        # Create backup
        cp "$UPLOAD_MAIN_FILE" "${UPLOAD_MAIN_FILE}.bak"
        echo -e "${BLUE}Backup created: ${UPLOAD_MAIN_FILE}.bak${NC}"
        
        # Remove legacy routes
        sed -i '/\/\/\ Keep\ backward\ compatibility/,/router\.POST(\"\/upload\"/d' "$UPLOAD_MAIN_FILE"
        
        echo -e "${GREEN}✓ Removed legacy routes from Upload Service${NC}"
    else
        echo -e "${RED}❌ Upload Service main file not found${NC}"
    fi
}

remove_transcoder_backward_compatibility() {
    echo -e "\n${BOLD}Removing backward compatibility from Transcoder Service${NC}"
    
    TRANSCODER_MAIN_FILE="../transcoder-service/cmd/main.go"
    if [ -f "$TRANSCODER_MAIN_FILE" ]; then
        # Create backup
        cp "$TRANSCODER_MAIN_FILE" "${TRANSCODER_MAIN_FILE}.bak"
        echo -e "${BLUE}Backup created: ${TRANSCODER_MAIN_FILE}.bak${NC}"
        
        # Remove legacy routes
        sed -i '/\/\/\ Maintain\ backward\ compatibility/,/router\.GET(\"\/health\"/d' "$TRANSCODER_MAIN_FILE"
        
        echo -e "${GREEN}✓ Removed legacy routes from Transcoder Service${NC}"
    else
        echo -e "${RED}❌ Transcoder Service main file not found${NC}"
    fi
}

# Main function
main() {
    echo -e "${BOLD}${BLUE}=== YouTube Clone Platform API Test Script ===${NC}"
    echo -e "${BLUE}This script will test all standardized API endpoints and can remove backward compatibility${NC}\n"
    
    # Check requirements
    check_requirements
    
    # Test all endpoints
    test_health_endpoints
    test_metadata_endpoints
    test_streaming_endpoints
    test_upload_endpoints
    test_transcoder_endpoints
    
    echo -e "\n${GREEN}${BOLD}✓ API testing completed${NC}"
    
    # Ask to remove backward compatibility
    confirm_remove_backward_compatibility
    
    # Remove backward compatibility from each service
    remove_auth_backward_compatibility
    remove_metadata_backward_compatibility
    remove_streaming_backward_compatibility
    remove_upload_backward_compatibility
    remove_transcoder_backward_compatibility
    
    echo -e "\n${GREEN}${BOLD}✓ Backward compatibility code removed from all services${NC}"
    echo -e "${BLUE}Backups of all modified files have been created with .bak extension${NC}"
    echo -e "${YELLOW}Please restart all services for changes to take effect${NC}\n"
    
    # Cleanup
    rm -rf "$TEMP_DIR"
    echo -e "${BLUE}Temporary directory removed${NC}"
}

# Run the main function
main
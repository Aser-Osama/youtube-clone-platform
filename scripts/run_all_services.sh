#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

# Function to print error
print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if a process is running
is_running() {
    pgrep -f "$1" > /dev/null
}

# Function to kill a process if it's running
kill_if_running() {
    if is_running "$1"; then
        print_status "Stopping $1..."
        pkill -f "$1"
        sleep 2
    fi
}

# Function to check if a port is in use
is_port_in_use() {
    local port=$1
    if lsof -i :$port > /dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function to kill process using a port
kill_port() {
    local port=$1
    if is_port_in_use $port; then
        print_status "Port $port is in use. Attempting to free it..."
        sudo lsof -ti :$port | xargs -r sudo kill -9
        sleep 2
    fi
}

# Function to check if Docker containers are running
are_docker_containers_running() {
    if sudo docker compose ps --services --filter "status=running" | grep -q "kafka\|minio"; then
        return 0
    else
        return 1
    fi
}

# Kill all existing services
print_status "Cleaning up existing services..."
kill_if_running "auth-service"
kill_if_running "api-gateway"
kill_if_running "metadata-service"
kill_if_running "streaming-service"
kill_if_running "video-upload-service"
kill_if_running "transcoder-service"

# Free up commonly used ports
print_status "Checking and freeing up ports..."
kill_port 8080  # Auth Service
kill_port 8082  # Metadata Service
kill_port 8083  # Transcoder Service
kill_port 8085  # API Gateway
kill_port 8090  # Streaming Service

# Check if Docker containers are running
if ! are_docker_containers_running; then
    print_status "Starting MinIO and Kafka..."
    sudo docker compose up -d
    
    # Wait for MinIO and Kafka to be ready
    print_status "Waiting for MinIO and Kafka to be ready..."
    sleep 10
else
    print_status "MinIO and Kafka are already running, skipping startup..."
fi

# Function to start a service in a new terminal tab
start_service() {
    local service_name=$1
    local command=$2
    local working_dir=$3

    # Create new terminal tab
    gnome-terminal --tab --title="$service_name" -- bash -c "cd $working_dir && $command; exec bash"
    sleep 2
}

# Start services in order
print_status "Starting services..."

# Start auth-service
start_service "Auth Service" "go run cmd/main.go" "../auth-service"

# Start metadata-service
start_service "Metadata Service" "PORT=8082 go run cmd/main.go" "../metadata-service"

# Start streaming-service
start_service "Streaming Service" "PORT=8090 go run cmd/main.go" "../streaming-service"

# Start upload-service
start_service "Upload Service" "PORT=8081 go run cmd/main.go" "../video-upload-service"

# Start api-gateway
start_service "API Gateway" "PORT=8085 go run cmd/main.go" "../api-gateway"

# Start transcoder-service
start_service "Transcoder Service" "PORT=8083 go run cmd/main.go" "../transcoder-service"

print_status "All services started!"
print_status "You can view each service's output in its respective terminal tab."
print_status "To stop all services, run: ./scripts/stop_all_services.sh"
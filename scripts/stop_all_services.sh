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

# Kill all services
print_status "Stopping all services..."
kill_if_running "auth-service"
kill_if_running "api-gateway"
kill_if_running "metadata-service"
kill_if_running "streaming-service"
kill_if_running "upload-service"

# Free up commonly used ports
print_status "Checking and freeing up ports..."
kill_port 8080  # Auth Service
kill_port 8082  # Metadata Service
kill_port 8083  # Streaming Service
kill_port 8084  # Upload Service
kill_port 8085  # API Gateway
kill_port 9092  # Kafka
kill_port 9000  # MinIO

# Stop Docker containers
print_status "Stopping Docker containers..."
sudo docker compose down

print_status "All services stopped!" 
#!/bin/bash

# Check if video_id is provided
if [ -z "$1" ]; then
    echo "Usage: $0 <video_id>"
    exit 1
fi

VIDEO_ID=$1

# Check if MinIO client is installed
if ! command -v mc &> /dev/null; then
    echo "MinIO client (mc) is not installed. Installing..."
    wget https://dl.min.io/client/mc/release/linux-amd64/mc
    chmod +x mc
    sudo mv mc /usr/local/bin/
fi

# Check if kcat is installed
if ! command -v kcat &> /dev/null; then
    echo "kcat is not installed. Please install it using:"
    echo "sudo dnf install kcat"
    exit 1
fi

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "jq is not installed. Please install it using:"
    echo "sudo dnf install jq"
    exit 1
fi

# Configure MinIO client if not already configured
if ! mc alias ls | grep -q "minio"; then
    echo "Configuring MinIO client..."
    mc alias set minio http://localhost:9000 minioadmin minioadmin
fi

# List objects in MinIO bucket
echo "Checking MinIO for video with ID: $VIDEO_ID"
if mc ls minio/rawvideos/original/$VIDEO_ID &> /dev/null; then
    echo "✅ Video found in MinIO at rawvideos/original/$VIDEO_ID"
    # Show the actual file details
    mc ls minio/rawvideos/original/$VIDEO_ID
else
    echo "❌ Video not found in MinIO at rawvideos/original/$VIDEO_ID"
    echo "Checking other locations..."
    mc ls minio/rawvideos/
fi

# Check if Kafka is running
echo -e "\nChecking Kafka connection..."
if ! kcat -b localhost:29092 -L &> /dev/null; then
    echo "❌ Cannot connect to Kafka. Please ensure Kafka is running:"
    echo "cd .. && docker compose up -d kafka zookeeper"
    exit 1
fi

# Check Kafka topic for the upload event
echo -e "\nChecking Kafka for upload event..."
if kcat -b localhost:29092 -t video-uploads -C -o beginning -e | grep -v "^test$" | jq -r '.video_id' 2>/dev/null | grep -q "^${VIDEO_ID}$"; then
    echo "✅ Upload event found in Kafka"
    echo "Event details:"
    kcat -b localhost:29092 -t video-uploads -C -o beginning -e | grep -v "^test$" | jq 'select(.video_id == "'$VIDEO_ID'")'
else
    echo "❌ Upload event not found in Kafka"
    echo "This might be because:"
    echo "1. The event wasn't published successfully"
    echo "2. The event has already been consumed"
    echo "3. The topic doesn't exist yet"
    
    # Try to create the topic
    echo -e "\nAttempting to create Kafka topic..."
    kcat -b localhost:29092 -t video-uploads -P -e <<< "test"
    if [ $? -eq 0 ]; then
        echo "✅ Successfully created/verified Kafka topic"
        echo -e "\nCurrent messages in topic:"
        kcat -b localhost:29092 -t video-uploads -C -o beginning -e | grep -v "^test$" | jq '.'
    else
        echo "❌ Failed to create Kafka topic"
    fi
fi 
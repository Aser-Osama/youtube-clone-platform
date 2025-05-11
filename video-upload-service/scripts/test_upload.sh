#!/bin/bash

# Check if ffmpeg is installed
if ! command -v ffmpeg &> /dev/null; then
    echo "ffmpeg is not installed. Please install it using:"
    echo "sudo dnf install ffmpeg"
    exit 1
fi

# Create a test video file if it doesn't exist
if [ ! -f "test_video.mp4" ]; then
    echo "Creating test video file..."
    # Create a 1-second test video using ffmpeg
    ffmpeg -f lavfi -i testsrc=duration=100:size=1280x720:rate=30 -c:v libx264 test_video.mp4
    if [ $? -ne 0 ]; then
        echo "❌ Failed to create test video"
        exit 1
    fi
    echo "✅ Test video created successfully"
fi

# Check if the upload service is running
if ! curl -s http://localhost:8080/health &> /dev/null; then
    echo "❌ Upload service is not running. Please start it first:"
    echo "go run cmd/main.go"
    exit 1
fi

# Generate a unique title with timestamp
timestamp=$(date +"%Y%m%d_%H%M%S")
title="Test Video ${timestamp}"

# Use a test user ID that matches a potential Google OAuth ID format
test_user_id="test_user_123456789"

# Upload the video
echo "Uploading test video..."
echo "Title: $title"
echo "User ID: $test_user_id"

response=$(curl -s -X POST http://localhost:8080/upload \
    -F "title=$title" \
    -F "user_id=$test_user_id" \
    -F "video=@test_video.mp4")

# Extract video_id from response
video_id=$(echo $response | grep -o '"video_id":"[^"]*' | cut -d'"' -f4)

if [ -z "$video_id" ]; then
    echo "❌ Upload failed. Response:"
    echo $response
    exit 1
fi

echo "✅ Upload successful!"
echo "Video ID: $video_id"
echo "Title: $title"
echo "User ID: $test_user_id"
echo -e "\nTo verify the upload, run:"
echo "./scripts/verify_upload.sh $video_id" 
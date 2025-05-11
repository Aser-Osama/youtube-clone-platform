#!/bin/bash

# Check if ffmpeg is installed
if ! command -v ffmpeg &> /dev/null; then
    echo "ffmpeg is not installed. Please install it using:"
    echo "sudo dnf install ffmpeg"
    exit 1
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

# Check if MinIO client is installed
if ! command -v mc &> /dev/null; then
    echo "MinIO client (mc) is not installed. Installing..."
    wget https://dl.min.io/client/mc/release/linux-amd64/mc
    chmod +x mc
    sudo mv mc /usr/local/bin/
fi

# Configure MinIO client if not already configured
if ! mc alias ls | grep -q "minio"; then
    echo "Configuring MinIO client..."
    mc alias set minio http://localhost:9000 minioadmin minioadmin
fi

# Create test videos of different resolutions
echo "Creating test videos of different resolutions..."

# 4K video
ffmpeg -f lavfi -i testsrc=duration=5:size=3840x2160:rate=30 -c:v libx264 test_video_4k.mp4
if [ $? -ne 0 ]; then
    echo "❌ Failed to create 4K test video"
    exit 1
fi

# 1080p video
ffmpeg -f lavfi -i testsrc=duration=5:size=1920x1080:rate=30 -c:v libx264 test_video_1080p.mp4
if [ $? -ne 0 ]; then
    echo "❌ Failed to create 1080p test video"
    exit 1
fi

# 720p video
ffmpeg -f lavfi -i testsrc=duration=5:size=1280x720:rate=30 -c:v libx264 test_video_720p.mp4
if [ $? -ne 0 ]; then
    echo "❌ Failed to create 720p test video"
    exit 1
fi

# 480p video
ffmpeg -f lavfi -i testsrc=duration=5:size=854x480:rate=30 -c:v libx264 test_video_480p.mp4
if [ $? -ne 0 ]; then
    echo "❌ Failed to create 480p test video"
    exit 1
fi

echo "✅ Test videos created successfully"

# Function to test a video
test_video() {
    local input_file=$1
    local resolution=$2
    local width=$3
    local height=$4

    echo -e "\nTesting $resolution video..."
    echo "Input file: $input_file"
    echo "Resolution: ${width}x${height}"

    # Generate a unique video ID
    VIDEO_ID=$(uuidgen)
    USER_ID="test_user_123456789"
    TITLE="Test Video $resolution $(date +"%Y%m%d_%H%M%S")"

    echo "Video ID: $VIDEO_ID"
    echo "User ID: $USER_ID"
    echo "Title: $TITLE"

    # Upload the video to MinIO
    echo -e "\nUploading video to MinIO..."
    mc cp $input_file minio/videos/original/$VIDEO_ID.mp4
    if [ $? -ne 0 ]; then
        echo "❌ Failed to upload video to MinIO"
        return 1
    fi
    echo "✅ Video uploaded to MinIO"

    # Create and publish a video upload event to Kafka
    echo -e "\nPublishing video upload event to Kafka..."
    EVENT=$(cat <<EOF
{
    "video_id": "$VIDEO_ID",
    "user_id": "$USER_ID",
    "title": "$TITLE",
    "content_type": "video/mp4",
    "size": $(stat -f %z $input_file),
    "metadata": {
        "duration": 5.0,
        "width": $width,
        "height": $height,
        "format": "mp4",
        "bitrate": 2000000,
        "file_size": $(stat -f %z $input_file),
        "checksum": "$(md5sum $input_file | cut -d' ' -f1)",
        "created_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
        "codec": "h264",
        "frame_rate": 30.0,
        "aspect_ratio": "16:9",
        "audio_codec": "aac",
        "audio_bitrate": 128000,
        "audio_channels": 2,
        "content_type": "video/mp4",
        "original_filename": "$input_file",
        "file_extension": ".mp4",
        "sanitized_filename": "$input_file"
    },
    "uploaded_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
)

    echo $EVENT | kcat -b localhost:29092 -t video-uploads -P
    if [ $? -ne 0 ]; then
        echo "❌ Failed to publish event to Kafka"
        return 1
    fi
    echo "✅ Event published to Kafka"

    # Wait for transcoding to complete
    echo -e "\nWaiting for transcoding to complete..."
    echo "This may take a few minutes..."

    # Poll for the transcoded files in MinIO
    MAX_ATTEMPTS=30
    ATTEMPT=0
    while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
        if mc ls minio/videos/hls/$VIDEO_ID &> /dev/null && mc ls minio/videos/mp4/$VIDEO_ID &> /dev/null; then
            echo "✅ Transcoding completed!"
            break
        fi
        ATTEMPT=$((ATTEMPT + 1))
        echo "Waiting... (attempt $ATTEMPT/$MAX_ATTEMPTS)"
        sleep 10
    done

    if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
        echo "❌ Transcoding did not complete in time"
        return 1
    fi

    # Check for transcoded files
    echo -e "\nChecking transcoded files..."
    echo "HLS files:"
    mc ls minio/videos/hls/$VIDEO_ID

    echo -e "\nMP4 files:"
    mc ls minio/videos/mp4/$VIDEO_ID

    echo -e "\nThumbnail:"
    mc ls minio/videos/thumbnails/$VIDEO_ID

    # Check for transcoding completion event
    echo -e "\nChecking for transcoding completion event..."
    kcat -b localhost:29092 -t transcoding-complete -C -o beginning -e | jq 'select(.video_id == "'$VIDEO_ID'")'

    echo -e "\n✅ $resolution video test completed successfully!"
    return 0
}

# Test each resolution
test_video "test_video_4k.mp4" "4K" 3840 2160
test_video "test_video_1080p.mp4" "1080p" 1920 1080
test_video "test_video_720p.mp4" "720p" 1280 720
test_video "test_video_480p.mp4" "480p" 854 480

# Clean up test files
echo -e "\nCleaning up test files..."
rm test_video_*.mp4

echo -e "\n✅ All tests completed successfully!" 
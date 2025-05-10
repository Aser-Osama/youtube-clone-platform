#!/bin/bash

VIDEO_FILE="/home/aser/coding/arch/Uni_Youtube_Clone/youtube-clone-platform/youtube-clone-platform/video-upload-service/scripts/youtube_9hhMUT2U2L4_854x480_h264.mp4"
if [ ! -f "$VIDEO_FILE" ]; then
    echo "❌ Error: Video file $VIDEO_FILE not found"
    exit 1
fi

FILE_SIZE=$(stat -f%z "$VIDEO_FILE" 2>/dev/null || stat -c%s "$VIDEO_FILE" 2>/dev/null)
echo "File size: $(numfmt --to=iec-i --suffix=B $FILE_SIZE)"

echo -e "\nVideo information:"
ffprobe -v error -select_streams v:0 \
    -show_entries stream=width,height,r_frame_rate,codec_name:format=duration,size,bit_rate \
    -of json "$VIDEO_FILE"

if [ $? -ne 0 ]; then
    echo "❌ ffprobe failed — check the video file format"
    exit 1
fi

echo -e "\nUploading custom video..."
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
TITLE="Custom Video Upload $TIMESTAMP"
USER_ID="test_user_123456789"

echo "Title: $TITLE"
echo "User ID: $USER_ID"

echo -e "\nSending request to upload service..."
RESPONSE=$(curl -s -f -X POST \
    -F "video=@$VIDEO_FILE" \
    -F "title=$TITLE" \
    -F "user_id=$USER_ID" \
    http://localhost:8080/upload)

if [ $? -ne 0 ]; then
    echo "❌ Upload failed: bad HTTP response"
    exit 1
fi

if echo "$RESPONSE" | jq . >/dev/null 2>&1; then
    echo -e "\nParsed response:"
    echo "$RESPONSE" | jq .

    if echo "$RESPONSE" | jq -e .video_id >/dev/null 2>&1; then
        echo "✅ Upload successful!"
        VIDEO_ID=$(echo "$RESPONSE" | jq -r .video_id)
        echo -e "\nTo verify the upload, run:"
        echo "./scripts/verify_upload.sh $VIDEO_ID"
    else
        echo "❌ Upload failed!"
        echo "Error details:"
        echo "$RESPONSE" | jq .
    fi
else
    echo "❌ Upload failed!"
    echo "Error: Invalid JSON response"
    echo "Raw response:"
    echo "$RESPONSE"
fi

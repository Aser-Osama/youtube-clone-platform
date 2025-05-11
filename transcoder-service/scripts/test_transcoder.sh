#!/bin/bash

# Check if ffmpeg is installed
if ! command -v ffmpeg &> /dev/null; then
    echo "ffmpeg is not installed. Please install it using:"
    echo "sudo dnf install ffmpeg"
    exit 1
fi

# Check if the transcoder service is running
if ! curl -s http://localhost:8080/health &> /dev/null; then
    echo "❌ Transcoder service is not running. Please start it first:"
    echo "go run cmd/server/main.go"
    exit 1
fi

# Check if MinIO is running
if ! curl -s http://localhost:9000/minio/health/live &> /dev/null; then
    echo "❌ MinIO is not running. Please start it first:"
    echo "docker compose up -d minio"
    exit 1
fi

# Check if Kafka is running
if ! kcat -b localhost:29092 -L &> /dev/null; then
    echo "❌ Kafka is not running. Please start it first:"
    echo "docker compose up -d kafka zookeeper"
    exit 1
fi

# Check health endpoint
echo "Checking transcoder service health..."
health_response=$(curl -s http://localhost:8080/health)
if [ $? -ne 0 ]; then
    echo "❌ Failed to get health status"
    exit 1
fi

echo "✅ Health check response:"
echo $health_response | jq '.'

# Check if all required services are healthy
if echo $health_response | jq -e '.services.minio.status == "healthy" and .services.kafka.status == "healthy" and .services.ffmpeg.status == "healthy"' > /dev/null; then
    echo "✅ All services are healthy"
else
    echo "❌ Some services are not healthy"
    echo "Please check the service logs for more information"
    exit 1
fi

echo -e "\nTo test the full transcoding pipeline, run:"
echo "./scripts/test_transcoding_pipeline.sh" 
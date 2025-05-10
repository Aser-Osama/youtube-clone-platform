# Video Upload Service

A microservice responsible for handling user video uploads, storing them in MinIO, and publishing upload events to Kafka for further processing.

## Features

- Accepts video uploads via HTTP
- Validates video files (size, format, etc.)
- Extracts metadata using ffprobe
- Stores videos in MinIO object storage
- Publishes upload events to Kafka

## Configuration

The service uses environment variables for configuration:

| Variable         | Description                   | Default         |
| ---------------- | ----------------------------- | --------------- |
| PORT             | HTTP port to listen on        | 8080            |
| MAX_BYTES        | Maximum upload size in bytes  | 1GB             |
| MINIO_ENDPOINT   | MinIO server address          | localhost:9000  |
| MINIO_ACCESS_KEY | MinIO access key              | minioadmin      |
| MINIO_SECRET_KEY | MinIO secret key              | minioadmin      |
| MINIO_USE_SSL    | Use SSL for MinIO             | false           |
| MINIO_BUCKET     | MinIO bucket name             | rawvideos       |
| KAFKA_BROKERS    | Kafka broker addresses        | localhost:29092 |
| KAFKA_TOPIC      | Kafka topic for upload events | video-uploads   |

## API Endpoints

### Upload Video

```
POST /upload
```

Form parameters:

- `title`: Video title (required)
- `user_id`: User ID (required)
- `video`: Video file (required)

Response:

```json
{
  "video_id": "uuid",
  "title": "Video Title",
  "user_id": "user_123",
  "uploaded_at": "2025-05-09T19:48:13Z",
  "metadata": {
    "duration": 243.4,
    "width": 854,
    "height": 480,
    "format": "mp4"
    // other metadata...
  }
}
```

### Health Check

```
GET /health
```

Response:

```json
{
  "status": "ok",
  "dependencies": {
    "minio": "ok",
    "kafka": "ok"
  }
}
```

## Development

### Testing

Use the provided scripts:

- `./scripts/test_upload.sh` - Creates and uploads a test video
- `./scripts/test_custom_upload.sh` - Uploads an existing video file
- `./scripts/verify_upload.sh <video_id>` - Verifies a video was uploaded correctly

### Dependencies

- ffprobe (for metadata extraction)
- MinIO (for storage)
- Kafka (for event publishing)

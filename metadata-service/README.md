# Metadata Service

The Metadata Service is responsible for storing and managing video metadata in the YouTube clone platform. It provides endpoints for retrieving video information and handles video upload events from Kafka.

## Features

- Stores video metadata in SQLite database
- Consumes video upload events from Kafka
- Provides REST API endpoints for video metadata
- Tracks video views
- Integrates with MinIO for video storage

## API Endpoints

### GET /api/v1/videos/:id

Retrieves metadata for a specific video.

**Request Example:**

```bash
curl -X GET http://localhost:8082/api/v1/videos/12345
```

**Response Example:**

```json
{
  "id": "12345",
  "title": "Sample Video",
  "description": "A sample video description",
  "views": 100,
  "status": "ready"
}
```

### GET /api/v1/videos

Retrieves a list of recent videos. Supports pagination with the `limit` query parameter.

**Request Example:**

```bash
curl -X GET "http://localhost:8082/api/v1/videos?limit=10"
```

**Response Example:**

```json
[
  {
    "id": "12345",
    "title": "Sample Video",
    "views": 100
  },
  {
    "id": "67890",
    "title": "Another Video",
    "views": 50
  }
]
```

### POST /api/v1/videos/:id/views

Increments the view count for a video. Requires `X-User-ID` header.

**Request Example:**

```bash
curl -X POST http://localhost:8082/api/v1/videos/12345/views -H "X-User-ID: user123"
```

**Response Example:**

```json
{
  "message": "View count incremented."
}
```

## Configuration

The service can be configured using a `.env` file:

**.env Example:**

```env
SERVER_PORT=8082

DATABASE_PATH=./data/metadata.db

KAFKA_BROKERS=localhost:29092
KAFKA_TOPICS_VIDEO_UPLOAD=video-uploads
KAFKA_GROUP_ID=metadata-service

MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_USE_SSL=false
MINIO_BUCKET=videos
```

## Development

1. Install dependencies:

   ```bash
   go mod download
   ```

2. Run the service:

   ```bash
   go run cmd/main.go
   ```

3. Build the Docker image:

   ```bash
   docker build -t metadata-service .
   ```

4. Set up the database:

   ```bash
   sqlite3 ./data/metadata.db < internal/db/schema.sql
   ```

5. Run tests:

   ```bash
   go test ./...
   ```

## Database Schema

The service uses SQLite with the following schema:

- `videos`: Stores video metadata
- `video_views`: Tracks video views per user

See `internal/db/schema.sql` for the complete schema definition.

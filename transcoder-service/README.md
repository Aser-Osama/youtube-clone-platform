# Transcoder Service

The Transcoder Service is a microservice that handles video transcoding for the YouTube Clone Platform. It listens for video upload events from Kafka, downloads videos from MinIO, transcodes them to HLS format using FFmpeg, and uploads the transcoded files back to MinIO.

## Features

- Listens for video upload events from Kafka
- Downloads videos from MinIO
- Transcodes videos to HLS format with multiple quality levels
- Generates thumbnails
- Uploads transcoded files to MinIO
- Publishes transcoding completion events to Kafka
- Supports concurrent transcoding jobs
- Configurable transcoding parameters
- Health check endpoint

## Architecture

The Transcoder Service follows a clean architecture pattern with the following components:

- **Config**: Loads and validates configuration from environment variables and .env file using Viper
- **Storage**: Handles interactions with MinIO for downloading and uploading files
- **Transcoder**: Wraps FFmpeg for video transcoding operations
- **Events**: Handles Kafka events for consuming upload events and producing completion events
- **Service**: Coordinates the transcoding process
- **Handler**: HTTP handlers for health checks and other endpoints

## Prerequisites

- Go 1.21 or higher
- FFmpeg
- MinIO
- Kafka

## Configuration

The service can be configured using environment variables or a `.env` file. The following variables are available:

| Variable                  | Description                                   | Default              |
| ------------------------- | --------------------------------------------- | -------------------- |
| `PORT`                    | HTTP server port                              | 8080                 |
| `KAFKA_BROKERS`           | Kafka broker addresses                        | localhost:9092       |
| `KAFKA_TOPIC`             | Kafka topic for video upload events           | video-uploads        |
| `KAFKA_GROUP_ID`          | Kafka consumer group ID                       | transcoder-service   |
| `MINIO_ENDPOINT`          | MinIO endpoint                                | localhost:9000       |
| `MINIO_ACCESS_KEY`        | MinIO access key                              | minioadmin           |
| `MINIO_SECRET_KEY`        | MinIO secret key                              | minioadmin           |
| `MINIO_USE_SSL`           | Whether to use SSL for MinIO                  | false                |
| `MINIO_BUCKET`            | MinIO bucket name                             | videos               |
| `MINIO_ORIGINAL_PREFIX`   | MinIO prefix for original videos              | original             |
| `MINIO_HLS_PREFIX`        | MinIO prefix for HLS files                    | hls                  |
| `MINIO_THUMBNAIL_PREFIX`  | MinIO prefix for thumbnails                   | thumbnails           |
| `FFMPEG_PATH`             | Path to FFmpeg executable                     | ffmpeg               |
| `FFMPEG_THREADS`          | Number of threads to use for FFmpeg           | 4                    |
| `FFMPEG_PRESET`           | FFmpeg preset                                 | medium               |
| `FFMPEG_CRF`              | FFmpeg CRF value                              | 23                   |
| `FFMPEG_SEGMENT_LENGTH`   | HLS segment length in seconds                 | 10                   |
| `FFMPEG_OUTPUT_FORMATS`   | Output formats                                | h264                 |
| `FFMPEG_OUTPUT_QUALITIES` | Output qualities                              | 1080p,720p,480p,360p |
| `MAX_CONCURRENT_JOBS`     | Maximum number of concurrent transcoding jobs | 2                    |
| `JOB_TIMEOUT`             | Timeout for transcoding jobs                  | 30m                  |
| `TEMP_DIR`                | Directory for temporary files                 | /tmp/transcoder      |

## Building

```bash
# Build the service
go build -o transcoder-service ./cmd/server
```

## Running

```bash
# Run the service
./transcoder-service
```

## API

The service exposes the following HTTP endpoints:

- `GET /health`: Health check endpoint that returns the status of the service and its dependencies

The service also communicates with other services through Kafka events.

## Events

### Consumed Events

The service consumes the following events:

- `VideoUploadEvent`: Triggered when a video is uploaded

```json
{
  "video_id": "string",
  "user_id": "string",
  "title": "string",
  "content_type": "string",
  "size": 0,
  "metadata": {
    "duration": 0,
    "width": 0,
    "height": 0,
    "format": "string",
    "bitrate": 0,
    "file_size": 0,
    "checksum": "string",
    "created_at": "string",
    "codec": "string",
    "frame_rate": 0,
    "aspect_ratio": "string",
    "audio_codec": "string",
    "audio_bitrate": 0,
    "audio_channels": 0,
    "content_type": "string",
    "original_filename": "string",
    "file_extension": "string",
    "sanitized_filename": "string"
  },
  "uploaded_at": "string"
}
```

### Produced Events

The service produces the following events:

- `TranscodingCompleteEvent`: Triggered when a video is transcoded

```json
{
  "video_id": "string",
  "user_id": "string",
  "title": "string",
  "hls_path": "string",
  "thumbnail_path": "string",
  "status": "string",
  "completed_at": "string"
}
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

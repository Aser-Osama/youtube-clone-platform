# API Documentation

This document provides comprehensive documentation for the standardized APIs of the YouTube Clone Platform.

## Table of Contents

- [API Gateway](#api-gateway)
- [Auth Service](#auth-service)
- [Metadata Service](#metadata-service)
- [Streaming Service](#streaming-service)
- [Upload Service](#upload-service)
- [Transcoder Service](#transcoder-service)

## API Gateway

The API Gateway serves as the entry point for all client requests, routing them to the appropriate microservices. All requests should be made to the gateway rather than directly to individual services.

**Base URL**: `http://localhost:8085`

### Health Check

- **GET** `/health`
  - Returns the health status of all services
  - Response: `200 OK` if all services are healthy, `503 Service Unavailable` otherwise
  - Example response:
    ```json
    {
      "status": true,
      "services": {
        "gateway": { "status": "healthy" },
        "auth": { "status": "healthy" },
        "metadata": { "status": "healthy" },
        "streaming": { "status": "healthy" },
        "upload": { "status": "healthy" },
        "transcoder": { "status": "healthy" }
      }
    }
    ```

## Auth Service

Handles user authentication and session management.

**Base Path**: `/api/v1/auth`

### Endpoints

#### OAuth Authentication

- **GET** `/api/v1/auth/google/login`

  - Redirects user to Google OAuth page
  - No request body required

- **GET** `/api/v1/auth/google/callback`
  - Callback endpoint for Google OAuth
  - Returns: JWT tokens upon successful authentication
  - Example response:
    ```json
    {
      "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
      "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
      "expires_in": 3600
    }
    ```

#### Token Management

- **POST** `/api/v1/auth/refresh`

  - Refreshes an access token using a valid refresh token
  - Request body:
    ```json
    {
      "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
    }
    ```
  - Response: New token pair
    ```json
    {
      "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
      "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
      "expires_in": 3600
    }
    ```

- **POST** `/api/v1/auth/logout`
  - Logs out a user by revoking their refresh token
  - Request body:
    ```json
    {
      "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
    }
    ```
  - Response: `200 OK` if successful

#### Health Check

- **GET** `/api/v1/auth/health`
  - Returns the health status of the auth service
  - Response: `200 OK` if healthy
    ```json
    {
      "status": "ok",
      "time": "2025-05-11T19:45:28Z"
    }
    ```

## Metadata Service

Manages video metadata including titles, descriptions, views, and other information.

**Base Path**: `/api/v1/metadata`

### Endpoints

#### Video Metadata

- **GET** `/api/v1/metadata/videos/:id`

  - Gets metadata for a specific video
  - URL Parameters:
    - `id`: Video ID
  - Response: Video metadata object
    ```json
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "My Video Title",
      "description": "Video description here",
      "user_id": "user123",
      "views": 42,
      "duration": 180.5,
      "created_at": "2025-05-11T18:30:00Z",
      "updated_at": "2025-05-11T18:30:00Z",
      "thumbnail_url": "http://example.com/thumbnail.jpg",
      "status": "ready"
    }
    ```

- **GET** `/api/v1/metadata/videos`

  - Gets a list of recent videos
  - Query Parameters:
    - `limit` (optional): Maximum number of videos to return (default: 10)
  - Response: Array of video metadata objects

- **POST** `/api/v1/metadata/videos/:id/views`

  - Increments the view count for a video
  - URL Parameters:
    - `id`: Video ID
  - Headers:
    - `X-User-ID`: ID of the user viewing the video
  - Response: `204 No Content` if successful

- **GET** `/api/v1/metadata/videos/search`

  - Searches for videos by query
  - Query Parameters:
    - `q`: Search query
    - `limit` (optional): Maximum number of results (default: 10)
  - Response: Array of matching video metadata objects

- **GET** `/api/v1/metadata/users/:id/videos`
  - Gets videos uploaded by a specific user
  - URL Parameters:
    - `id`: User ID
  - Query Parameters:
    - `limit` (optional): Maximum number of videos to return (default: 10)
  - Response: Array of video metadata objects

#### Health Check

- **GET** `/api/v1/metadata/health`
  - Returns the health status of the metadata service
  - Response: `200 OK` if healthy
    ```json
    {
      "status": "ok"
    }
    ```

## Streaming Service

Handles video streaming in different formats and resolutions.

**Base Path**: `/api/v1/streaming`

### Endpoints

#### Video Streaming

- **GET** `/api/v1/streaming/videos/:videoID/hls/manifest`

  - Gets the HLS master playlist for a video
  - URL Parameters:
    - `videoID`: Video ID
  - Response: HLS manifest content (m3u8)

- **GET** `/api/v1/streaming/videos/:videoID/hls/:resolution/playlist`

  - Gets the HLS playlist for a specific resolution
  - URL Parameters:
    - `videoID`: Video ID
    - `resolution`: Video resolution (e.g., "720p")
  - Response: HLS playlist content (m3u8)

- **GET** `/api/v1/streaming/videos/:videoID/hls/:resolution/:segment`

  - Gets an HLS segment file
  - URL Parameters:
    - `videoID`: Video ID
    - `resolution`: Video resolution
    - `segment`: Segment filename
  - Response: TS file content

- **GET** `/api/v1/streaming/videos/:videoID/mp4`

  - Gets an MP4 version of the video
  - URL Parameters:
    - `videoID`: Video ID
  - Query Parameters:
    - `quality` (optional): Desired quality (e.g., "720p")
  - Response: MP4 file or redirect to storage URL

- **GET** `/api/v1/streaming/videos/:videoID/mp4/qualities`

  - Lists available MP4 qualities for a video
  - URL Parameters:
    - `videoID`: Video ID
  - Response: List of available qualities
    ```json
    {
      "qualities": ["1080p", "720p", "480p", "360p"]
    }
    ```

- **GET** `/api/v1/streaming/videos/:videoID/thumbnail`
  - Gets the thumbnail for a video
  - URL Parameters:
    - `videoID`: Video ID
  - Response: Image file or redirect to storage URL

#### Health Check

- **GET** `/api/v1/streaming/health`
  - Returns the health status of the streaming service
  - Response: `200 OK` if healthy
    ```json
    {
      "status": "ok"
    }
    ```

## Upload Service

Handles video uploads and initial processing.

**Base Path**: `/api/v1/upload`

### Endpoints

#### Video Upload

- **POST** `/api/v1/upload/videos`

  - Uploads a new video
  - Request: `multipart/form-data`
    - `title`: Video title
    - `user_id`: ID of the uploading user
    - `video`: Video file
  - Response: Upload status and video ID
    ```json
    {
      "status": "success",
      "message": "Video uploaded successfully",
      "video_id": "550e8400-e29b-41d4-a716-446655440000"
    }
    ```

- **POST** `/api/v1/upload/videos/process`
  - Triggers processing for an already uploaded video (protected endpoint)
  - Request body:
    ```json
    {
      "video_id": "550e8400-e29b-41d4-a716-446655440000"
    }
    ```
  - Response: Process status
    ```json
    {
      "status": "processing",
      "message": "Video processing initiated"
    }
    ```

#### Health Check

- **GET** `/api/v1/upload/health`
  - Returns the health status of the upload service
  - Response: `200 OK` if healthy
    ```json
    {
      "status": "ok",
      "services": {
        "minio": "healthy",
        "kafka": "healthy"
      }
    }
    ```

## Transcoder Service

Transcodes videos into multiple formats and qualities for streaming.

**Base Path**: `/api/v1/transcoder`

### Endpoints

#### Transcoding Jobs

- **POST** `/api/v1/transcoder/jobs`

  - Creates a new transcoding job (protected endpoint)
  - Request body:
    ```json
    {
      "video_id": "550e8400-e29b-41d4-a716-446655440000",
      "source_path": "videos/original/550e8400-e29b-41d4-a716-446655440000.mp4",
      "output_formats": ["hls", "mp4"],
      "output_qualities": ["1080p", "720p", "480p", "360p"],
      "callback_url": "http://example.com/callback"
    }
    ```
  - Response: Job status
    ```json
    {
      "job_id": "job-123456",
      "status": "queued",
      "video_id": "550e8400-e29b-41d4-a716-446655440000"
    }
    ```

- **GET** `/api/v1/transcoder/jobs/:id`
  - Gets the status of a transcoding job
  - URL Parameters:
    - `id`: Job ID
  - Response: Job status and details
    ```json
    {
      "job_id": "job-123456",
      "status": "in_progress",
      "video_id": "550e8400-e29b-41d4-a716-446655440000",
      "progress": 65,
      "created_at": "2025-05-11T18:30:00Z",
      "updated_at": "2025-05-11T18:35:00Z"
    }
    ```

#### Health Check

- **GET** `/api/v1/transcoder/health`
  - Returns the health status of the transcoder service
  - Response: `200 OK` if healthy
    ```json
    {
      "status": "ok",
      "services": {
        "minio": {
          "status": "healthy"
        },
        "kafka": {
          "status": "healthy"
        },
        "ffmpeg": {
          "status": "healthy"
        }
      }
    }
    ```

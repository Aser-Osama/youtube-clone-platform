package storage

import (
	"context"
	"fmt"
	"io"
	"path"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage interface {
	UploadVideo(ctx context.Context, reader io.Reader, size int64, contentType string, originalFilename string) (string, error)
}

type MinIOStorage struct {
	client     *minio.Client
	bucketName string
}

func NewMinIOStorage(endpoint, accessKey, secretKey string, useSSL bool, bucketName string) (*MinIOStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &MinIOStorage{
		client:     client,
		bucketName: bucketName,
	}, nil
}

func (s *MinIOStorage) UploadVideo(ctx context.Context, reader io.Reader, size int64, contentType string, originalFilename string) (string, error) {
	// Generate a unique ID for the video
	videoID := uuid.New().String()

	// Extract and preserve extension from original filename
	fileExt := path.Ext(originalFilename)
	if fileExt == "" {
		// Default to .mp4 if no extension is found
		fileExt = ".mp4"
	}

	// Combine the ID with the original extension
	objectName := path.Join("original", videoID+fileExt)

	// Ensure we're using video/mp4 instead of application/octet-stream for better compatibility
	if contentType == "" || contentType == "application/octet-stream" {
		if fileExt == ".mp4" {
			contentType = "video/mp4"
		} else if fileExt == ".webm" {
			contentType = "video/webm"
		} else if fileExt == ".mov" || fileExt == ".qt" {
			contentType = "video/quicktime"
		} else {
			contentType = "video/mp4" // Default to MP4 for unknown extensions
		}
	}

	// Include original filename in metadata
	metadata := map[string]string{
		"original-filename": originalFilename,
	}

	// Upload the video to MinIO
	_, err := s.client.PutObject(ctx, s.bucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType:  contentType,
		UserMetadata: metadata,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload video: %w", err)
	}

	// Return just the videoID without extension for compatibility with existing code
	return videoID, nil
}

// CheckHealth verifies the MinIO connection is working
func (s *MinIOStorage) CheckHealth(ctx context.Context) error {
	// Check if the bucket exists as a simple health check
	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("bucket '%s' does not exist", s.bucketName)
	}
	return nil
}

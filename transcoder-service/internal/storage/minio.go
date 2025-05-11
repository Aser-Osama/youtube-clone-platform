package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/minio/minio-go/v7"
)

// MinIOStorage implements the Storage interface using MinIO
type MinIOStorage struct {
	client          *minio.Client
	bucketName      string
	processedBucket string
	originalPrefix  string
	hlsPrefix       string
	mp4Prefix       string
	thumbnailPrefix string
}

// NewMinIOStorage creates a new MinIOStorage instance
func NewMinIOStorage(
	client *minio.Client,
	bucketName string,
	processedBucket string,
	originalPrefix string,
	hlsPrefix string,
	mp4Prefix string,
	thumbnailPrefix string,
) Storage {
	// Create context for bucket operations
	ctx := context.Background()

	// Ensure both buckets exist
	buckets := []string{bucketName, processedBucket}
	for _, bucket := range buckets {
		exists, err := client.BucketExists(ctx, bucket)
		if err != nil {
			log.Printf("Error checking bucket %s: %v", bucket, err)
			continue
		}
		if !exists {
			err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
			if err != nil {
				log.Printf("Error creating bucket %s: %v", bucket, err)
			} else {
				log.Printf("Created bucket: %s", bucket)
			}
		}
	}

	return &MinIOStorage{
		client:          client,
		bucketName:      bucketName,
		processedBucket: processedBucket,
		originalPrefix:  originalPrefix,
		hlsPrefix:       hlsPrefix,
		mp4Prefix:       mp4Prefix,
		thumbnailPrefix: thumbnailPrefix,
	}
}

// DownloadVideo downloads a video from MinIO to a local file
func (s *MinIOStorage) DownloadVideo(ctx context.Context, videoID string, fileExtension string, localPath string) error {
	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	// Construct the object name
	objectName := filepath.Join(s.originalPrefix, videoID+fileExtension)

	// Get the object from MinIO
	object, err := s.client.GetObject(ctx, s.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get object from MinIO: %w", err)
	}
	defer object.Close()

	// Copy the object to the local file
	_, err = io.Copy(localFile, object)
	if err != nil {
		return fmt.Errorf("failed to copy object to local file: %w", err)
	}

	return nil
}

// UploadHLSFiles uploads HLS files to MinIO
func (s *MinIOStorage) UploadHLSFiles(ctx context.Context, videoID string, localDir string) (string, error) {
	// Create the HLS directory in MinIO
	hlsDir := filepath.Join(s.hlsPrefix, videoID)

	// Walk through the local directory and upload all files
	err := filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get the relative path
		relPath, err := filepath.Rel(localDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Construct the object name
		objectName := filepath.Join(hlsDir, relPath)

		// Open the file
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		// Get file stats
		fileInfo, err := file.Stat()
		if err != nil {
			return fmt.Errorf("failed to get file stats: %w", err)
		}

		// Determine content type
		contentType := "application/octet-stream"
		if filepath.Ext(path) == ".m3u8" {
			contentType = "application/vnd.apple.mpegurl"
		} else if filepath.Ext(path) == ".ts" {
			contentType = "video/mp2t"
		}

		// Upload the file to MinIO processed bucket
		_, err = s.client.PutObject(ctx, s.processedBucket, objectName, file, fileInfo.Size(), minio.PutObjectOptions{
			ContentType: contentType,
		})
		if err != nil {
			return fmt.Errorf("failed to upload file to MinIO: %w", err)
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload HLS files: %w", err)
	}

	// Return the HLS directory path
	return hlsDir, nil
}

// GetMP4Prefix returns the MP4 prefix
func (s *MinIOStorage) GetMP4Prefix() string {
	return s.mp4Prefix
}

// UploadMP4Files uploads MP4 files to MinIO
func (s *MinIOStorage) UploadMP4Files(ctx context.Context, videoID string, mp4Dir string) error {
	return filepath.Walk(mp4Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(mp4Dir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		objectName := fmt.Sprintf("%s/%s/%s", s.mp4Prefix, videoID, relPath)
		_, err = s.client.FPutObject(ctx, s.processedBucket, objectName, path, minio.PutObjectOptions{
			ContentType: "video/mp4",
		})
		if err != nil {
			return fmt.Errorf("failed to upload MP4 file %s: %w", relPath, err)
		}

		return nil
	})
}

// UploadThumbnail uploads a thumbnail to MinIO
func (s *MinIOStorage) UploadThumbnail(ctx context.Context, videoID string, localPath string) (string, error) {
	// Open the file
	file, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file stats
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file stats: %w", err)
	}

	// Construct the object name
	objectName := filepath.Join(s.thumbnailPrefix, videoID+filepath.Ext(localPath))

	// Upload the file to MinIO processed bucket
	_, err = s.client.PutObject(ctx, s.processedBucket, objectName, file, fileInfo.Size(), minio.PutObjectOptions{
		ContentType: "image/jpeg",
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload thumbnail to MinIO: %w", err)
	}

	// Return the thumbnail path
	return objectName, nil
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

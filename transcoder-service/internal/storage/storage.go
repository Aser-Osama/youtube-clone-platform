package storage

import (
	"context"
)

// Storage defines the interface for video storage operations
type Storage interface {
	// DownloadVideo downloads a video from storage
	DownloadVideo(ctx context.Context, videoID string, fileExtension string, localPath string) error
	// UploadHLSFiles uploads HLS files to storage
	UploadHLSFiles(ctx context.Context, videoID string, localDir string) (string, error)
	// UploadMP4Files uploads MP4 files to storage
	UploadMP4Files(ctx context.Context, videoID string, mp4Dir string) error
	// UploadThumbnail uploads a thumbnail to storage
	UploadThumbnail(ctx context.Context, videoID string, thumbnailPath string) (string, error)
	// CheckHealth checks if the storage is healthy
	CheckHealth(ctx context.Context) error
	// GetMP4Prefix returns the MP4 prefix
	GetMP4Prefix() string
}

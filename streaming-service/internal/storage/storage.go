package storage

import (
	"context"
	"time"
)

// Storage defines the interface for video streaming operations
type Storage interface {
	// GetHLSManifest returns the HLS manifest (.m3u8) for a video
	GetHLSManifest(ctx context.Context, videoID string) (string, error)

	// GetHLSSegment returns a signed URL for an HLS segment (.ts)
	GetHLSSegment(ctx context.Context, videoID string, segmentName string) (string, error)

	// GetMP4URL returns a signed URL for the MP4 version of a video
	GetMP4URL(ctx context.Context, videoID string) (string, error)

	// GetMP4URLWithQuality returns a signed URL for the MP4 version of a video with specified quality
	// If quality is empty, it will return the highest available quality
	GetMP4URLWithQuality(ctx context.Context, videoID string, quality string) (string, error)

	// GetThumbnailURL returns a signed URL for the video thumbnail
	GetThumbnailURL(ctx context.Context, videoID string) (string, error)

	// CheckHealth checks if the storage is healthy
	CheckHealth(ctx context.Context) error
}

// URLGenerator defines the interface for generating signed URLs
type URLGenerator interface {
	// GeneratePresignedURL generates a presigned URL for an object
	GeneratePresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error)
}

// HLSProcessor defines additional methods for HLS content processing
type HLSProcessor interface {
	// GetObjectContent fetches the content of an object as a string
	GetObjectContent(ctx context.Context, objectName string) (string, error)

	// ProcessM3U8 processes an m3u8 file to replace relative URLs with absolute URLs
	ProcessM3U8(content, videoID, resolution string) (string, error)

	// GetHLSObjectPath returns the full object path for HLS content
	GetHLSObjectPath(videoID string, relativePath string) string
}

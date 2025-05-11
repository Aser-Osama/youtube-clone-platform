package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage implements the Storage interface using MinIO
type MinIOStorage struct {
	client          *minio.Client
	bucketName      string
	hlsPrefix       string
	mp4Prefix       string
	thumbnailPrefix string
	urlExpiry       time.Duration
	baseURL         string // Base URL for playlist links
}

// MinIOConfig holds the configuration for MinIO
type MinIOConfig struct {
	Endpoint        string
	AccessKey       string
	SecretKey       string
	UseSSL          bool
	BucketName      string
	HLSPrefix       string
	MP4Prefix       string
	ThumbnailPrefix string
	URLExpiry       int
	BaseURL         string // Base URL to use instead of localhost when specified
}

// NewMinIOStorage creates a new MinIO storage instance
func NewMinIOStorage(cfg *MinIOConfig) (*MinIOStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		// Default to localhost with http/https based on UseSSL
		protocol := "http"
		if cfg.UseSSL {
			protocol = "https"
		}
		baseURL = fmt.Sprintf("%s://%s/%s", protocol, cfg.Endpoint, cfg.BucketName)
	}

	return &MinIOStorage{
		client:          client,
		bucketName:      cfg.BucketName,
		hlsPrefix:       cfg.HLSPrefix,
		mp4Prefix:       cfg.MP4Prefix,
		thumbnailPrefix: cfg.ThumbnailPrefix,
		urlExpiry:       time.Duration(cfg.URLExpiry) * time.Second,
		baseURL:         baseURL,
	}, nil
}

// GetHLSManifest returns the HLS manifest (.m3u8) for a video
func (s *MinIOStorage) GetHLSManifest(ctx context.Context, videoID string) (string, error) {
	fmt.Printf("Looking for HLS manifest for video ID: %s\n", videoID)
	fmt.Printf("Checking HLS manifests in bucket: %s with prefix: %s\n", s.bucketName, s.hlsPrefix)

	// First check if master playlist exists
	masterPath := path.Join(s.hlsPrefix, videoID, "master.m3u8")
	masterExists, err := s.objectExists(ctx, masterPath)
	if err == nil && masterExists {
		fmt.Printf("Found existing master.m3u8 for video %s\n", videoID)
		content, err := s.GetObjectContent(ctx, masterPath)
		if err != nil {
			return "", err
		}
		// Return the master playlist as is - it should already have the proper URLs
		return content, nil
	}

	// If no master playlist exists, we'll generate one based on available resolution playlists
	// First, find all available resolution playlists
	resolutions := []string{"1080p", "720p", "480p", "360p", "240p"}
	var availableResolutions []string
	var highestResolution string

	fmt.Printf("No master.m3u8 found, checking for resolution-specific playlists for video %s\n", videoID)
	for _, res := range resolutions {
		playlistPath := path.Join(s.hlsPrefix, videoID, res, "playlist.m3u8")
		exists, err := s.objectExists(ctx, playlistPath)
		if err == nil && exists {
			fmt.Printf("Found %s/playlist.m3u8 for video %s\n", res, videoID)
			availableResolutions = append(availableResolutions, res)
			if highestResolution == "" {
				highestResolution = res
			}
		}
	}

	fmt.Printf("Available resolutions for video %s: %v\n", videoID, availableResolutions)

	if len(availableResolutions) == 0 {
		// No resolution-specific playlists found, fall back to looking for a single playlist
		fallbackPaths := []string{
			path.Join(s.hlsPrefix, videoID, "playlist.m3u8"),
			path.Join(s.hlsPrefix, videoID, "index.m3u8"),
		}

		fmt.Printf("No resolution playlists found, checking for fallback playlists for video %s\n", videoID)
		for _, manifestPath := range fallbackPaths {
			exists, err := s.objectExists(ctx, manifestPath)
			if err == nil && exists {
				fmt.Printf("Found fallback playlist %s for video %s\n", manifestPath, videoID)
				content, err := s.GetObjectContent(ctx, manifestPath)
				if err != nil {
					return "", err
				}
				return content, nil
			}
		}

		return "", fmt.Errorf("no HLS manifest found for video ID %s", videoID)
	}

	// Generate a master playlist with available resolutions
	var bandwidth = map[string]int{
		"1080p": 5000000,
		"720p":  3000000,
		"480p":  1500000,
		"360p":  800000,
		"240p":  400000,
	}

	var resolution = map[string]string{
		"1080p": "1920x1080",
		"720p":  "1280x720",
		"480p":  "854x480",
		"360p":  "640x360",
		"240p":  "426x240",
	}

	masterPlaylist := []string{
		"#EXTM3U",
		"#EXT-X-VERSION:3",
	}

	// Add each available resolution to the master playlist
	for _, res := range availableResolutions {
		bw := bandwidth[res]
		resSplit := resolution[res]

		variant := fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%s", bw, resSplit)
		masterPlaylist = append(masterPlaylist, variant)

		// Generate a presigned URL for the playlist
		playlistPath := path.Join(res, "playlist.m3u8")
		playlistURL, err := s.GeneratePresignedURL(ctx, s.GetHLSObjectPath(videoID, playlistPath), s.urlExpiry)
		if err != nil {
			return "", fmt.Errorf("failed to generate signed URL for playlist: %w", err)
		}
		masterPlaylist = append(masterPlaylist, playlistURL)
	}

	generatedManifest := strings.Join(masterPlaylist, "\n")
	fmt.Printf("Generated master manifest for video %s with %d resolutions:\n%s\n",
		videoID, len(availableResolutions), generatedManifest)

	return generatedManifest, nil
}

// GetHLSSegment returns a signed URL for an HLS segment (.ts)
func (s *MinIOStorage) GetHLSSegment(ctx context.Context, videoID string, segmentName string) (string, error) {
	fmt.Printf("Getting HLS segment for video %s, segment %s\n", videoID, segmentName)

	// First try the direct path (no resolution subfolder)
	directObjectName := path.Join(s.hlsPrefix, videoID, segmentName)
	directExists, err := s.objectExists(ctx, directObjectName)
	if err == nil && directExists {
		fmt.Printf("Found segment at direct path: %s\n", directObjectName)
		return s.GeneratePresignedURL(ctx, directObjectName, s.urlExpiry)
	}

	// Try resolution-specific folders
	resolutions := []string{"1080p", "720p", "480p", "360p", "240p"}
	for _, resolution := range resolutions {
		// Try both with and without resolution prefix in segmentName
		var nestedObjectName string
		if strings.HasPrefix(segmentName, resolution+"/") {
			// If segmentName already includes resolution, use it as is
			nestedObjectName = path.Join(s.hlsPrefix, videoID, segmentName)
		} else {
			// Otherwise, add resolution prefix
			nestedObjectName = path.Join(s.hlsPrefix, videoID, resolution, segmentName)
		}

		fmt.Printf("Checking segment at path: %s\n", nestedObjectName)
		nestedExists, err := s.objectExists(ctx, nestedObjectName)
		if err == nil && nestedExists {
			fmt.Printf("Found segment at nested path: %s\n", nestedObjectName)
			return s.GeneratePresignedURL(ctx, nestedObjectName, s.urlExpiry)
		}
	}

	// If not found, try the original path as a last resort
	fmt.Printf("Segment not found in any location, trying original path: %s\n", directObjectName)
	return s.GeneratePresignedURL(ctx, directObjectName, s.urlExpiry)
}

// GetMP4URLWithQuality returns a signed URL for the MP4 version of a video with specified quality
func (s *MinIOStorage) GetMP4URLWithQuality(ctx context.Context, videoID string, quality string) (string, error) {
	fmt.Printf("Looking for MP4 video with ID: %s in bucket '%s' with mp4Prefix '%s', requested quality: '%s'\n",
		videoID, s.bucketName, s.mp4Prefix, quality)

	// If specific quality is requested, try that first
	if quality != "" {
		// Check if the requested quality exists
		objectName := path.Join(s.mp4Prefix, videoID, "mp4", quality+".mp4")
		fmt.Printf("Trying requested quality MP4 object path: '%s'\n", objectName)
		exists, err := s.objectExists(ctx, objectName)
		if err == nil && exists {
			fmt.Printf("Found MP4 at requested quality path: '%s'\n", objectName)
			return s.GeneratePresignedURL(ctx, objectName, s.urlExpiry)
		}
		// If requested quality not found, log but continue to fallback options
		if err != nil {
			fmt.Printf("Error checking object '%s': %v\n", objectName, err)
		} else {
			fmt.Printf("Requested quality '%s' not found for video ID '%s'\n", quality, videoID)
		}
	}

	// If specific quality wasn't requested or wasn't found, try default resolutions in order (highest to lowest)
	resolutions := []string{"1080p", "720p", "480p", "360p", "240p"}
	for _, resolution := range resolutions {
		objectName := path.Join(s.mp4Prefix, videoID, "mp4", resolution+".mp4")
		fmt.Printf("Trying MP4 object path: '%s'\n", objectName)
		exists, err := s.objectExists(ctx, objectName)
		if err != nil {
			fmt.Printf("Error checking object '%s': %v. Skipping.\n", objectName, err)
			continue
		}
		if exists {
			fmt.Printf("Found MP4 at object path: '%s'\n", objectName)
			return s.GeneratePresignedURL(ctx, objectName, s.urlExpiry)
		}
	}

	// Fallback: Check for a generic video.mp4
	genericObjectName := path.Join(s.mp4Prefix, videoID, "mp4", "video.mp4")
	fmt.Printf("Trying generic MP4 object path: '%s'\n", genericObjectName)
	exists, err := s.objectExists(ctx, genericObjectName)
	if err == nil && exists {
		fmt.Printf("Found generic MP4 at object path: '%s'\n", genericObjectName)
		return s.GeneratePresignedURL(ctx, genericObjectName, s.urlExpiry)
	}
	if err != nil {
		fmt.Printf("Error checking generic object '%s': %v.\n", genericObjectName, err)
	}

	fmt.Printf("No MP4 file found for video ID '%s' in bucket '%s' with mp4Prefix '%s' after checking all patterns.\n",
		videoID, s.bucketName, s.mp4Prefix)
	return "", fmt.Errorf("no MP4 file found for video ID %s after checking specific resolution and generic paths", videoID)
}

// GetMP4URL returns a signed URL for the MP4 version of a video
// For backward compatibility, calls GetMP4URLWithQuality with empty quality
func (s *MinIOStorage) GetMP4URL(ctx context.Context, videoID string) (string, error) {
	return s.GetMP4URLWithQuality(ctx, videoID, "")
}

// Helper function to check if an object exists
func (s *MinIOStorage) objectExists(ctx context.Context, objectName string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		// Check if the error is because the object doesn't exist
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetThumbnailURL returns a signed URL for the video thumbnail
func (s *MinIOStorage) GetThumbnailURL(ctx context.Context, videoID string) (string, error) {
	// Try both possible thumbnail paths
	paths := []string{
		path.Join(s.thumbnailPrefix, videoID, "thumbnail.jpg"), // New path format
		path.Join(s.thumbnailPrefix, videoID+".jpg"),           // Old path format
	}

	for _, objectName := range paths {
		exists, err := s.objectExists(ctx, objectName)
		if err == nil && exists {
			fmt.Printf("Found thumbnail at path: %s\n", objectName)
			return s.GeneratePresignedURL(ctx, objectName, s.urlExpiry)
		}
	}

	// If no thumbnail found, return error
	return "", fmt.Errorf("no thumbnail found for video ID %s", videoID)
}

// GeneratePresignedURL generates a presigned URL for an object
func (s *MinIOStorage) GeneratePresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	url, err := s.client.PresignedGetObject(ctx, s.bucketName, objectName, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url.String(), nil
}

// CheckHealth checks if the storage is healthy
func (s *MinIOStorage) CheckHealth(ctx context.Context) error {
	_, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	return nil
}

// GetObjectContent fetches the content of an object as a string
func (s *MinIOStorage) GetObjectContent(ctx context.Context, objectName string) (string, error) {
	obj, err := s.client.GetObject(ctx, s.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return "", err
	}
	defer obj.Close()
	buf := new(strings.Builder)
	_, err = io.Copy(buf, obj)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ProcessM3U8 processes an m3u8 file to replace relative URLs with absolute URLs
func (s *MinIOStorage) ProcessM3U8(content, videoID, resolution string) (string, error) {
	// Regular expression to match segment file references
	segmentRegex := regexp.MustCompile(`([^/\n]+\.ts)`)

	// Process the content line by line
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		// Skip comments and directives
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Replace segment references with absolute URLs
		if segmentRegex.MatchString(line) {
			segmentName := line
			// Generate a presigned URL for the segment
			segmentPath := path.Join(resolution, segmentName)
			signedURL, err := s.GeneratePresignedURL(context.Background(), s.GetHLSObjectPath(videoID, segmentPath), s.urlExpiry)
			if err != nil {
				return "", fmt.Errorf("failed to generate signed URL for segment: %w", err)
			}
			lines[i] = signedURL
		}
	}

	return strings.Join(lines, "\n"), nil
}

// GetHLSObjectPath returns the full object path for HLS content
func (s *MinIOStorage) GetHLSObjectPath(videoID string, relativePath string) string {
	return path.Join(s.hlsPrefix, videoID, relativePath)
}

// GetMP4Prefix returns the prefix used for MP4 files
func (s *MinIOStorage) GetMP4Prefix() string {
	return s.mp4Prefix
}

// ObjectExists checks if an object exists in the storage
func (s *MinIOStorage) ObjectExists(ctx context.Context, objectName string) (bool, error) {
	return s.objectExists(ctx, objectName)
}

package handler

import (
	"fmt"
	"net/http"
	"path"

	"youtube-clone-platform/streaming-service/internal/storage"

	"github.com/gin-gonic/gin"
)

// StreamHandler handles video streaming requests
type StreamHandler struct {
	storage storage.Storage
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler(storage storage.Storage) *StreamHandler {
	return &StreamHandler{
		storage: storage,
	}
}

// HandleHLSManifest handles requests for HLS manifest files
func (h *StreamHandler) HandleHLSManifest(c *gin.Context) {
	videoID := c.Param("videoID")
	if videoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video ID is required"})
		return
	}

	manifest, err := h.storage.GetHLSManifest(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get HLS manifest"})
		return
	}

	// If the manifest looks like a URL, redirect. Otherwise, serve as content.
	if len(manifest) > 0 && (manifest[:4] == "http" || manifest[:5] == "https") {
		c.Redirect(http.StatusTemporaryRedirect, manifest)
		return
	}

	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Cache-Control", "max-age=300") // Cache for 5 minutes
	c.String(http.StatusOK, manifest)
}

// HandleHLSSegment handles requests for HLS segment files
func (h *StreamHandler) HandleHLSSegment(c *gin.Context) {
	videoID := c.Param("videoID")
	segment := c.Param("segment")
	resolution := c.Param("resolution")
	if videoID == "" || segment == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video ID and segment are required"})
		return
	}

	var segmentPath string
	if resolution != "" {
		segmentPath = resolution + "/" + segment
	} else {
		segmentPath = segment
	}

	url, err := h.storage.GetHLSSegment(c.Request.Context(), videoID, segmentPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get HLS segment"})
		return
	}

	// Set cache headers for segments
	c.Header("Cache-Control", "max-age=604800") // Cache for one week
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// HandleHLSPlaylist handles requests for HLS sub-playlist files (e.g., 720p/playlist.m3u8)
func (h *StreamHandler) HandleHLSPlaylist(c *gin.Context) {
	videoID := c.Param("videoID")
	resolution := c.Param("resolution")
	if videoID == "" || resolution == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video ID and resolution are required"})
		return
	}

	// Log the request
	fmt.Printf("HandleHLSPlaylist: Request for videoID=%s, resolution=%s\n", videoID, resolution)

	// Get the resolution-specific playlist
	playlistName := "playlist.m3u8"
	playlistPath := resolution + "/" + playlistName

	// Get the MinIO storage client
	minioStorage, ok := h.storage.(*storage.MinIOStorage)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage implementation error"})
		return
	}

	// Try to get the playlist content directly
	objectName := minioStorage.GetHLSObjectPath(videoID, playlistPath)
	fmt.Printf("HandleHLSPlaylist: Checking for playlist at path: %s\n", objectName)

	content, err := minioStorage.GetObjectContent(c.Request.Context(), objectName)
	if err != nil {
		fmt.Printf("HandleHLSPlaylist: Error getting content: %v\n", err)
		// If we can't get the content directly, try getting the signed URL
		url, err := h.storage.GetHLSSegment(c.Request.Context(), videoID, playlistPath)
		if err != nil {
			fmt.Printf("HandleHLSPlaylist: Error getting signed URL: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get HLS playlist"})
			return
		}
		fmt.Printf("HandleHLSPlaylist: Redirecting to: %s\n", url)
		c.Redirect(http.StatusTemporaryRedirect, url)
		return
	}

	// Process the playlist to update segment URLs
	processedContent, err := minioStorage.ProcessM3U8(content, videoID, resolution)
	if err != nil {
		fmt.Printf("HandleHLSPlaylist: Error processing M3U8: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process playlist"})
		return
	}

	fmt.Printf("HandleHLSPlaylist: Successfully processed playlist for videoID=%s, resolution=%s\n", videoID, resolution)
	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Cache-Control", "max-age=300") // Cache for 5 minutes
	c.String(http.StatusOK, processedContent)
}

// HandleMP4 handles requests for MP4 video files
func (h *StreamHandler) HandleMP4(c *gin.Context) {
	videoID := c.Param("videoID")
	if videoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video ID is required"})
		return
	}

	// Check for quality parameter (e.g., 1080p, 720p, etc.)
	quality := c.Query("quality")

	var url string
	var err error

	if quality != "" {
		// If quality parameter is provided, use the new method
		url, err = h.storage.GetMP4URLWithQuality(c.Request.Context(), videoID, quality)
	} else {
		// For backward compatibility
		url, err = h.storage.GetMP4URL(c.Request.Context(), videoID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get MP4 URL"})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// HandleThumbnail handles requests for video thumbnails
func (h *StreamHandler) HandleThumbnail(c *gin.Context) {
	videoID := c.Param("videoID")
	if videoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video ID is required"})
		return
	}

	url, err := h.storage.GetThumbnailURL(c.Request.Context(), videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get thumbnail URL"})
		return
	}

	// Set appropriate cache headers for thumbnails
	c.Header("Cache-Control", "max-age=86400") // Cache for one day
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// ListMP4Qualities handles requests to list available MP4 qualities for a video
func (h *StreamHandler) ListMP4Qualities(c *gin.Context) {
	videoID := c.Param("videoID")
	if videoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video ID is required"})
		return
	}

	// Get the MinIO storage client
	minioStorage, ok := h.storage.(*storage.MinIOStorage)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage implementation error"})
		return
	}

	// Define standard qualities to check
	standardQualities := []string{"1080p", "720p", "480p", "360p", "240p"}
	availableQualities := []string{}

	// Check each quality
	for _, quality := range standardQualities {
		objectName := path.Join(minioStorage.GetMP4Prefix(), videoID, "mp4", quality+".mp4")
		exists, err := minioStorage.ObjectExists(c.Request.Context(), objectName)
		if err == nil && exists {
			availableQualities = append(availableQualities, quality)
		}
	}

	// Also check for generic video.mp4
	genericObjectName := path.Join(minioStorage.GetMP4Prefix(), videoID, "mp4", "video.mp4")
	genericExists, _ := minioStorage.ObjectExists(c.Request.Context(), genericObjectName)
	if genericExists {
		availableQualities = append(availableQualities, "default")
	}

	// Return the available qualities
	c.JSON(http.StatusOK, gin.H{
		"video_id":  videoID,
		"qualities": availableQualities,
	})
}

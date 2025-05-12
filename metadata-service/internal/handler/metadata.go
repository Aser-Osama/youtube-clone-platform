package handler

import (
	"net/http"
	"strconv"

	"youtube-clone-platform/metadata-service/internal/service"

	"github.com/gin-gonic/gin"
)

// MetadataHandler handles HTTP requests for video metadata
type MetadataHandler struct {
	metadataService *service.MetadataService
}

// NewMetadataHandler creates a new metadata handler
func NewMetadataHandler(metadataService *service.MetadataService) *MetadataHandler {
	return &MetadataHandler{
		metadataService: metadataService,
	}
}

// RegisterRoutes registers the HTTP routes for the metadata service
func (h *MetadataHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/metadata")
	{
		// Public endpoints
		api.GET("/public/videos", h.GetRecentVideos)
		api.GET("/public/videos/:id", h.GetVideoMetadata)

		// Regular endpoints (now all public)
		api.GET("/videos/:id", h.GetVideoMetadata)
		api.GET("/videos", h.GetRecentVideos)
		api.POST("/videos/:id/views", h.IncrementViews)
		api.GET("/videos/search", h.SearchVideos)
		api.GET("/users/:id/videos", h.GetUserVideos)
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}
}

// GetVideoMetadata handles GET /api/v1/videos/:id
func (h *MetadataHandler) GetVideoMetadata(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video ID is required"})
		return
	}

	metadata, err := h.metadataService.GetVideoMetadata(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metadata)
}

// GetRecentVideos handles GET /api/v1/videos
func (h *MetadataHandler) GetRecentVideos(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	videos, err := h.metadataService.GetRecentVideos(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, videos)
}

// IncrementViews handles POST /api/v1/videos/:id/views
func (h *MetadataHandler) IncrementViews(c *gin.Context) {
	videoID := c.Param("id")
	if videoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video ID is required"})
		return
	}

	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID is required"})
		return
	}

	if err := h.metadataService.IncrementViews(c.Request.Context(), videoID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// SearchVideos handles GET /api/v1/videos/search
func (h *MetadataHandler) SearchVideos(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query is required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	videos, err := h.metadataService.SearchVideos(c.Request.Context(), query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, videos)
}

// GetUserVideos handles GET /api/v1/users/:id/videos
func (h *MetadataHandler) GetUserVideos(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user ID is required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	videos, err := h.metadataService.GetVideosByUser(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, videos)
}

package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aser/youtube-clone-platform/video-upload-service/internal/service"
	"github.com/aser/youtube-clone-platform/video-upload-service/internal/validation"
	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	service *service.UploadService
}

func NewUploadHandler(s *service.UploadService) *UploadHandler {
	return &UploadHandler{service: s}
}

type UploadError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *UploadError) Error() string {
	return e.Message
}

func (h *UploadHandler) HandleUpload(c *gin.Context) {
	startTime := time.Now()

	// Get video title from form
	title := c.PostForm("title")
	if title == "" {
		c.JSON(http.StatusBadRequest, &UploadError{
			Code:    http.StatusBadRequest,
			Message: "title is required",
		})
		return
	}

	// Get user ID from form (will be set by gateway in production)
	userID := c.PostForm("user_id")
	if userID == "" {
		userID = "anonymous" // Default value if not provided
	}

	// Get the video file
	file, header, err := c.Request.FormFile("video")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			c.JSON(http.StatusBadRequest, &UploadError{
				Code:    http.StatusBadRequest,
				Message: "video file is required",
			})
			return
		}
		c.JSON(http.StatusBadRequest, &UploadError{
			Code:    http.StatusBadRequest,
			Message: "invalid file upload",
			Details: err.Error(),
		})
		return
	}
	defer file.Close()

	// Validate video file
	if err := validation.ValidateVideoFile(header); err != nil {
		c.JSON(http.StatusBadRequest, &UploadError{
			Code:    http.StatusBadRequest,
			Message: "invalid video file",
			Details: err.Error(),
		})
		return
	}

	// Process upload through service
	result, err := h.service.HandleUpload(
		c.Request.Context(),
		userID,
		title,
		file,
		header.Size,
		header.Header.Get("Content-Type"),
		header.Filename,
	)
	if err != nil {
		var uploadErr *UploadError
		if errors.As(err, &uploadErr) {
			c.JSON(uploadErr.Code, uploadErr)
			return
		}

		var validationErr *validation.ValidationError
		if errors.As(err, &validationErr) {
			c.JSON(http.StatusBadRequest, &UploadError{
				Code:    http.StatusBadRequest,
				Message: "validation failed",
				Details: validationErr.Error(),
			})
			return
		}

		// Log the error for debugging
		fmt.Printf("Upload error: %v\n", err)

		c.JSON(http.StatusInternalServerError, &UploadError{
			Code:    http.StatusInternalServerError,
			Message: "failed to process upload",
			Details: "an unexpected error occurred",
		})
		return
	}

	// Calculate processing time
	processingTime := time.Since(startTime)

	response := gin.H{
		"video_id":        result.VideoID,
		"title":           title,
		"user_id":         userID,
		"processing_time": processingTime.String(),
		"uploaded_at":     time.Now().UTC().Format(time.RFC3339),
	}

	if result.Warning != "" {
		response["warning"] = result.Warning
	}

	if result.Metadata != nil {
		response["metadata"] = result.Metadata
	}

	c.JSON(http.StatusOK, response)
}

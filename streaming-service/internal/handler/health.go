package handler

import (
	"context"
	"net/http"

	"youtube-clone-platform/streaming-service/internal/storage"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	storage storage.Storage
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(storage storage.Storage) *HealthHandler {
	return &HealthHandler{
		storage: storage,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status       string            `json:"status"`
	Dependencies map[string]string `json:"dependencies"`
}

// HandleHealthCheck handles health check requests
func (h *HealthHandler) HandleHealthCheck(c *gin.Context) {
	status := h.RunHealthCheck(c.Request.Context())
	if status.Status == "ok" {
		c.JSON(http.StatusOK, status)
	} else {
		c.JSON(http.StatusServiceUnavailable, status)
	}
}

// RunHealthCheck performs a health check
func (h *HealthHandler) RunHealthCheck(ctx context.Context) HealthResponse {
	dependencies := make(map[string]string)

	// Check MinIO health
	if err := h.storage.CheckHealth(ctx); err != nil {
		dependencies["minio"] = "error"
		return HealthResponse{
			Status:       "error",
			Dependencies: dependencies,
		}
	}
	dependencies["minio"] = "ok"

	return HealthResponse{
		Status:       "ok",
		Dependencies: dependencies,
	}
}

package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	sharedlog "github.com/aser/youtube-clone-platform/internal/shared/log"
	"github.com/aser/youtube-clone-platform/video-upload-service/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	minioStorage *storage.MinIOStorage
	kafkaBrokers []string
	kafkaTopic   string
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(minioStorage *storage.MinIOStorage, kafkaBrokers []string, kafkaTopic string) *HealthHandler {
	return &HealthHandler{
		minioStorage: minioStorage,
		kafkaBrokers: kafkaBrokers,
		kafkaTopic:   kafkaTopic,
	}
}

// HealthStatus represents the service health status
type HealthStatus struct {
	Status       string            `json:"status"`
	Dependencies map[string]string `json:"dependencies"`
	Timestamp    string            `json:"timestamp"`
	Version      string            `json:"version"`
}

// HandleHealthCheck checks the health of the service and its dependencies
func (h *HealthHandler) HandleHealthCheck(c *gin.Context) {
	status := HealthStatus{
		Status:       "ok",
		Dependencies: make(map[string]string),
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Version:      "1.0.0", // Should be loaded from build info in production
	}

	// Check MinIO connection
	minioStatus := "ok"
	if err := h.checkMinIO(); err != nil {
		minioStatus = "error: " + err.Error()
		status.Status = "degraded"
		sharedlog.Error(fmt.Sprintf("Health check: MinIO connection failed: %v", err))
	}
	status.Dependencies["minio"] = minioStatus

	// Check Kafka connection
	kafkaStatus := "ok"
	if err := h.checkKafka(); err != nil {
		kafkaStatus = "error: " + err.Error()
		status.Status = "degraded"
		sharedlog.Error(fmt.Sprintf("Health check: Kafka connection failed: %v", err))
	}
	status.Dependencies["kafka"] = kafkaStatus

	c.JSON(http.StatusOK, status)
}

// checkMinIO verifies the MinIO connection
func (h *HealthHandler) checkMinIO() error {
	// Use a short timeout for health check
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return h.minioStorage.CheckHealth(ctx)
}

// checkKafka verifies the Kafka connection
func (h *HealthHandler) checkKafka() error {
	// Try to connect to Kafka with timeout
	conn, err := kafka.DialLeader(context.Background(), "tcp", h.kafkaBrokers[0], h.kafkaTopic, 0)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Set a short read deadline to check if connection is alive
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	return nil
}

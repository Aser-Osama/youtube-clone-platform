package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"youtube-clone-platform/transcoder-service/internal/storage"

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

// RunHealthCheck performs a health check and returns the status
func (h *HealthHandler) RunHealthCheck() HealthStatus {
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
		fmt.Printf("Health check: MinIO connection failed: %v\n", err)
	}
	status.Dependencies["minio"] = minioStatus

	// Check Kafka connection and topic
	kafkaStatus := "ok"
	if err := h.checkKafka(); err != nil {
		kafkaStatus = "error: " + err.Error()
		status.Status = "degraded"
		fmt.Printf("Health check: Kafka connection failed: %v\n", err)
	}
	status.Dependencies["kafka"] = kafkaStatus

	return status
}

// HandleHealthCheck checks the health of the service and its dependencies
func (h *HealthHandler) HandleHealthCheck(c *gin.Context) {
	status := h.RunHealthCheck()
	c.JSON(http.StatusOK, status)
}

// checkMinIO verifies the MinIO connection
func (h *HealthHandler) checkMinIO() error {
	// Use a short timeout for health check
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return h.minioStorage.CheckHealth(ctx)
}

// checkKafka verifies the Kafka connection and topic existence
func (h *HealthHandler) checkKafka() error {
	// Try to connect to Kafka with timeout
	conn, err := kafka.Dial("tcp", h.kafkaBrokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to Kafka: %w", err)
	}
	defer conn.Close()

	// Set a short read deadline to check if connection is alive
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Try to create the topic (this is idempotent - won't fail if topic exists)
	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             h.kafkaTopic,
			NumPartitions:     3,
			ReplicationFactor: 1,
		},
	}

	err = conn.CreateTopics(topicConfigs...)
	if err != nil {
		return fmt.Errorf("failed to create/verify topic: %w", err)
	}
	fmt.Printf("Verified/created Kafka topic: %s\n", h.kafkaTopic)

	return nil
}

package main

import (
	"fmt"

	sharedlog "github.com/aser/youtube-clone-platform/internal/shared/log"
	"github.com/aser/youtube-clone-platform/video-upload-service/internal/config"
	"github.com/aser/youtube-clone-platform/video-upload-service/internal/events"
	"github.com/aser/youtube-clone-platform/video-upload-service/internal/handler"
	"github.com/aser/youtube-clone-platform/video-upload-service/internal/service"
	"github.com/aser/youtube-clone-platform/video-upload-service/internal/storage"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		sharedlog.Error(fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	// Initialize MinIO storage
	minioStorage, err := storage.NewMinIOStorage(
		cfg.MinIO.Endpoint,
		cfg.MinIO.AccessKeyID,
		cfg.MinIO.SecretAccessKey,
		cfg.MinIO.UseSSL,
		cfg.MinIO.BucketName,
	)
	if err != nil {
		sharedlog.Error(fmt.Sprintf("Failed to initialize MinIO storage: %v", err))
		return
	}

	// Initialize Kafka publisher
	kafkaPublisher := events.NewKafkaPublisher(cfg.Kafka.Brokers, cfg.Kafka.Topic)
	defer kafkaPublisher.Close()

	// Initialize upload service
	uploadService := service.NewUploadService(minioStorage, kafkaPublisher, cfg.MaxBytes)

	// Initialize handlers
	uploadHandler := handler.NewUploadHandler(uploadService)
	healthHandler := handler.NewHealthHandler(minioStorage, cfg.Kafka.Brokers, cfg.Kafka.Topic)

	// Setup Gin router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", healthHandler.HandleHealthCheck)

	// Register routes
	router.POST("/upload", uploadHandler.HandleUpload)

	// Start server
	sharedlog.Info(fmt.Sprintf("Starting video upload service on port %s", cfg.Port))
	if err := router.Run(":" + cfg.Port); err != nil {
		sharedlog.Error(fmt.Sprintf("Failed to start server: %v", err))
		return
	}
}

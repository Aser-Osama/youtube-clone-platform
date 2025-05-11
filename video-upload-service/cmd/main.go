package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	sharedlog "youtube-clone-platform/internal/shared/log"
	"youtube-clone-platform/video-upload-service/internal/config"
	"youtube-clone-platform/video-upload-service/internal/events"
	"youtube-clone-platform/video-upload-service/internal/handler"
	"youtube-clone-platform/video-upload-service/internal/service"
	"youtube-clone-platform/video-upload-service/internal/storage"

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

	// API v1 routes
	api := router.Group("/api/v1/upload")
	{
		// Health check endpoint
		api.GET("/health", healthHandler.HandleHealthCheck)

		// Upload endpoint
		api.POST("/videos", uploadHandler.HandleUpload)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		sharedlog.Info(fmt.Sprintf("Starting video upload service on port %s", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sharedlog.Error(fmt.Sprintf("Failed to start server: %v", err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	// Accept syscall.SIGINT and syscall.SIGTERM (CTRL+C)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// Block until a signal is received
	<-quit

	sharedlog.Info("Shutting down video upload service...")

	// Create a deadline to wait for current operations to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown the server
	if err := srv.Shutdown(ctx); err != nil {
		sharedlog.Error(fmt.Sprintf("Server forced to shutdown: %v", err))
	}

	sharedlog.Info("Video upload service exited gracefully")
}

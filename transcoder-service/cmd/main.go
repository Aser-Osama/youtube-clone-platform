package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"youtube-clone-platform/transcoder-service/internal/config"
	"youtube-clone-platform/transcoder-service/internal/events"
	"youtube-clone-platform/transcoder-service/internal/handler"
	"youtube-clone-platform/transcoder-service/internal/service"
	"youtube-clone-platform/transcoder-service/internal/storage"
	"youtube-clone-platform/transcoder-service/internal/transcoder"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize MinIO client
	minioClient, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKeyID, cfg.MinIO.SecretAccessKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}

	// Create MinIO storage
	minioStorage := storage.NewMinIOStorage(
		minioClient,
		cfg.MinIO.BucketName,
		cfg.MinIO.ProcessedBucket,
		cfg.MinIO.OriginalPrefix,
		cfg.MinIO.HLSPrefix,
		cfg.MinIO.MP4Prefix,
		cfg.MinIO.ThumbnailPrefix,
	)

	// Check MinIO health
	if err := minioStorage.CheckHealth(context.Background()); err != nil {
		log.Fatalf("MinIO health check failed: %v", err)
	}

	// Create transcoder
	transcoderInstance, err := transcoder.NewTranscoder(
		cfg.FFmpeg.Path,
		cfg.FFmpeg.Threads,
		cfg.FFmpeg.Preset,
		cfg.FFmpeg.CRF,
		cfg.FFmpeg.SegmentLength,
		cfg.FFmpeg.OutputFormats,
		cfg.FFmpeg.OutputQualities,
		cfg.Processing.TempDir,
	)
	if err != nil {
		log.Fatalf("Failed to create transcoder: %v", err)
	}

	// Create Kafka consumer
	consumer, err := events.NewKafkaConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.Topic,
		cfg.Kafka.GroupID,
	)
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	}

	// Create Kafka producer
	producer := events.NewKafkaProducer(
		cfg.Kafka.Brokers,
		"transcoding-complete",
	)

	// Create transcoder service
	transcoderService := service.NewTranscoderService(
		minioStorage,
		transcoderInstance,
		consumer,
		producer,
		cfg.Processing.MaxConcurrentJobs,
		cfg.Processing.JobTimeout,
		cfg.Processing.TempDir,
	)

	// Create Gin router
	router := gin.Default()

	// Initialize handlers
	minioStorageImpl, ok := minioStorage.(*storage.MinIOStorage)
	if !ok {
		log.Fatalf("Failed to cast storage to MinIOStorage")
	}

	healthHandler := handler.NewHealthHandler(
		minioStorageImpl,
		cfg.Kafka.Brokers,
		"transcoding-complete",
	)

	// Run initial health check
	log.Printf("Running initial health check...")
	status := healthHandler.RunHealthCheck()
	if status.Status != "ok" {
		log.Printf("Initial health check failed:")
		for service, state := range status.Dependencies {
			log.Printf("- %s: %s", service, state)
		}
		log.Fatal("Service cannot start due to health check failures")
	}
	log.Printf("Initial health check passed")

	// Setup routes
	api := router.Group("/api/v1/transcoder")
	{
		api.GET("/health", healthHandler.HandleHealthCheck)
		// TODO: Add transcoder routes once handler is implemented
		api.POST("/jobs", func(c *gin.Context) {
			// TODO: Implement job creation handler
			c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
		})
		api.GET("/jobs/:id", func(c *gin.Context) {
			// TODO: Implement job status handler
			c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
		})
	}

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Create a context that will be canceled on SIGINT or SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received signal: %v", sig)
		cancel()
	}()

	// Start the service
	log.Printf("Starting transcoder service on port %s...", cfg.Port)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Start consuming messages
	go func() {
		if err := transcoderService.Start(ctx); err != nil {
			log.Printf("Error starting transcoder service: %v", err)
			cancel()
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Create a shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Stop the server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}

	// Stop the service
	log.Printf("Stopping transcoder service...")
	transcoderService.Stop()

	log.Printf("Transcoder service stopped")
}

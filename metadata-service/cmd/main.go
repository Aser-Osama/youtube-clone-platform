package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"youtube-clone-platform/metadata-service/internal/config"
	"youtube-clone-platform/metadata-service/internal/db"
	kafkautil "youtube-clone-platform/metadata-service/internal/events"
	"youtube-clone-platform/metadata-service/internal/handler"
	"youtube-clone-platform/metadata-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/segmentio/kafka-go"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Initialize database
	db, err := initDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Kafka consumer
	kafkaConsumer, err := initKafkaConsumer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Kafka consumer: %v", err)
	}
	defer kafkaConsumer.Close()

	// Initialize MinIO client
	minioClient, err := initMinioClient(cfg.MinIO)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}

	// Create service instances
	metadataService := service.NewMetadataService(db, minioClient)
	metadataHandler := handler.NewMetadataHandler(metadataService)

	// Setup HTTP server
	router := gin.Default()
	metadataHandler.RegisterRoutes(router)

	// Create context for consumer that can be canceled
	consumerCtx, consumerCancel := context.WithCancel(context.Background())

	// Start Kafka consumer in a goroutine
	go func() {
		if err := kafkautil.StartConsumer(consumerCtx, kafkaConsumer, metadataService); err != nil && err != context.Canceled {
			log.Printf("Kafka consumer error: %v", err)
		}
	}()

	// Create HTTP server
	port := cfg.ServerPort
	if port == "" {
		port = "8081"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("Metadata service running on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down metadata service...")

	// Cancel the consumer context to stop the Kafka consumer
	consumerCancel()

	// Create a deadline to wait for current operations to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown the server
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Metadata service exited gracefully")
}

func initDB(databasePath string) (*db.Store, error) {
	return db.New(databasePath)
}

func initKafkaConsumer(cfg *config.Config) (*kafka.Reader, error) {
	return kafkautil.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID)
}

func initMinioClient(minioCfg config.MinIOConfig) (*minio.Client, error) {
	return minio.New(minioCfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioCfg.AccessKey, minioCfg.SecretKey, ""),
		Secure: minioCfg.UseSSL,
	})
}

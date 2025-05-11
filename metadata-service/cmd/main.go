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

	// Initialize Kafka consumers
	uploadConsumer, err := initKafkaConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID)
	if err != nil {
		log.Fatalf("Failed to initialize video upload Kafka consumer: %v", err)
	}
	defer uploadConsumer.Close()

	// Initialize transcoding complete consumer
	transcodingConsumer, err := initKafkaConsumer(cfg.KafkaBrokers, cfg.TranscodingTopic, cfg.TranscodingGroupID)
	if err != nil {
		log.Fatalf("Failed to initialize transcoding complete Kafka consumer: %v", err)
	}
	defer transcodingConsumer.Close()

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

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	metadataHandler.RegisterRoutes(router)

	// Create context for consumer that can be canceled
	consumerCtx, consumerCancel := context.WithCancel(context.Background())

	// Start Kafka consumers in goroutines
	go func() {
		if err := kafkautil.StartConsumer(consumerCtx, uploadConsumer, metadataService); err != nil && err != context.Canceled {
			log.Printf("Video upload Kafka consumer error: %v", err)
		}
	}()

	go func() {
		if err := kafkautil.StartTranscodingCompleteConsumer(consumerCtx, transcodingConsumer, metadataService); err != nil && err != context.Canceled {
			log.Printf("Transcoding complete Kafka consumer error: %v", err)
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

func initKafkaConsumer(brokers []string, topic, groupID string) (*kafka.Reader, error) {
	return kafkautil.NewConsumer(brokers, topic, groupID)
}

func initMinioClient(minioCfg config.MinIOConfig) (*minio.Client, error) {
	return minio.New(minioCfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioCfg.AccessKey, minioCfg.SecretKey, ""),
		Secure: minioCfg.UseSSL,
	})
}

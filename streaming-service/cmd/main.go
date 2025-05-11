package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"youtube-clone-platform/streaming-service/internal/config"
	"youtube-clone-platform/streaming-service/internal/handler"
	"youtube-clone-platform/streaming-service/internal/storage"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize MinIO storage
	minioStorage, err := storage.NewMinIOStorage(&storage.MinIOConfig{
		Endpoint:        cfg.MinIO.Endpoint,
		AccessKey:       cfg.MinIO.AccessKey,
		SecretKey:       cfg.MinIO.SecretKey,
		UseSSL:          cfg.MinIO.UseSSL,
		BucketName:      cfg.MinIO.BucketName,
		HLSPrefix:       cfg.MinIO.HLSPrefix,
		MP4Prefix:       cfg.MinIO.MP4Prefix,
		ThumbnailPrefix: cfg.MinIO.ThumbnailPrefix,
		URLExpiry:       cfg.MinIO.URLExpiry,
	})
	if err != nil {
		log.Fatalf("Failed to initialize MinIO storage: %v", err)
	}

	// Check MinIO health
	if err := minioStorage.CheckHealth(context.Background()); err != nil {
		log.Fatalf("MinIO health check failed: %v", err)
	}

	// Create Gin router
	router := gin.Default()

	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Create handlers
	streamHandler := handler.NewStreamHandler(minioStorage)
	healthHandler := handler.NewHealthHandler(minioStorage)

	// Serve static files at root level
	router.Static("/static", "./static")
	router.StaticFile("/", "./static/index.html")
	router.StaticFile("/app.js", "./static/app.js")
	router.StaticFile("/favicon.ico", "./static/favicon.ico")

	// Register API routes
	api := router.Group("/api/v1/streaming")
	{
		api.GET("/health", healthHandler.HandleHealthCheck)
		api.GET("/videos/:videoID/hls/manifest", streamHandler.HandleHLSManifest)
		api.GET("/videos/:videoID/hls/segments/:segment", streamHandler.HandleHLSSegment)
		api.GET("/videos/:videoID/hls/:resolution/playlist", streamHandler.HandleHLSPlaylist)
		api.GET("/videos/:videoID/hls/:resolution/:segment", streamHandler.HandleHLSSegment)
		api.GET("/videos/:videoID/mp4", streamHandler.HandleMP4)
		api.GET("/videos/:videoID/mp4/qualities", streamHandler.ListMP4Qualities)
		api.GET("/videos/:videoID/thumbnail", streamHandler.HandleThumbnail)

		// Also serve static files under /api/v1/streaming
		api.Static("/static", "./static")
		api.StaticFile("/", "./static/index.html")
		api.StaticFile("/app.js", "./static/app.js")
		api.StaticFile("/favicon.ico", "./static/favicon.ico")
	}

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
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

	// Start the server
	log.Printf("Starting streaming service on port %s...", cfg.ServerPort)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
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

	log.Printf("Streaming service stopped")
}

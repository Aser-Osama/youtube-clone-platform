package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"youtube-clone-platform/api-gateway/internal/config"
	"youtube-clone-platform/api-gateway/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create and start the gateway service
	gateway, err := service.NewGatewayService(cfg)
	if err != nil {
		log.Fatalf("Failed to create gateway service: %v", err)
	}

	// Create context that listens for the interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start the service
	go func() {
		if err := gateway.Start(ctx); err != nil {
			log.Printf("Error starting gateway service: %v", err)
			stop()
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	log.Println("Shutting down gracefully...")
}

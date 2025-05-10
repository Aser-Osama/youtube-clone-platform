package main

import (
	"auth-service/internal/auth"
	"auth-service/internal/config"
	"auth-service/internal/db"
	"auth-service/internal/handler"
	"auth-service/internal/middleware"
	"auth-service/internal/service"

	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth/gothic"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize session store
	store := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	gothic.Store = store

	// Initialize JWT (keys)
	if err := auth.InitJWT(cfg); err != nil {
		log.Fatalf("Failed to initialize JWT: %v", err)
	}

	// Connect to SQLite database
	sqlDB, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}

	// Ensure database is closed properly
	defer sqlDB.Close()

	// Initialize repository & services
	repo := db.NewRepo(sqlDB)
	authService := service.NewAuthService(repo)

	// Initialize Goth for Google OAuth
	auth.InitGoth(cfg)

	authHandler := handler.NewAuthHandler(authService)

	// Start Gin engine
	r := gin.Default()

	// Routes
	r.GET("/auth/google/login", authHandler.GoogleLogin)
	r.GET("/auth/google/callback", authHandler.GoogleCallback)
	r.POST("/auth/logout", middleware.JWTAuth(), authHandler.Logout)
	r.POST("/auth/refresh", authHandler.Refresh)

	// Health check
	r.GET("/auth/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now()})
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Println("Auth service running on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	// Accept syscall.SIGINT and syscall.SIGTERM (CTRL+C)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// Block until a signal is received
	<-quit
	log.Println("Shutting down auth server...")

	// Create a deadline to wait for current operations to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown the server
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Auth server exited gracefully")
}

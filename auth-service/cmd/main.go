package main

import (
	"auth-service/internal/auth"
	"auth-service/internal/config"
	"auth-service/internal/db"
	"auth-service/internal/handler"
	"auth-service/internal/middleware"
	"auth-service/internal/service"

	"database/sql"
	"log"
	"net/http"
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

	// Start server
	log.Println("Auth service running on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

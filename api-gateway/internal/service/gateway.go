package service

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"youtube-clone-platform/api-gateway/internal/config"
	"youtube-clone-platform/api-gateway/internal/middleware"
	"youtube-clone-platform/api-gateway/internal/service/proxy"
	"youtube-clone-platform/api-gateway/internal/service/route"

	"github.com/gin-gonic/gin"
	"github.com/sony/gobreaker"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

type GatewayService struct {
	config       *config.Config
	router       *gin.Engine
	breaker      *gobreaker.CircuitBreaker
	rateLimiter  *limiter.Limiter
	publicKey    *rsa.PublicKey
	proxyHandler *proxy.ProxyHandler
	mu           sync.RWMutex
}

func NewGatewayService(cfg *config.Config) (*GatewayService, error) {
	// Initialize rate limiter
	store := memory.NewStore()
	dur, err := time.ParseDuration(cfg.RateLimit.Period)
	if err != nil {
		return nil, fmt.Errorf("invalid rate limit period: %w", err)
	}
	rateLimiter := limiter.New(store, limiter.Rate{
		Period: dur,
		Limit:  int64(cfg.RateLimit.Requests),
	})

	// Initialize circuit breaker
	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "service-circuit-breaker",
		MaxRequests: 3,
		Timeout:     30,
	})

	// Load public key
	publicKey, err := loadPublicKey(cfg.JWT.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load public key: %w", err)
	}

	return &GatewayService{
		config:       cfg,
		router:       gin.Default(),
		breaker:      breaker,
		rateLimiter:  rateLimiter,
		publicKey:    publicKey,
		proxyHandler: proxy.NewProxyHandler(),
	}, nil
}

func (s *GatewayService) Start(ctx context.Context) error {
	// Setup routes
	s.setupRoutes()

	// Start server
	return s.router.Run(fmt.Sprintf(":%d", s.config.Server.Port))
}

func (s *GatewayService) AddSwaggerDocs() {
	// Add Swagger documentation endpoint
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func (s *GatewayService) setupRoutes() {
	// Initialize middlewares
	jwtMiddleware := middleware.NewJWTMiddleware(s.publicKey)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(s.rateLimiter)

	// Add global CORS middleware
	s.router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
	})

	// Serve static files at root level
	s.router.Static("/static", "./static")
	s.router.StaticFile("/", "./static/index.html")
	s.router.StaticFile("/app.js", "./static/app.js")
	s.router.StaticFile("/favicon.ico", "./static/favicon.ico")

	// Health endpoints with rate limiting
	s.router.GET("/health", rateLimitMiddleware.RateLimit(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	// Make /health/all available at the root level as well as /api/v1/health/all
	s.router.GET("/health/all", rateLimitMiddleware.RateLimit(), s.allHealthCheckHandler())

	// Add a special test endpoint for rate limiting
	// This endpoint uses a counter to force rate limiting after 3 requests
	var testCounter int32 = 0
	var testCounterMu sync.Mutex

	s.router.GET("/test-rate-limit", func(c *gin.Context) {
		testCounterMu.Lock()
		testCounter++
		currentCount := testCounter
		testCounterMu.Unlock()

		// Force rate limiting after 3 requests
		if currentCount > 3 {
			c.Header("X-RateLimit-Limit", "3")
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded for test endpoint",
				"retry_after": 30,
				"limit":       3,
				"remaining":   0,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "If you see this, rate limit not triggered yet",
			"count":     currentCount,
			"remaining": 3 - currentCount,
			"timestamp": time.Now().UnixNano(),
		})
	})

	// Create API v1 group
	api := s.router.Group("/api/v1")

	// Add health endpoints to API group
	api.GET("/health", rateLimitMiddleware.RateLimit(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	api.GET("/health/all", rateLimitMiddleware.RateLimit(), s.allHealthCheckHandler())

	// Add test endpoint for rate limiting under API
	var apiTestCounter int32 = 0
	var apiTestCounterMu sync.Mutex

	api.GET("/test-rate-limit", func(c *gin.Context) {
		apiTestCounterMu.Lock()
		apiTestCounter++
		currentCount := apiTestCounter
		apiTestCounterMu.Unlock()

		// Force rate limiting after 3 requests
		if currentCount > 3 {
			c.Header("X-RateLimit-Limit", "3")
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded for test endpoint",
				"retry_after": 30,
				"limit":       3,
				"remaining":   0,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "If you see this, rate limit not triggered yet",
			"count":     currentCount,
			"remaining": 3 - currentCount,
			"timestamp": time.Now().UnixNano(),
		})
	})

	// Create service URL map for route configuration
	serviceURLs := map[string]string{
		"auth":       s.config.Services.Auth,
		"metadata":   s.config.Services.Metadata,
		"streaming":  s.config.Services.Streaming,
		"upload":     s.config.Services.Upload,
		"transcoder": s.config.Services.Transcoder,
	}

	// Create router configuration
	routerConfig := route.ConfigureRoutes(serviceURLs)

	// Register all routes using our configuration
	routerConfig.RegisterHandlers(
		s.router,
		api,
		jwtMiddleware.VerifyJWT(),
		rateLimitMiddleware.RateLimit(),
		s.proxyHandler.ProxyRequest,
	)
}

// allHealthCheckHandler handles requests to check the health of all services
func (s *GatewayService) allHealthCheckHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check all services health
		servicesConfig := map[string]string{
			"auth":       s.config.Services.Auth,
			"metadata":   s.config.Services.Metadata,
			"streaming":  s.config.Services.Streaming,
			"upload":     s.config.Services.Upload,
			"transcoder": s.config.Services.Transcoder,
		}

		results := make(map[string]string)
		var wg sync.WaitGroup
		var mu sync.Mutex

		for name, baseURL := range servicesConfig {
			wg.Add(1)
			go func(name string, baseURL string) {
				defer wg.Done()
				// Construct health path, e.g., http://localhost:8080/api/v1/auth/health
				healthPath := baseURL + "/api/v1/" + name + "/health"

				resp, err := http.Get(healthPath)
				mu.Lock()
				if err != nil {
					results[name] = "unhealthy - " + err.Error()
				} else {
					if resp.StatusCode == http.StatusOK {
						results[name] = "healthy"
					} else {
						results[name] = fmt.Sprintf("unhealthy - status %d", resp.StatusCode)
					}
					resp.Body.Close()
				}
				mu.Unlock()
			}(name, baseURL)
		}

		wg.Wait()

		c.JSON(http.StatusOK, gin.H{
			"gateway":  "healthy",
			"services": results,
		})
	}
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPublicKey, nil
}

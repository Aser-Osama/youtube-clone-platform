package service

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"youtube-clone-platform/api-gateway/internal/config"
	"youtube-clone-platform/api-gateway/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sony/gobreaker"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

type GatewayService struct {
	config      *config.Config
	router      *gin.Engine
	breaker     *gobreaker.CircuitBreaker
	rateLimiter *limiter.Limiter
	publicKey   *rsa.PublicKey
	mu          sync.RWMutex
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
		config:      cfg,
		router:      gin.Default(),
		breaker:     breaker,
		rateLimiter: rateLimiter,
		publicKey:   publicKey,
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

	// Streaming service routes
	s.router.GET("/streaming/*path", s.proxyRequest(s.config.Services.Streaming))

	// Metadata service routes
	s.router.GET("/metadata/*path", s.proxyRequest(s.config.Services.Metadata))

	// API v1 routes
	api := s.router.Group("/api/v1")
	{
		// Health endpoints with rate limiting
		api.GET("/health", rateLimitMiddleware.RateLimit(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		api.GET("/health/all", rateLimitMiddleware.RateLimit(), func(c *gin.Context) {
			// Check all services health
			services := map[string]string{
				"auth":      s.config.Services.Auth,
				"metadata":  s.config.Services.Metadata,
				"streaming": s.config.Services.Streaming,
				"upload":    s.config.Services.Upload,
			}

			results := make(map[string]string)
			for name, url := range services {
				resp, err := http.Get(url + "/api/v1/health")
				if err != nil {
					results[name] = "unhealthy"
					continue
				}
				if resp.StatusCode == http.StatusOK {
					results[name] = "healthy"
				} else {
					results[name] = "unhealthy"
				}
				resp.Body.Close()
			}

			c.JSON(http.StatusOK, gin.H{
				"gateway":  "healthy",
				"services": results,
			})
		})

		// Auth service routes
		auth := api.Group("/auth")
		{
			// Public auth endpoints
			auth.GET("/google/login", s.authReverseProxy())
			auth.GET("/google/callback", s.authReverseProxy())
			auth.GET("/health", s.authReverseProxy())

			// Protected auth endpoints
			protectedAuth := auth.Group("")
			protectedAuth.Use(jwtMiddleware.VerifyJWT())
			{
				protectedAuth.POST("/refresh", s.authReverseProxy())
				protectedAuth.POST("/logout", s.authReverseProxy())
			}
		}

		// Streaming service routes
		streaming := api.Group("/streaming")
		streaming.Use(rateLimitMiddleware.RateLimit())
		{
			streaming.GET("/videos/:id", s.proxyRequest(s.config.Services.Streaming))
		}

		// Upload service routes
		upload := api.Group("/upload")
		upload.Use(rateLimitMiddleware.RateLimit())
		{
			upload.POST("/videos", s.proxyRequest(s.config.Services.Upload))
		}

		// Metadata service routes
		metadata := api.Group("/metadata")
		metadata.Use(rateLimitMiddleware.RateLimit(), jwtMiddleware.VerifyJWT())
		{
			metadata.GET("/videos", s.proxyRequest(s.config.Services.Metadata))
			metadata.GET("/videos/:id", s.proxyRequest(s.config.Services.Metadata))
		}
	}
}

func (s *GatewayService) proxyRequest(targetURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		target, err := url.Parse(targetURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid target URL"})
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(target)

		// Customize the proxy's director to modify the request
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)

			// Debug logging
			fmt.Printf("Proxying request to %s: %s %s\n", targetURL, req.Method, req.URL.Path)

			// Add any necessary headers
			req.Header.Set("X-Forwarded-Host", req.Host)
			req.Header.Set("X-Forwarded-Proto", "http")

			// Preserve the original method
			req.Method = c.Request.Method

			// Copy all headers from the original request
			for key, values := range c.Request.Header {
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}

			// Modify the path to include /api/v1 prefix
			req.URL.Path = "/api/v1" + req.URL.Path
		}

		// Add error handling
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			fmt.Printf("Proxy error for %s: %v\n", targetURL, err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(fmt.Sprintf("Proxy error: %v", err)))
		}

		// Add response modifier
		proxy.ModifyResponse = func(resp *http.Response) error {
			fmt.Printf("Received response from %s: %d\n", targetURL, resp.StatusCode)
			return nil
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func (s *GatewayService) authReverseProxy() gin.HandlerFunc {
	target, _ := url.Parse(s.config.Services.Auth)
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Customize the proxy's director to modify the request
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Debug logging
		fmt.Printf("Proxying request to auth service: %s %s\n", req.Method, req.URL.Path)

		// Add any necessary headers
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Forwarded-Proto", "http")

		// Preserve the original method
		req.Method = req.Method

		// Copy all headers from the original request
		for key, values := range req.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		// Modify the path to include /api/v1 prefix
		req.URL.Path = "/api/v1" + req.URL.Path
	}

	// Add error handling
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		fmt.Printf("Auth proxy error: %v\n", err)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(fmt.Sprintf("Proxy error: %v", err)))
	}

	// Add response modifier
	proxy.ModifyResponse = func(resp *http.Response) error {
		fmt.Printf("Received response from auth service: %d\n", resp.StatusCode)
		return nil
	}

	return func(c *gin.Context) {
		// Debug logging
		fmt.Printf("Received request: %s %s\n", c.Request.Method, c.Request.URL.Path)

		// For protected endpoints, verify JWT first
		if strings.HasSuffix(c.Request.URL.Path, "/refresh") || strings.HasSuffix(c.Request.URL.Path, "/logout") {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header is required"})
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
				return
			}

			tokenString := parts[1]
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return s.publicKey, nil
			})

			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
				return
			}

			if !token.Valid {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
				return
			}
		}

		proxy.ServeHTTP(c.Writer, c.Request)
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

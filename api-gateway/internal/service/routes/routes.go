package routes

import (
	"net/http"
	"strings"
	"youtube-clone-platform/api-gateway/internal/config"
	"youtube-clone-platform/api-gateway/internal/middleware"
	"youtube-clone-platform/api-gateway/internal/service/health"
	"youtube-clone-platform/api-gateway/internal/service/proxy"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/ulule/limiter/v3"
)

// Router handles the setup of all API routes
type Router struct {
	config        *config.Config
	engine        *gin.Engine
	proxy         *proxy.ReverseProxy
	healthChecker *health.HealthChecker
}

// NewRouter creates a new Router instance
func NewRouter(cfg *config.Config, engine *gin.Engine) *Router {
	// Create service URLs map for health checker
	serviceURLs := map[string]string{
		"auth":       cfg.Services.Auth,
		"metadata":   cfg.Services.Metadata,
		"streaming":  cfg.Services.Streaming,
		"upload":     cfg.Services.Upload,
		"transcoder": cfg.Services.Transcoder,
	}

	return &Router{
		config:        cfg,
		engine:        engine,
		proxy:         proxy.NewReverseProxy(),
		healthChecker: health.NewHealthChecker(serviceURLs),
	}
}

// Setup configures all the routes and middlewares
func (r *Router) Setup(jwtMiddleware *middleware.JWTMiddleware, rateLimiter *limiter.Limiter) {
	// Create rate limit middleware
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimiter)

	// Add global CORS middleware
	r.engine.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.Status(http.StatusOK)
			c.Abort()
			return
		}

		c.Next()
	})

	// Setup API routes first
	r.setupPublicRoutes(rateLimitMiddleware)
	r.setupProtectedRoutes(jwtMiddleware, rateLimitMiddleware)

	// Setup frontend routes last
	r.setupFrontendRoutes()
}

// Setup public routes
func (r *Router) setupPublicRoutes(rateLimitMiddleware *middleware.RateLimitMiddleware) {
	public := r.engine.Group("/")
	{
		// Gateway health endpoints with rate limiting
		public.GET("/health", rateLimitMiddleware.RateLimit(), r.healthChecker.HealthCheckHandler())
		public.GET("/health/all", rateLimitMiddleware.RateLimit(), r.healthChecker.AllServicesHealthCheckHandler())

		// Auth endpoints
		r.setupAuthPublicRoutes(public.Group("/api/v1/auth"))

		// Streaming service routes
		r.setupStreamingPublicRoutes(public.Group("/api/v1/streaming"), rateLimitMiddleware)

		// Upload service routes
		r.setupUploadPublicRoutes(public.Group("/api/v1/upload"), rateLimitMiddleware)

		// Metadata service public routes
		r.setupMetadataPublicRoutes(public.Group("/api/v1/metadata"), rateLimitMiddleware)

		// Transcoder service routes
		r.setupTranscoderPublicRoutes(public.Group("/api/v1/transcoder"), rateLimitMiddleware)
	}
}

// Setup protected routes that require authentication
func (r *Router) setupProtectedRoutes(jwtMiddleware *middleware.JWTMiddleware, rateLimitMiddleware *middleware.RateLimitMiddleware) {
	protected := r.engine.Group("/")
	protected.Use(jwtMiddleware.VerifyJWT())
	{
		// Auth service protected routes
		r.setupAuthProtectedRoutes(protected.Group("/api/v1/auth"))

		// Metadata service protected routes
		r.setupMetadataProtectedRoutes(protected.Group("/api/v1/metadata"), rateLimitMiddleware)

		// Upload service protected routes
		r.setupUploadProtectedRoutes(protected.Group("/api/v1/upload"), rateLimitMiddleware)

		// Transcoder service protected routes
		r.setupTranscoderProtectedRoutes(protected.Group("/api/v1/transcoder"), rateLimitMiddleware)
	}
}

// Setup frontend routes to serve the UI directly from the API Gateway
func (r *Router) setupFrontendRoutes() {
	// Initialize Swagger
	swaggerConfig := ginSwagger.URL("/swagger/doc.json")
	swaggerHandler := ginSwagger.WrapHandler(swaggerFiles.Handler, swaggerConfig)

	// Handle /api to Swagger documentation redirect
	r.engine.GET("/api", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	// Handle /api/v1 to prevent unwanted redirects
	r.engine.GET("/api/v1", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "API Gateway is running",
			"version": "v1",
		})
	})

	// Serve Swagger UI
	r.engine.GET("/swagger/*any", swaggerHandler)

	// Serve static files
	r.engine.Static("/static", "./static")
	r.engine.StaticFile("/app.js", "./static/app.js")
	r.engine.StaticFile("/favicon.ico", "./static/favicon.ico")

	// Serve index.html for root path
	r.engine.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	// Handle all other routes by serving index.html
	r.engine.NoRoute(func(c *gin.Context) {
		// Skip API routes
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1") {
			c.Next()
			return
		}
		// Skip Swagger routes
		if strings.HasPrefix(c.Request.URL.Path, "/swagger") {
			c.Next()
			return
		}
		// Skip static file routes
		if strings.HasPrefix(c.Request.URL.Path, "/static") {
			c.Next()
			return
		}
		// Serve index.html for all other routes
		c.File("./static/index.html")
	})
}

// Setup auth service public routes
func (r *Router) setupAuthPublicRoutes(group *gin.RouterGroup) {
	group.GET("/google/login", r.proxy.ProxyRequest(r.config.Services.Auth, "/api/v1/auth/google/login", true))
	group.GET("/google/callback", r.proxy.ProxyRequest(r.config.Services.Auth, "/api/v1/auth/google/callback", true))
	group.GET("/health", r.proxy.ProxyRequest(r.config.Services.Auth, "/api/v1/auth/health", true))
}

// Setup auth service protected routes
func (r *Router) setupAuthProtectedRoutes(group *gin.RouterGroup) {
	group.POST("/refresh", r.proxy.ProxyRequest(r.config.Services.Auth, "/api/v1/auth/refresh", true))
	group.POST("/logout", r.proxy.ProxyRequest(r.config.Services.Auth, "/api/v1/auth/logout", true))
}

// Setup streaming service public routes
func (r *Router) setupStreamingPublicRoutes(group *gin.RouterGroup, rateLimitMiddleware *middleware.RateLimitMiddleware) {
	group.Use(rateLimitMiddleware.RateLimit())
	group.GET("/health", r.proxy.ProxyRequest(r.config.Services.Streaming, "/health", true))
	group.GET("/videos/:id", r.proxy.ProxyRequest(r.config.Services.Streaming, "/videos/:id", true))
}

// Setup metadata service public routes
func (r *Router) setupMetadataPublicRoutes(group *gin.RouterGroup, rateLimitMiddleware *middleware.RateLimitMiddleware) {
	group.Use(rateLimitMiddleware.RateLimit())
	group.GET("/health", r.proxy.ProxyRequest(r.config.Services.Metadata, "/health", true))
}

// Setup metadata service protected routes
func (r *Router) setupMetadataProtectedRoutes(group *gin.RouterGroup, rateLimitMiddleware *middleware.RateLimitMiddleware) {
	group.Use(rateLimitMiddleware.RateLimit())
	group.GET("/videos", r.proxy.ProxyRequest(r.config.Services.Metadata, "/api/v1/metadata/videos", true))
	group.GET("/videos/:id", r.proxy.ProxyRequest(r.config.Services.Metadata, "/api/v1/metadata/videos/:id", true))
	group.POST("/videos", r.proxy.ProxyRequest(r.config.Services.Metadata, "/api/v1/metadata/videos", true))
	group.PUT("/videos/:id", r.proxy.ProxyRequest(r.config.Services.Metadata, "/api/v1/metadata/videos/:id", true))
	group.DELETE("/videos/:id", r.proxy.ProxyRequest(r.config.Services.Metadata, "/api/v1/metadata/videos/:id", true))
}

// Setup upload service public routes
func (r *Router) setupUploadPublicRoutes(group *gin.RouterGroup, rateLimitMiddleware *middleware.RateLimitMiddleware) {
	group.Use(rateLimitMiddleware.RateLimit())
	group.GET("/health", r.proxy.ProxyRequest(r.config.Services.Upload, "/health", true))
}

// Setup upload service protected routes
func (r *Router) setupUploadProtectedRoutes(group *gin.RouterGroup, rateLimitMiddleware *middleware.RateLimitMiddleware) {
	group.Use(rateLimitMiddleware.RateLimit())
	group.POST("/videos", r.proxy.ProxyRequest(r.config.Services.Upload, "/api/v1/upload/videos", true))
	group.POST("/videos/process", r.proxy.ProxyRequest(r.config.Services.Upload, "/api/v1/upload/videos/process", true))
}

// Setup transcoder service public routes
func (r *Router) setupTranscoderPublicRoutes(group *gin.RouterGroup, rateLimitMiddleware *middleware.RateLimitMiddleware) {
	group.Use(rateLimitMiddleware.RateLimit())
	group.GET("/health", r.proxy.ProxyRequest(r.config.Services.Transcoder, "/health", true))
}

// Setup transcoder service protected routes
func (r *Router) setupTranscoderProtectedRoutes(group *gin.RouterGroup, rateLimitMiddleware *middleware.RateLimitMiddleware) {
	group.Use(rateLimitMiddleware.RateLimit())
	group.POST("/jobs", r.proxy.ProxyRequest(r.config.Services.Transcoder, "/api/v1/transcoder/jobs", true))
	group.GET("/jobs/:id", r.proxy.ProxyRequest(r.config.Services.Transcoder, "/api/v1/transcoder/jobs/:id", true))
}

package route

// helper functions for boolean pointers
func boolPtr(b bool) *bool {
	return &b
}

// ConfigureRoutes creates a complete router configuration with all services
func ConfigureRoutes(config map[string]string) *RouterConfig {
	router := NewRouterConfig()

	// Add each service configuration
	configureAuthRoutes(router, config["auth"])
	configureStreamingRoutes(router, config["streaming"])
	configureMetadataRoutes(router, config["metadata"])
	configureUploadRoutes(router, config["upload"])
	configureTranscoderRoutes(router, config["transcoder"])

	return router
}

// configureAuthRoutes configures routes for the auth service
func configureAuthRoutes(router *RouterConfig, baseURL string) {
	auth := router.AddService("auth", baseURL, false)

	// Public endpoints
	auth.AddEndpoint("GET", "/google/login", "Google OAuth login", boolPtr(false))
	auth.AddEndpoint("GET", "/google/callback", "Google OAuth callback", boolPtr(false))
	auth.AddEndpoint("GET", "/health", "Health check endpoint", boolPtr(false))

	// Protected endpoints
	auth.AddEndpoint("POST", "/refresh", "Refresh authentication token", boolPtr(true))
	auth.AddEndpoint("POST", "/logout", "Logout and invalidate token", boolPtr(true))
}

// configureStreamingRoutes configures routes for the streaming service
func configureStreamingRoutes(router *RouterConfig, baseURL string) {
	streaming := router.AddService("streaming", baseURL, false)

	// Health check
	streaming.AddEndpoint("GET", "/health", "Health check endpoint", boolPtr(false))

	// Video access endpoints - all use videoID consistently
	streaming.AddEndpoint("GET", "/videos/:videoID", "Get video by ID", boolPtr(false))
	streaming.AddEndpoint("GET", "/videos/:videoID/thumbnail", "Get video thumbnail", boolPtr(false))

	// HLS streaming endpoints
	streaming.AddEndpoint("GET", "/videos/:videoID/hls/manifest", "Get HLS manifest", boolPtr(false))
	streaming.AddEndpoint("GET", "/videos/:videoID/hls/:resolution/playlist", "Get HLS playlist for specific resolution", boolPtr(false))
	streaming.AddEndpoint("GET", "/videos/:videoID/hls/:resolution/:segment", "Get HLS segment", boolPtr(false))
	streaming.AddEndpoint("GET", "/videos/:videoID/hls/segments/:segment", "Get HLS segment directly", boolPtr(false))

	// MP4 endpoints
	streaming.AddEndpoint("GET", "/videos/:videoID/mp4", "Get MP4 video", boolPtr(false))
	streaming.AddEndpoint("GET", "/videos/:videoID/mp4/qualities", "List available MP4 qualities", boolPtr(false))
}

// configureMetadataRoutes configures routes for the metadata service
func configureMetadataRoutes(router *RouterConfig, baseURL string) {
	metadata := router.AddService("metadata", baseURL, false)

	// Health check
	metadata.AddEndpoint("GET", "/health", "Health check endpoint", boolPtr(false))

	// Public video metadata endpoints
	metadata.AddEndpoint("GET", "/public/videos", "List public videos", boolPtr(false))
	metadata.AddEndpoint("GET", "/public/videos/:videoID", "Get public video details", boolPtr(false))

	// Protected video metadata endpoints - using videoID consistently
	metadata.AddEndpoint("GET", "/videos", "List user's videos", boolPtr(false))
	metadata.AddEndpoint("GET", "/videos/:videoID", "Get video details", boolPtr(false))
	metadata.AddEndpoint("POST", "/videos", "Create new video metadata", boolPtr(false))
	metadata.AddEndpoint("PUT", "/videos/:videoID", "Update video metadata", boolPtr(false))
	metadata.AddEndpoint("DELETE", "/videos/:videoID", "Delete video", boolPtr(false))
}

// configureUploadRoutes configures routes for the upload service
func configureUploadRoutes(router *RouterConfig, baseURL string) {
	upload := router.AddService("upload", baseURL, true)

	// Public endpoints
	upload.AddEndpoint("GET", "/health", "Health check endpoint", boolPtr(false))

	// Protected endpoints
	upload.AddEndpoint("POST", "/videos", "Upload a new video", nil)
	upload.AddEndpoint("POST", "/videos/process", "Process uploaded video", nil)
}

// configureTranscoderRoutes configures routes for the transcoder service
func configureTranscoderRoutes(router *RouterConfig, baseURL string) {
	transcoder := router.AddService("transcoder", baseURL, true)

	// Public endpoints
	transcoder.AddEndpoint("GET", "/health", "Health check endpoint", boolPtr(false))
	transcoder.AddEndpoint("GET", "/status", "Get transcoder status", boolPtr(false))

	// Protected endpoints
	transcoder.AddEndpoint("POST", "/jobs", "Create new transcoding job", nil)
	transcoder.AddEndpoint("GET", "/jobs/:jobID", "Get transcoding job status", nil)
}

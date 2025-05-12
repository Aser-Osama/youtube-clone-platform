package route

import (
	"github.com/gin-gonic/gin"
)

// RouterConfig holds the complete configuration for all service routes
type RouterConfig struct {
	// Services maps service name to its endpoint configuration
	Services map[string]*ServiceConfig
}

// ServiceConfig holds the configuration for a specific service's routes
type ServiceConfig struct {
	// BaseURL is the base URL of the service (e.g., http://localhost:8080)
	BaseURL string

	// Endpoints defines all endpoints for this service
	Endpoints []*EndpointConfig

	// RequireAuth indicates whether all endpoints in this service require authentication by default
	RequireAuth bool
}

// EndpointConfig holds the configuration for a specific endpoint
type EndpointConfig struct {
	// Method is the HTTP method (GET, POST, etc.)
	Method string

	// Path is the path of the endpoint (e.g., /videos/:videoID)
	Path string

	// RequireAuth indicates whether this endpoint requires authentication
	// If nil, the service's RequireAuth value is used
	RequireAuth *bool

	// Description provides a brief description of the endpoint's purpose
	Description string
}

// NewRouterConfig creates a new RouterConfig with default configurations
func NewRouterConfig() *RouterConfig {
	return &RouterConfig{
		Services: make(map[string]*ServiceConfig),
	}
}

// AddService adds a service configuration
func (r *RouterConfig) AddService(name string, baseURL string, requireAuth bool) *ServiceConfig {
	svc := &ServiceConfig{
		BaseURL:     baseURL,
		Endpoints:   make([]*EndpointConfig, 0),
		RequireAuth: requireAuth,
	}
	r.Services[name] = svc
	return svc
}

// AddEndpoint adds an endpoint to a service
func (s *ServiceConfig) AddEndpoint(method, path, description string, requireAuth *bool) *EndpointConfig {
	endpoint := &EndpointConfig{
		Method:      method,
		Path:        path,
		RequireAuth: requireAuth,
		Description: description,
	}
	s.Endpoints = append(s.Endpoints, endpoint)
	return endpoint
}

// RegisterHandlers registers all route handlers using the provided router and handler function
func (r *RouterConfig) RegisterHandlers(
	router *gin.Engine,
	apiGroup *gin.RouterGroup,
	jwtMiddleware gin.HandlerFunc,
	rateLimitMiddleware gin.HandlerFunc,
	handlerFunc func(string) gin.HandlerFunc,
) {
	// Register direct service access routes at root level (for non-api requests)
	for svcName, svc := range r.Services {
		router.Any("/"+svcName+"/*path", handlerFunc(svc.BaseURL))
	}

	// Register API routes for each service
	for svcName, svc := range r.Services {
		group := apiGroup.Group("/" + svcName)

		// Apply rate limiting to all service endpoints
		group.Use(rateLimitMiddleware)

		// Register each endpoint
		for _, endpoint := range svc.Endpoints {
			// Determine if authentication is required
			requireAuth := svc.RequireAuth
			if endpoint.RequireAuth != nil {
				requireAuth = *endpoint.RequireAuth
			}

			// Register the endpoint with or without auth middleware
			if requireAuth {
				// Create a protected group with JWT middleware
				protectedGroup := group.Group("")
				protectedGroup.Use(jwtMiddleware)
				protectedGroup.Handle(endpoint.Method, endpoint.Path, handlerFunc(svc.BaseURL))
			} else {
				// Register without auth middleware
				group.Handle(endpoint.Method, endpoint.Path, handlerFunc(svc.BaseURL))
			}
		}
	}
}

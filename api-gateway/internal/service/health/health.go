package health

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ServiceHealth represents the health status of a service
type ServiceHealth struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HealthChecker provides health check functionality for services
type HealthChecker struct {
	serviceURLs map[string]string
}

// Service health paths - different services may have different health check endpoints
var healthPaths = map[string]string{
	"auth":       "/api/v1/auth/health",
	"metadata":   "/api/v1/metadata/health",
	"streaming":  "/api/v1/streaming/health",
	"upload":     "/api/v1/upload/health",
	"transcoder": "/api/v1/transcoder/health",
}

// NewHealthChecker creates a new HealthChecker with the provided service URLs
func NewHealthChecker(serviceURLs map[string]string) *HealthChecker {
	return &HealthChecker{
		serviceURLs: serviceURLs,
	}
}

// CheckService checks the health of a single service
func (h *HealthChecker) CheckService(serviceName, serviceURL string) (string, string) {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Get the correct health path for this service
	healthPath, ok := healthPaths[serviceName]
	if !ok {
		healthPath = "/health" // fallback to default
	}

	fullURL := fmt.Sprintf("%s%s", serviceURL, healthPath)
	fmt.Printf("Checking health for %s at %s\n", serviceName, fullURL)

	resp, err := client.Get(fullURL)
	if err != nil {
		return "unhealthy", fmt.Sprintf("Error connecting: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "unhealthy", fmt.Sprintf("Received status code: %d", resp.StatusCode)
	}

	return "healthy", ""
}

// HealthCheckHandler returns a handler for checking the health of the gateway
func (h *HealthChecker) HealthCheckHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		services := map[string]ServiceHealth{
			"gateway": {Status: "healthy"},
		}

		// Check health of each service
		for name, url := range h.serviceURLs {
			if url != "" {
				status, msg := h.CheckService(name, url)
				services[name] = ServiceHealth{Status: status, Message: msg}
			}
		}

		// Calculate overall health
		allHealthy := true
		for svc, health := range services {
			if health.Status != "healthy" && svc != "gateway" {
				allHealthy = false
				break
			}
		}

		statusCode := http.StatusOK
		if !allHealthy {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, gin.H{
			"status":   allHealthy,
			"services": services,
		})
	}
}

// AllServicesHealthCheckHandler returns a handler for checking the health of all services
func (h *HealthChecker) AllServicesHealthCheckHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		services := map[string]ServiceHealth{
			"gateway": {Status: "healthy"},
		}

		// Check health of all services in parallel
		type serviceResult struct {
			name    string
			status  string
			message string
		}

		resultChan := make(chan serviceResult, len(h.serviceURLs))

		for name, url := range h.serviceURLs {
			if url == "" {
				continue
			}

			go func(name, url string) {
				status, msg := h.CheckService(name, url)
				resultChan <- serviceResult{name: name, status: status, message: msg}
			}(name, url)
		}

		// Collect results
		for i := 0; i < len(h.serviceURLs); i++ {
			select {
			case result := <-resultChan:
				if result.name != "" {
					services[result.name] = ServiceHealth{Status: result.status, Message: result.message}
				}
			case <-time.After(3 * time.Second):
				// Timeout for any remaining services
				break
			}
		}

		// Calculate overall health
		allHealthy := true
		for svc, health := range services {
			if health.Status != "healthy" && svc != "gateway" {
				allHealthy = false
				break
			}
		}

		statusCode := http.StatusOK
		if !allHealthy {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, gin.H{
			"status":   allHealthy,
			"services": services,
		})
	}
}

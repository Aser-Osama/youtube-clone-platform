package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// ReverseProxy handles the proxying of requests to backend services
type ReverseProxy struct{}

// NewReverseProxy creates a new ReverseProxy
func NewReverseProxy() *ReverseProxy {
	return &ReverseProxy{}
}

// ProxyRequest creates a handler function that proxies requests to a target service
func (p *ReverseProxy) ProxyRequest(targetURL, targetPath string, preservePath bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		target, err := url.Parse(targetURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid target URL"})
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(target)

		// Keep the original director
		originalDirector := proxy.Director

		// Replace the director to customize the request
		proxy.Director = func(req *http.Request) {
			// Call original director to set up basic URL properties
			originalDirector(req)

			// Path handling logic
			finalPath := targetPath

			// If targetPath already starts with /api/v1, don't duplicate
			if strings.HasPrefix(finalPath, "/api/v1") && strings.HasPrefix(c.Request.URL.Path, "/api/v1") {
				// Remove the redundant prefix from targetPath
				finalPath = strings.TrimPrefix(finalPath, "/api/v1")
			}

			// Replace any path variables with actual values from the request
			for _, param := range c.Params {
				finalPath = strings.Replace(finalPath, ":"+param.Key, param.Value, -1)
			}

			// Handle wildcard path parameters (for routes like /videos/:id/hls/*path)
			if strings.Contains(finalPath, "*path") && len(c.Params) > 0 {
				for _, param := range c.Params {
					if param.Key == "path" {
						finalPath = strings.Replace(finalPath, "*path", param.Value, -1)
					}
				}
			}

			// If we need to preserve query params from original request
			req.URL.RawQuery = c.Request.URL.RawQuery

			// Set the final URL path
			req.URL.Path = finalPath

			// Copy request method
			req.Method = c.Request.Method

			// Copy original headers
			req.Header = make(http.Header)
			for key, values := range c.Request.Header {
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}

			// Add forwarding headers
			req.Header.Set("X-Forwarded-Host", c.Request.Host)
			req.Header.Set("X-Forwarded-Proto", c.Request.Proto)
			req.Header.Set("X-Forwarded-For", c.ClientIP())
			req.Header.Set("X-Real-IP", c.ClientIP())

			// Debug logging
			fmt.Printf("Proxying request to %s: %s %s\n", targetURL, req.Method, req.URL.Path)
		}

		// Add error handling
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			fmt.Printf("Proxy error for %s: %v\n", targetURL, err)
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Service unavailable: %v", err)})
			c.Abort()
		}

		// Add response modifier
		proxy.ModifyResponse = func(resp *http.Response) error {
			fmt.Printf("Received response from %s: %d\n", targetURL, resp.StatusCode)
			return nil
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

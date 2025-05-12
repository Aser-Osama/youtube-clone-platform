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
func (p *ReverseProxy) ProxyRequest(targetURL, path string, preservePath bool) gin.HandlerFunc {
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

			// Always use the target path directly without duplicating api/v1 prefix
			processedPath := path
			
			// Replace any path variables with actual values
			for _, param := range c.Params {
				processedPath = strings.Replace(processedPath, ":"+param.Key, param.Value, -1)
			}

			// Set the final URL path
			req.URL.Path = processedPath

			// Copy request method
			req.Method = c.Request.Method

			// Copy original headers
			for key, values := range c.Request.Header {
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}

			// Add forwarding headers
			req.Header.Set("X-Forwarded-Host", c.Request.Host)
			req.Header.Set("X-Forwarded-Proto", "http")
			req.Header.Set("X-Forwarded-For", c.ClientIP())

			// Debug logging
			fmt.Printf("Proxying request to %s: %s %s\n", targetURL, req.Method, req.URL.Path)
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

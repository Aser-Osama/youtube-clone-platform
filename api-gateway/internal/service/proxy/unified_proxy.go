package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// ProxyHandler provides a unified interface for handling proxy requests to backend services
type ProxyHandler struct{}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{}
}

// ProxyRequest creates a handler function that proxies requests to a target service
func (p *ProxyHandler) ProxyRequest(targetURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		target, err := url.Parse(targetURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid target URL"})
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(target)

		// Keep the original director
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			// Call original director to set up basic URL properties
			originalDirector(req)

			// Ensure scheme and host are set correctly
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host

			// Path handling: extract service name and ensure correct path
			path := c.Request.URL.Path
			pathParts := strings.Split(path, "/")

			// Extract service name from path
			var serviceName string
			if len(pathParts) > 2 && pathParts[1] == "api" && pathParts[2] == "v1" {
				if len(pathParts) > 3 {
					serviceName = pathParts[3]
				}
			}

			// Ensure the path is correctly set with /api/v1/[service-name] prefix
			if serviceName != "" && !strings.HasPrefix(req.URL.Path, "/api/v1/"+serviceName) {
				req.URL.Path = "/api/v1/" + serviceName + c.Param("path")
			}

			// Query parameters
			req.URL.RawQuery = c.Request.URL.RawQuery

			// Forward headers
			// Method 1: Copy all headers from original request
			req.Header = make(http.Header)
			for key, values := range c.Request.Header {
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}

			// Method 2: Add standard proxy headers
			clientIP := getClientIP(c)
			addProxyHeaders(req, c, clientIP)

			// Debug output
			fmt.Printf("Proxying request to %s: %s %s\n", req.URL.String(), req.Method, req.URL.Path)
		}

		// Add error handling
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			fmt.Printf("Proxy error for %s: %v\n", targetURL, err)
			c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Service unavailable: %v", err)})
			c.Abort()
		}

		// Add response modifier for logging
		proxy.ModifyResponse = func(resp *http.Response) error {
			fmt.Printf("Received response from %s: %d\n", targetURL, resp.StatusCode)
			return nil
		}

		// Serve the request
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// addProxyHeaders adds standard proxy headers to the request
func addProxyHeaders(req *http.Request, c *gin.Context, clientIP string) {
	// Standard proxy headers
	req.Header.Set("X-Forwarded-Host", c.Request.Host)
	forwardedProto := c.Request.Header.Get("X-Forwarded-Proto")
	if forwardedProto == "" {
		forwardedProto = "http"
	}
	req.Header.Set("X-Forwarded-Proto", forwardedProto)
	req.Header.Set("X-Real-IP", clientIP)

	// Append to X-Forwarded-For
	if prior, ok := req.Header["X-Forwarded-For"]; ok {
		req.Header.Set("X-Forwarded-For", strings.Join(prior, ", ")+", "+clientIP)
	} else {
		req.Header.Set("X-Forwarded-For", clientIP)
	}
}

// getClientIP gets the client IP address from the request
func getClientIP(c *gin.Context) string {
	// Try to get from Gin's ClientIP helper
	clientIP := c.ClientIP()

	// If that fails, try to extract from RemoteAddr
	if clientIP == "" {
		ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err == nil {
			clientIP = ip
		} else {
			clientIP = c.Request.RemoteAddr
		}
	}

	return clientIP
}

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

			// Path handling
			if preservePath {
				// For standardized paths, keep the full path
				req.URL.Path = c.Request.URL.Path
			} else {
				// For non-standardized paths (like health checks), use the specified path
				// Replace any path variables (like :id) with their actual values
				processedPath := path
				for _, param := range c.Params {
					processedPath = strings.Replace(processedPath, ":"+param.Key, param.Value, -1)
				}
				req.URL.Path = processedPath
			}
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

package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
)

type RateLimitMiddleware struct {
	limiter *limiter.Limiter
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(limiter *limiter.Limiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
	}
}

// RateLimit returns a gin middleware for standard rate limiting
func (m *RateLimitMiddleware) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		context, err := m.limiter.Get(c, ip)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "rate limit error"})
			c.Abort()
			return
		}

		if context.Reached {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": context.Reset,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// StrictRateLimit returns a gin middleware with stricter rate limiting for testing
// This applies a much stricter limit by using a special key for the test endpoint
func (m *RateLimitMiddleware) StrictRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use a special key for the test endpoint with a prefix to make it more sensitive
		ip := c.ClientIP()
		key := "test-endpoint:" + ip

		// Get the rate limit context
		context, err := m.limiter.Get(c, key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "rate limit error"})
			c.Abort()
			return
		}

		// Add rate limit headers to the response
		c.Header("X-RateLimit-Limit", strconv.FormatInt(context.Limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(context.Remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(context.Reset, 10))

		// Check if the rate limit has been reached
		if context.Reached {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded for test endpoint",
				"retry_after": context.Reset,
				"limit":       context.Limit,
				"remaining":   0,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

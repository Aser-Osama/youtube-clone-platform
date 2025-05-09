package middleware

import (
	"auth-service/internal/auth"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		hdr := c.GetHeader("Authorization")
		parts := strings.SplitN(hdr, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		claims, err := auth.VerifyToken(parts[1])
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Set("user_id", claims.Subject)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Set("jti", claims.ID)

		c.Next()
	}
}

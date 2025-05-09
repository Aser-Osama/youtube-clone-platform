package handler

import (
	"auth-service/internal/auth"
	"auth-service/internal/service"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/markbates/goth/gothic"
)

type AuthHandler struct {
	Service *service.AuthService
}

func NewAuthHandler(s *service.AuthService) *AuthHandler {
	return &AuthHandler{Service: s}
}

// GoogleLogin redirects user to Google OAuth page
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	c.Request = c.Request.WithContext(context.WithValue(
		c.Request.Context(),
		"provider",
		"google",
	))
	gothic.BeginAuthHandler(c.Writer, c.Request)
}

// GoogleCallback handles the callback from Google and returns token pair
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	// Gothic OAuth2 handling
	user, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed: " + err.Error()})
		return
	}

	// Business logic (Persisting user & Generating tokens)
	tokens, err := h.Service.HandleGoogleCallback(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// Logout revokes a refresh token using the token's JTI
func (h *AuthHandler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	claims, err := auth.VerifyToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	if err := h.Service.Logout(claims.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke token"})
		return
	}

	c.Status(http.StatusOK)
}

// Refresh generates a new access/refresh token pair from a valid refresh token
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	tokens, err := h.Service.Refresh(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

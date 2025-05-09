package service

import (
	"auth-service/internal/auth"
	"auth-service/internal/db"
	"auth-service/internal/db/models"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/markbates/goth"
)

type AuthService struct {
	repo *db.Repository
}

func NewAuthService(r *db.Repository) *AuthService {
	return &AuthService{repo: r}
}

// HandleGoogleCallback processes a login via Google OAuth
func (s *AuthService) HandleGoogleCallback(u goth.User) (map[string]string, error) {
	userID := fmt.Sprintf("%s_%s", u.Provider, u.UserID)
	user, err := s.repo.GetUserByID(userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if user == nil {
		newUser := models.User{
			ID:       userID,
			Email:    u.Email,
			Name:     u.Name,
			Role:     "user",
			Provider: "google",
		}
		user, err = s.repo.CreateUser(newUser)
		if err != nil {
			return nil, err
		}
	}

	accessToken, err := auth.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	refreshToken, jti, err := auth.GenerateRefreshToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	err = s.repo.StoreRefreshToken(models.RefreshToken{
		TokenID:   jti,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	}, nil
}

// Refresh issues a new access/refresh token pair
func (s *AuthService) Refresh(refreshToken string) (map[string]string, error) {
	claims, err := auth.VerifyToken(refreshToken)
	if err != nil {
		return nil, err
	}

	stored, err := s.repo.GetRefreshToken(claims.ID)
	if err != nil || stored.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("invalid or expired refresh token")
	}

	if err := s.repo.DeleteRefreshToken(claims.ID); err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByID(claims.UserID)
	if err != nil {
		return nil, err
	}

	newAccess, err := auth.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	newRefresh, newJTI, err := auth.GenerateRefreshToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	err = s.repo.StoreRefreshToken(models.RefreshToken{
		TokenID:   newJTI,
		UserID:    claims.UserID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"access_token":  newAccess,
		"refresh_token": newRefresh,
	}, nil
}

// Logout deletes a specific refresh token by its jti
func (s *AuthService) Logout(jti string) error {
	println("Logging out token:", jti)
	return s.repo.DeleteRefreshToken(jti)
}

// LogoutAll deletes all refresh tokens for a user
func (s *AuthService) LogoutAll(userID string) error {
	return s.repo.DeleteAllUserRefreshTokens(userID)
}

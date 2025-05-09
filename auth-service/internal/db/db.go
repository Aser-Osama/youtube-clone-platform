package db

import (
	"auth-service/internal/db/models"
	"auth-service/internal/db/repository"
	"context"
	"database/sql"
)

type Repository struct {
	q *repository.Queries
}

// NewRepo creates a new Repository instance
func NewRepo(db *sql.DB) *Repository {
	return &Repository{
		q: repository.New(db),
	}
}

// CreateUser inserts a new user into the database
func (r *Repository) CreateUser(u models.User) (*models.User, error) {
	res, err := r.q.CreateUser(context.Background(), repository.CreateUserParams{
		ID:       u.ID,
		Email:    u.Email,
		Name:     sql.NullString{String: u.Name, Valid: u.Name != ""},
		Role:     sql.NullString{String: u.Role, Valid: u.Role != ""},
		Provider: u.Provider,
	})
	if err != nil {
		return nil, err
	}
	return &models.User{
		ID:       res.ID,
		Email:    res.Email,
		Name:     res.Name.String,
		Role:     res.Role.String,
		Provider: res.Provider,
	}, nil
}

// GetUserByEmail returns a user by email
func (r *Repository) GetUserByEmail(email string) (*models.User, error) {
	res, err := r.q.GetUserByEmail(context.Background(), email)
	if err != nil {
		return nil, err
	}
	return &models.User{
		ID:       res.ID,
		Email:    res.Email,
		Name:     res.Name.String,
		Role:     res.Role.String,
		Provider: res.Provider,
	}, nil
}

// GetUserByID returns a user by ID
func (r *Repository) GetUserByID(id string) (*models.User, error) {
	res, err := r.q.GetUserById(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return &models.User{
		ID:       res.ID,
		Email:    res.Email,
		Name:     res.Name.String,
		Role:     res.Role.String,
		Provider: res.Provider,
	}, nil
}

// StoreRefreshToken adds a refresh token to the DB
func (r *Repository) StoreRefreshToken(rt models.RefreshToken) error {
	return r.q.StoreRefreshToken(context.Background(), repository.StoreRefreshTokenParams{
		TokenID:   rt.TokenID,
		UserID:    rt.UserID,
		ExpiresAt: rt.ExpiresAt,
	})
}

// GetRefreshToken retrieves a refresh token by its jti
func (r *Repository) GetRefreshToken(tokenID string) (*models.RefreshToken, error) {
	res, err := r.q.GetRefreshToken(context.Background(), tokenID)
	if err != nil {
		return nil, err
	}
	return &models.RefreshToken{
		TokenID:   res.TokenID,
		UserID:    res.UserID,
		ExpiresAt: res.ExpiresAt,
	}, nil
}

// DeleteRefreshToken deletes a refresh token by its jti
func (r *Repository) DeleteRefreshToken(tokenID string) error {
	return r.q.DeleteRefreshToken(context.Background(), tokenID)
}

// DeleteAllUserRefreshTokens revokes all refresh tokens for a given user
func (r *Repository) DeleteAllUserRefreshTokens(userID string) error {
	return r.q.DeleteAllUserRefreshTokens(context.Background(), userID)
}

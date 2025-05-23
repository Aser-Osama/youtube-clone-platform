// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: queries.sql

package repository

import (
	"context"
	"database/sql"
	"time"
)

const createUser = `-- name: CreateUser :one
INSERT INTO users (id, email, name, role, provider)
VALUES (?, ?, ?, ?, ?)
RETURNING id, email, name, role, provider, created_at
`

type CreateUserParams struct {
	ID       string         `json:"id"`
	Email    string         `json:"email"`
	Name     sql.NullString `json:"name"`
	Role     sql.NullString `json:"role"`
	Provider string         `json:"provider"`
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	row := q.db.QueryRowContext(ctx, createUser,
		arg.ID,
		arg.Email,
		arg.Name,
		arg.Role,
		arg.Provider,
	)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.Name,
		&i.Role,
		&i.Provider,
		&i.CreatedAt,
	)
	return i, err
}

const deleteAllUserRefreshTokens = `-- name: DeleteAllUserRefreshTokens :exec
DELETE FROM refresh_tokens WHERE user_id = ?
`

func (q *Queries) DeleteAllUserRefreshTokens(ctx context.Context, userID string) error {
	_, err := q.db.ExecContext(ctx, deleteAllUserRefreshTokens, userID)
	return err
}

const deleteRefreshToken = `-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens WHERE token_id = ?
`

func (q *Queries) DeleteRefreshToken(ctx context.Context, tokenID string) error {
	_, err := q.db.ExecContext(ctx, deleteRefreshToken, tokenID)
	return err
}

const getRefreshToken = `-- name: GetRefreshToken :one
SELECT token_id, user_id, expires_at FROM refresh_tokens WHERE token_id = ?
`

func (q *Queries) GetRefreshToken(ctx context.Context, tokenID string) (RefreshToken, error) {
	row := q.db.QueryRowContext(ctx, getRefreshToken, tokenID)
	var i RefreshToken
	err := row.Scan(&i.TokenID, &i.UserID, &i.ExpiresAt)
	return i, err
}

const getUserByEmail = `-- name: GetUserByEmail :one
SELECT id, email, name, role, provider, created_at FROM users WHERE email = ?
`

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (User, error) {
	row := q.db.QueryRowContext(ctx, getUserByEmail, email)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.Name,
		&i.Role,
		&i.Provider,
		&i.CreatedAt,
	)
	return i, err
}

const getUserById = `-- name: GetUserById :one
SELECT id, email, name, role, provider, created_at FROM users WHERE id = ?
`

func (q *Queries) GetUserById(ctx context.Context, id string) (User, error) {
	row := q.db.QueryRowContext(ctx, getUserById, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.Name,
		&i.Role,
		&i.Provider,
		&i.CreatedAt,
	)
	return i, err
}

const storeRefreshToken = `-- name: StoreRefreshToken :exec
INSERT INTO refresh_tokens (token_id, user_id, expires_at)
VALUES (?, ?, ?)
`

type StoreRefreshTokenParams struct {
	TokenID   string    `json:"token_id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (q *Queries) StoreRefreshToken(ctx context.Context, arg StoreRefreshTokenParams) error {
	_, err := q.db.ExecContext(ctx, storeRefreshToken, arg.TokenID, arg.UserID, arg.ExpiresAt)
	return err
}

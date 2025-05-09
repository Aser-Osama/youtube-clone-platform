-- name: CreateUser :one
INSERT INTO users (id, email, name, role, provider)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: GetUserById :one
SELECT * FROM users WHERE id = ?;

-- name: StoreRefreshToken :exec
INSERT INTO refresh_tokens (token_id, user_id, expires_at)
VALUES (?, ?, ?);

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens WHERE token_id = ?;

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens WHERE token_id = ?;

-- name: DeleteAllUserRefreshTokens :exec
DELETE FROM refresh_tokens WHERE user_id = ?;


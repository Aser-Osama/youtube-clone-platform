package models

import "time"

type RefreshToken struct {
	TokenID   string
	UserID    string
	ExpiresAt time.Time
}

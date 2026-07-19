package dto

import (
	"time"

	"github.com/google/uuid"
)

// RefreshResponse is the output for token refresh.
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"` // seconds
}

// LoginResponseV2 is the output for successful login with refresh token.
type LoginResponseV2 struct {
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	ExpiresIn   int      `json:"expires_in"` // seconds
	User        UserInfo `json:"user"`
}

// SessionInfo represents an active session/device.
type SessionInfo struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
}

// SessionsResponse is the output for listing sessions.
type SessionsResponse struct {
	Sessions []SessionInfo `json:"sessions"`
}

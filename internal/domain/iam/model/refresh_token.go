package model

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a stored refresh token entity.
type RefreshToken struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash string     `json:"-"` // SHA-256 hex, never exposed
	FamilyID  uuid.UUID  `json:"family_id"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	IPAddress string     `json:"ip_address,omitempty"`
	UserAgent string     `json:"user_agent,omitempty"`
}

// IsRevoked returns true if the token has been revoked.
func (t *RefreshToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

// IsExpired returns true if the token has passed its expiry time.
func (t *RefreshToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

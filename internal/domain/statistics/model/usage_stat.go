package model

import (
	"time"

	"github.com/google/uuid"
)

// UsageStat represents a single model usage statistics record.
type UsageStat struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	ModelID          string     `json:"model_id"`
	TokenCount       int        `json:"token_count"`
	FirstTokenTimeMs *int       `json:"first_token_time_ms,omitempty"`
	InferenceTimeMs  int        `json:"inference_time_ms"`
	TokensPerSecond  *float64   `json:"tokens_per_second,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

package dto

import (
	"github.com/masterfabric-go/masterfabric/internal/domain/statistics/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/statistics/repository"
)

// RecordUsageRequest is the request body for recording model usage.
type RecordUsageRequest struct {
	ModelID          string   `json:"model_id"`
	TokenCount       int      `json:"token_count"`
	FirstTokenTimeMs *int     `json:"first_token_time_ms,omitempty"`
	InferenceTimeMs  int      `json:"inference_time_ms"`
	TokensPerSecond  *float64 `json:"tokens_per_second,omitempty"`
}

// Validate validates the record usage request.
func (r *RecordUsageRequest) Validate() error {
	if r.ModelID == "" {
		return ErrValidation("model_id is required")
	}
	if r.TokenCount < 0 {
		return ErrValidation("token_count must be >= 0")
	}
	if r.InferenceTimeMs <= 0 {
		return ErrValidation("inference_time_ms must be > 0")
	}
	if r.FirstTokenTimeMs != nil && *r.FirstTokenTimeMs < 0 {
		return ErrValidation("first_token_time_ms must be >= 0")
	}
	if r.TokensPerSecond != nil && *r.TokensPerSecond < 0 {
		return ErrValidation("tokens_per_second must be >= 0")
	}
	return nil
}

// UsageStatResponse is the response for a single usage stat record.
type UsageStatResponse = model.UsageStat

// UsageStatsResponse is the paginated response for usage stats queries.
type UsageStatsResponse struct {
	Data     []*model.UsageStat    `json:"data"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PerPage  int                   `json:"per_page"`
	Summary  *repository.UsageSummary `json:"summary"`
}

// UsageStatsQuery holds query parameters for listing usage stats.
type UsageStatsQuery struct {
	ModelID   *string `json:"model_id"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
	Page      int     `json:"page"`
	PerPage   int     `json:"per_page"`
}

// ToFilter converts the query to a repository filter.
func (q *UsageStatsQuery) ToFilter() repository.UsageStatFilter {
	return repository.UsageStatFilter{
		ModelID:   q.ModelID,
		Page:      q.Page,
		PerPage:   q.PerPage,
	}
}

// ValidationErr represents a validation error.
type ValidationErr struct {
	Message string
}

func (e *ValidationErr) Error() string {
	return e.Message
}

// ErrValidation creates a new ValidationErr.
func ErrValidation(msg string) *ValidationErr {
	return &ValidationErr{Message: msg}
}

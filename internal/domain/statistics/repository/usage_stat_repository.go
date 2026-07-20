package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/statistics/model"
)

// UsageStatRepository defines the interface for model usage statistics persistence.
type UsageStatRepository interface {
	Create(ctx context.Context, stat *model.UsageStat) error
	ListByUser(ctx context.Context, userID uuid.UUID, filter UsageStatFilter) ([]*model.UsageStat, int, error)
	GetSummaryByUser(ctx context.Context, userID uuid.UUID, filter UsageStatFilter) (*UsageSummary, error)
}

// UsageStatFilter holds optional filters for querying usage statistics.
type UsageStatFilter struct {
	ModelID   *string
	StartDate *time.Time
	EndDate   *time.Time
	Page      int
	PerPage   int
}

// UsageSummary holds aggregated usage statistics for a user.
type UsageSummary struct {
	TotalGenerations int              `json:"total_generations"`
	TotalTokens      int64            `json:"total_tokens"`
	AvgTokensPerSec  float64          `json:"avg_tokens_per_second"`
	ModelsUsed       []string         `json:"models_used"`
	ModelBreakdown   []ModelBreakdown `json:"model_breakdown"`
}

// ModelBreakdown holds per-model aggregated statistics.
type ModelBreakdown struct {
	ModelID         string  `json:"model_id"`
	Count           int     `json:"count"`
	AvgTokensPerSec float64 `json:"avg_tokens_per_second"`
}

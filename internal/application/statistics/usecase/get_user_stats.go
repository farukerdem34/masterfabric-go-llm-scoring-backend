package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/statistics/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/statistics/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// GetUserStatsUseCase retrieves usage statistics for a user.
type GetUserStatsUseCase struct {
	repo repository.UsageStatRepository
}

// NewGetUserStatsUseCase creates a new GetUserStatsUseCase.
func NewGetUserStatsUseCase(repo repository.UsageStatRepository) *GetUserStatsUseCase {
	return &GetUserStatsUseCase{repo: repo}
}

// Execute returns paginated usage stats and summary for the given user.
func (uc *GetUserStatsUseCase) Execute(ctx context.Context, userID uuid.UUID, query *dto.UsageStatsQuery) (*dto.UsageStatsResponse, error) {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PerPage < 1 || query.PerPage > 100 {
		query.PerPage = 50
	}

	filter := query.ToFilter()

	stats, total, err := uc.repo.ListByUser(ctx, userID, filter)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list usage stats", err)
	}

	summary, err := uc.repo.GetSummaryByUser(ctx, userID, filter)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get usage summary", err)
	}

	return &dto.UsageStatsResponse{
		Data:    stats,
		Total:   total,
		Page:    query.Page,
		PerPage: query.PerPage,
		Summary: summary,
	}, nil
}

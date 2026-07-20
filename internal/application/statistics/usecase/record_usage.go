package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/statistics/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/statistics/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/statistics/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// RecordUsageUseCase records a single model usage statistics entry.
type RecordUsageUseCase struct {
	repo repository.UsageStatRepository
}

// NewRecordUsageUseCase creates a new RecordUsageUseCase.
func NewRecordUsageUseCase(repo repository.UsageStatRepository) *RecordUsageUseCase {
	return &RecordUsageUseCase{repo: repo}
}

// Execute records a usage stat for the given user.
func (uc *RecordUsageUseCase) Execute(ctx context.Context, userID uuid.UUID, req *dto.RecordUsageRequest) (*model.UsageStat, error) {
	if err := req.Validate(); err != nil {
		return nil, domainErr.New(domainErr.ErrValidation, err.Error(), nil)
	}

	stat := &model.UsageStat{
		ID:               uuid.New(),
		UserID:           userID,
		ModelID:          req.ModelID,
		TokenCount:       req.TokenCount,
		FirstTokenTimeMs: req.FirstTokenTimeMs,
		InferenceTimeMs:  req.InferenceTimeMs,
		TokensPerSecond:  req.TokensPerSecond,
	}

	if err := uc.repo.Create(ctx, stat); err != nil {
		return nil, err
	}

	return stat, nil
}

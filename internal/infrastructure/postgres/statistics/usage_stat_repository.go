package statistics

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/masterfabric-go/masterfabric/internal/domain/statistics/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/statistics/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// UsageStatRepo implements repository.UsageStatRepository with PostgreSQL.
type UsageStatRepo struct {
	db *pgxpool.Pool
}

// NewUsageStatRepo creates a new UsageStatRepo.
func NewUsageStatRepo(db *pgxpool.Pool) *UsageStatRepo {
	return &UsageStatRepo{db: db}
}

func (r *UsageStatRepo) Create(ctx context.Context, stat *model.UsageStat) error {
	if stat.ID == uuid.Nil {
		stat.ID = uuid.New()
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO model_usage_stats (id, user_id, model_id, token_count, first_token_time_ms, inference_time_ms, tokens_per_second)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		stat.ID, stat.UserID, stat.ModelID, stat.TokenCount, stat.FirstTokenTimeMs, stat.InferenceTimeMs, stat.TokensPerSecond,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to create usage stat", err)
	}
	return nil
}

func (r *UsageStatRepo) ListByUser(ctx context.Context, userID uuid.UUID, filter repository.UsageStatFilter) ([]*model.UsageStat, int, error) {
	where, args := r.buildWhereClause(userID, filter)

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM model_usage_stats%s", where)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to count usage stats", err)
	}

	page := filter.Page
	perPage := filter.PerPage
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	dataQuery := fmt.Sprintf(
		"SELECT id, user_id, model_id, token_count, first_token_time_ms, inference_time_ms, tokens_per_second, created_at FROM model_usage_stats%s ORDER BY created_at DESC LIMIT %d OFFSET %d",
		where, perPage, offset,
	)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to list usage stats", err)
	}
	defer rows.Close()

	var stats []*model.UsageStat
	for rows.Next() {
		var s model.UsageStat
		if err := rows.Scan(&s.ID, &s.UserID, &s.ModelID, &s.TokenCount, &s.FirstTokenTimeMs, &s.InferenceTimeMs, &s.TokensPerSecond, &s.CreatedAt); err != nil {
			return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to scan usage stat", err)
		}
		stats = append(stats, &s)
	}

	return stats, total, nil
}

func (r *UsageStatRepo) GetSummaryByUser(ctx context.Context, userID uuid.UUID, filter repository.UsageStatFilter) (*repository.UsageSummary, error) {
	where, args := r.buildWhereClause(userID, filter)

	// Total generations and total tokens
	summaryQuery := fmt.Sprintf(
		"SELECT COUNT(*), COALESCE(SUM(token_count), 0), COALESCE(AVG(tokens_per_second), 0) FROM model_usage_stats%s",
		where,
	)
	var summary repository.UsageSummary
	if err := r.db.QueryRow(ctx, summaryQuery, args...).Scan(
		&summary.TotalGenerations, &summary.TotalTokens, &summary.AvgTokensPerSec,
	); err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get usage summary", err)
	}

	// Models used
	modelsQuery := fmt.Sprintf("SELECT DISTINCT model_id FROM model_usage_stats%s ORDER BY model_id", where)
	rows, err := r.db.Query(ctx, modelsQuery, args...)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list models used", err)
	}
	defer rows.Close()

	for rows.Next() {
		var modelID string
		if err := rows.Scan(&modelID); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan model_id", err)
		}
		summary.ModelsUsed = append(summary.ModelsUsed, modelID)
	}

	// Model breakdown
	breakdownQuery := fmt.Sprintf(
		"SELECT model_id, COUNT(*), COALESCE(AVG(tokens_per_second), 0) FROM model_usage_stats%s GROUP BY model_id ORDER BY COUNT(*) DESC",
		where,
	)
	breakdownRows, err := r.db.Query(ctx, breakdownQuery, args...)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get model breakdown", err)
	}
	defer breakdownRows.Close()

	for breakdownRows.Next() {
		var b repository.ModelBreakdown
		if err := breakdownRows.Scan(&b.ModelID, &b.Count, &b.AvgTokensPerSec); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan model breakdown", err)
		}
		summary.ModelBreakdown = append(summary.ModelBreakdown, b)
	}

	return &summary, nil
}

func (r *UsageStatRepo) buildWhereClause(userID uuid.UUID, filter repository.UsageStatFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
	args = append(args, userID)
	argIdx++

	if filter.ModelID != nil {
		conditions = append(conditions, fmt.Sprintf("model_id = $%d", argIdx))
		args = append(args, *filter.ModelID)
		argIdx++
	}
	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.StartDate)
		argIdx++
	}
	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.EndDate)
		argIdx++
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

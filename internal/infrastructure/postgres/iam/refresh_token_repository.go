package iam

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// RefreshTokenRepo implements repository.RefreshTokenRepository with PostgreSQL.
type RefreshTokenRepo struct {
	db *pgxpool.Pool
}

// NewRefreshTokenRepo creates a new RefreshTokenRepo.
func NewRefreshTokenRepo(db *pgxpool.Pool) *RefreshTokenRepo {
	return &RefreshTokenRepo{db: db}
}

func (r *RefreshTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now().UTC()
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, family_id, expires_at, created_at, ip_address, user_agent)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		token.ID, token.UserID, token.TokenHash, token.FamilyID, token.ExpiresAt, token.CreatedAt, token.IPAddress, token.UserAgent,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to create refresh token", err)
	}
	return nil
}

func (r *RefreshTokenRepo) GetByTokenHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	var t model.RefreshToken
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, token_hash, family_id, expires_at, revoked_at, created_at, ip_address, user_agent
		 FROM refresh_tokens WHERE token_hash = $1`, hash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.FamilyID, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt, &t.IPAddress, &t.UserAgent)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, "refresh token not found", nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, "failed to get refresh token", err)
	}
	return &t, nil
}

func (r *RefreshTokenRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $1 WHERE id = $2 AND revoked_at IS NULL`,
		now, id,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to revoke refresh token", err)
	}
	return nil
}

func (r *RefreshTokenRepo) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`,
		now, userID,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to revoke all refresh tokens", err)
	}
	return nil
}

func (r *RefreshTokenRepo) RevokeFamily(ctx context.Context, familyID uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $1 WHERE family_id = $2 AND revoked_at IS NULL`,
		now, familyID,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to revoke token family", err)
	}
	return nil
}

func (r *RefreshTokenRepo) ListActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*model.RefreshToken, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, token_hash, family_id, expires_at, revoked_at, created_at, ip_address, user_agent
		 FROM refresh_tokens
		 WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to list refresh tokens", err)
	}
	defer rows.Close()

	var tokens []*model.RefreshToken
	for rows.Next() {
		var t model.RefreshToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.FamilyID, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt, &t.IPAddress, &t.UserAgent); err != nil {
			return nil, domainErr.New(domainErr.ErrInternal, "failed to scan refresh token", err)
		}
		tokens = append(tokens, &t)
	}
	return tokens, nil
}

func (r *RefreshTokenRepo) DeleteExpired(ctx context.Context) (int64, error) {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM refresh_tokens WHERE expires_at < NOW() OR (revoked_at IS NOT NULL AND revoked_at < NOW() - INTERVAL '7 days')`,
	)
	if err != nil {
		return 0, domainErr.New(domainErr.ErrInternal, "failed to delete expired refresh tokens", err)
	}
	return tag.RowsAffected(), err
}

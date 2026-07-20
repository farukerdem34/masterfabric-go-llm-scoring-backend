package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/iam/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/service"
	"github.com/masterfabric-go/masterfabric/internal/shared/config"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// RefreshTokenUseCase handles refresh token operations.
type RefreshTokenUseCase struct {
	refreshTokenRepo repository.RefreshTokenRepository
	userRepo         repository.UserRepository
	auth             service.AuthService
	cfg              config.RefreshTokenConfig
	jwtIssuer        string
	jwtSecret        string
}

// NewRefreshTokenUseCase creates a new RefreshTokenUseCase.
func NewRefreshTokenUseCase(
	refreshTokenRepo repository.RefreshTokenRepository,
	userRepo repository.UserRepository,
	auth service.AuthService,
	cfg config.RefreshTokenConfig,
	jwtIssuer string,
) *RefreshTokenUseCase {
	return &RefreshTokenUseCase{
		refreshTokenRepo: refreshTokenRepo,
		userRepo:         userRepo,
		auth:             auth,
		cfg:              cfg,
		jwtIssuer:        jwtIssuer,
	}
}

// GenerateRefreshToken creates a new opaque refresh token and stores its hash.
func (uc *RefreshTokenUseCase) GenerateRefreshToken(ctx context.Context, userID uuid.UUID, r *http.Request) (string, *model.RefreshToken, error) {
	rawToken, err := generateOpaqueToken(32)
	if err != nil {
		return "", nil, domainErr.New(domainErr.ErrInternal, "failed to generate refresh token", err)
	}

	tokenHash := hashToken(rawToken)
	familyID := uuid.New()
	expiresAt := time.Now().UTC().Add(uc.cfg.Duration)

	refreshToken := &model.RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		FamilyID:  familyID,
		ExpiresAt: expiresAt,
		IPAddress: extractIP(r),
		UserAgent: r.UserAgent(),
	}

	if err := uc.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return "", nil, err
	}

	return rawToken, refreshToken, nil
}

// Rotate validates the current refresh token, revokes it, and issues a new pair.
func (uc *RefreshTokenUseCase) Rotate(ctx context.Context, rawToken string, r *http.Request) (*dto.RefreshResponse, error) {
	tokenHash := hashToken(rawToken)

	stored, err := uc.refreshTokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrUnauthorized, "invalid refresh token", nil)
	}

	if stored.IsRevoked() {
		// Replay detected — revoke entire family
		_ = uc.refreshTokenRepo.RevokeFamily(ctx, stored.FamilyID)
		return nil, domainErr.New(domainErr.ErrUnauthorized, "refresh token has been revoked", nil)
	}

	if stored.IsExpired() {
		return nil, domainErr.New(domainErr.ErrUnauthorized, "refresh token has expired", nil)
	}

	// Revoke current token
	if err := uc.refreshTokenRepo.Revoke(ctx, stored.ID); err != nil {
		return nil, err
	}

	// Generate new access token
	user, err := uc.userRepo.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrUnauthorized, "user not found", nil)
	}

	accessToken, err := uc.auth.GenerateToken(ctx, service.TokenClaims{
		UserID: user.ID,
		Email:  user.Email,
	})
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to generate access token", err)
	}

	// Generate new refresh token (same family)
	newRawToken, err := generateOpaqueToken(32)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to generate refresh token", err)
	}

	newRefreshToken := &model.RefreshToken{
		UserID:    stored.UserID,
		TokenHash: hashToken(newRawToken),
		FamilyID:  stored.FamilyID, // Same family for rotation
		ExpiresAt: time.Now().UTC().Add(uc.cfg.Duration),
		IPAddress: extractIP(r),
		UserAgent: r.UserAgent(),
	}

	if err := uc.refreshTokenRepo.Create(ctx, newRefreshToken); err != nil {
		return nil, err
	}

	return &dto.RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: newRawToken,
		TokenType:    "Bearer",
		ExpiresIn:    uc.cfg.AccessTokenTTL * 60,
	}, nil
}

// Logout revokes a single refresh token.
func (uc *RefreshTokenUseCase) Logout(ctx context.Context, rawToken string) error {
	tokenHash := hashToken(rawToken)
	stored, err := uc.refreshTokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil // token not found, already logged out
	}
	return uc.refreshTokenRepo.Revoke(ctx, stored.ID)
}

// LogoutAll revokes all refresh tokens for a user.
func (uc *RefreshTokenUseCase) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return uc.refreshTokenRepo.RevokeAllByUserID(ctx, userID)
}

// ListSessions returns all active sessions for a user.
func (uc *RefreshTokenUseCase) ListSessions(ctx context.Context, userID uuid.UUID) (*dto.SessionsResponse, error) {
	tokens, err := uc.refreshTokenRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	sessions := make([]dto.SessionInfo, 0, len(tokens))
	for _, t := range tokens {
		sessions = append(sessions, dto.SessionInfo{
			ID:        t.ID,
			CreatedAt: t.CreatedAt,
			IPAddress: t.IPAddress,
			UserAgent: t.UserAgent,
		})
	}

	return &dto.SessionsResponse{Sessions: sessions}, nil
}

// generateOpaqueToken creates a cryptographically random token.
func generateOpaqueToken(numBytes int) (string, error) {
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand read: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken returns the SHA-256 hex digest of a token.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}

// extractIP gets the client IP from X-Forwarded-For or RemoteAddr.
func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For may contain multiple IPs; take the first one
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// RemoteAddr is in "ip:port" format; extract just the IP
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		ip := addr[:idx]
		// Strip brackets for IPv6
		ip = strings.TrimPrefix(ip, "[")
		ip = strings.TrimSuffix(ip, "]")
		return ip
	}
	return addr
}

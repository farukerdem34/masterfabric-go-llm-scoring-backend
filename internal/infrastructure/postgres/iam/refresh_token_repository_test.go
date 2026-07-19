package iam_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/model"
	"github.com/stretchr/testify/assert"
)

func TestRefreshTokenRepo_CreateAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// This test requires a test database setup
	// Uncomment when test infrastructure is available:
	// db := setupTestDB(t)
	// repo := iam.NewRefreshTokenRepo(db)
	//
	// token := &model.RefreshToken{
	//     UserID:    uuid.New(),
	//     TokenHash: "test-hash-" + uuid.New().String(),
	//     FamilyID:  uuid.New(),
	//     ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	// }
	//
	// err := repo.Create(context.Background(), token)
	// require.NoError(t, err)
	//
	// got, err := repo.GetByTokenHash(context.Background(), token.TokenHash)
	// require.NoError(t, err)
	// assert.Equal(t, token.ID, got.ID)
	// assert.Equal(t, token.UserID, got.UserID)
	// assert.False(t, got.IsRevoked())
	// assert.False(t, got.IsExpired())
}

func TestRefreshToken_IsRevoked(t *testing.T) {
	token := &model.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: "hash",
		FamilyID:  uuid.New(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	assert.False(t, token.IsRevoked())

	now := time.Now()
	token.RevokedAt = &now
	assert.True(t, token.IsRevoked())
}

func TestRefreshToken_IsExpired(t *testing.T) {
	token := &model.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: "hash",
		FamilyID:  uuid.New(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	assert.False(t, token.IsExpired())

	token.ExpiresAt = time.Now().Add(-time.Hour)
	assert.True(t, token.IsExpired())
}

//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pradella/fluentdev/backend/internal/adapter/postgres"
	"github.com/pradella/fluentdev/backend/internal/adapter/postgres/testutil"
	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/domain/user"
)

func newUser(email string) user.User {
	id, _ := uuid.NewV7()
	return user.User{
		ID: id, Email: email, PasswordHash: "$argon2id$fake",
		DisplayName: "Test", Timezone: "America/Sao_Paulo", CreatedAt: time.Now(),
	}
}

func TestUserRepoSmoke(t *testing.T) {
	pool := testutil.NewTestPool(t)
	store := postgres.NewStore(pool)
	repo := postgres.NewUserRepo(store)
	ctx := context.Background()

	u := newUser("ana@example.com")
	require.NoError(t, repo.Create(ctx, u))

	// citext: case-insensitive lookup and duplicate detection.
	got, err := repo.GetByEmail(ctx, "ANA@example.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)
	assert.Equal(t, "America/Sao_Paulo", got.Timezone)
	assert.Empty(t, got.Level, "level is NULL before placement")

	err = repo.Create(ctx, newUser("Ana@Example.com"))
	assert.ErrorIs(t, err, domain.ErrConflict)

	_, err = repo.GetByEmail(ctx, "nobody@example.com")
	assert.ErrorIs(t, err, domain.ErrNotFound)

	// Identity linking.
	identID, _ := uuid.NewV7()
	require.NoError(t, repo.CreateIdentity(ctx, user.Identity{
		ID: identID, UserID: u.ID, Provider: "github", Subject: "gh-1",
	}))
	ident, err := repo.GetIdentity(ctx, "github", "gh-1")
	require.NoError(t, err)
	assert.Equal(t, u.ID, ident.UserID)
}

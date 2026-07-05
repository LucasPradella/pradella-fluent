package auth_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/testutil"
	"github.com/pradella/fluentdev/backend/internal/usecase/auth"
)

func newService() (*auth.Service, *testutil.MemUserRepo, *testutil.MemSessionRepo) {
	users := testutil.NewMemUserRepo()
	sessions := testutil.NewMemSessionRepo()
	return auth.New(users, sessions, 30*24*time.Hour), users, sessions
}

func TestRegisterHashesWithArgon2id(t *testing.T) {
	svc, users, _ := newService()
	u, token, err := svc.Register(context.Background(), "ana@example.com", "supersecret123", "Ana")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	stored := users.Users[u.ID]
	assert.True(t, strings.HasPrefix(stored.PasswordHash, "$argon2id$"), "must store an argon2id hash")
	assert.NotContains(t, stored.PasswordHash, "supersecret123")

	ok, err := auth.VerifyPassword("supersecret123", stored.PasswordHash)
	require.NoError(t, err)
	assert.True(t, ok)
	ok, _ = auth.VerifyPassword("wrong-password", stored.PasswordHash)
	assert.False(t, ok)
}

func TestRegisterRejectsShortPassword(t *testing.T) {
	svc, _, _ := newService()
	_, _, err := svc.Register(context.Background(), "ana@example.com", "short", "Ana")
	assert.ErrorIs(t, err, domain.ErrInvalid)
}

func TestRegisterDuplicateEmailConflicts(t *testing.T) {
	svc, _, _ := newService()
	_, _, err := svc.Register(context.Background(), "ana@example.com", "supersecret123", "Ana")
	require.NoError(t, err)
	_, _, err = svc.Register(context.Background(), "ANA@example.com", "supersecret123", "Ana 2")
	assert.ErrorIs(t, err, domain.ErrConflict)
}

func TestLoginGenericErrors(t *testing.T) {
	svc, _, _ := newService()
	_, _, err := svc.Register(context.Background(), "ana@example.com", "supersecret123", "Ana")
	require.NoError(t, err)

	_, _, err = svc.Login(context.Background(), "ana@example.com", "wrong-password")
	assert.ErrorIs(t, err, domain.ErrUnauthorized)

	_, _, err = svc.Login(context.Background(), "nobody@example.com", "whatever-pass")
	assert.ErrorIs(t, err, domain.ErrUnauthorized, "unknown user must yield the same error")
}

func TestSessionLifecycle(t *testing.T) {
	svc, _, sessions := newService()
	u, token, err := svc.Register(context.Background(), "ana@example.com", "supersecret123", "Ana")
	require.NoError(t, err)

	// Raw token never stored — only its hash.
	for k := range sessions.Sessions {
		assert.NotEqual(t, token, k)
	}
	_, hasRaw := sessions.Sessions[token]
	assert.False(t, hasRaw)
	_, hasHash := sessions.Sessions[string(auth.HashToken(token))]
	assert.True(t, hasHash)

	got, err := svc.Authenticate(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)

	// Logout revokes; token no longer authenticates.
	require.NoError(t, svc.Logout(context.Background(), token))
	_, err = svc.Authenticate(context.Background(), token)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestExpiredSessionRejected(t *testing.T) {
	users := testutil.NewMemUserRepo()
	sessions := testutil.NewMemSessionRepo()
	svc := auth.New(users, sessions, 1*time.Nanosecond) // instant expiry
	_, token, err := svc.Register(context.Background(), "ana@example.com", "supersecret123", "Ana")
	require.NoError(t, err)

	time.Sleep(2 * time.Millisecond)
	_, err = svc.Authenticate(context.Background(), token)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestOAuthNewUserThenRepeatSignIn(t *testing.T) {
	svc, users, _ := newService()

	profile := auth.OAuthProfile{
		Provider: "google", Subject: "goog-1", Email: "Novo@Example.com", Name: "Novo Usuário",
	}
	u1, token, isNew, err := svc.HandleOAuthUser(context.Background(), profile)
	require.NoError(t, err)
	assert.True(t, isNew)
	assert.NotEmpty(t, token)
	assert.Equal(t, "novo@example.com", u1.Email, "e-mail must be normalized")
	assert.Len(t, users.Identities, 1)

	// Second sign-in with the same subject reuses the account.
	u2, _, isNew, err := svc.HandleOAuthUser(context.Background(), profile)
	require.NoError(t, err)
	assert.False(t, isNew)
	assert.Equal(t, u1.ID, u2.ID)
	assert.Len(t, users.Identities, 1, "no duplicate identity on repeat sign-in")
}

func TestOAuthRejectsUnknownProviderAndMissingEmail(t *testing.T) {
	svc, _, _ := newService()
	_, _, _, err := svc.HandleOAuthUser(context.Background(), auth.OAuthProfile{
		Provider: "gitlab", Subject: "x", Email: "a@b.com",
	})
	assert.ErrorIs(t, err, domain.ErrInvalid)

	_, _, _, err = svc.HandleOAuthUser(context.Background(), auth.OAuthProfile{
		Provider: "github", Subject: "x", Email: "",
	})
	assert.ErrorIs(t, err, domain.ErrInvalid, "unverified/missing e-mail must be rejected")
}

func TestWithClock(t *testing.T) {
	svc, _, _ := newService()
	fixed := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	svc.WithClock(func() time.Time { return fixed })
	_, token, err := svc.Register(context.Background(), "clock@example.com", "supersecret123", "Ana")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestOAuthLinksByVerifiedEmail(t *testing.T) {
	svc, users, _ := newService()
	u, _, err := svc.Register(context.Background(), "ana@example.com", "supersecret123", "Ana")
	require.NoError(t, err)

	got, token, isNew, err := svc.HandleOAuthUser(context.Background(), auth.OAuthProfile{
		Provider: "github", Subject: "gh-123", Email: "ana@example.com", Name: "Ana GH",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.False(t, isNew, "same verified e-mail must link, not duplicate")
	assert.Equal(t, u.ID, got.ID)
	assert.Len(t, users.Identities, 2) // email + github
}

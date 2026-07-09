package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/domain/user"
)

// newToken returns a 256-bit opaque session token, base64url encoded.
func newToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// HashToken derives the storage key for a raw session token.
func HashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

// startSession creates a session for the user and returns the raw token.
func (s *Service) startSession(ctx context.Context, userID uuid.UUID) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	sess := Session{
		ID:        id,
		UserID:    userID,
		TokenHash: HashToken(token),
		ExpiresAt: s.now().Add(s.sessionTTL),
	}
	if err := s.sessions.Create(ctx, sess); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return token, nil
}

// Authenticate resolves a raw cookie token into a user, sliding the session
// expiry forward. Unknown or expired tokens yield domain.ErrUnauthorized.
func (s *Service) Authenticate(ctx context.Context, rawToken string) (user.User, error) {
	if rawToken == "" {
		return user.User{}, domain.ErrUnauthorized
	}
	sess, err := s.sessions.GetByTokenHash(ctx, HashToken(rawToken))
	if err != nil {
		return user.User{}, domain.ErrUnauthorized
	}
	// Sliding expiry: extend on activity.
	if err := s.sessions.Touch(ctx, sess.ID, s.now().Add(s.sessionTTL)); err != nil {
		return user.User{}, fmt.Errorf("touch session: %w", err)
	}
	u, err := s.users.GetByID(ctx, sess.UserID)
	if err != nil {
		return user.User{}, domain.ErrUnauthorized
	}
	return u, nil
}

// Logout revokes the session behind the raw token. Idempotent.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	return s.sessions.DeleteByTokenHash(ctx, HashToken(rawToken))
}

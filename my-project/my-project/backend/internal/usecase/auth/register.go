package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/domain/user"
)

// argon2Params follow the OWASP password-storage recommendation (research R5).
var argon2Params = &argon2id.Params{
	Memory:      64 * 1024,
	Iterations:  2,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

const (
	MinPasswordLen = 10
	MaxPasswordLen = 128
)

// Register creates an e-mail/password account and starts a session.
// A duplicate e-mail yields domain.ErrConflict (409).
func (s *Service) Register(ctx context.Context, email, password, displayName string) (user.User, string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	displayName = strings.TrimSpace(displayName)

	if len(password) < MinPasswordLen || len(password) > MaxPasswordLen {
		return user.User{}, "", fmt.Errorf("%w: password must have %d-%d characters", domain.ErrInvalid, MinPasswordLen, MaxPasswordLen)
	}
	if email == "" || !strings.Contains(email, "@") {
		return user.User{}, "", fmt.Errorf("%w: malformed e-mail", domain.ErrInvalid)
	}
	if displayName == "" || len(displayName) > 60 {
		return user.User{}, "", fmt.Errorf("%w: display name must have 1-60 characters", domain.ErrInvalid)
	}

	hash, err := argon2id.CreateHash(password, argon2Params)
	if err != nil {
		return user.User{}, "", fmt.Errorf("hash password: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return user.User{}, "", err
	}
	u := user.User{
		ID:           id,
		Email:        email,
		PasswordHash: hash,
		DisplayName:  displayName,
		Timezone:     "America/Sao_Paulo",
		CreatedAt:    s.now(),
	}
	if err := s.users.Create(ctx, u); err != nil {
		return user.User{}, "", err // ErrConflict propagates
	}

	identID, err := uuid.NewV7()
	if err != nil {
		return user.User{}, "", err
	}
	ident := user.Identity{ID: identID, UserID: u.ID, Provider: "email", Subject: email}
	if err := s.users.CreateIdentity(ctx, ident); err != nil {
		return user.User{}, "", fmt.Errorf("create identity: %w", err)
	}

	token, err := s.startSession(ctx, u.ID)
	if err != nil {
		return user.User{}, "", err
	}
	return u, token, nil
}

// VerifyPassword wraps argon2id comparison, which is constant-time on the
// derived key.
func VerifyPassword(password, encodedHash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, encodedHash)
}

func createHash(password string) (string, error) {
	return argon2id.CreateHash(password, argon2Params)
}

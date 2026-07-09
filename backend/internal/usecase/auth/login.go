package auth

import (
	"context"
	"strings"

	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/domain/user"
)

// Login verifies e-mail/password credentials and starts a session.
// All failure modes return domain.ErrUnauthorized so responses stay
// generic (OWASP A07 — no user enumeration).
func (s *Service) Login(ctx context.Context, email, password string) (user.User, string, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// Equalize work between unknown-user and wrong-password paths.
		_, _ = VerifyPassword(password, dummyHash)
		return user.User{}, "", domain.ErrUnauthorized
	}
	if u.PasswordHash == "" { // OAuth-only account
		_, _ = VerifyPassword(password, dummyHash)
		return user.User{}, "", domain.ErrUnauthorized
	}
	ok, err := VerifyPassword(password, u.PasswordHash)
	if err != nil || !ok {
		return user.User{}, "", domain.ErrUnauthorized
	}

	token, err := s.startSession(ctx, u.ID)
	if err != nil {
		return user.User{}, "", err
	}
	return u, token, nil
}

// dummyHash burns comparable CPU when the account does not exist.
var dummyHash = func() string {
	h, err := createHash("timing-equalizer-only")
	if err != nil {
		return "$argon2id$v=19$m=65536,t=2,p=2$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	}
	return h
}()

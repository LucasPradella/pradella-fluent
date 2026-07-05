// Package auth implements registration, login, OAuth linking and
// cookie-session management (FR-001, FR-002; research R5).
package auth

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/domain/user"
)

// UserRepo persists users and their auth identities.
type UserRepo interface {
	// Create returns domain.ErrConflict when the e-mail is already taken.
	Create(ctx context.Context, u user.User) error
	GetByEmail(ctx context.Context, email string) (user.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (user.User, error)
	CreateIdentity(ctx context.Context, ident user.Identity) error
	// GetIdentity returns domain.ErrNotFound when no identity matches.
	GetIdentity(ctx context.Context, provider, subject string) (user.Identity, error)
}

// Session is a server-side session record; the raw token never persists.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash []byte
	ExpiresAt time.Time
}

// SessionRepo persists sessions keyed by SHA-256 token hash.
type SessionRepo interface {
	Create(ctx context.Context, s Session) error
	// GetByTokenHash returns domain.ErrNotFound for unknown/expired tokens.
	GetByTokenHash(ctx context.Context, hash []byte) (Session, error)
	Touch(ctx context.Context, id uuid.UUID, expiresAt time.Time) error
	DeleteByTokenHash(ctx context.Context, hash []byte) error
}

// Service wires the auth use cases.
type Service struct {
	users      UserRepo
	sessions   SessionRepo
	sessionTTL time.Duration
	now        func() time.Time
}

// New builds the auth service. now is injectable for tests.
func New(users UserRepo, sessions SessionRepo, sessionTTL time.Duration) *Service {
	return &Service{users: users, sessions: sessions, sessionTTL: sessionTTL, now: time.Now}
}

// WithClock overrides the clock (tests only).
func (s *Service) WithClock(now func() time.Time) *Service {
	s.now = now
	return s
}

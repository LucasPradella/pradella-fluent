package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/adapter/postgres/gen"
	"github.com/pradella/fluentdev/backend/internal/usecase/auth"
)

// SessionRepo implements auth.SessionRepo.
type SessionRepo struct{ *Store }

func NewSessionRepo(s *Store) *SessionRepo { return &SessionRepo{s} }

func (r *SessionRepo) Create(ctx context.Context, s auth.Session) error {
	return mapErr(r.q.CreateSession(ctx, gen.CreateSessionParams{
		ID:        s.ID,
		UserID:    s.UserID,
		TokenHash: s.TokenHash,
		ExpiresAt: ts(s.ExpiresAt),
	}))
}

func (r *SessionRepo) GetByTokenHash(ctx context.Context, hash []byte) (auth.Session, error) {
	row, err := r.q.GetSessionByTokenHash(ctx, hash)
	if err != nil {
		return auth.Session{}, mapErr(err)
	}
	return auth.Session{
		ID:        row.ID,
		UserID:    row.UserID,
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt.Time,
	}, nil
}

func (r *SessionRepo) Touch(ctx context.Context, id uuid.UUID, expiresAt time.Time) error {
	return mapErr(r.q.TouchSession(ctx, gen.TouchSessionParams{ID: id, ExpiresAt: ts(expiresAt)}))
}

func (r *SessionRepo) DeleteByTokenHash(ctx context.Context, hash []byte) error {
	return mapErr(r.q.DeleteSessionByTokenHash(ctx, hash))
}

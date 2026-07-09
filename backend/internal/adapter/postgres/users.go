package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/adapter/postgres/gen"
	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/domain/user"
)

// UserRepo implements auth.UserRepo and lessons.UserReader.
type UserRepo struct{ *Store }

func NewUserRepo(s *Store) *UserRepo { return &UserRepo{s} }

func toDomainUser(u gen.User) user.User {
	out := user.User{
		ID:            u.ID,
		Email:         u.Email,
		DisplayName:   u.DisplayName,
		CurrentStreak: int(u.CurrentStreak),
		LongestStreak: int(u.LongestStreak),
		Timezone:      u.Timezone,
		CreatedAt:     u.CreatedAt.Time,
	}
	if u.PasswordHash.Valid {
		out.PasswordHash = u.PasswordHash.String
	}
	if u.ProficiencyLevel.Valid {
		out.Level = domain.Level(u.ProficiencyLevel.ProficiencyLevel)
	}
	return out
}

func (r *UserRepo) Create(ctx context.Context, u user.User) error {
	_, err := r.q.CreateUser(ctx, gen.CreateUserParams{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: text(u.PasswordHash),
		DisplayName:  u.DisplayName,
		Timezone:     u.Timezone,
	})
	return mapErr(err)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (user.User, error) {
	u, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return user.User{}, mapErr(err)
	}
	return toDomainUser(u), nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (user.User, error) {
	u, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		return user.User{}, mapErr(err)
	}
	return toDomainUser(u), nil
}

func (r *UserRepo) CreateIdentity(ctx context.Context, ident user.Identity) error {
	return mapErr(r.q.CreateAuthIdentity(ctx, gen.CreateAuthIdentityParams{
		ID:              ident.ID,
		UserID:          ident.UserID,
		Provider:        gen.AuthProvider(ident.Provider),
		ProviderSubject: ident.Subject,
	}))
}

func (r *UserRepo) GetIdentity(ctx context.Context, provider, subject string) (user.Identity, error) {
	row, err := r.q.GetAuthIdentity(ctx, gen.GetAuthIdentityParams{
		Provider:        gen.AuthProvider(provider),
		ProviderSubject: subject,
	})
	if err != nil {
		return user.Identity{}, mapErr(err)
	}
	return user.Identity{
		ID:       row.ID,
		UserID:   row.UserID,
		Provider: string(row.Provider),
		Subject:  row.ProviderSubject,
	}, nil
}

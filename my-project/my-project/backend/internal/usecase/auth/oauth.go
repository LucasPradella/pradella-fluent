package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/pradella/fluentdev/backend/internal/domain"
	"github.com/pradella/fluentdev/backend/internal/domain/user"
)

// OAuthProfile is what an identity provider asserts about the user after the
// authorization-code exchange (infra/oauth adapters produce it).
type OAuthProfile struct {
	Provider string // github | google
	Subject  string // provider's stable user id
	Email    string // verified e-mail only
	Name     string
}

// HandleOAuthUser signs in (or up) a user from a completed OAuth flow.
// Linking rule (data-model auth_identities): an existing account with the
// same verified e-mail gains a new identity instead of a duplicate account.
func (s *Service) HandleOAuthUser(ctx context.Context, p OAuthProfile) (user.User, string, bool, error) {
	if p.Provider != "github" && p.Provider != "google" {
		return user.User{}, "", false, fmt.Errorf("%w: unknown provider %q", domain.ErrInvalid, p.Provider)
	}
	if p.Subject == "" || p.Email == "" {
		return user.User{}, "", false, fmt.Errorf("%w: provider returned no verified e-mail", domain.ErrInvalid)
	}
	email := strings.TrimSpace(strings.ToLower(p.Email))

	// 1. Known identity → straight sign-in.
	if ident, err := s.users.GetIdentity(ctx, p.Provider, p.Subject); err == nil {
		u, err := s.users.GetByID(ctx, ident.UserID)
		if err != nil {
			return user.User{}, "", false, err
		}
		token, err := s.startSession(ctx, u.ID)
		return u, token, false, err
	}

	// 2. Same verified e-mail → link identity to the existing account.
	if u, err := s.users.GetByEmail(ctx, email); err == nil {
		identID, err := uuid.NewV7()
		if err != nil {
			return user.User{}, "", false, err
		}
		ident := user.Identity{ID: identID, UserID: u.ID, Provider: p.Provider, Subject: p.Subject}
		if err := s.users.CreateIdentity(ctx, ident); err != nil {
			return user.User{}, "", false, err
		}
		token, err := s.startSession(ctx, u.ID)
		return u, token, false, err
	}

	// 3. Brand-new user.
	name := strings.TrimSpace(p.Name)
	if name == "" {
		name = strings.SplitN(email, "@", 2)[0]
	}
	if len(name) > 60 {
		name = name[:60]
	}
	id, err := uuid.NewV7()
	if err != nil {
		return user.User{}, "", false, err
	}
	u := user.User{
		ID:          id,
		Email:       email,
		DisplayName: name,
		Timezone:    "America/Sao_Paulo",
		CreatedAt:   s.now(),
	}
	if err := s.users.Create(ctx, u); err != nil {
		return user.User{}, "", false, err
	}
	identID, err := uuid.NewV7()
	if err != nil {
		return user.User{}, "", false, err
	}
	ident := user.Identity{ID: identID, UserID: u.ID, Provider: p.Provider, Subject: p.Subject}
	if err := s.users.CreateIdentity(ctx, ident); err != nil {
		return user.User{}, "", false, err
	}
	token, err := s.startSession(ctx, u.ID)
	return u, token, true, err
}

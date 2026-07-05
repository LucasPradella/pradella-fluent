// Package oauth configures the GitHub and Google OAuth 2.0 providers
// (authorization-code flow with state + PKCE — research R5, FR-001).
package oauth

import (
	"context"

	"golang.org/x/oauth2"

	"github.com/pradella/fluentdev/backend/internal/usecase/auth"
)

// Provider bundles an oauth2 config with a profile fetcher.
type Provider struct {
	Name   string
	Config *oauth2.Config
	// FetchProfile exchanges an access token for a verified profile.
	FetchProfile func(ctx context.Context, ts oauth2.TokenSource) (auth.OAuthProfile, error)
}

// Providers is the provider registry keyed by path parameter.
type Providers map[string]*Provider

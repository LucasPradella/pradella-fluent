package oauth

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/pradella/fluentdev/backend/internal/usecase/auth"
)

// NewGoogle builds the Google provider. redirectBase is the backend origin.
func NewGoogle(clientID, clientSecret, redirectBase string) *Provider {
	return &Provider{
		Name: "google",
		Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     google.Endpoint,
			RedirectURL:  redirectBase + "/api/v1/auth/oauth/google/callback",
			Scopes:       []string{"openid", "email", "profile"},
		},
		FetchProfile: fetchGoogleProfile,
	}
}

func fetchGoogleProfile(ctx context.Context, ts oauth2.TokenSource) (auth.OAuthProfile, error) {
	client := oauth2.NewClient(ctx, ts)

	var u struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := getJSON(ctx, client, "https://openidconnect.googleapis.com/v1/userinfo", &u); err != nil {
		return auth.OAuthProfile{}, err
	}
	if !u.EmailVerified {
		return auth.OAuthProfile{}, fmt.Errorf("google account e-mail is not verified")
	}
	return auth.OAuthProfile{
		Provider: "google",
		Subject:  u.Sub,
		Email:    u.Email,
		Name:     u.Name,
	}, nil
}

package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/pradella/fluentdev/backend/internal/usecase/auth"
)

// NewGitHub builds the GitHub provider. redirectBase is the backend origin.
func NewGitHub(clientID, clientSecret, redirectBase string) *Provider {
	return &Provider{
		Name: "github",
		Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     github.Endpoint,
			RedirectURL:  redirectBase + "/api/v1/auth/oauth/github/callback",
			Scopes:       []string{"read:user", "user:email"},
		},
		FetchProfile: fetchGitHubProfile,
	}
}

func fetchGitHubProfile(ctx context.Context, ts oauth2.TokenSource) (auth.OAuthProfile, error) {
	client := oauth2.NewClient(ctx, ts)

	var u struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
	}
	if err := getJSON(ctx, client, "https://api.github.com/user", &u); err != nil {
		return auth.OAuthProfile{}, err
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := getJSON(ctx, client, "https://api.github.com/user/emails", &emails); err != nil {
		return auth.OAuthProfile{}, err
	}
	email := ""
	for _, e := range emails {
		if e.Primary && e.Verified {
			email = e.Email
			break
		}
	}
	if email == "" {
		for _, e := range emails {
			if e.Verified {
				email = e.Email
				break
			}
		}
	}

	name := u.Name
	if name == "" {
		name = u.Login
	}
	return auth.OAuthProfile{
		Provider: "github",
		Subject:  strconv.FormatInt(u.ID, 10),
		Email:    email,
		Name:     name,
	}, nil
}

func getJSON(ctx context.Context, client *http.Client, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("oauth profile fetch %s: status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	serverConfig "github.com/amalshaji/portr/internal/server/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type GitHubService struct {
	config *oauth2.Config
}

type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

func NewGitHubService(cfg *serverConfig.AdminConfig) *GitHubService {
	if cfg.GithubClientID == "" || cfg.GithubSecret == "" {
		return nil
	}

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GithubClientID,
		ClientSecret: cfg.GithubSecret,
		RedirectURL:  cfg.DomainAddress() + "/api/v1/auth/github/callback",
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}

	return &GitHubService{
		config: oauthConfig,
	}
}

func (g *GitHubService) GetAuthURL(state string) string {
	return g.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (g *GitHubService) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.config.Exchange(ctx, code)
}

func (g *GitHubService) GetUser(ctx context.Context, token *oauth2.Token) (*GitHubUser, error) {
	client := g.config.Client(ctx, token)

	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	if user.Email == "" {
		emails, err := g.getUserEmails(ctx, client)
		if err == nil && len(emails) > 0 {
			for _, email := range emails {
				if email.Primary {
					user.Email = email.Email
					break
				}
			}
			if user.Email == "" {
				user.Email = emails[0].Email
			}
		}
	}

	return &user, nil
}

type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func (g *GitHubService) getUserEmails(ctx context.Context, client *http.Client) ([]GitHubEmail, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github emails API returned %d", resp.StatusCode)
	}

	var emails []GitHubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, err
	}

	return emails, nil
}

func (g *GitHubService) IsEnabled() bool {
	return g != nil && g.config != nil
}

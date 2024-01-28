package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	db "github.com/amalshaji/portr/internal/server/db/models"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/go-resty/resty/v2"
	"golang.org/x/oauth2"
)

const GITHUB_REDIRECT_URI = "/auth/github/callback"

func (s *Service) GetOauth2Client() oauth2.Config {
	return oauth2.Config{
		ClientID:     s.config.Admin.OAuth.ClientID,
		ClientSecret: s.config.Admin.OAuth.ClientSecret,
		RedirectURL:  s.config.AdminUrl() + GITHUB_REDIRECT_URI,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
		Scopes: []string{"user:email"},
	}
}

func (s *Service) GetAccessToken(code, state string) (string, error) {
	requestBodyMap := map[string]string{
		"client_id":     s.config.Admin.OAuth.ClientID,
		"client_secret": s.config.Admin.OAuth.ClientSecret,
		"code":          code,
	}

	var response = struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
		TokenType   string `json:"token_type"`
	}{}

	client := resty.New()
	resp, err := client.R().
		SetHeader("Accept", "application/json").
		SetBody(requestBodyMap).
		SetResult(response).
		Post("https://github.com/login/oauth/access_token")
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("github api returned status code %d", resp.StatusCode())
	}

	return response.AccessToken, nil
}

type GithubUserDetails struct {
	Email     string `json:"email"`
	AvatarUrl string `json:"avatar_url"`
}

func (s *Service) GetGithubUserDetails(accessToken string) (GithubUserDetails, error) {
	var result GithubUserDetails

	client := resty.New()
	resp, err := client.R().
		SetResult(&result).
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", accessToken)).
		SetHeader("Accept", "application/vnd.github+json").
		SetHeader("X-GitHub-Api-Version", "2022-11-28").
		Get("https://api.github.com/user")
	if err != nil {
		return GithubUserDetails{}, err
	}
	if resp.StatusCode() != http.StatusOK {
		return GithubUserDetails{}, fmt.Errorf("github api returned status code %d", resp.StatusCode())
	}

	return result, nil
}

type GithubUserEmails struct {
	Email      string `json:"email"`
	Verified   bool   `json:"verified"`
	Primary    bool   `json:"primary"`
	Visibility string `json:"visibility"`
}

func (s *Service) GetGithubUserEmails(accessToken string) (*[]GithubUserEmails, error) {
	url := "https://api.github.com/user/emails"
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result []GithubUserEmails
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *Service) LoginUser(ctx context.Context, user *db.User) (db.Session, error) {
	sessionToken := utils.GenerateSessionToken()
	return s.db.Queries.CreateSession(ctx, db.CreateSessionParams{
		Token:  sessionToken,
		UserID: user.ID,
	})
}

func (s *Service) IsSuperUserSignUp(ctx context.Context) bool {
	count, _ := s.db.Queries.GetUsersCount(ctx)
	return count == 0
}

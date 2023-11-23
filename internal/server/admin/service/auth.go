package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/amalshaji/localport/internal/server/db"
	"github.com/amalshaji/localport/internal/utils"
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
	}
}

func (s *Service) GetAccessToken(code, state string) (string, error) {
	requestBodyMap := map[string]string{
		"client_id":     s.config.Admin.OAuth.ClientID,
		"client_secret": s.config.Admin.OAuth.ClientSecret,
		"code":          code,
	}
	requestJSON, _ := json.Marshal(requestBodyMap)

	req, err := http.NewRequest(
		"POST",
		"https://github.com/login/oauth/access_token",
		bytes.NewBuffer(requestJSON),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
		TokenType   string `json:"token_type"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	return response.AccessToken, nil
}

type GithubUserDetails struct {
	Email     string `json:"email"`
	AvatarUrl string `json:"avatar_url"`
}

func (s *Service) GetGithubUserDetails(accessToken string) (GithubUserDetails, error) {
	url := "https://api.github.com/user"
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return GithubUserDetails{}, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return GithubUserDetails{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return GithubUserDetails{}, fmt.Errorf("github api returned status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GithubUserDetails{}, err
	}

	var result GithubUserDetails
	err = json.Unmarshal(body, &result)
	if err != nil {
		return GithubUserDetails{}, err
	}

	return result, nil
}

func (s *Service) LoginUser(user db.User) string {
	sessionToken := utils.GenerateSessionToken()
	s.db.Conn.Create(&db.Session{
		Token: sessionToken,
		User:  user,
	})
	return sessionToken
}

func (s *Service) IsSuperUserSignUp() bool {
	var count int64
	s.db.Conn.Find(&db.User{}).Count(&count)
	return count == 0
}

package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/amalshaji/localport/internal/server/db"
	"github.com/amalshaji/localport/internal/utils"
)

const GITHUB_REDIRECT_URI = "/admin/auth/github/callback"

func (s *Service) GetAuthorizationUrl() (string, string) {
	state := utils.GenerateOAuthState()
	return fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&state=%s",
		s.config.Admin.OAuth.ClientID,
		s.config.AdminUrl()+GITHUB_REDIRECT_URI,
		state,
	), state
}

func (s *Service) GetAccessToken(code, state string) (string, error) {
	resp, err := http.Post(
		"https://github.com/login/oauth/access_token?client_id=%s&client_secret=%s&code=%s",
		"application/json",
		nil,
	)
	if err != nil {
		return "", err
	}

	var response struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
		TokenType   string `json:"token_type"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
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

	var result GithubUserDetails
	err = json.NewDecoder(resp.Body).Decode(&result)
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

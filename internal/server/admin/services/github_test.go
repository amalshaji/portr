package services

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGetUserDoesNotFetchEmails(t *testing.T) {
	requests := make([]string, 0, 1)
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id":123,"email":"public@example.com"}`)),
			Header:     make(http.Header),
		}, nil
	})}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
	service := &GitHubService{config: &oauth2.Config{}}

	if _, err := service.GetUser(ctx, &oauth2.Token{AccessToken: "token"}); err != nil {
		t.Fatalf("failed to get GitHub user: %v", err)
	}
	if len(requests) != 1 || requests[0] != "/user" {
		t.Fatalf("expected only GitHub profile request, got %v", requests)
	}
}

func TestSelectVerifiedEmailMatchesPublicEmailCaseInsensitively(t *testing.T) {
	email, ok := selectVerifiedEmail("User@Example.com", []GitHubEmail{
		{Email: "user@example.com", Primary: true, Verified: true},
	})

	if email != "user@example.com" {
		t.Fatalf("expected verified email from GitHub email list, got %q", email)
	}
	if !ok {
		t.Fatalf("expected email to be marked verified")
	}
}

func TestSelectVerifiedEmailIgnoresUnverifiedPublicEmail(t *testing.T) {
	_, ok := selectVerifiedEmail("user@example.com", []GitHubEmail{
		{Email: "user@example.com", Primary: true, Verified: false},
	})

	if ok {
		t.Fatalf("expected unverified email to stay untrusted")
	}
}

func TestSelectVerifiedEmailFallsBackToVerifiedEmail(t *testing.T) {
	email, ok := selectVerifiedEmail("", []GitHubEmail{
		{Email: "unverified@example.com", Primary: true, Verified: false},
		{Email: "verified@example.com", Primary: false, Verified: true},
	})

	if email != "verified@example.com" {
		t.Fatalf("expected verified fallback email, got %q", email)
	}
	if !ok {
		t.Fatalf("expected fallback email to be marked verified")
	}
}

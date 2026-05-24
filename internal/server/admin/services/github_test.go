package services

import "testing"

func TestApplyVerifiedEmailMarksMatchingPublicEmail(t *testing.T) {
	user := &GitHubUser{Email: "User@Example.com"}

	applyVerifiedEmail(user, []GitHubEmail{
		{Email: "user@example.com", Primary: true, Verified: true},
	})

	if user.Email != "user@example.com" {
		t.Fatalf("expected verified email to normalize from GitHub email list, got %q", user.Email)
	}
	if !user.EmailVerified {
		t.Fatalf("expected email to be marked verified")
	}
}

func TestApplyVerifiedEmailIgnoresUnverifiedPublicEmail(t *testing.T) {
	user := &GitHubUser{Email: "user@example.com"}

	applyVerifiedEmail(user, []GitHubEmail{
		{Email: "user@example.com", Primary: true, Verified: false},
	})

	if user.EmailVerified {
		t.Fatalf("expected unverified email to stay untrusted")
	}
}

func TestApplyVerifiedEmailFallsBackToVerifiedEmail(t *testing.T) {
	user := &GitHubUser{}

	applyVerifiedEmail(user, []GitHubEmail{
		{Email: "unverified@example.com", Primary: true, Verified: false},
		{Email: "verified@example.com", Primary: false, Verified: true},
	})

	if user.Email != "verified@example.com" {
		t.Fatalf("expected verified fallback email, got %q", user.Email)
	}
	if !user.EmailVerified {
		t.Fatalf("expected fallback email to be marked verified")
	}
}

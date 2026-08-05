package auth

import (
	"context"
	"errors"

	"github.com/amalshaji/portr/internal/server/admin/models"
	"github.com/amalshaji/portr/internal/server/admin/services"
	"github.com/charmbracelet/log"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type githubLoginResult struct {
	User             models.User
	RedirectTeamSlug string
}

type githubLoginDeniedReason string

const (
	githubLoginDeniedPrivateEmail        githubLoginDeniedReason = "private-email"
	githubLoginDeniedAutoSignupDisabled  githubLoginDeniedReason = "auto-signup-disabled"
	githubLoginDeniedAutoSignupDomain    githubLoginDeniedReason = "auto-signup-domain-denied"
	githubLoginDeniedAutoSignupTeam      githubLoginDeniedReason = "auto-signup-team-missing"
	githubLoginDeniedAutoSignupUnhandled githubLoginDeniedReason = "auto-signup-unavailable"
)

type githubLoginDeniedError struct {
	reason githubLoginDeniedReason
}

type githubEmailVerifier interface {
	GetVerifiedEmail(ctx context.Context, token *oauth2.Token, publicEmail string) (string, error)
}

type githubEmailVerificationError struct {
	err error
}

func (e githubEmailVerificationError) Error() string {
	return e.err.Error()
}

func (e githubEmailVerificationError) Unwrap() error {
	return e.err
}

func (e githubLoginDeniedError) Error() string {
	return string(e.reason)
}

func (e githubLoginDeniedError) Code() string {
	return string(e.reason)
}

type githubLoginResolver struct {
	db                *gorm.DB
	autoSignupService *services.AutoSignupService
}

func newGitHubLoginResolver(db *gorm.DB) *githubLoginResolver {
	return &githubLoginResolver{
		db:                db,
		autoSignupService: services.NewAutoSignupService(db),
	}
}

func (r *githubLoginResolver) resolve(ctx context.Context, verifier githubEmailVerifier, githubUser *services.GitHubUser, token *oauth2.Token) (*githubLoginResult, error) {
	var user models.User
	var githubUserRecord models.GithubUser

	err := r.db.Preload("User").Where("github_id = ?", githubUser.ID).First(&githubUserRecord).Error
	if err == nil {
		return &githubLoginResult{User: githubUserRecord.User}, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	verifiedEmail, err := verifier.GetVerifiedEmail(ctx, token, githubUser.Email)
	if err != nil {
		return nil, githubEmailVerificationError{err: err}
	}
	if verifiedEmail == "" {
		log.Warn("GitHub user attempted login with an unverified email", "email", githubUser.Email)
		return nil, githubLoginDeniedError{reason: githubLoginDeniedPrivateEmail}
	}
	email := models.NormalizeEmail(verifiedEmail)
	err = r.db.Where("LOWER(email) = ?", email).First(&user).Error
	if err == nil {
		githubUserRecord = models.GithubUser{
			GithubID:          githubUser.ID,
			GithubAccessToken: token.AccessToken,
			GithubAvatarURL:   githubUser.AvatarURL,
			UserID:            user.ID,
		}

		if err := r.db.Create(&githubUserRecord).Error; err != nil {
			if updateErr := r.db.Where("user_id = ?", user.ID).Updates(&githubUserRecord).Error; updateErr != nil {
				return nil, updateErr
			}
		}

		return &githubLoginResult{User: user}, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	githubUser.Email = email
	return r.autoSignupUser(githubUser, token.AccessToken)
}

func (r *githubLoginResolver) autoSignupUser(githubUser *services.GitHubUser, accessToken string) (*githubLoginResult, error) {
	provisioned, err := r.autoSignupService.ProvisionGitHubUser(services.GitHubAutoSignupInput{
		Email:             githubUser.Email,
		GithubID:          githubUser.ID,
		GithubAccessToken: accessToken,
		GithubAvatarURL:   githubUser.AvatarURL,
	})
	if err != nil {
		var deniedErr services.AutoSignupDeniedError
		if errors.As(err, &deniedErr) {
			reason := githubLoginDeniedAutoSignupReason(deniedErr.Reason)
			switch reason {
			case githubLoginDeniedAutoSignupDisabled:
				log.Warn("GitHub user attempted login but no account exists", "email", githubUser.Email)
			case githubLoginDeniedAutoSignupDomain:
				log.Warn("GitHub auto signup rejected email domain", "email", githubUser.Email)
			case githubLoginDeniedAutoSignupTeam:
				log.Error("GitHub auto signup domain is configured without a target team", "email", githubUser.Email)
			}
			return nil, githubLoginDeniedError{reason: reason}
		}

		return nil, err
	}

	return &githubLoginResult{
		User:             provisioned.User,
		RedirectTeamSlug: provisioned.Team.Slug,
	}, nil
}

func githubLoginDeniedAutoSignupReason(reason services.AutoSignupDenialReason) githubLoginDeniedReason {
	switch reason {
	case services.AutoSignupDeniedDisabled:
		return githubLoginDeniedAutoSignupDisabled
	case services.AutoSignupDeniedDomain:
		return githubLoginDeniedAutoSignupDomain
	case services.AutoSignupDeniedTeamMissing:
		return githubLoginDeniedAutoSignupTeam
	default:
		return githubLoginDeniedAutoSignupUnhandled
	}
}

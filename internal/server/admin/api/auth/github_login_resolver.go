package auth

import (
	"errors"
	"strings"

	"github.com/amalshaji/portr/internal/server/admin/models"
	"github.com/amalshaji/portr/internal/server/admin/services"
	"github.com/charmbracelet/log"
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

func (r *githubLoginResolver) resolve(githubUser *services.GitHubUser, accessToken string) (*githubLoginResult, error) {
	var user models.User
	var githubUserRecord models.GithubUser

	err := r.db.Preload("User").Where("github_id = ?", githubUser.ID).First(&githubUserRecord).Error
	if err == nil {
		return &githubLoginResult{User: githubUserRecord.User}, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if !githubUser.EmailVerified {
		log.Warn("GitHub user attempted login with an unverified email", "email", githubUser.Email)
		return nil, githubLoginDeniedError{reason: githubLoginDeniedPrivateEmail}
	}

	err = r.db.Where("email = ?", githubUser.Email).First(&user).Error
	if err == nil {
		githubUserRecord = models.GithubUser{
			GithubID:          githubUser.ID,
			GithubAccessToken: accessToken,
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

	return r.autoSignupUser(githubUser, accessToken)
}

func (r *githubLoginResolver) autoSignupUser(githubUser *services.GitHubUser, accessToken string) (*githubLoginResult, error) {
	team, err := r.autoSignupService.TeamForEmail(githubUser.Email)
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

	tx := r.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			tx.Rollback()
			panic(recovered)
		}
	}()

	var teamForUpdate models.Team
	if err := tx.First(&teamForUpdate, team.ID).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return nil, githubLoginDeniedError{reason: githubLoginDeniedAutoSignupTeam}
		}
		return nil, err
	}

	user := models.User{
		Email: strings.TrimSpace(githubUser.Email),
	}
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	githubUserRecord := models.GithubUser{
		GithubID:          githubUser.ID,
		GithubAccessToken: accessToken,
		GithubAvatarURL:   githubUser.AvatarURL,
		UserID:            user.ID,
	}
	if err := tx.Create(&githubUserRecord).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	teamUser := models.TeamUser{
		UserID: user.ID,
		TeamID: teamForUpdate.ID,
		Role:   models.RoleMember,
	}
	if err := tx.Create(&teamUser).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &githubLoginResult{
		User:             user,
		RedirectTeamSlug: teamForUpdate.Slug,
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

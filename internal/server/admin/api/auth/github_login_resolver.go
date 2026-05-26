package auth

import (
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

type githubLoginResolver struct {
	db *gorm.DB
}

func newGitHubLoginResolver(db *gorm.DB) *githubLoginResolver {
	return &githubLoginResolver{db: db}
}

func (r *githubLoginResolver) resolve(githubUser *services.GitHubUser, accessToken string) (*githubLoginResult, string, error) {
	var user models.User
	var githubUserRecord models.GithubUser

	err := r.db.Preload("User").Where("github_id = ?", githubUser.ID).First(&githubUserRecord).Error
	if err == nil {
		return &githubLoginResult{User: githubUserRecord.User}, "", nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, "", err
	}

	if !githubUser.EmailVerified {
		log.Warn("GitHub user attempted login with an unverified email", "email", githubUser.Email)
		return nil, "private-email", nil
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
				return nil, "", updateErr
			}
		}

		return &githubLoginResult{User: user}, "", nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, "", err
	}

	return r.autoSignup(githubUser, accessToken)
}

func (r *githubLoginResolver) autoSignup(githubUser *services.GitHubUser, accessToken string) (*githubLoginResult, string, error) {
	settings, err := models.GetOrCreateInstanceSettings(r.db)
	if err != nil {
		return nil, "", err
	}
	if !settings.AutoSignupEnabled {
		log.Warn("GitHub user attempted login but no account exists", "email", githubUser.Email)
		return nil, "auto-signup-disabled", nil
	}

	emailDomain, ok := models.EmailDomain(githubUser.Email)
	if !ok {
		log.Warn("GitHub auto signup rejected invalid email", "email", githubUser.Email)
		return nil, "auto-signup-domain-denied", nil
	}

	var domainMapping models.AutoSignupDomain
	if err := r.db.Preload("Team").Where("domain = ?", emailDomain).First(&domainMapping).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Warn("GitHub auto signup rejected email domain", "email", githubUser.Email, "domain", emailDomain)
			return nil, "auto-signup-domain-denied", nil
		}
		return nil, "", err
	}
	if domainMapping.Team.ID == 0 {
		log.Error("GitHub auto signup domain is configured without a target team", "domain", emailDomain)
		return nil, "auto-signup-team-missing", nil
	}

	tx := r.db.Begin()
	if tx.Error != nil {
		return nil, "", tx.Error
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			tx.Rollback()
			panic(recovered)
		}
	}()

	var team models.Team
	if err := tx.First(&team, domainMapping.TeamID).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return nil, "auto-signup-team-missing", nil
		}
		return nil, "", err
	}

	user := models.User{
		Email: strings.TrimSpace(githubUser.Email),
	}
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return nil, "", err
	}

	githubUserRecord := models.GithubUser{
		GithubID:          githubUser.ID,
		GithubAccessToken: accessToken,
		GithubAvatarURL:   githubUser.AvatarURL,
		UserID:            user.ID,
	}
	if err := tx.Create(&githubUserRecord).Error; err != nil {
		tx.Rollback()
		return nil, "", err
	}

	teamUser := models.TeamUser{
		UserID: user.ID,
		TeamID: team.ID,
		Role:   models.RoleMember,
	}
	if err := tx.Create(&teamUser).Error; err != nil {
		tx.Rollback()
		return nil, "", err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, "", err
	}

	return &githubLoginResult{
		User:             user,
		RedirectTeamSlug: team.Slug,
	}, "", nil
}

package services

import (
	"fmt"

	"github.com/amalshaji/portr/internal/server/admin/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AutoSignupService struct {
	db *gorm.DB
}

type AutoSignupDomainInput struct {
	Domain string
	TeamID uint
}

type AutoSignupSettings struct {
	Settings models.AutoSignupSettings
	Domains  []models.AutoSignupDomain
}

type GitHubAutoSignupInput struct {
	Email             string
	GithubID          int64
	GithubAccessToken string
	GithubAvatarURL   string
}

type GitHubAutoSignupResult struct {
	User models.User
	Team models.Team
}

type AutoSignupValidationError struct {
	Message string
}

func (e AutoSignupValidationError) Error() string {
	return e.Message
}

type AutoSignupDenialReason string

const (
	AutoSignupDeniedDisabled    AutoSignupDenialReason = "disabled"
	AutoSignupDeniedDomain      AutoSignupDenialReason = "domain_denied"
	AutoSignupDeniedTeamMissing AutoSignupDenialReason = "team_missing"
)

type AutoSignupDeniedError struct {
	Reason AutoSignupDenialReason
}

func (e AutoSignupDeniedError) Error() string {
	return string(e.Reason)
}

func NewAutoSignupService(db *gorm.DB) *AutoSignupService {
	return &AutoSignupService{db: db}
}

func (s *AutoSignupService) GetSettings() (*AutoSignupSettings, error) {
	settings, err := models.GetOrCreateAutoSignupSettings(s.db)
	if err != nil {
		return nil, err
	}

	domains, err := s.loadDomains(s.db)
	if err != nil {
		return nil, err
	}

	return &AutoSignupSettings{
		Settings: *settings,
		Domains:  domains,
	}, nil
}

func (s *AutoSignupService) UpdateSettings(enabled bool, input []AutoSignupDomainInput) (*AutoSignupSettings, error) {
	domains, err := s.buildDomainModels(input, enabled)
	if err != nil {
		return nil, err
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		settings, err := autoSignupSettingsForUpdate(tx)
		if err != nil {
			return err
		}

		settings.AutoSignupEnabled = enabled
		if err := tx.Save(settings).Error; err != nil {
			return err
		}

		if err := tx.Where("1 = 1").Delete(&models.AutoSignupDomain{}).Error; err != nil {
			return err
		}
		if len(domains) > 0 {
			if err := tx.Create(&domains).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return s.GetSettings()
}

func (s *AutoSignupService) ProvisionGitHubUser(input GitHubAutoSignupInput) (*GitHubAutoSignupResult, error) {
	input.Email = models.NormalizeEmail(input.Email)
	var result GitHubAutoSignupResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		settings, err := autoSignupSettingsForUpdate(tx)
		if err != nil {
			return err
		}

		var existingUser models.User
		err = tx.Where("LOWER(email) = ?", input.Email).First(&existingUser).Error
		if err == nil {
			if err := createGitHubUser(tx, input, existingUser.ID); err != nil {
				return err
			}
			result.User = existingUser
			return nil
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}

		team, err := autoSignupTeamForEmail(tx, settings, input.Email)
		if err != nil {
			return err
		}

		user := models.User{Email: input.Email}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		if err := createGitHubUser(tx, input, user.ID); err != nil {
			return err
		}

		teamUser := models.TeamUser{
			UserID: user.ID,
			TeamID: team.ID,
			Role:   models.RoleMember,
		}
		if err := tx.Create(&teamUser).Error; err != nil {
			return err
		}

		result = GitHubAutoSignupResult{User: user, Team: *team}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func createGitHubUser(tx *gorm.DB, input GitHubAutoSignupInput, userID uint) error {
	return tx.Create(&models.GithubUser{
		GithubID:          input.GithubID,
		GithubAccessToken: input.GithubAccessToken,
		GithubAvatarURL:   input.GithubAvatarURL,
		UserID:            userID,
	}).Error
}

func autoSignupTeamForEmail(tx *gorm.DB, settings *models.AutoSignupSettings, email string) (*models.Team, error) {
	if !settings.AutoSignupEnabled {
		return nil, AutoSignupDeniedError{Reason: AutoSignupDeniedDisabled}
	}

	emailDomain, ok := models.EmailDomain(email)
	if !ok {
		return nil, AutoSignupDeniedError{Reason: AutoSignupDeniedDomain}
	}

	var domainMapping models.AutoSignupDomain
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("domain = ?", emailDomain).First(&domainMapping).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, AutoSignupDeniedError{Reason: AutoSignupDeniedDomain}
		}
		return nil, err
	}

	var team models.Team
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&team, domainMapping.TeamID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, AutoSignupDeniedError{Reason: AutoSignupDeniedTeamMissing}
		}
		return nil, err
	}

	return &team, nil
}

func autoSignupSettingsForUpdate(tx *gorm.DB) (*models.AutoSignupSettings, error) {
	settings := models.DefaultAutoSignupSettings()
	settings.ID = 1

	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&settings, settings.ID).Error
	if err == nil {
		return &settings, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err := tx.Create(&settings).Error; err != nil {
		return nil, err
	}

	return &settings, nil
}

func (s *AutoSignupService) loadDomains(db *gorm.DB) ([]models.AutoSignupDomain, error) {
	var domains []models.AutoSignupDomain
	if err := db.Preload("Team").Order("domain ASC").Find(&domains).Error; err != nil {
		return nil, err
	}
	return domains, nil
}

func (s *AutoSignupService) buildDomainModels(input []AutoSignupDomainInput, requireDomains bool) ([]models.AutoSignupDomain, error) {
	seenDomains := make(map[string]uint, len(input))
	teamIDs := make(map[uint]struct{}, len(input))
	autoSignupDomains := make([]models.AutoSignupDomain, 0, len(input))

	for _, item := range input {
		domain, ok := models.NormalizeAutoSignupDomain(item.Domain)
		if !ok {
			return nil, AutoSignupValidationError{Message: "Auto signup domain is invalid"}
		}
		if item.TeamID == 0 {
			return nil, AutoSignupValidationError{Message: "Auto signup team is required for every domain"}
		}

		if existingTeamID, ok := seenDomains[domain]; ok {
			if existingTeamID != item.TeamID {
				return nil, AutoSignupValidationError{Message: fmt.Sprintf("Domain %s is already configured for another team", domain)}
			}
			return nil, AutoSignupValidationError{Message: fmt.Sprintf("Domain %s is configured more than once", domain)}
		}

		seenDomains[domain] = item.TeamID
		teamIDs[item.TeamID] = struct{}{}
		autoSignupDomains = append(autoSignupDomains, models.AutoSignupDomain{
			Domain: domain,
			TeamID: item.TeamID,
		})
	}

	if requireDomains && len(autoSignupDomains) == 0 {
		return nil, AutoSignupValidationError{Message: "At least one auto signup domain is required when auto signup is enabled"}
	}

	if err := s.validateTeams(teamIDs); err != nil {
		return nil, err
	}

	return autoSignupDomains, nil
}

func (s *AutoSignupService) validateTeams(teamIDs map[uint]struct{}) error {
	if len(teamIDs) == 0 {
		return nil
	}

	ids := make([]uint, 0, len(teamIDs))
	for id := range teamIDs {
		ids = append(ids, id)
	}

	var count int64
	if err := s.db.Model(&models.Team{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return AutoSignupValidationError{Message: "Auto signup team does not exist"}
	}

	return nil
}

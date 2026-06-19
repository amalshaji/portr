package services

import (
	"fmt"

	"github.com/amalshaji/portr/internal/server/admin/models"
	"gorm.io/gorm"
)

type AutoSignupService struct {
	db *gorm.DB
}

type AutoSignupDomainInput struct {
	Domain string
	TeamID uint
}

type AutoSignupSettings struct {
	Settings models.InstanceSettings
	Domains  []models.AutoSignupDomain
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
	settings, err := models.GetOrCreateInstanceSettings(s.db)
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
		settings, err := models.GetOrCreateInstanceSettings(tx)
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

func (s *AutoSignupService) TeamForEmail(email string) (*models.Team, error) {
	settings, err := models.GetOrCreateInstanceSettings(s.db)
	if err != nil {
		return nil, err
	}
	if !settings.AutoSignupEnabled {
		return nil, AutoSignupDeniedError{Reason: AutoSignupDeniedDisabled}
	}

	emailDomain, ok := models.EmailDomain(email)
	if !ok {
		return nil, AutoSignupDeniedError{Reason: AutoSignupDeniedDomain}
	}

	var domainMapping models.AutoSignupDomain
	if err := s.db.Preload("Team").Where("domain = ?", emailDomain).First(&domainMapping).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, AutoSignupDeniedError{Reason: AutoSignupDeniedDomain}
		}
		return nil, err
	}
	if domainMapping.Team.ID == 0 {
		return nil, AutoSignupDeniedError{Reason: AutoSignupDeniedTeamMissing}
	}

	return &domainMapping.Team, nil
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
	domains := make([]string, 0, len(input))
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
		domains = append(domains, domain)
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
	if err := s.validateExistingDomainOwnership(domains, seenDomains); err != nil {
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

func (s *AutoSignupService) validateExistingDomainOwnership(domains []string, requested map[string]uint) error {
	if len(domains) == 0 {
		return nil
	}

	var existing []models.AutoSignupDomain
	if err := s.db.Where("domain IN ?", domains).Find(&existing).Error; err != nil {
		return err
	}
	for _, mapping := range existing {
		if requested[mapping.Domain] != mapping.TeamID {
			return AutoSignupValidationError{Message: fmt.Sprintf("Domain %s is already configured for another team", mapping.Domain)}
		}
	}

	return nil
}

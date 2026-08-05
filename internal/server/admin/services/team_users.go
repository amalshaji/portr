package services

import (
	"context"
	"errors"
	"strings"

	"github.com/amalshaji/portr/internal/server/admin/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInvalidEmail            = errors.New("invalid email")
	ErrInvalidRole             = errors.New("invalid team role")
	ErrTeamNotFound            = errors.New("team not found")
	ErrAdminAccessRequired     = errors.New("admin access required")
	ErrSuperuserAccessRequired = errors.New("superuser access required")
	ErrUserAlreadyInTeam       = errors.New("user is already in team")
)

type TeamUserService struct {
	db *gorm.DB
}

type InviteUserInput struct {
	Email        string
	TeamSlug     string
	Role         string
	SetSuperuser bool
}

type InviteUserResult struct {
	TeamUser models.TeamUser
	Password string
}

func NewTeamUserService(db *gorm.DB) *TeamUserService {
	return &TeamUserService{db: db}
}

func (s *TeamUserService) Invite(ctx context.Context, actorTeamUserID uint, input InviteUserInput) (*InviteUserResult, error) {
	input.Email = models.NormalizeEmail(input.Email)
	input.TeamSlug = strings.TrimSpace(input.TeamSlug)
	input.Role = strings.TrimSpace(input.Role)
	if input.TeamSlug == "" {
		input.TeamSlug = models.DefaultTeamSlug
	}
	if input.Role == "" {
		input.Role = models.RoleMember
	}
	if !models.IsValidEmail(input.Email) {
		return nil, ErrInvalidEmail
	}
	if !models.IsValidTeamRole(input.Role) {
		return nil, ErrInvalidRole
	}

	result := &InviteUserResult{}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		actor, err := inviteActor(tx, actorTeamUserID)
		if err != nil {
			return err
		}

		var team models.Team
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("slug = ?", input.TeamSlug).First(&team).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTeamNotFound
			}
			return err
		}
		if !actor.User.IsSuperuser && (actor.TeamID != team.ID || !actor.IsAdmin()) {
			return ErrAdminAccessRequired
		}
		if input.SetSuperuser && !actor.User.IsSuperuser {
			return ErrSuperuserAccessRequired
		}

		settings, err := autoSignupSettingsForUpdate(tx)
		if err != nil {
			return err
		}

		user, password, err := findOrCreateInvitedUser(tx, input, settings.AutoSignupEnabled)
		if err != nil {
			return err
		}
		result.Password = password

		teamUser := models.TeamUser{UserID: user.ID, TeamID: team.ID, Role: input.Role}
		createResult := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "team_id"}, {Name: "user_id"}},
			DoNothing: true,
		}).Create(&teamUser)
		if createResult.Error != nil {
			return createResult.Error
		}
		if createResult.RowsAffected == 0 {
			return ErrUserAlreadyInTeam
		}
		if err := tx.Preload("User").Preload("User.GithubUser").Preload("Team").First(&teamUser, teamUser.ID).Error; err != nil {
			return err
		}

		result.TeamUser = teamUser
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func inviteActor(tx *gorm.DB, teamUserID uint) (*models.TeamUser, error) {
	var actor models.TeamUser
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&actor, teamUserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminAccessRequired
		}
		return nil, err
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&actor.User, actor.UserID).Error; err != nil {
		return nil, err
	}
	return &actor, nil
}

func findOrCreateInvitedUser(tx *gorm.DB, input InviteUserInput, autoSignupEnabled bool) (*models.User, string, error) {
	var user models.User
	err := tx.Where("LOWER(email) = ?", input.Email).First(&user).Error
	if err == nil {
		if input.SetSuperuser && !user.IsSuperuser {
			user.IsSuperuser = true
			if err := tx.Save(&user).Error; err != nil {
				return nil, "", err
			}
		}
		return &user, "", nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", err
	}

	candidate := models.User{Email: input.Email, IsSuperuser: input.SetSuperuser}
	password := ""
	if !autoSignupEnabled {
		password, err = GenerateRandomPassword()
		if err != nil {
			return nil, "", err
		}
		if err := candidate.SetPassword(password); err != nil {
			return nil, "", err
		}
	}

	createResult := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "email"}},
		DoNothing: true,
	}).Create(&candidate)
	if createResult.Error != nil {
		return nil, "", createResult.Error
	}
	if createResult.RowsAffected > 0 {
		return &candidate, password, nil
	}

	if err := tx.Where("LOWER(email) = ?", input.Email).First(&user).Error; err != nil {
		return nil, "", err
	}
	if input.SetSuperuser && !user.IsSuperuser {
		user.IsSuperuser = true
		if err := tx.Save(&user).Error; err != nil {
			return nil, "", err
		}
	}
	return &user, "", nil
}

package db

import (
	"github.com/amalshaji/portr/internal/admin/models"
	"github.com/charmbracelet/log"
	"gorm.io/gorm"
)

type AdminDB struct {
	*gorm.DB
}

func New(db *gorm.DB) *AdminDB {
	return &AdminDB{db}
}

func (db *AdminDB) CreateFirstUser(email, password string) (*models.User, error) {
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return nil, err
	}

	if count > 0 {
		return nil, nil
	}

	user := &models.User{
		Email:       email,
		IsSuperuser: true,
	}

	if err := user.SetPassword(password); err != nil {
		return nil, err
	}

	if err := db.Create(user).Error; err != nil {
		return nil, err
	}

	team := &models.Team{
		Name: "Default Team",
	}

	if err := db.Create(team).Error; err != nil {
		return nil, err
	}

	teamUser := &models.TeamUser{
		UserID: user.ID,
		TeamID: team.ID,
		Role:   models.RoleAdmin,
	}

	if err := db.Create(teamUser).Error; err != nil {
		return nil, err
	}

	log.Info("Created first superuser", "email", email)
	return user, nil
}

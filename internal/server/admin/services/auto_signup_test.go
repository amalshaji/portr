package services

import (
	"errors"
	"testing"

	"github.com/amalshaji/portr/internal/server/admin/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestProvisionGitHubUserRechecksDisabledPolicyBeforeWriting(t *testing.T) {
	db := newAutoSignupServiceTestDB(t)

	team := models.Team{Name: "Engineering"}
	if err := db.Create(&team).Error; err != nil {
		t.Fatalf("failed to create team: %v", err)
	}
	settings := models.DefaultAutoSignupSettings()
	settings.ID = 1
	settings.AutoSignupEnabled = true
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("failed to create settings: %v", err)
	}
	if err := db.Create(&models.AutoSignupDomain{Domain: "example.com", TeamID: team.ID}).Error; err != nil {
		t.Fatalf("failed to create domain mapping: %v", err)
	}

	settings.AutoSignupEnabled = false
	if err := db.Save(&settings).Error; err != nil {
		t.Fatalf("failed to disable auto signup: %v", err)
	}

	_, err := NewAutoSignupService(db).ProvisionGitHubUser(GitHubAutoSignupInput{
		Email:             "new-user@example.com",
		GithubID:          12345,
		GithubAccessToken: "github-token",
	})
	var deniedErr AutoSignupDeniedError
	if !errors.As(err, &deniedErr) || deniedErr.Reason != AutoSignupDeniedDisabled {
		t.Fatalf("expected disabled policy denial, got %v", err)
	}

	for _, model := range []any{&models.User{}, &models.GithubUser{}, &models.TeamUser{}} {
		var count int64
		if err := db.Model(model).Count(&count).Error; err != nil {
			t.Fatalf("failed to count %T rows: %v", model, err)
		}
		if count != 0 {
			t.Fatalf("expected no %T rows, got %d", model, count)
		}
	}
}

func TestProvisionGitHubUserReusesLegacyMixedCaseUser(t *testing.T) {
	db := newAutoSignupServiceTestDB(t)

	team := models.Team{Name: "Engineering"}
	if err := db.Create(&team).Error; err != nil {
		t.Fatalf("failed to create team: %v", err)
	}
	settings := models.DefaultAutoSignupSettings()
	settings.ID = 1
	settings.AutoSignupEnabled = true
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("failed to create settings: %v", err)
	}
	if err := db.Create(&models.AutoSignupDomain{Domain: "example.com", TeamID: team.ID}).Error; err != nil {
		t.Fatalf("failed to create domain mapping: %v", err)
	}

	existingUser := models.User{Email: "Member@Example.com"}
	if err := db.Session(&gorm.Session{SkipHooks: true}).Create(&existingUser).Error; err != nil {
		t.Fatalf("failed to create legacy user: %v", err)
	}

	result, err := NewAutoSignupService(db).ProvisionGitHubUser(GitHubAutoSignupInput{
		Email:             "member@example.com",
		GithubID:          54321,
		GithubAccessToken: "github-token",
	})
	if err != nil {
		t.Fatalf("failed to provision GitHub user: %v", err)
	}
	if result.User.ID != existingUser.ID {
		t.Fatalf("expected existing user %d, got %d", existingUser.ID, result.User.ID)
	}

	var userCount int64
	if err := db.Model(&models.User{}).Count(&userCount).Error; err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("expected one user, got %d", userCount)
	}
}

func newAutoSignupServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.GithubUser{},
		&models.Team{},
		&models.TeamUser{},
		&models.AutoSignupSettings{},
		&models.AutoSignupDomain{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to access SQL database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	return db
}

package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserEmailIsNormalizedBeforeSave(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("failed to migrate users: %v", err)
	}

	user := User{Email: " Member@Example.COM "}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if user.Email != "member@example.com" {
		t.Fatalf("expected normalized email, got %q", user.Email)
	}
}

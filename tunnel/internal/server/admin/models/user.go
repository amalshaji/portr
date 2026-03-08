package models

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type User struct {
	Model
	Timestamps
	Email       string      `gorm:"uniqueIndex;not null" json:"email"`
	FirstName   *string     `json:"first_name"`
	LastName    *string     `json:"last_name"`
	Password    *string     `json:"-"`
	IsSuperuser bool        `gorm:"default:false" json:"is_superuser"`
	GithubUser  *GithubUser `json:"github_user,omitempty"`
	Teams       []Team      `gorm:"many2many:team_users;" json:"teams,omitempty"`
	TeamUsers   []TeamUser  `json:"-"`
	Sessions    []Session   `json:"-"`
}

func (User) TableName() string {
	return "user"
}

type GithubUser struct {
	Model
	GithubID          int64  `gorm:"uniqueIndex;not null" json:"github_id"`
	GithubAccessToken string `json:"-"`
	GithubAvatarURL   string `json:"github_avatar_url"`
	UserID            uint   `gorm:"uniqueIndex;not null" json:"user_id"`
	User              User   `json:"-"`
}

func (GithubUser) TableName() string {
	return "githubuser"
}

const (
	saltLength = 16
	keyLength  = 32
	timeParam  = 3
	memory     = 64 * 1024
	threads    = 4
)

func (u *User) SetPassword(password string) error {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return err
	}

	hash := argon2.IDKey([]byte(password), salt, timeParam, memory, threads, keyLength)

	// Generate PHC format: $argon2id$v=19$m=65536,t=3,p=4$base64salt$base64hash
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)
	phc := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, timeParam, threads, saltB64, hashB64)
	u.Password = &phc

	return nil
}

func (u *User) CheckPassword(password string) bool {
	if u.Password == nil {
		return false
	}

	// First try the PHC format (Python compatible)
	if u.checkPasswordPHC(password) {
		return true
	}

	// If PHC fails, try the old custom base64 format
	if u.checkPasswordOldFormat(password) {
		return true
	}

	return false
}

// checkPasswordPHC handles the PHC format (Python compatible)
func (u *User) checkPasswordPHC(password string) bool {
	// PHC format: $argon2id$v=19$m=65536,t=3,p=4$base64salt$base64hash
	parts := strings.Split(*u.Password, "$")
	if len(parts) != 6 {
		return false
	}

	if parts[1] != "argon2id" {
		return false
	}

	// Extract salt and hash
	saltB64 := parts[4]
	hashB64 := parts[5]

	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		return false
	}

	storedHash, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil {
		return false
	}

	hash := argon2.IDKey([]byte(password), salt, timeParam, memory, threads, keyLength)

	if len(hash) != len(storedHash) {
		return false
	}

	var diff byte
	for i := 0; i < len(hash); i++ {
		diff |= hash[i] ^ storedHash[i]
	}

	return diff == 0
}

// checkPasswordOldFormat handles the old custom base64 format
func (u *User) checkPasswordOldFormat(password string) bool {
	decoded, err := base64.RawStdEncoding.DecodeString(*u.Password)
	if err != nil {
		return false
	}

	if len(decoded) < saltLength {
		return false
	}

	salt := decoded[:saltLength]
	storedHash := decoded[saltLength:]

	hash := argon2.IDKey([]byte(password), salt, timeParam, memory, threads, keyLength)

	if len(hash) != len(storedHash) {
		return false
	}

	var diff byte
	for i := 0; i < len(hash); i++ {
		diff |= hash[i] ^ storedHash[i]
	}

	return diff == 0
}

func (u *User) FullName() string {
	if u.FirstName != nil && u.LastName != nil {
		return fmt.Sprintf("%s %s", *u.FirstName, *u.LastName)
	}
	if u.FirstName != nil {
		return *u.FirstName
	}
	if u.LastName != nil {
		return *u.LastName
	}
	return u.Email
}

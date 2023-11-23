package service

import (
	"fmt"

	"github.com/amalshaji/localport/internal/server/db"
	"github.com/amalshaji/localport/internal/utils"
	"gorm.io/gorm"
)

func (s *Service) ListUsers() []db.User {
	var users []db.User
	s.db.Conn.Find(&users)
	return users
}

func (s *Service) GetUserBySession(token string) (db.User, error) {
	var session = db.Session{}
	result := s.db.Conn.Joins("User").First(&session, "token = ?", token)
	if result.Error == gorm.ErrRecordNotFound {
		return db.User{}, fmt.Errorf("session not found")
	}
	return session.User, nil
}

func (s *Service) CreateUser(githubUserDetails GithubUserDetails, accessToken string, role db.UserRole) (db.User, error) {
	secretKey := utils.GenerateSecretKeyForUser()
	user := db.User{
		Email:             githubUserDetails.Email,
		Role:              role,
		SecretKey:         secretKey,
		GithubAccessToken: accessToken,
		GithubAvatarUrl:   githubUserDetails.AvatarUrl,
	}
	result := s.db.Conn.Create(&user)
	if result.Error != nil {
		return db.User{}, result.Error
	}
	return user, nil
}

func (s *Service) GetOrCreateUserForGithubLogin(accessToken string) (db.User, error) {
	userDetails, err := s.GetGithubUserDetails(accessToken)
	if err != nil {
		return db.User{}, err
	}
	var count int64
	s.db.Conn.Find(&db.User{}).Count(&count)
	if count == 0 {
		// This is the first user, make it super user
		return s.CreateUser(userDetails, accessToken, db.SuperUser)
	}

	var user db.User
	result := s.db.Conn.Where("email = ?", userDetails.Email).First(&user)
	if result.Error == gorm.ErrRecordNotFound {
		// No user found, signup
		// check for user restrictions
		return s.CreateUser(userDetails, accessToken, db.Member)
	}

	// TODO: update github details
	return user, nil
}

func (s *Service) Logout(token string) error {
	result := s.db.Conn.Where("token = ?", token).Delete(&db.Session{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

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

func (s *Service) CreateUser(githubUserDetails GithubUserDetails, role db.UserRole) (db.User, error) {
	secretKey := utils.GenerateSecretKeyForUser()
	user := db.User{
		Email:     githubUserDetails.Email,
		Role:      role,
		SecretKey: &secretKey,
	}
	result := s.db.Conn.Create(&user)
	if result.Error != nil {
		return db.User{}, result.Error
	}
	return user, nil
}

func (s *Service) CreateOrUpdateOauthStateForUser(
	user db.User,
	accessToken string,
	githubUserDetails GithubUserDetails,
) error {
	var oauthState = db.OAuthState{}
	result := s.db.Conn.Where("user_id = ?", user.ID).First(&oauthState)
	if result.Error != nil && result.Error == gorm.ErrRecordNotFound {
		result := s.db.Conn.Create(&db.OAuthState{
			AccessToken: accessToken,
			AvatarUrl:   githubUserDetails.AvatarUrl,
			User:        user,
		})
		if result.Error != nil {
			return result.Error
		}
	} else {
		result := s.db.Conn.Model(&oauthState).Updates(db.OAuthState{
			AccessToken: accessToken,
			AvatarUrl:   githubUserDetails.AvatarUrl,
		})
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
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
		user, err := s.CreateUser(userDetails, db.SuperUser)
		if err != nil {
			return db.User{}, err
		}
		err = s.CreateOrUpdateOauthStateForUser(user, accessToken, userDetails)
		if err != nil {
			return db.User{}, err
		}
		return user, nil
	}

	var user db.User
	result := s.db.Conn.Where("email = ?", userDetails.Email).First(&user)
	if result.Error == gorm.ErrRecordNotFound {
		// No user found, signup
		// check for user restrictions
		user, err := s.CreateUser(userDetails, db.Member)
		if err != nil {
			return db.User{}, err
		}
		err = s.CreateOrUpdateOauthStateForUser(user, accessToken, userDetails)
		if err != nil {
			return db.User{}, err
		}
		return user, nil
	}

	return user, nil
}

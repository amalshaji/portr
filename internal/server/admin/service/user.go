package service

import (
	"fmt"
	"slices"
	"strings"

	"github.com/amalshaji/localport/internal/server/db"
	"github.com/amalshaji/localport/internal/utils"
	"gorm.io/gorm"
)

var (
	ErrRequiresInvite   = fmt.Errorf("requires invite")
	ErrDomainNotAllowed = fmt.Errorf("domain not allowed")
	ErrPrivateEmail     = fmt.Errorf("private email")
)

func (s *Service) ListUsers() []db.User {
	var users []db.User
	s.db.Conn.Find(&users)
	return users
}

func (s *Service) GetUserBySession(token string) (*db.User, error) {
	var session = db.Session{}
	result := s.db.Conn.Joins("User").First(&session, "token = ?", token)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("session not found")
	}
	return &session.User, nil
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
		return db.User{}, fmt.Errorf("error while creating user")
	}
	return user, nil
}

func (s *Service) checkEligibleSignup(userDetails GithubUserDetails) (db.UserRole, error) {
	var invite db.Invite

	settings := s.ListSettingsForSignup()

	if settings.SignupRequiresInvite {
		result := s.db.Conn.First(&invite, "email = ? AND status = ?", userDetails.Email, "accepted")
		if result.Error != nil && result.Error == gorm.ErrRecordNotFound {
			return "", ErrRequiresInvite
		}
		return invite.Role, nil
	}

	allowedDomains := strings.Split(settings.RandomUserSignupAllowedDomains, ",")
	userEmailDomain := strings.Split(userDetails.Email, "@")[1]
	if !slices.Contains(allowedDomains, userEmailDomain) {
		return "", ErrDomainNotAllowed
	}
	return db.Member, nil
}

func (s *Service) GetOrCreateUserForGithubLogin(accessToken string) (db.User, error) {
	userDetails, err := s.GetGithubUserDetails(accessToken)
	if err != nil {
		return db.User{}, fmt.Errorf("error while creating user")
	}

	if userDetails.Email == "" {
		return db.User{}, ErrPrivateEmail
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
		var role db.UserRole
		if role, err = s.checkEligibleSignup(userDetails); err != nil {
			return db.User{}, err
		}
		return s.CreateUser(userDetails, accessToken, role)
	}

	// TODO: update github details
	return user, nil
}

func (s *Service) Logout(token string) error {
	result := s.db.Conn.Where("token = ?", token).Delete(&db.Session{})
	return result.Error
}

func (s *Service) UpdateUser(user *db.User, firstName, lastName string) (*db.User, error) {
	user.FirstName = &firstName
	user.LastName = &lastName
	result := s.db.Conn.Save(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return user, nil
}

func (s *Service) RotateSecretKey(user *db.User) (*db.User, error) {
	user.SecretKey = utils.GenerateSecretKeyForUser()
	result := s.db.Conn.Save(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return user, nil
}

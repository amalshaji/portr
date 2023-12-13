package service

import (
	"fmt"

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

func (s *Service) ListTeamUsers(teamName string) []db.TeamUser {
	var users []db.TeamUser
	s.db.Conn.Model(&db.TeamUser{}).
		Select("role").
		Joins("User").
		Joins("Team").
		Where("team.slug = ?", teamName).
		Find(&users)
	return users
}

func (s *Service) GetUserBySession(token string) (*db.User, error) {
	var session = db.User{}
	result := s.db.Conn.Preload("Teams").
		Joins("JOIN sessions on users.id = sessions.user_id").
		Where("sessions.token = ?", token).
		First(&session)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("session not found")
	}
	return &session, nil
}

func (s *Service) GetTeamUser(user *db.User, teamName string) (*db.TeamUser, error) {
	var teamUser db.TeamUser
	result := s.db.Conn.Joins("Team").Joins("User").First(&teamUser, "team.slug = ? AND user_id = ?", teamName, user.ID)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("teamUser not found")
	}
	return &teamUser, nil
}

func (s *Service) CreateUser(
	githubUserDetails GithubUserDetails,
	accessToken string,
	isSuperUser bool,
) (
	*db.User, error,
) {
	user := db.User{
		Email:             githubUserDetails.Email,
		IsSuperUser:       isSuperUser,
		GithubAccessToken: accessToken,
		GithubAvatarUrl:   githubUserDetails.AvatarUrl,
	}
	result := s.db.Conn.Create(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("error while creating user")
	}
	return &user, nil
}

func (s *Service) checkEligibleSignup(userDetails GithubUserDetails) error {
	var count int64

	result := s.db.Conn.Model(&db.Invite{}).
		Where("email = ? AND status = ?", userDetails.Email, "active").
		Count(&count)
	if result.Error != nil {
		return result.Error
	}
	if count == 0 {
		return ErrRequiresInvite
	}
	return nil
}

func (s *Service) GetOrCreateUserForGithubLogin(accessToken string) (*db.User, error) {
	userDetails, err := s.GetGithubUserDetails(accessToken)
	if err != nil {
		s.log.Error("error while getting user details", "error", err)
		return nil, fmt.Errorf("error while creating user")
	}

	if userDetails.Email == "" {
		// no emails in user api
		// get all emails from the emails api
		email, err := s.GetGithubUserEmails(accessToken)
		if err != nil {
			s.log.Error("error while getting user emails", "error", err)
			return nil, fmt.Errorf("error while creating user")
		}

		// get the primary email
		for _, e := range *email {
			if e.Verified && e.Primary {
				userDetails.Email = e.Email
				break
			}
		}

		if userDetails.Email == "" {
			// no primary email found
			s.log.Error("no primary email found", "error", err)
			return nil, fmt.Errorf("failed to fetch email from github")
		}

	}

	var count int64
	s.db.Conn.Find(&db.User{}).Count(&count)
	if count == 0 {
		// This is the first user, make it super user
		return s.CreateUser(userDetails, accessToken, true)
	}

	tx := s.db.Conn.Begin()

	var user db.User
	result := s.db.Conn.Where("email = ?", userDetails.Email).First(&user)
	if result.Error == gorm.ErrRecordNotFound {
		// No user found, signup
		// check for user restrictions
		if err = s.checkEligibleSignup(userDetails); err != nil {
			return nil, err
		}
		user, err := s.CreateUser(userDetails, accessToken, false)
		if err != nil {
			s.log.Error("error while creating user", "error", err)
			tx.Rollback()
			return nil, err
		}
		for _, invite := range s.TeamsInvitedTo(user.Email) {
			_, err := s.CreateTeamUser(user, &invite.Team, invite.Role)
			if err != nil {
				s.log.Error("error while creating team user", "error", err)
				tx.Rollback()
				return nil, err
			}
			// mark invite as accepted
			invite.Status = db.Accepted
			result := s.db.Conn.Save(&invite)
			if result.Error != nil {
				s.log.Error("error while updating invite", "error", result.Error)
				tx.Rollback()
				return nil, result.Error
			}
		}
	}

	tx.Commit()
	// TODO: update github details
	return &user, nil
}

func (s *Service) TeamsInvitedTo(email string) []db.Invite {
	var invites []db.Invite
	s.db.Conn.Joins("Team").Model(&db.Invite{}).Where("email = ?", email).Find(&invites)
	return invites
}

func (s *Service) CreateTeamUser(user *db.User, team *db.Team, role db.UserRole) (*db.TeamUser, error) {
	teamUser := db.TeamUser{
		TeamID:    team.ID,
		UserID:    user.ID,
		Role:      role,
		SecretKey: utils.GenerateSecretKeyForUser(),
	}
	result := s.db.Conn.Create(&teamUser)
	if result.Error != nil {
		return nil, result.Error
	}
	return &teamUser, nil
}

func (s *Service) Logout(token string) error {
	result := s.db.Conn.Where("token = ?", token).Delete(&db.Session{})
	return result.Error
}

func (s *Service) UpdateUser(user *db.User, firstName, lastName string) (*db.User, error) {
	user.FirstName = &firstName
	user.LastName = &lastName
	result := s.db.Conn.Model(&db.User{}).Where("id = ?", user.ID).Updates(user)
	if result.Error != nil {
		return nil, result.Error
	}
	return user, nil
}

func (s *Service) RotateSecretKey(user *db.TeamUser) (*db.TeamUser, error) {
	user.SecretKey = utils.GenerateSecretKeyForUser()
	result := s.db.Conn.Save(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return user, nil
}

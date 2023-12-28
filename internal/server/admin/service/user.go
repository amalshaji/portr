package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	db "github.com/amalshaji/localport/internal/server/db/models"
	"github.com/amalshaji/localport/internal/utils"
)

var (
	ErrUserNotFound     = fmt.Errorf("user not found")
	ErrDomainNotAllowed = fmt.Errorf("domain not allowed")
	ErrPrivateEmail     = fmt.Errorf("private email")
)

func (s *Service) ListTeamUsers(ctx context.Context, teamID int64) []db.GetTeamMembersRow {
	teamUsers, _ := s.db.Queries.GetTeamMembers(ctx, teamID)
	return teamUsers
}

func (s *Service) GetUserBySession(ctx context.Context, token string) (*db.UserWithTeams, error) {
	result, err := s.db.Queries.GetUserBySession(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.Error("invalid session token", "token", token)
			return nil, fmt.Errorf("invalid session token")
		}
	}
	// optimize this, single query
	teams, _ := s.db.Queries.GetTeamsOfUser(ctx, result.ID)
	return &db.UserWithTeams{
		GetUserBySessionRow: result,
		Teams:               teams,
	}, nil
}

type UserWithTeamsUpdateResponse struct {
	db.GetUserByIdRow
	Teams []db.Team
}

func (s *Service) GetUserById(ctx context.Context, userID int64) (UserWithTeamsUpdateResponse, error) {
	result, err := s.db.Queries.GetUserById(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.Error("invalid session token", "userID", userID)
			return UserWithTeamsUpdateResponse{}, fmt.Errorf("invalid session token")
		}
	}
	// optimize this, single query
	teams, _ := s.db.Queries.GetTeamsOfUser(ctx, result.ID)
	return UserWithTeamsUpdateResponse{
		GetUserByIdRow: result,
		Teams:          teams,
	}, nil
}

func (s *Service) GetTeamUser(
	ctx context.Context,
	userID int64,
	teamName string,
) (*db.GetTeamMemberByUserIdAndTeamSlugRow, error) {
	teamUser, err := s.db.Queries.GetTeamMemberByUserIdAndTeamSlug(ctx, db.GetTeamMemberByUserIdAndTeamSlugParams{
		ID:   userID,
		Slug: teamName,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.Error("teamUser not found", "user", userID, "team", teamName)
			return nil, fmt.Errorf("teamUser not found")
		}
	}
	return &teamUser, nil
}

func (s *Service) CreateUser(
	ctx context.Context,
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
	user, err := s.db.Queries.CreateUser(ctx, db.CreateUserParams{
		Email:             user.Email,
		IsSuperUser:       user.IsSuperUser,
		GithubAccessToken: user.GithubAccessToken,
		GithubAvatarUrl:   user.GithubAvatarUrl,
	})
	if err != nil {
		s.log.Error("error while creating user", "error", err)
		return nil, err
	}
	return &user, nil
}

func (s *Service) GetOrCreateUserForGithubLogin(ctx context.Context, accessToken string) (*db.User, error) {
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

	count, _ := s.db.Queries.GetUsersCount(ctx)
	if count == 0 {
		// This is the first user, make it super user
		return s.CreateUser(ctx, userDetails, accessToken, true)
	}

	user, err := s.db.Queries.GetUserByEmail(ctx, userDetails.Email)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}

	err = s.db.Queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:                user.ID,
		GithubAccessToken: accessToken,
		GithubAvatarUrl:   userDetails.AvatarUrl,
	})
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Service) CreateTeamUser(ctx context.Context, userID, teamID int64, role string) (*db.TeamMember, error) {
	teamUser, err := s.db.Queries.CreateTeamMember(ctx, db.CreateTeamMemberParams{
		TeamID:    teamID,
		UserID:    userID,
		Role:      role,
		SecretKey: utils.GenerateSecretKeyForUser(),
	})
	if err != nil {
		return nil, err
	}
	return &teamUser, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.db.Queries.DeleteSession(ctx, token)
}

func (s *Service) UpdateUser(ctx context.Context, userID int64, firstName, lastName string) (UserWithTeamsUpdateResponse, error) {
	err := s.db.Queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:        userID,
		FirstName: firstName,
		LastName:  lastName,
	})
	if err != nil {
		return UserWithTeamsUpdateResponse{}, err
	}

	return s.GetUserById(ctx, userID)
}

func (s *Service) RotateSecretKey(ctx context.Context, teamUserID int64) (db.GetTeamMemberByIdRow, error) {
	secretKey := utils.GenerateSecretKeyForUser()
	err := s.db.Queries.UpdateSecretKey(ctx, db.UpdateSecretKeyParams{
		ID:        teamUserID,
		SecretKey: secretKey,
	})
	if err != nil {
		return db.GetTeamMemberByIdRow{}, err
	}
	return s.db.Queries.GetTeamMemberById(ctx, teamUserID)
}

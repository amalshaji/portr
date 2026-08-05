package services

import (
	"context"
	"errors"
	"testing"

	"github.com/amalshaji/portr/internal/server/admin/models"
	"gorm.io/gorm"
)

func TestInviteUserCreatesPasswordWhenAutoSignupDisabled(t *testing.T) {
	db := newAutoSignupServiceTestDB(t)
	team := createInviteTeam(t, db, "Default Team")
	actor := createInviteActor(t, db, team, "owner@example.com", false, models.RoleAdmin)

	result, err := NewTeamUserService(db).Invite(context.Background(), actor.ID, InviteUserInput{
		Email: "New.User@Example.com",
	})
	if err != nil {
		t.Fatalf("invite user: %v", err)
	}
	if result.Password == "" {
		t.Fatal("expected a generated password")
	}
	if result.TeamUser.User.Email != "new.user@example.com" {
		t.Fatalf("expected normalized email, got %q", result.TeamUser.User.Email)
	}
	if !result.TeamUser.User.CheckPassword(result.Password) {
		t.Fatal("expected generated password to match stored hash")
	}
	if result.TeamUser.Role != models.RoleMember || result.TeamUser.Team.Slug != models.DefaultTeamSlug {
		t.Fatalf("unexpected defaults: role=%q team=%q", result.TeamUser.Role, result.TeamUser.Team.Slug)
	}
}

func TestInviteUserOmitsPasswordWhenAutoSignupEnabled(t *testing.T) {
	db := newAutoSignupServiceTestDB(t)
	team := createInviteTeam(t, db, "Default Team")
	actor := createInviteActor(t, db, team, "owner@example.com", false, models.RoleAdmin)
	settings := models.DefaultAutoSignupSettings()
	settings.ID = 1
	settings.AutoSignupEnabled = true
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("create settings: %v", err)
	}

	result, err := NewTeamUserService(db).Invite(context.Background(), actor.ID, InviteUserInput{
		Email: "github-user@example.com",
		Role:  models.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("invite user: %v", err)
	}
	if result.Password != "" {
		t.Fatalf("expected no password, got %q", result.Password)
	}
	if result.TeamUser.User.Password != nil {
		t.Fatal("expected passwordless user when auto signup is enabled")
	}
}

func TestInviteUserDoesNotResetExistingUserPassword(t *testing.T) {
	db := newAutoSignupServiceTestDB(t)
	team := createInviteTeam(t, db, "Engineering")
	actor := createInviteActor(t, db, team, "owner@example.com", false, models.RoleAdmin)
	user := models.User{Email: "existing@example.com"}
	if err := user.SetPassword("existing-password"); err != nil {
		t.Fatalf("set password: %v", err)
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	originalHash := *user.Password

	result, err := NewTeamUserService(db).Invite(context.Background(), actor.ID, InviteUserInput{
		Email:    user.Email,
		TeamSlug: team.Slug,
		Role:     models.RoleMember,
	})
	if err != nil {
		t.Fatalf("invite user: %v", err)
	}
	if result.Password != "" {
		t.Fatal("expected no password for an existing user")
	}
	if result.TeamUser.User.Password == nil || *result.TeamUser.User.Password != originalHash {
		t.Fatal("expected existing password hash to remain unchanged")
	}
}

func TestInviteUserRejectsDuplicateMembership(t *testing.T) {
	db := newAutoSignupServiceTestDB(t)
	team := createInviteTeam(t, db, "Engineering")
	actor := createInviteActor(t, db, team, "owner@example.com", false, models.RoleAdmin)

	_, err := NewTeamUserService(db).Invite(context.Background(), actor.ID, InviteUserInput{
		Email:    actor.User.Email,
		TeamSlug: team.Slug,
		Role:     models.RoleMember,
	})
	if !errors.Is(err, ErrUserAlreadyInTeam) {
		t.Fatalf("expected ErrUserAlreadyInTeam, got %v", err)
	}
}

func TestInviteUserRejectsInvalidEmailAndRole(t *testing.T) {
	tests := []struct {
		name  string
		email string
		role  string
		want  error
	}{
		{name: "invalid email", email: "not-an-email", role: models.RoleMember, want: ErrInvalidEmail},
		{name: "invalid role", email: "member@example.com", role: "owner", want: ErrInvalidRole},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newAutoSignupServiceTestDB(t)
			_, err := NewTeamUserService(db).Invite(context.Background(), 0, InviteUserInput{Email: tt.email, Role: tt.role})
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestInviteUserEnforcesCredentialTeamScope(t *testing.T) {
	db := newAutoSignupServiceTestDB(t)
	source := createInviteTeam(t, db, "Source Team")
	target := createInviteTeam(t, db, "Target Team")
	actor := createInviteActor(t, db, source, "admin@example.com", false, models.RoleAdmin)
	if err := db.Create(&models.TeamUser{UserID: actor.UserID, TeamID: target.ID, Role: models.RoleAdmin}).Error; err != nil {
		t.Fatalf("create target membership: %v", err)
	}

	_, err := NewTeamUserService(db).Invite(context.Background(), actor.ID, InviteUserInput{
		Email:    "invitee@example.com",
		TeamSlug: target.Slug,
	})
	if !errors.Is(err, ErrAdminAccessRequired) {
		t.Fatalf("expected credential scope denial, got %v", err)
	}
}

func TestInviteUserRequiresSuperuserForPromotion(t *testing.T) {
	db := newAutoSignupServiceTestDB(t)
	team := createInviteTeam(t, db, "Engineering")
	actor := createInviteActor(t, db, team, "admin@example.com", false, models.RoleAdmin)

	_, err := NewTeamUserService(db).Invite(context.Background(), actor.ID, InviteUserInput{
		Email:        "invitee@example.com",
		TeamSlug:     team.Slug,
		SetSuperuser: true,
	})
	if !errors.Is(err, ErrSuperuserAccessRequired) {
		t.Fatalf("expected superuser access denial, got %v", err)
	}
}

func TestTeamUserMembershipIsUniquePerTeamAndUser(t *testing.T) {
	db := newAutoSignupServiceTestDB(t)
	team := createInviteTeam(t, db, "Engineering")
	user := models.User{Email: "member@example.com"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	first := models.TeamUser{TeamID: team.ID, UserID: user.ID, Role: models.RoleMember}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first membership: %v", err)
	}
	second := models.TeamUser{TeamID: team.ID, UserID: user.ID, Role: models.RoleAdmin}
	if err := db.Create(&second).Error; err == nil {
		t.Fatal("expected duplicate team membership to violate a database constraint")
	}
}

func createInviteTeam(t *testing.T, db *gorm.DB, name string) *models.Team {
	t.Helper()
	team := &models.Team{Name: name}
	if err := db.Create(team).Error; err != nil {
		t.Fatalf("create team: %v", err)
	}
	return team
}

func createInviteActor(t *testing.T, db *gorm.DB, team *models.Team, email string, superuser bool, role string) *models.TeamUser {
	t.Helper()
	user := models.User{Email: email, IsSuperuser: superuser}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create actor: %v", err)
	}
	actor := &models.TeamUser{UserID: user.ID, TeamID: team.ID, Role: role}
	if err := db.Create(actor).Error; err != nil {
		t.Fatalf("create actor membership: %v", err)
	}
	actor.User = user
	actor.Team = *team
	return actor
}

package handler

import (
	"errors"
	"time"

	"github.com/amalshaji/localport/internal/server/admin/service"
	"github.com/amalshaji/localport/internal/utils"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) StartGithubAuth(c *fiber.Ctx) error {
	state := utils.GenerateOAuthState()
	oauth2Client := h.service.GetOauth2Client()
	url := oauth2Client.AuthCodeURL(state)

	c.Cookie(&fiber.Cookie{
		Name:     "localport-oauth-state",
		Value:    state,
		HTTPOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(10 * time.Minute),
		SameSite: "Lax",
	})
	return c.Redirect(url)
}

func (h *Handler) GithubAuthCallback(c *fiber.Ctx) error {
	state := c.Cookies("localport-oauth-state")
	if state == "" {
		h.log.Error("malformed oauth flow", "error", "missing state in cookie")
		return c.Redirect("/?code=github-oauth-error")
	}

	c.ClearCookie("localport-oauth-state")

	code := c.Query("code")
	if code == "" {
		h.log.Error("malformed oauth flow", "error", "missing code in query params")
		return c.Redirect("/?code=github-oauth-error")
	}

	oauth2Client := h.service.GetOauth2Client()

	token, err := oauth2Client.Exchange(c.Context(), code)
	if err != nil {
		h.log.Error("error while getting access token", "error", err)
		return c.Redirect("/?code=github-oauth-error")
	}

	user, err := h.service.GetOrCreateUserForGithubLogin(c.Context(), token.AccessToken)
	if err != nil {
		h.log.Error("error while creating user", "error", err)
		if errors.Is(err, service.ErrRequiresInvite) {
			return c.Redirect("/?code=requires-invite")
		} else if errors.Is(err, service.ErrDomainNotAllowed) {
			return c.Redirect("/?code=domain-not-allowed")
		} else if errors.Is(err, service.ErrPrivateEmail) {
			return c.Redirect("/?code=private-email")
		}
	}

	session, _ := h.service.LoginUser(c.Context(), user)
	c.Cookie(&fiber.Cookie{
		Name:     "localport-session",
		Value:    session.Token,
		HTTPOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		SameSite: "Lax",
	})
	return c.Redirect("/")
}

func (h *Handler) IsSuperUserSignup(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"isSuperUserSignup": h.service.IsSuperUserSignUp(c.Context())})
}

package handler

import (
	"fmt"
	"time"

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
		return c.Redirect("/?github-oauth-error=error while authenticating with github")
	}

	c.ClearCookie("localport-oauth-state")

	code := c.Query("code")
	if code == "" {
		h.log.Error("malformed oauth flow", "error", "missing code in query params")
		return c.Redirect("/?github-oauth-error=error while authenticating with github")
	}

	oauth2Client := h.service.GetOauth2Client()

	token, err := oauth2Client.Exchange(c.Context(), code)
	if err != nil {
		h.log.Error("error while getting access token", "error", err)
		return c.Redirect("/?github-oauth-error=error while authenticating with github")
	}

	user, err := h.service.GetOrCreateUserForGithubLogin(token.AccessToken)
	if err != nil {
		h.log.Error("error while creating user", "error", err)
		return c.Redirect(fmt.Sprintf("/?github-oauth-error=%s", err.Error()))
	}

	sessionToken := h.service.LoginUser(user)
	c.Cookie(&fiber.Cookie{
		Name:     "localport-session",
		Value:    sessionToken,
		HTTPOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		SameSite: "Lax",
	})
	return c.Redirect("/connections")
}

func (h *Handler) IsSuperUserSignup(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"isSuperUserSignup": h.service.IsSuperUserSignUp()})
}
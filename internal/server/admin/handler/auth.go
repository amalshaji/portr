package handler

import "github.com/gofiber/fiber/v2"

func (h *Handler) StartGithubAuth(c *fiber.Ctx) error {
	url, state := h.service.GetAuthorizationUrl()
	c.Cookie(&fiber.Cookie{
		Name:     "localport-oauth-state",
		Value:    state,
		HTTPOnly: true,
	})
	return c.Redirect(url)
}

func (h *Handler) GithubAuthCallback(c *fiber.Ctx) error {
	state := c.Cookies("localport-oauth-state")
	if state == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "malformed oauth flow"})
	}

	code := c.Query("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "malformed oauth flow"})
	}

	token, err := h.service.GetAccessToken(code, state)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "internal server error"})
	}

	user, err := h.service.GetOrCreateUserForGithubLogin(token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "internal server error"})
	}

	sessionToken := h.service.LoginUser(user)
	c.Cookie(&fiber.Cookie{
		Name:     "localport-session",
		Value:    sessionToken,
		HTTPOnly: true,
	})
	return c.Redirect("/dashboard")
}

func (h *Handler) IsSuperUserSignup(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"isSuperUserSignup": h.service.IsSuperUserSignUp()})
}

package handler

import (
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"
)

func (h *Handler) ValidateClientConfig(c *fiber.Ctx) error {
	var payload struct {
		Key string `json:"key"`
	}
	if err := c.BodyParser(&payload); err != nil {
		h.log.Error("failed to parse payload", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "invalid payload"})
	}

	err := h.service.ValidateClientConfig(payload.Key)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "failed to validate client config"})
	}

	content, err := os.ReadFile(h.config.Ssh.KeysDir + "/id_rsa")
	if err != nil {
		h.log.Error("failed to locate credentials", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to locate credentials"})
	}

	return c.Send(content)
}

func (h *Handler) GetServerAddress(c *fiber.Ctx) error {
	AdminUrl := h.config.Admin.Host + ":" + fmt.Sprint(h.config.Admin.Port)
	if h.config.Secure {
		AdminUrl = h.config.Domain
	}

	sshHost := h.config.Ssh.Host
	if h.config.Secure {
		sshHost = h.config.Domain
	}

	sshUrl := sshHost + ":" + fmt.Sprint(h.config.Ssh.Port)

	return c.JSON(fiber.Map{
		"AdminUrl": AdminUrl,
		"SshUrl":   sshUrl,
	})
}

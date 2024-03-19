package handler

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func (h *Handler) GetTunnels(c *fiber.Ctx) error {
	tunnels, err := h.service.GetTunnels()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to get tunnels"})
	}

	return c.JSON(fiber.Map{"tunnels": tunnels})
}

func (h *Handler) GetRequests(c *fiber.Ctx) error {
	subdomain := c.Params("subdomain")
	port := c.Params("port")
	tunnels, err := h.service.GetRequests(subdomain, port)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to get requests"})
	}

	return c.JSON(fiber.Map{"requests": tunnels})
}

func (h *Handler) RenderResponse(c *fiber.Ctx) error {
	requestId := c.Params("id")
	request, err := h.service.GetRequestById(requestId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to get requests"})
	}

	headersMap := make(map[string][]string)
	err = json.Unmarshal([]byte(request.ResponseHeaders), &headersMap)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to parse response headers"})
	}

	contentType := headersMap["Content-Type"]
	if len(contentType) == 0 {
		contentType = []string{"text/html; charset=utf-8"}
	}

	contentLength := headersMap["Content-Length"]
	if len(contentLength) == 0 {
		contentLength = []string{fmt.Sprintf("%d", len(request.ResponseBody))}
	}

	c.Response().Header.Set("Content-Type", contentType[0])
	c.Response().Header.Set("Content-Length", contentLength[0])

	c.Response().BodyWriter().Write(request.ResponseBody)
	return nil
}

package handler

import (
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
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

func (h *Handler) ReplayRequest(c *fiber.Ctx) error {
	requestId := c.Params("id")
	request, err := h.service.GetRequestById(requestId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to get requests"})
	}

	headersMap := make(map[string][]string)
	err = json.Unmarshal([]byte(request.Headers), &headersMap)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to parse response headers"})
	}

	client := resty.New().R()
	for key, values := range headersMap {
		if len(values) == 0 {
			continue
		}
		client.SetHeader(key, values[0])
	}

	requestUrl := fmt.Sprintf("https://%s%s", request.Host, request.Url)

	switch request.Method {
	case "GET":
		client.Get(requestUrl)
	case "POST":
		client.Post(requestUrl)
	case "PUT":
		client.Put(requestUrl)
	case "DELETE":
		client.Delete(requestUrl)
	case "PATCH":
		client.Patch(requestUrl)
	case "OPTIONS":
		client.Options(requestUrl)
	case "HEAD":
		client.Head(requestUrl)
	}

	return c.JSON(fiber.Map{"message": "replayed request"})
}

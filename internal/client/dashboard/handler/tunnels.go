package handler

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
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

	_type := c.Query("type")
	if _type == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "type query param is required"})
	}

	var headers datatypes.JSON
	var body []byte

	if _type == "request" {
		headers = request.Headers
		body = request.Body
	} else {
		headers = request.ResponseHeaders
		body = request.ResponseBody
	}

	headersMap := make(map[string][]string)
	err = json.Unmarshal([]byte(headers), &headersMap)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to parse response headers"})
	}

	contentType := headersMap["Content-Type"]
	if len(contentType) == 0 {
		contentType = []string{"text/html; charset=utf-8"}
	}

	contentLength := headersMap["Content-Length"]
	if len(contentLength) == 0 {
		contentLength = []string{fmt.Sprintf("%d", len(body))}
	}

	c.Response().Header.Set("Content-Type", contentType[0])
	c.Response().Header.Set("Content-Length", contentLength[0])

	c.Response().BodyWriter().Write(body)
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

	client.SetHeader("X-Portr-Replayed-Request-Id", requestId)

	client.SetBody(request.Body)

	requestUrl := fmt.Sprintf("https://%s%s", request.Host, request.Url)

	var response *resty.Response

	switch request.Method {
	case "GET":
		response, err = client.Get(requestUrl)
	case "POST":
		response, err = client.Post(requestUrl)
	case "PUT":
		response, err = client.Put(requestUrl)
	case "DELETE":
		response, err = client.Delete(requestUrl)
	case "PATCH":
		response, err = client.Patch(requestUrl)
	case "OPTIONS":
		response, err = client.Options(requestUrl)
	case "HEAD":
		response, err = client.Head(requestUrl)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to replay request"})
	}

	xPortrErrorReason := response.Header().Get("X-Portr-Error-Reason")
	if xPortrErrorReason != "" {
		switch xPortrErrorReason {
		case "unregistered-subdomain":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "The tunnel is not active. Please start the tunnel and try again"})
		case "local-server-not-online":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "The local server is not online. Please start the local server and try again"})
		case "connection-lost":
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"message": "The tunnel connection was lost. Please try again in a bit."})
		}
	}

	return c.JSON(fiber.Map{"message": "replayed request"})
}

func (h *Handler) DeleteRequests(c *fiber.Ctx) error {
	subdomain := c.Params("subdomain")
	port := c.Params("port")

	localPort, err := strconv.Atoi(port)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "invalid port"})
	}

	deletedCount, err := h.service.DeleteTunnelLogs(subdomain, localPort)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to delete tunnel logs"})
	}

	return c.JSON(fiber.Map{"deleted_count": deletedCount})
}

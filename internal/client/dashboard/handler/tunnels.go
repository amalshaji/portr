package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/amalshaji/portr/internal/client/db"
	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
)

type replayRequestInput struct {
	Method       string            `json:"method"`
	Path         string            `json:"path"`
	Headers      map[string]string `json:"headers"`
	Body         string            `json:"body"`
	BodyEncoding string            `json:"body_encoding"`
}

type websocketEventPayload struct {
	ID            string `json:"id"`
	Direction     string `json:"direction"`
	Opcode        int    `json:"opcode"`
	OpcodeName    string `json:"opcode_name"`
	IsFinal       bool   `json:"is_final"`
	Payload       string `json:"payload"`
	PayloadText   string `json:"payload_text,omitempty"`
	PayloadLength int    `json:"payload_length"`
	LoggedAt      string `json:"logged_at"`
}

func decodeWebSocketPayloadText(event db.WebSocketEvent) (string, bool) {
	switch event.Opcode {
	case 0, 1, 2:
		if len(event.Payload) == 0 || !utf8.Valid(event.Payload) {
			return "", false
		}
		return string(event.Payload), true
	default:
		return "", false
	}
}

var allowedReplayMethods = map[string]struct{}{
	"DELETE":  {},
	"GET":     {},
	"HEAD":    {},
	"OPTIONS": {},
	"PATCH":   {},
	"POST":    {},
	"PUT":     {},
	"TRACE":   {},
}

func decodeHeaderValues(raw datatypes.JSON) (map[string][]string, error) {
	headers := make(map[string][]string)
	if len(raw) == 0 {
		return headers, nil
	}

	if err := json.Unmarshal(raw, &headers); err != nil {
		return nil, err
	}

	return headers, nil
}

func flattenHeaders(headers map[string][]string) map[string]string {
	flat := make(map[string]string, len(headers))
	for key, values := range headers {
		if len(values) == 0 {
			continue
		}
		flat[key] = values[0]
	}
	return flat
}

func decodeReplayBody(body string, encoding string) ([]byte, error) {
	if strings.EqualFold(encoding, "base64") {
		return base64.StdEncoding.DecodeString(body)
	}

	return []byte(body), nil
}

func normalizeReplayMethod(method string) string {
	method = strings.TrimSpace(strings.ToUpper(method))
	if method == "" {
		return ""
	}
	return method
}

func executeReplay(method string, targetURL string, headers map[string]string, body []byte) (*resty.Response, error) {
	client := resty.New().R()
	for key, value := range headers {
		if value == "" {
			continue
		}
		if strings.EqualFold(key, "Content-Length") {
			continue
		}
		client.SetHeader(key, value)
	}

	client.SetBody(body)
	return client.Execute(method, targetURL)
}

func replayErrorMessage(response *resty.Response) (int, string, bool) {
	xPortrErrorReason := response.Header().Get("X-Portr-Error-Reason")
	if xPortrErrorReason == "" {
		return 0, "", false
	}

	switch xPortrErrorReason {
	case "unregistered-subdomain":
		return fiber.StatusInternalServerError, "The tunnel is not active. Please start the tunnel and try again", true
	case "local-server-not-online":
		return fiber.StatusInternalServerError, "The local server is not online. Please start the local server and try again", true
	case "connection-lost":
		return fiber.StatusServiceUnavailable, "The tunnel connection was lost. Please try again in a bit.", true
	default:
		return fiber.StatusBadGateway, "Failed to replay request", true
	}
}

func serializeWebSocketEvent(event db.WebSocketEvent) websocketEventPayload {
	payload := websocketEventPayload{
		ID:            event.ID,
		Direction:     event.Direction,
		Opcode:        event.Opcode,
		OpcodeName:    event.OpcodeName,
		IsFinal:       event.IsFinal,
		Payload:       base64.StdEncoding.EncodeToString(event.Payload),
		PayloadLength: event.PayloadLength,
		LoggedAt:      event.LoggedAt.UTC().Format(timeLayout),
	}

	if text, ok := decodeWebSocketPayloadText(event); ok {
		payload.PayloadText = text
	}

	return payload
}

const timeLayout = "2006-01-02T15:04:05.999999999Z07:00"

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
	requests, err := h.service.GetRequests(subdomain, port)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to get requests"})
	}

	return c.JSON(fiber.Map{"requests": requests})
}

func (h *Handler) GetWebSocketSessions(c *fiber.Ctx) error {
	subdomain := c.Params("subdomain")
	port := c.Params("port")

	sessions, err := h.service.GetWebSocketSessions(subdomain, port)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to get websocket sessions"})
	}

	return c.JSON(fiber.Map{"sessions": sessions})
}

func (h *Handler) GetWebSocketSession(c *fiber.Ctx) error {
	sessionID := c.Params("id")

	sessionWithEvents, err := h.service.GetWebSocketSessionByID(sessionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to get websocket session"})
	}

	events := make([]websocketEventPayload, 0, len(sessionWithEvents.Events))
	for _, event := range sessionWithEvents.Events {
		events = append(events, serializeWebSocketEvent(event))
	}

	return c.JSON(fiber.Map{
		"session": sessionWithEvents.Session,
		"events":  events,
	})
}

func (h *Handler) RenderResponse(c *fiber.Ctx) error {
	requestID := c.Params("id")
	request, err := h.service.GetRequestById(requestID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to get requests"})
	}

	renderType := c.Query("type")
	if renderType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "type query param is required"})
	}

	var headers datatypes.JSON
	var body []byte

	if renderType == "request" {
		headers = request.Headers
		body = request.Body
	} else {
		headers = request.ResponseHeaders
		body = request.ResponseBody
	}

	headersMap, err := decodeHeaderValues(headers)
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
	requestID := c.Params("id")
	request, err := h.service.GetRequestById(requestID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to get requests"})
	}

	headersMap, err := decodeHeaderValues(request.Headers)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to parse request headers"})
	}

	headers := flattenHeaders(headersMap)
	headers["X-Portr-Replayed-Request-Id"] = requestID

	requestURL := fmt.Sprintf("https://%s%s", request.Host, request.Url)
	response, err := executeReplay(request.Method, requestURL, headers, request.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to replay request"})
	}

	if status, message, ok := replayErrorMessage(response); ok {
		return c.Status(status).JSON(fiber.Map{"message": message})
	}

	return c.JSON(fiber.Map{"message": "replayed request"})
}

func (h *Handler) ReplayRequestWithEdits(c *fiber.Ctx) error {
	requestID := c.Params("id")
	request, err := h.service.GetRequestById(requestID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to get requests"})
	}

	var input replayRequestInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "invalid replay payload"})
	}

	method := normalizeReplayMethod(input.Method)
	if method == "" {
		method = request.Method
	}
	if _, ok := allowedReplayMethods[method]; !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "unsupported replay method"})
	}

	path := strings.TrimSpace(input.Path)
	if path == "" {
		path = request.Url
	}
	if !strings.HasPrefix(path, "/") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "path must start with /"})
	}

	headersMap, err := decodeHeaderValues(request.Headers)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to parse request headers"})
	}

	headers := flattenHeaders(headersMap)
	if input.Headers != nil {
		headers = make(map[string]string, len(input.Headers))
		for key, value := range input.Headers {
			headers[key] = value
		}
	}
	headers["X-Portr-Replayed-Request-Id"] = requestID

	body := request.Body
	if input.Body != "" || input.BodyEncoding != "" {
		body, err = decodeReplayBody(input.Body, input.BodyEncoding)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "invalid request body encoding"})
		}
	}

	requestURL := fmt.Sprintf("https://%s%s", request.Host, path)
	response, err := executeReplay(method, requestURL, headers, body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to replay request"})
	}

	if status, message, ok := replayErrorMessage(response); ok {
		return c.Status(status).JSON(fiber.Map{"message": message})
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

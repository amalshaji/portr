package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/amalshaji/portr/internal/client/db"
	clientreplay "github.com/amalshaji/portr/internal/client/replay"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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

const (
	defaultPageSize = 100
	maxPageSize     = 500
)

func pageParams(c *fiber.Ctx) (limit, offset int) {
	limit = c.QueryInt("limit", defaultPageSize)
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}

	offset = c.QueryInt("offset", 0)
	if offset < 0 {
		offset = 0
	}

	return limit, offset
}

func (h *Handler) GetTunnels(c *fiber.Ctx) error {
	limit, offset := pageParams(c)
	page, err := h.service.GetTunnels(limit, offset, c.Query("search"), c.Query("status"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to get tunnels"})
	}

	return c.JSON(fiber.Map{
		"tunnels": page.Tunnels,
		"total":   page.Total,
		"stats":   page.Stats,
	})
}

func (h *Handler) GetRequests(c *fiber.Ctx) error {
	subdomain := c.Params("subdomain")
	port := c.Params("port")
	limit, offset := pageParams(c)

	requests, total, err := h.service.GetRequests(subdomain, port, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to get requests"})
	}

	return c.JSON(fiber.Map{"requests": requests, "total": total})
}

func (h *Handler) GetRequest(c *fiber.Ctx) error {
	request, err := h.service.GetRequestById(c.Params("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "request not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to get request"})
	}

	return c.JSON(fiber.Map{"request": request})
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

	if _, err := clientreplay.Execute(request, clientreplay.EditOptions{Scheme: h.replayScheme()}); err != nil {
		return replayHTTPError(c, err)
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

	edit := clientreplay.EditOptions{
		Method:         input.Method,
		Path:           input.Path,
		Scheme:         h.replayScheme(),
		Headers:        input.Headers,
		ReplaceHeaders: input.Headers != nil,
		Body: clientreplay.BodyOverride{
			Set:      input.Body != "" || input.BodyEncoding != "",
			Value:    input.Body,
			Encoding: input.BodyEncoding,
		},
	}

	if _, err := clientreplay.Execute(request, edit); err != nil {
		return replayHTTPError(c, err)
	}

	return c.JSON(fiber.Map{"message": "replayed request"})
}

func (h *Handler) replayScheme() string {
	if h.config.UseLocalHost {
		return "http"
	}
	return "https"
}

func replayHTTPError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, clientreplay.ErrUnsupportedMethod):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "unsupported replay method"})
	case errors.Is(err, clientreplay.ErrInvalidPath):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "path must start with /"})
	case errors.Is(err, clientreplay.ErrInvalidBodyEncoding):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "invalid request body encoding"})
	default:
		var failure *clientreplay.Failure
		if errors.As(err, &failure) {
			return c.Status(failure.StatusCode).JSON(fiber.Map{"message": failure.Message})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "failed to replay request"})
	}
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

package ssh

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/amalshaji/portr/internal/client/db"
	"github.com/amalshaji/portr/internal/utils"
	"github.com/oklog/ulid/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type webSocketFrame struct {
	Raw           []byte
	Opcode        byte
	IsFinal       bool
	Payload       []byte
	PayloadLength int
}

func isWebSocketUpgrade(request *http.Request) bool {
	if request == nil {
		return false
	}

	if !strings.EqualFold(request.Header.Get("Upgrade"), "websocket") {
		return false
	}

	return strings.Contains(strings.ToLower(request.Header.Get("Connection")), "upgrade")
}

func drainBufferedBytes(reader *bufio.Reader) []byte {
	if reader == nil || reader.Buffered() == 0 {
		return nil
	}

	buffered := make([]byte, reader.Buffered())
	if _, err := io.ReadFull(reader, buffered); err != nil {
		return nil
	}

	return buffered
}

func websocketOpcodeName(opcode byte) string {
	switch opcode {
	case 0x0:
		return "continuation"
	case 0x1:
		return "text"
	case 0x2:
		return "binary"
	case 0x8:
		return "close"
	case 0x9:
		return "ping"
	case 0xA:
		return "pong"
	default:
		return "unknown"
	}
}

func readWebSocketFrame(reader io.Reader) (*webSocketFrame, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}

	raw := append([]byte{}, header...)
	isFinal := header[0]&0x80 != 0
	opcode := header[0] & 0x0F

	payloadLength := int64(header[1] & 0x7F)
	switch payloadLength {
	case 126:
		extended := make([]byte, 2)
		if _, err := io.ReadFull(reader, extended); err != nil {
			return nil, err
		}
		raw = append(raw, extended...)
		payloadLength = int64(binary.BigEndian.Uint16(extended))
	case 127:
		extended := make([]byte, 8)
		if _, err := io.ReadFull(reader, extended); err != nil {
			return nil, err
		}
		raw = append(raw, extended...)
		payloadLength = int64(binary.BigEndian.Uint64(extended))
	}

	masked := header[1]&0x80 != 0
	var maskingKey []byte
	if masked {
		maskingKey = make([]byte, 4)
		if _, err := io.ReadFull(reader, maskingKey); err != nil {
			return nil, err
		}
		raw = append(raw, maskingKey...)
	}

	payload := make([]byte, payloadLength)
	if payloadLength > 0 {
		if _, err := io.ReadFull(reader, payload); err != nil {
			return nil, err
		}
		raw = append(raw, payload...)
	}

	decodedPayload := append([]byte{}, payload...)
	if masked {
		for idx := range decodedPayload {
			decodedPayload[idx] ^= maskingKey[idx%4]
		}
	}

	return &webSocketFrame{
		Raw:           raw,
		Opcode:        opcode,
		IsFinal:       isFinal,
		Payload:       decodedPayload,
		PayloadLength: len(decodedPayload),
	}, nil
}

func isIgnorableWebSocketError(err error) bool {
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "use of closed network connection")
}

func (s *SshClient) logWebSocketSession(handshakeRequestID string, request *http.Request, response *http.Response) string {
	requestHeadersBytes, err := json.Marshal(request.Header)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to marshal websocket request headers", err)
		}
		return ""
	}

	responseHeadersBytes, err := json.Marshal(response.Header)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to marshal websocket response headers", err)
		}
		return ""
	}

	now := time.Now().UTC()
	session := db.WebSocketSession{
		ID:                 ulid.Make().String(),
		HandshakeRequestID: handshakeRequestID,
		Subdomain:          s.config.Tunnel.Subdomain,
		Localport:          s.config.Tunnel.Port,
		Host:               request.Host,
		URL:                request.URL.String(),
		Method:             request.Method,
		RequestHeaders:     datatypes.JSON(requestHeadersBytes),
		ResponseStatusCode: response.StatusCode,
		ResponseHeaders:    datatypes.JSON(responseHeadersBytes),
		StartedAt:          now,
	}

	if err := s.db.Conn.Create(&session).Error; err != nil {
		if s.config.Debug {
			s.logDebug("Failed to persist websocket session", err)
		}
		return ""
	}

	return session.ID
}

func (s *SshClient) recordWebSocketEvent(sessionID string, direction string, frame *webSocketFrame) {
	if sessionID == "" || frame == nil {
		return
	}

	now := time.Now().UTC()
	event := db.WebSocketEvent{
		ID:            ulid.Make().String(),
		SessionID:     sessionID,
		Direction:     direction,
		Opcode:        int(frame.Opcode),
		OpcodeName:    websocketOpcodeName(frame.Opcode),
		IsFinal:       frame.IsFinal,
		Payload:       frame.Payload,
		PayloadLength: frame.PayloadLength,
		LoggedAt:      now,
	}

	if err := s.db.Conn.Create(&event).Error; err != nil {
		if s.config.Debug {
			s.logDebug("Failed to persist websocket event", err)
		}
		return
	}

	updates := map[string]interface{}{
		"last_event_at": now,
		"event_count":   gorm.Expr("event_count + 1"),
	}
	if direction == "client" {
		updates["client_event_count"] = gorm.Expr("client_event_count + 1")
	} else {
		updates["server_event_count"] = gorm.Expr("server_event_count + 1")
	}

	if frame.Opcode == 0x8 {
		updates["closed_at"] = now
		if len(frame.Payload) >= 2 {
			closeCode := int(binary.BigEndian.Uint16(frame.Payload[:2]))
			updates["close_code"] = closeCode
			if len(frame.Payload) > 2 {
				updates["close_reason"] = string(frame.Payload[2:])
			}
		}
	}

	if err := s.db.Conn.Model(&db.WebSocketSession{}).
		Where("id = ?", sessionID).
		Updates(updates).Error; err != nil && s.config.Debug {
		s.logDebug("Failed to update websocket session", err)
	}
}

func (s *SshClient) closeWebSocketSession(sessionID string, err error) {
	if sessionID == "" {
		return
	}

	now := time.Now().UTC()
	updates := map[string]interface{}{
		"closed_at": now,
	}
	if err != nil && !isIgnorableWebSocketError(err) {
		updates["close_reason"] = err.Error()
	}

	if updateErr := s.db.Conn.Model(&db.WebSocketSession{}).
		Where("id = ? AND closed_at IS NULL", sessionID).
		Updates(updates).Error; updateErr != nil && s.config.Debug {
		s.logDebug("Failed to close websocket session", updateErr)
	}
}

func (s *SshClient) proxyWebSocketFrames(sessionID string, direction string, reader io.Reader, writer net.Conn) error {
	for {
		frame, err := readWebSocketFrame(reader)
		if err != nil {
			return err
		}

		if _, err := writer.Write(frame.Raw); err != nil {
			return err
		}

		s.recordWebSocketEvent(sessionID, direction, frame)
	}
}

func (s *SshClient) websocketTunnel(sessionID string, clientReader io.Reader, serverConn net.Conn, serverReader io.Reader, clientConn net.Conn) {
	var (
		once      sync.Once
		firstErr  error
		errMu     sync.Mutex
		completed sync.WaitGroup
	)

	closeAll := func() {
		once.Do(func() {
			_ = clientConn.Close()
			_ = serverConn.Close()
		})
	}

	recordErr := func(err error) {
		if isIgnorableWebSocketError(err) {
			return
		}
		errMu.Lock()
		defer errMu.Unlock()
		if firstErr == nil {
			firstErr = err
		}
	}

	completed.Add(2)
	s.goSafe("websocket tunnel client->server", func() {
		defer completed.Done()
		recordErr(s.proxyWebSocketFrames(sessionID, "client", clientReader, serverConn))
		closeAll()
	})
	s.goSafe("websocket tunnel server->client", func() {
		defer completed.Done()
		recordErr(s.proxyWebSocketFrames(sessionID, "server", serverReader, clientConn))
		closeAll()
	})

	completed.Wait()
	closeAll()
	s.closeWebSocketSession(sessionID, firstErr)
}

func (s *SshClient) handleWebSocketRequest(
	src net.Conn,
	srcReader *bufio.Reader,
	srcWriter *bufio.Writer,
	request *http.Request,
	localEndpoint string,
) error {
	dst, err := net.Dial("tcp", localEndpoint)
	if err != nil {
		htmlContent := []byte(utils.LocalServerNotOnline(localEndpoint))
		response := &http.Response{
			Status:        "503 Service Unavailable",
			StatusCode:    http.StatusServiceUnavailable,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        http.Header{},
			ContentLength: int64(len(htmlContent)),
			Body:          io.NopCloser(bytes.NewReader(htmlContent)),
		}
		response.Header.Set("Content-Type", "text/html")
		response.Header.Set("X-Portr-Error", "true")
		response.Header.Set("X-Portr-Error-Reason", "local-server-not-online")
		if writeErr := response.Write(srcWriter); writeErr != nil {
			return writeErr
		}
		return srcWriter.Flush()
	}

	shouldCloseDst := true
	defer func() {
		if shouldCloseDst {
			_ = dst.Close()
		}
	}()

	dstReader := bufio.NewReader(dst)
	dstWriter := bufio.NewWriter(dst)

	requestBody, err := io.ReadAll(request.Body)
	if err != nil {
		return err
	}
	_ = request.Body.Close()
	request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

	if err := request.Write(dstWriter); err != nil {
		return err
	}
	if err := dstWriter.Flush(); err != nil {
		return err
	}

	response, err := http.ReadResponse(dstReader, request)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusSwitchingProtocols {
		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		_ = response.Body.Close()
		response.Body = io.NopCloser(bytes.NewBuffer(responseBody))

		if err := response.Write(srcWriter); err != nil {
			return err
		}
		if err := srcWriter.Flush(); err != nil {
			return err
		}

		s.logHttpRequest(ulid.Make().String(), request, requestBody, response, responseBody)
		return nil
	}

	if err := response.Write(srcWriter); err != nil {
		return err
	}
	if err := srcWriter.Flush(); err != nil {
		return err
	}

	handshakeRequestID := ulid.Make().String()
	s.logHttpRequest(handshakeRequestID, request, requestBody, response, nil)
	sessionID := s.logWebSocketSession(handshakeRequestID, request, response)

	clientBuffered := drainBufferedBytes(srcReader)
	serverBuffered := drainBufferedBytes(dstReader)

	shouldCloseDst = false
	s.websocketTunnel(
		sessionID,
		io.MultiReader(bytes.NewReader(clientBuffered), src),
		dst,
		io.MultiReader(bytes.NewReader(serverBuffered), dst),
		src,
	)

	return nil
}

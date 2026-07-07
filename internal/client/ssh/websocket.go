package ssh

import (
	"bufio"
	"bytes"
	"context"
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
	"github.com/oklog/ulid/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type webSocketFrame struct {
	Opcode        byte
	IsFinal       bool
	Payload       []byte
	PayloadLength int
}

type webSocketFrameHeader struct {
	raw           []byte
	opcode        byte
	isFinal       bool
	masked        bool
	maskingKey    [4]byte
	payloadLength uint64
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

func readWebSocketFrameHeader(reader io.Reader) (*webSocketFrameHeader, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}

	raw := append([]byte{}, header...)
	isFinal := header[0]&0x80 != 0
	opcode := header[0] & 0x0F

	payloadLength := uint64(header[1] & 0x7F)
	switch payloadLength {
	case 126:
		extended := make([]byte, 2)
		if _, err := io.ReadFull(reader, extended); err != nil {
			return nil, err
		}
		raw = append(raw, extended...)
		payloadLength = uint64(binary.BigEndian.Uint16(extended))
	case 127:
		extended := make([]byte, 8)
		if _, err := io.ReadFull(reader, extended); err != nil {
			return nil, err
		}
		raw = append(raw, extended...)
		payloadLength = binary.BigEndian.Uint64(extended)
		if payloadLength&(uint64(1)<<63) != 0 {
			return nil, errors.New("invalid websocket payload length")
		}
	}

	masked := header[1]&0x80 != 0
	var maskingKey [4]byte
	if masked {
		if _, err := io.ReadFull(reader, maskingKey[:]); err != nil {
			return nil, err
		}
		raw = append(raw, maskingKey[:]...)
	}

	return &webSocketFrameHeader{
		raw:           raw,
		opcode:        opcode,
		isFinal:       isFinal,
		masked:        masked,
		maskingKey:    maskingKey,
		payloadLength: payloadLength,
	}, nil
}

func isIgnorableWebSocketError(err error) bool {
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "use of closed network connection")
}

func (s *SshClient) logWebSocketSession(handshakeRequestID string, request *http.Request, response *http.Response) string {
	return s.logWebSocketSessionWithID(ulid.Make().String(), handshakeRequestID, request, response)
}

func (s *SshClient) logWebSocketSessionWithID(sessionID, handshakeRequestID string, request *http.Request, response *http.Response) string {
	if !s.requestLoggingEnabled() {
		return ""
	}

	requestHeaders := redactHeaderValues(request.Header, s.config.RedactHeaders)
	requestHeadersBytes, err := json.Marshal(requestHeaders)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to marshal websocket request headers", err)
		}
		return ""
	}

	responseHeaders := redactHeaderValues(response.Header, s.config.RedactHeaders)
	responseHeadersBytes, err := json.Marshal(responseHeaders)
	if err != nil {
		if s.config.Debug {
			s.logDebug("Failed to marshal websocket response headers", err)
		}
		return ""
	}

	now := time.Now().UTC()
	session := db.WebSocketSession{
		ID:                 sessionID,
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
	if !s.requestLoggingEnabled() {
		return
	}
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
	if !s.requestLoggingEnabled() {
		return
	}
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
		frame, err := forwardWebSocketFrame(reader, writer)
		if err != nil {
			return err
		}
		if sessionID != "" {
			s.submitCapture(websocketEventCaptureTask{sessionID: sessionID, direction: direction, frame: frame})
		}
	}
}

func forwardWebSocketFrame(reader io.Reader, writer io.Writer) (*webSocketFrame, error) {
	header, err := readWebSocketFrameHeader(reader)
	if err != nil {
		return nil, err
	}
	if header.payloadLength > uint64(^uint(0)>>1) {
		return nil, errors.New("websocket payload length exceeds platform capacity")
	}
	if err := writeAll(writer, header.raw); err != nil {
		return nil, err
	}

	captureLength := int(header.payloadLength)
	if captureLength > maxCapturedBodyBytes {
		captureLength = maxCapturedBodyBytes
	}
	captured := make([]byte, 0, captureLength)
	buffer := make([]byte, 32*1024)
	remaining := header.payloadLength
	var payloadOffset uint64
	for remaining > 0 {
		chunkLength := uint64(len(buffer))
		if remaining < chunkLength {
			chunkLength = remaining
		}
		chunk := buffer[:int(chunkLength)]
		if _, err := io.ReadFull(reader, chunk); err != nil {
			return nil, err
		}
		if err := writeAll(writer, chunk); err != nil {
			return nil, err
		}
		for index := 0; index < len(chunk) && len(captured) < captureLength; index++ {
			value := chunk[index]
			if header.masked {
				value ^= header.maskingKey[(payloadOffset+uint64(index))%4]
			}
			captured = append(captured, value)
		}
		payloadOffset += chunkLength
		remaining -= chunkLength
	}

	return &webSocketFrame{
		Opcode:        header.opcode,
		IsFinal:       header.isFinal,
		Payload:       captured,
		PayloadLength: int(header.payloadLength),
	}, nil
}

func writeAll(writer io.Writer, payload []byte) error {
	for len(payload) > 0 {
		written, err := writer.Write(payload)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		payload = payload[written:]
	}
	return nil
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
	if sessionID != "" {
		captureCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		s.submitCaptureContext(captureCtx, websocketCloseCaptureTask{sessionID: sessionID, err: firstErr})
	}
}

func (s *SshClient) handleWebSocketRequest(
	src net.Conn,
	srcReader *bufio.Reader,
	srcWriter *bufio.Writer,
	request *http.Request,
	localEndpoint string,
) error {
	dialCtx, cancelDial := context.WithTimeout(request.Context(), 5*time.Second)
	defer cancelDial()
	dst, err := (&net.Dialer{KeepAlive: 30 * time.Second}).DialContext(dialCtx, "tcp", localEndpoint)
	if err != nil {
		if writeErr := writeLocalServerUnavailable(srcWriter, localEndpoint); writeErr != nil {
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

	requestCapture := &bodyCapture{}
	if request.Body != nil {
		request.Body = &capturingReadCloser{ReadCloser: request.Body, capture: requestCapture, onDone: func() {}}
		defer request.Body.Close()
	}

	if err := request.Write(dstWriter); err != nil {
		_ = writeLocalServerUnavailable(srcWriter, localEndpoint)
		_ = srcWriter.Flush()
		return err
	}
	requestBody := requestCapture.Bytes()
	if err := dstWriter.Flush(); err != nil {
		_ = writeLocalServerUnavailable(srcWriter, localEndpoint)
		_ = srcWriter.Flush()
		return err
	}

	response, err := http.ReadResponse(dstReader, request)
	if err != nil {
		_ = writeLocalServerUnavailable(srcWriter, localEndpoint)
		_ = srcWriter.Flush()
		return err
	}

	if response.StatusCode != http.StatusSwitchingProtocols {
		responseCapture := &bodyCapture{}
		response.Body = &capturingReadCloser{ReadCloser: response.Body, capture: responseCapture, onDone: func() {}}
		defer response.Body.Close()
		if err := response.Write(srcWriter); err != nil {
			return err
		}
		if err := srcWriter.Flush(); err != nil {
			return err
		}

		responseBody := responseCapture.Bytes()
		requestID := ulid.Make().String()
		s.submitCapture(httpCaptureTask{
			id:           requestID,
			request:      request,
			requestBody:  requestBody,
			response:     response,
			responseBody: responseBody,
			bytesIn:      requestCapture.Size(),
			bytesOut:     responseCapture.Size(),
		})
		return nil
	}

	if err := response.Write(srcWriter); err != nil {
		return err
	}
	if err := srcWriter.Flush(); err != nil {
		return err
	}

	handshakeRequestID := ulid.Make().String()
	sessionID := ulid.Make().String()
	if !s.submitCapture(websocketOpenCaptureTask{
		handshake: httpCaptureTask{
			id:          handshakeRequestID,
			request:     request,
			requestBody: requestBody,
			response:    response,
			bytesIn:     requestCapture.Size(),
		},
		sessionID: sessionID,
		request:   request,
		response:  response,
	}) {
		sessionID = ""
	}

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

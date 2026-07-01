package sshd

import (
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/charmbracelet/log"
	sshserver "github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

const forwardedTCPChannelType = "forwarded-tcpip"

type remoteForwardRequest struct {
	BindAddr string
	BindPort uint32
}

type remoteForwardSuccess struct {
	BindPort uint32
}

type remoteForwardCancelRequest struct {
	BindAddr string
	BindPort uint32
}

type remoteForwardChannelData struct {
	DestAddr   string
	DestPort   uint32
	OriginAddr string
	OriginPort uint32
}

type boundForward struct {
	closed sync.Once
	close  func()
}

type forwardedTCPHandler struct {
	mu       sync.Mutex
	forwards map[string]*boundForward
	onBound  func(sshserver.Context, string, uint32) error
	onClosed func(sshserver.Context, string, uint32)
}

func (h *forwardedTCPHandler) HandleSSHRequest(ctx sshserver.Context, _ *sshserver.Server, req *gossh.Request) (bool, []byte) {
	switch req.Type {
	case "tcpip-forward":
		return h.open(ctx, req)
	case "cancel-tcpip-forward":
		return h.cancel(req)
	default:
		return false, nil
	}
}

func (h *forwardedTCPHandler) open(ctx sshserver.Context, req *gossh.Request) (bool, []byte) {
	var payload remoteForwardRequest
	if err := gossh.Unmarshal(req.Payload, &payload); err != nil {
		return false, nil
	}
	if h.onBound == nil {
		return false, []byte("port forwarding is disabled")
	}

	requestedAddr := net.JoinHostPort(payload.BindAddr, strconv.Itoa(int(payload.BindPort)))
	listener, err := net.Listen("tcp", requestedAddr)
	if err != nil {
		return false, nil
	}

	_, portString, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		_ = listener.Close()
		return false, nil
	}
	port, err := strconv.ParseUint(portString, 10, 32)
	if err != nil {
		_ = listener.Close()
		return false, nil
	}
	boundPort := uint32(port)
	boundAddr := net.JoinHostPort(payload.BindAddr, portString)

	if err := h.onBound(ctx, payload.BindAddr, boundPort); err != nil {
		_ = listener.Close()
		log.Error("Failed to register bound SSH forward", "address", boundAddr, "error", err)
		return false, nil
	}

	forward := &boundForward{}
	h.mu.Lock()
	if h.forwards == nil {
		h.forwards = make(map[string]*boundForward)
	}
	h.forwards[boundAddr] = forward
	h.mu.Unlock()

	closeForward := func() {
		forward.closed.Do(func() {
			_ = listener.Close()
			h.mu.Lock()
			current := h.forwards[boundAddr]
			if current == forward {
				delete(h.forwards, boundAddr)
			}
			h.mu.Unlock()
			if current == forward && h.onClosed != nil {
				h.onClosed(ctx, payload.BindAddr, boundPort)
			}
		})
	}
	forward.close = closeForward

	go func() {
		<-ctx.Done()
		closeForward()
	}()
	go h.accept(ctx, listener, payload.BindAddr, boundPort, closeForward)

	return true, gossh.Marshal(&remoteForwardSuccess{BindPort: boundPort})
}

func (h *forwardedTCPHandler) cancel(req *gossh.Request) (bool, []byte) {
	var payload remoteForwardCancelRequest
	if err := gossh.Unmarshal(req.Payload, &payload); err != nil {
		return false, nil
	}

	addr := net.JoinHostPort(payload.BindAddr, strconv.Itoa(int(payload.BindPort)))
	h.mu.Lock()
	forward := h.forwards[addr]
	h.mu.Unlock()
	if forward == nil {
		return true, nil
	}
	forward.close()
	return true, nil
}

func (h *forwardedTCPHandler) accept(
	ctx sshserver.Context,
	listener net.Listener,
	destAddr string,
	destPort uint32,
	closeForward func(),
) {
	defer closeForward()
	connection, ok := ctx.Value(sshserver.ContextKeyConn).(*gossh.ServerConn)
	if !ok || connection == nil {
		return
	}

	for {
		localConn, err := listener.Accept()
		if err != nil {
			return
		}

		originHost, originPortString, err := net.SplitHostPort(localConn.RemoteAddr().String())
		if err != nil {
			_ = localConn.Close()
			continue
		}
		originPort, err := strconv.ParseUint(originPortString, 10, 32)
		if err != nil {
			_ = localConn.Close()
			continue
		}

		payload := gossh.Marshal(&remoteForwardChannelData{
			DestAddr:   destAddr,
			DestPort:   destPort,
			OriginAddr: originHost,
			OriginPort: uint32(originPort),
		})
		go proxyForwardedConnection(connection, localConn, payload)
	}
}

func proxyForwardedConnection(connection *gossh.ServerConn, localConn net.Conn, payload []byte) {
	channel, requests, err := connection.OpenChannel(forwardedTCPChannelType, payload)
	if err != nil {
		_ = localConn.Close()
		return
	}
	go gossh.DiscardRequests(requests)

	results := make(chan error, 2)
	go func() {
		_, copyErr := io.Copy(channel, localConn)
		_ = channel.CloseWrite()
		results <- copyErr
	}()
	go func() {
		_, copyErr := io.Copy(localConn, channel)
		if tcp, ok := localConn.(*net.TCPConn); ok {
			_ = tcp.CloseWrite()
		}
		results <- copyErr
	}()
	if firstErr := <-results; firstErr != nil {
		_ = channel.Close()
		_ = localConn.Close()
	}
	<-results
	_ = channel.Close()
	_ = localConn.Close()
}

func forwardKey(host string, port uint32) string {
	return net.JoinHostPort(host, strconv.Itoa(int(port)))
}

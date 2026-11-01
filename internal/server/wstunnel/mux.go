package wstunnel

import (
	"context"
	"net"

	"github.com/hashicorp/yamux"
)

// streamMux carries a session's tunnel streams.
type streamMux interface {
	OpenStream(context.Context) (muxStream, error)
	CloseChan() <-chan struct{}
	Close() error
}

// muxStream is one tunnel stream. CloseWrite ends the outgoing direction while
// the response keeps flowing back; Close releases the stream.
type muxStream interface {
	net.Conn
	CloseWrite() error
}

type yamuxMux struct {
	*yamux.Session
}

func (m yamuxMux) OpenStream(context.Context) (muxStream, error) {
	stream, err := m.Session.OpenStream()
	if err != nil {
		return nil, err
	}
	return yamuxStream{stream}, nil
}

// yamuxStream maps CloseWrite to yamux Close, which only half-closes.
type yamuxStream struct {
	*yamux.Stream
}

func (s yamuxStream) CloseWrite() error {
	return s.Stream.Close()
}

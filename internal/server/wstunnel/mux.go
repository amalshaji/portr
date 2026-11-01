package wstunnel

import (
	"context"
	"net"

	"github.com/hashicorp/yamux"
)

// streamMux carries a session's tunnel streams. Close on a stream returned by
// OpenStream must only half-close it, so responses keep flowing back.
type streamMux interface {
	OpenStream(context.Context) (net.Conn, error)
	CloseChan() <-chan struct{}
	Close() error
}

type yamuxMux struct {
	*yamux.Session
}

func (m yamuxMux) OpenStream(context.Context) (net.Conn, error) {
	return m.Session.OpenStream()
}

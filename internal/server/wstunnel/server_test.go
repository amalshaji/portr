package wstunnel

import (
	"context"
	"net"
	"testing"
	"time"

	serverdb "github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/server/service"
	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type closeRecorder struct {
	closed chan struct{}
}

func (c *closeRecorder) Close() error {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
	return nil
}

type closeListener struct {
	closeRecorder
}

func (l *closeListener) Accept() (net.Conn, error) {
	<-l.closed
	return nil, net.ErrClosed
}

func (*closeListener) Addr() net.Addr {
	return testAddr("test")
}

type testAddr string

func (a testAddr) Network() string { return string(a) }
func (a testAddr) String() string  { return string(a) }

func newTestService(t *testing.T) (*service.Service, *gorm.DB) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := database.AutoMigrate(&serverdb.TeamUser{}, &serverdb.Connection{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return service.New(&serverdb.Db{Conn: database}), database
}

func TestManagerShutdownClosesActiveSessions(t *testing.T) {
	tunnelService, database := newTestService(t)
	connection := serverdb.Connection{ID: "conn-1", Type: "tcp", Status: "active"}
	if err := database.Create(&connection).Error; err != nil {
		t.Fatalf("create connection: %v", err)
	}

	listener := &closeListener{closeRecorder: closeRecorder{closed: make(chan struct{})}}
	transport := &closeRecorder{closed: make(chan struct{})}
	sess := &session{
		id:         "session-1",
		connection: &connection,
		conn:       transport,
		streams:    make(map[string]*streamQueue),
		listener:   listener,
		closed:     make(chan struct{}),
	}
	manager := New(nil, tunnelService)
	manager.sessions[sess.id] = sess

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown manager: %v", err)
	}
	if err := manager.Shutdown(ctx); err != nil {
		t.Fatalf("repeat shutdown manager: %v", err)
	}

	select {
	case <-transport.closed:
	default:
		t.Fatal("expected websocket transport to close")
	}
	select {
	case <-sess.closed:
	default:
		t.Fatal("expected session to close")
	}
	if _, err := listener.Accept(); err == nil {
		t.Fatal("expected tcp listener to close")
	}
	if len(manager.sessions) != 0 {
		t.Fatalf("expected sessions to be removed, got %d", len(manager.sessions))
	}

	var stored serverdb.Connection
	if err := database.First(&stored, "id = ?", connection.ID).Error; err != nil {
		t.Fatalf("reload connection: %v", err)
	}
	if stored.Status != "closed" {
		t.Fatalf("expected closed connection status, got %q", stored.Status)
	}
}

func TestSessionDeliverBackpressuresInsteadOfDroppingFrames(t *testing.T) {
	sess := &session{
		streams: make(map[string]*streamQueue),
		closed:  make(chan struct{}),
	}
	stream := sess.addStream("slow-stream")
	for i := 0; i < cap(stream.frames); i++ {
		sess.deliver(wsproto.Frame{Type: wsproto.TypeData, StreamID: "slow-stream", Data: []byte{byte(i)}})
	}

	delivered := make(chan struct{})
	go func() {
		sess.deliver(wsproto.Frame{Type: wsproto.TypeData, StreamID: "slow-stream", Data: []byte("last")})
		close(delivered)
	}()

	select {
	case <-delivered:
		t.Fatal("deliver returned while the stream queue was full")
	case <-time.After(25 * time.Millisecond):
	}

	<-stream.frames
	select {
	case <-delivered:
	case <-time.After(time.Second):
		t.Fatal("deliver remained blocked after stream capacity became available")
	}

	if got := len(stream.frames); got != cap(stream.frames) {
		t.Fatalf("expected all frames to remain queued, got %d of %d", got, cap(stream.frames))
	}
}

func TestSessionDeliverUnblocksWhenStreamCloses(t *testing.T) {
	sess := &session{
		streams: make(map[string]*streamQueue),
		closed:  make(chan struct{}),
	}
	stream := sess.addStream("closing-stream")
	for i := 0; i < cap(stream.frames); i++ {
		sess.deliver(wsproto.Frame{Type: wsproto.TypeData, StreamID: "closing-stream"})
	}

	delivered := make(chan struct{})
	go func() {
		sess.deliver(wsproto.Frame{Type: wsproto.TypeData, StreamID: "closing-stream"})
		close(delivered)
	}()

	sess.removeStream("closing-stream")
	select {
	case <-delivered:
	case <-time.After(time.Second):
		t.Fatal("deliver remained blocked after the stream closed")
	}
}

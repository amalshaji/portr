package wstunnel

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/websocket"

	serverdb "github.com/amalshaji/portr/internal/server/db"
	"github.com/amalshaji/portr/internal/server/service"
	"github.com/amalshaji/portr/internal/tunnel/wsproto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type blockingConnectionService struct {
	mu             sync.Mutex
	status         string
	closedStarted  chan struct{}
	releaseClosed  chan struct{}
	activeCalled   chan struct{}
	activeCallOnce sync.Once
}

func (*blockingConnectionService) GetReservedConnectionById(context.Context, string) (*serverdb.Connection, error) {
	return nil, nil
}

func (s *blockingConnectionService) MarkConnectionAsActive(context.Context, string) error {
	s.mu.Lock()
	s.status = "active"
	s.mu.Unlock()
	s.activeCallOnce.Do(func() { close(s.activeCalled) })
	return nil
}

func (s *blockingConnectionService) MarkTCPConnectionAsActive(context.Context, string, uint32) error {
	return nil
}

func (s *blockingConnectionService) MarkConnectionAsClosed(context.Context, string) error {
	close(s.closedStarted)
	<-s.releaseClosed
	s.mu.Lock()
	s.status = "closed"
	s.mu.Unlock()
	return nil
}

func (s *blockingConnectionService) currentStatus() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

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

func TestReconnectCannotBeOverwrittenByPreviousSessionClose(t *testing.T) {
	service := &blockingConnectionService{
		status:        "active",
		closedStarted: make(chan struct{}),
		releaseClosed: make(chan struct{}),
		activeCalled:  make(chan struct{}),
	}
	subdomain := "reconnect"
	connection := &serverdb.Connection{ID: "conn-1", Type: "http", Subdomain: &subdomain}
	oldSession := &session{
		id:         "old",
		connection: connection,
		streams:    make(map[string]*streamQueue),
		closed:     make(chan struct{}),
	}
	manager := New(nil, service)
	manager.bySub[subdomain] = []*session{oldSession}
	manager.sessions[oldSession.id] = oldSession

	unregistered := make(chan struct{})
	go func() {
		manager.unregisterSession(oldSession)
		close(unregistered)
	}()
	select {
	case <-service.closedStarted:
	case <-time.After(time.Second):
		t.Fatal("old session did not begin its closed transition")
	}

	newSession := &session{
		id:         "new",
		connection: connection,
		streams:    make(map[string]*streamQueue),
		closed:     make(chan struct{}),
	}
	registered := make(chan error, 1)
	go func() {
		registered <- manager.registerSession(context.Background(), newSession)
	}()

	select {
	case <-service.activeCalled:
		t.Fatal("new session became active before the old closed transition finished")
	case <-time.After(50 * time.Millisecond):
	}
	close(service.releaseClosed)

	select {
	case <-unregistered:
	case <-time.After(time.Second):
		t.Fatal("old session did not finish unregistering")
	}
	select {
	case err := <-registered:
		if err != nil {
			t.Fatalf("register new session: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("new session did not register after the old session closed")
	}
	if got := service.currentStatus(); got != "active" {
		t.Fatalf("expected reconnected status to remain active, got %q", got)
	}
}

func TestSessionDeliverKeepsOtherStreamsResponsiveWhenOneIsFull(t *testing.T) {
	sess := &session{
		streams: make(map[string]*streamQueue),
		closed:  make(chan struct{}),
	}
	slow := sess.addStream("slow-stream")
	fast := sess.addStream("fast-stream")
	for i := 0; i < cap(slow.frames); i++ {
		sess.deliver(wsproto.Frame{Type: wsproto.TypeData, StreamID: "slow-stream", Data: []byte{byte(i)}})
	}
	sess.deliver(wsproto.Frame{Type: wsproto.TypeData, StreamID: "slow-stream", Data: []byte("overflow")})
	sess.deliver(wsproto.Frame{Type: wsproto.TypeData, StreamID: "fast-stream", Data: []byte("ready")})
	select {
	case frame := <-fast.frames:
		if string(frame.Data) != "ready" {
			t.Fatalf("unexpected fast-stream frame %q", frame.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("full stream blocked delivery to another stream")
	}

	sess.streamMu.Lock()
	_, slowPresent := sess.streams["slow-stream"]
	sess.streamMu.Unlock()
	if slowPresent {
		t.Fatal("expected overflowing stream to be removed")
	}
}

func TestStreamCreditWaitUnblocksWhenStreamCloses(t *testing.T) {
	sess := &session{
		streams: make(map[string]*streamQueue),
		closed:  make(chan struct{}),
	}
	stream := sess.addStream("closing-stream")

	waiting := make(chan bool, 1)
	go func() {
		waiting <- stream.takeCredit(sess.closed)
	}()

	sess.removeStream("closing-stream")
	select {
	case acquired := <-waiting:
		if acquired {
			t.Fatal("credit wait succeeded after stream close")
		}
	case <-time.After(time.Second):
		t.Fatal("credit wait remained blocked after the stream closed")
	}
}

func TestHandlerRejectsClientWithoutProtocolVersion(t *testing.T) {
	tunnelService, _ := newTestService(t)
	manager := New(nil, tunnelService)
	server := httptest.NewServer(manager.Handler())
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + wsproto.ConnectPath
	wsConfig, err := websocket.NewConfig(wsURL, server.URL)
	if err != nil {
		t.Fatalf("create websocket config: %v", err)
	}
	// An older client sends credentials but no protocol version header.
	wsConfig.Header = http.Header{}
	wsConfig.Header.Set(wsproto.ConnectionIDHeader, "conn-1")
	wsConfig.Header.Set(wsproto.SecretKeyHeader, "secret")
	conn, err := websocket.DialConfig(wsConfig)
	if err != nil {
		t.Fatalf("dial websocket handler: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	frame, err := wsproto.Receive(conn)
	if err != nil {
		t.Fatalf("receive handshake frame: %v", err)
	}
	if frame.Type != wsproto.TypeError {
		t.Fatalf("expected an error frame, got %q", frame.Type)
	}
	if !strings.Contains(frame.Message, "protocol mismatch") || !strings.Contains(frame.Message, "upgrade the portr client") {
		t.Fatalf("unexpected handshake error %q", frame.Message)
	}
}

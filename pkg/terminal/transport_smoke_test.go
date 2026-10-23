//go:build smoke

package terminal

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"k8s.io/client-go/tools/remotecommand"
)

type readTracker struct {
	enabled atomic.Bool
	started chan struct{}
}
type trackedListener struct {
	net.Listener
	tracker *readTracker
}

func (l trackedListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return trackedConn{Conn: conn, tracker: l.tracker}, nil
}

type trackedConn struct {
	net.Conn
	tracker *readTracker
}

func (c trackedConn) Read(p []byte) (int, error) {
	if c.tracker.enabled.Load() {
		select {
		case c.tracker.started <- struct{}{}:
		default:
		}
	}
	return c.Conn.Read(p)
}

func transportPair(t *testing.T, logger *zap.Logger) (*Session, *websocket.Conn, *readTracker) {
	t.Helper()
	tracker := &readTracker{started: make(chan struct{}, 8)}
	ready := make(chan *Session, 1)
	ended := make(chan struct{})
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		session := &Session{conn: conn, sizeChan: make(chan remotecommand.TerminalSize, 1), logger: logger}
		ready <- session
		<-ended
	}))
	server.Listener = trackedListener{Listener: server.Listener, tracker: tracker}
	server.Start()
	peer, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	session := <-ready
	t.Cleanup(func() { _ = peer.Close(); _ = session.conn.Close(); close(ended); server.Close() })
	return session, peer, tracker
}

func TestCloseInterruptsAlreadyBlockedRead(t *testing.T) {
	session, peer, tracker := transportPair(t, zap.NewNop())
	tracker.enabled.Store(true)
	readDone := make(chan struct{})
	go func() { _, _ = session.Read(make([]byte, 8)); close(readDone) }()
	select {
	case <-tracker.started:
	case <-time.After(time.Second):
		t.Fatal("reader did not enter underlying network read")
	}
	closed := make(chan error, 1)
	go func() { closed <- session.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(300 * time.Millisecond):
		_ = peer.Close()
		<-closed
		t.Fatal("Close deadlocked behind blocked websocket Read")
	}
	select {
	case <-readDone:
	case <-time.After(time.Second):
		t.Fatal("closed reader did not return")
	}
}

func TestLongRawAndJSONInputIsDeliveredAcrossReadsWithoutLogging(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	session, peer, _ := transportPair(t, zap.New(core))
	defer session.Close()
	raw := "raw-private-marker:" + strings.Repeat("中文😀'", 4096)
	structured := "json-private-marker:" + strings.Repeat("line\nquoted\"", 4096)
	privateOperation := "private-operation-marker"
	if err := peer.WriteJSON(Message{Op: "resize", RowSize: 40, ColSize: 120}); err != nil {
		t.Fatal(err)
	}
	if err := peer.WriteJSON(Message{Op: privateOperation, Data: "private-operation-data"}); err != nil {
		t.Fatal(err)
	}
	if err := peer.WriteMessage(websocket.TextMessage, []byte(raw)); err != nil {
		t.Fatal(err)
	}
	if err := peer.WriteJSON(Message{Op: "stdin", Data: structured}); err != nil {
		t.Fatal(err)
	}
	_ = session.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if n, err := session.Read(nil); n != 0 || err != nil {
		t.Fatal("zero-length Read consumed input")
	}
	expected := raw + structured
	var actual strings.Builder
	for actual.Len() < len(expected) {
		buffer := make([]byte, 17)
		n, err := session.Read(buffer)
		if err != nil {
			t.Fatal("long input ended early")
		}
		actual.Write(buffer[:n])
	}
	if actual.String() != expected {
		t.Fatal("long frame input lost or reordered bytes")
	}
	size := session.Next()
	if size == nil || size.Height != 40 || size.Width != 120 {
		t.Fatal("resize preceding stdin was lost")
	}
	for _, entry := range logs.All() {
		text := entry.Message + fmt.Sprint(entry.ContextMap())
		for _, secret := range []string{"raw-private-marker", "json-private-marker", privateOperation, "private-operation-data"} {
			if strings.Contains(text, secret) {
				t.Fatal("raw stdin/unknown operation leaked into logs")
			}
		}
	}
}

func TestSessionSerializesStdoutErrorAndHeartbeatWrites(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	session, peer, _ := transportPair(t, zap.New(core))
	defer session.Close()
	handler := &terminaler{logger: zap.New(core)}
	results := make(chan error, 4)
	for _, name := range []string{"stdout", "stderr"} {
		go func(name string) {
			for i := 0; i < 64; i++ {
				if _, err := session.Write([]byte(name)); err != nil {
					results <- err
					return
				}
			}
			results <- nil
		}(name)
	}
	go func() {
		for i := 0; i < 32; i++ {
			if err := handler.writeErrorMessage(session, "private-error-marker"); err != nil {
				results <- err
				return
			}
		}
		results <- nil
	}()
	go func() {
		for i := 0; i < 32; i++ {
			if err := session.writeControl(websocket.PingMessage, nil); err != nil {
				results <- err
				return
			}
		}
		results <- nil
	}()
	_ = peer.SetReadDeadline(time.Now().Add(3 * time.Second))
	counts := map[string]int{}
	for i := 0; i < 160; i++ {
		_, payload, err := peer.ReadMessage()
		if err != nil {
			t.Fatal("concurrent output lost frames")
		}
		var message Message
		if json.Unmarshal(payload, &message) != nil || message.Op != "stdout" {
			t.Fatal("stdout wire format changed")
		}
		counts[message.Data]++
	}
	if counts["stdout"] != 64 || counts["stderr"] != 64 || counts["\r\n错误: private-error-marker\r\n"] != 32 {
		t.Fatal("output streams crossed or lost frames")
	}
	for i := 0; i < 4; i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	for _, entry := range logs.All() {
		if strings.Contains(entry.Message+fmt.Sprint(entry.ContextMap()), "private-error-marker") {
			t.Fatal("error message content leaked into logs")
		}
	}
}

func TestResizeQueueWaitsForEventsAndClosesWithBufferedValue(t *testing.T) {
	session, _, _ := transportPair(t, zap.NewNop())
	next := make(chan *remotecommand.TerminalSize, 1)
	go func() { next <- session.Next() }()
	select {
	case <-next:
		t.Fatal("empty resize queue incorrectly stopped monitoring")
	case <-time.After(30 * time.Millisecond):
	}
	session.queueSize(remotecommand.TerminalSize{Width: 140, Height: 50})
	select {
	case size := <-next:
		if size == nil || size.Width != 140 || size.Height != 50 {
			t.Fatal("resize event lost")
		}
	case <-time.After(time.Second):
		t.Fatal("resize waiter did not wake")
	}
	session.queueSize(remotecommand.TerminalSize{Width: 80, Height: 25})
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	// 缓冲有值时也必须真正close，而不是消费一个值后留下永远开放的通道
	for i := 0; i < 2; i++ {
		select {
		case _, ok := <-session.sizeChan:
			if !ok {
				goto closed
			}
		default:
			t.Fatal("resize channel remained open")
		}
	}
	t.Fatal("resize channel did not close")
closed:
	if session.Next() != nil {
		t.Fatal("closed resize queue did not stop")
	}
	if err := session.Close(); err != nil {
		t.Fatal("repeated Close was not stable")
	}
}

func TestCloseUnblocksResizeWaiter(t *testing.T) {
	session, _, _ := transportPair(t, zap.NewNop())
	done := make(chan *remotecommand.TerminalSize, 1)
	go func() { done <- session.Next() }()
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case size := <-done:
		if size != nil {
			t.Fatal("closed monitor returned a resize")
		}
	case <-time.After(time.Second):
		t.Fatal("closed Next waiter remained blocked")
	}
}

func TestCloseReasonDoesNotLeakAsInputLog(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	session, peer, _ := transportPair(t, zap.New(core))
	defer session.Close()
	if err := peer.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "private-close-reason"), time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	_ = session.conn.SetReadDeadline(time.Now().Add(time.Second))
	_, err := session.Read(make([]byte, 8))
	if err == nil {
		t.Fatal("close frame did not terminate reader")
	}
	if strings.Contains(err.Error(), "private-close-reason") {
		t.Fatal("untrusted close reason escaped read boundary")
	}
	for _, entry := range logs.All() {
		if strings.Contains(entry.Message+fmt.Sprint(entry.ContextMap()), "private-close-reason") {
			t.Fatal("client close reason leaked into logs")
		}
	}
}

func TestRequestCancellationReclaimsBlockedReader(t *testing.T) {
	session, _, tracker := transportPair(t, zap.NewNop())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := session.closeOnCancel(ctx)
	defer stop()
	tracker.enabled.Store(true)
	done := make(chan struct{})
	go func() { _, _ = session.Read(make([]byte, 8)); close(done) }()
	select {
	case <-tracker.started:
	case <-time.After(time.Second):
		t.Fatal("stdin did not reach network read")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("request cancellation did not reclaim blocked stdin")
	}
	stop()
	if session.Next() != nil {
		t.Fatal("request cancellation left resize monitoring open")
	}
}

func TestPongCallbackAndHeartbeatShareSafeTransport(t *testing.T) {
	session, peer, _ := transportPair(t, zap.NewNop())
	defer session.Close()
	handler := &terminaler{logger: zap.NewNop()}
	handler.setupPongHandler(session)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := session.closeOnCancel(ctx)
	defer stop()
	ping := make(chan struct{}, 1)
	peer.SetPingHandler(func(string) error {
		select {
		case ping <- struct{}{}:
		default:
		}
		return peer.WriteControl(websocket.PongMessage, nil, time.Now().Add(time.Second))
	})
	peerDone := make(chan struct{})
	go func() { _, _, _ = peer.ReadMessage(); close(peerDone) }()
	heartbeatDone := make(chan struct{})
	go func() { handler.startHeartbeat(ctx, session, cancel); close(heartbeatDone) }()
	stdin := make(chan string, 1)
	go func() { buffer := make([]byte, 32); n, _ := session.Read(buffer); stdin <- string(buffer[:n]) }()
	select {
	case <-ping:
	case <-time.After(time.Second):
		t.Fatal("heartbeat did not send initial ping")
	}
	if err := peer.WriteJSON(Message{Op: "stdin", Data: "after-pong"}); err != nil {
		t.Fatal(err)
	}
	select {
	case input := <-stdin:
		if input != "after-pong" {
			t.Fatal("Pong callback interfered with stdin")
		}
	case <-time.After(time.Second):
		t.Fatal("Pong callback blocked reader")
	}
	cancel()
	select {
	case <-heartbeatDone:
	case <-time.After(time.Second):
		t.Fatal("heartbeat did not stop with request")
	}
	select {
	case <-peerDone:
	case <-time.After(time.Second):
		t.Fatal("peer was not closed with request")
	}
}

func TestPeerDisconnectCancelsOwnedExecContext(t *testing.T) {
	session, peer, tracker := transportPair(t, zap.NewNop())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	session.cancel = cancel
	stop := session.closeOnCancel(ctx)
	defer stop()
	tracker.enabled.Store(true)
	done := make(chan struct{})
	go func() { _, _ = session.Read(make([]byte, 8)); close(done) }()
	select {
	case <-tracker.started:
	case <-time.After(time.Second):
		t.Fatal("stdin did not enter network read")
	}
	_ = peer.Close()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("peer disconnect did not cancel owned exec context")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("disconnected stdin did not finish")
	}
	stop()
	if session.Next() != nil {
		t.Fatal("disconnected exec retained resize monitor")
	}
}

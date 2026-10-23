//go:build smoke

package terminal

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/remotecommand"
)

func inputLimitsPair(t *testing.T, compression bool) (*Session, *websocket.Conn, context.Context) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan *Session, 1)
	release := make(chan struct{})
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{EnableCompression: compression}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		ready <- &Session{conn: conn, sizeChan: make(chan remotecommand.TerminalSize, 1), logger: zap.NewNop(), cancel: cancel}
		<-release
	}))
	peer, _, err := (&websocket.Dialer{EnableCompression: compression}).Dial("ws"+strings.TrimPrefix(web.URL, "http"), nil)
	if err != nil {
		web.Close()
		t.Fatal(err)
	}
	session := <-ready
	t.Cleanup(func() { cancel(); _ = peer.Close(); _ = session.Close(); close(release); web.Close() })
	return session, peer, ctx
}

func TestOversizedPodInputClosesWithoutPartialDelivery(t *testing.T) {
	oversizedJSON, _ := json.Marshal(Message{Op: "stdin", Data: strings.Repeat("x", 64*1024+1)})
	resize, _ := json.Marshal(Message{Op: "resize", RowSize: 25, ColSize: 80, Data: strings.Repeat("x", 1024)})
	for _, compression := range []bool{false, true} {
		for _, data := range [][]byte{
			[]byte(strings.Repeat("x", 64*1024+1)), oversizedJSON, resize, []byte(strings.Repeat("x", 2*1024*1024)),
		} {
			session, peer, ctx := inputLimitsPair(t, compression)
			done := make(chan bool, 1)
			go func() { var p [17]byte; n, err := session.Read(p[:]); done <- n == 0 && err != nil }()
			_ = peer.WriteMessage(websocket.TextMessage, data)
			// 旧实现对resize会继续读下一帧；新实现必须在其之前关闭
			_ = peer.WriteMessage(websocket.TextMessage, []byte(`{"op":"stdin","data":"next"}`))
			select {
			case rejected := <-done:
				if !rejected {
					t.Error("oversized message was accepted or partially delivered")
				}
			case <-time.After(2 * time.Second):
				t.Error("oversized input left reader blocked")
			}
			if ctx.Err() == nil {
				t.Error("oversized input did not cancel exec")
			}
			if len(session.sizeChan) != 0 {
				t.Error("oversized resize entered size queue")
			}
			_ = session.Close()
			if session.Next() != nil {
				t.Error("oversized resize crossed size boundary")
			}
		}
	}
}

func TestValidPodInputAtLimitPreservesEscapedJSONAndRaw(t *testing.T) {
	for _, compression := range []bool{false, true} {
		for _, structured := range []bool{false, true} {
			expected := bytes.Repeat([]byte{0}, 64*1024)
			data := expected
			if structured {
				var err error
				data, err = json.Marshal(Message{Op: "stdin", Data: string(expected)})
				if err != nil {
					t.Fatal(err)
				}
			}
			session, peer, _ := inputLimitsPair(t, compression)
			done := make(chan bool, 1)
			go func() {
				var actual []byte
				var p [113]byte
				for len(actual) < len(expected) {
					n, err := session.Read(p[:])
					if err != nil {
						done <- false
						return
					}
					actual = append(actual, p[:n]...)
				}
				done <- bytes.Equal(actual, expected)
			}()
			if err := peer.WriteMessage(websocket.TextMessage, data); err != nil {
				t.Fatal(err)
			}
			select {
			case correct := <-done:
				if !correct {
					t.Error("valid boundary input changed")
				}
			case <-time.After(2 * time.Second):
				t.Error("valid input stalled")
			}
		}
	}
}

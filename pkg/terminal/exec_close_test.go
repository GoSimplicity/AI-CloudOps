package terminal

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// internalExecDetail 模拟 exec 失败时 API Server 返回的内部错误细节，不得出现在关闭帧 reason 中
const internalExecDetail = "INTERNAL-DETAIL runtime exec failed: oci runtime error at /run/containerd/secret-path"

// fakeKubeletExec 模拟 API Server 的 pods/exec SPDY 子协议（v4.channel.k8s.io）：
// 非 TTY 的探测命令直接成功退出；TTY 交互 shell 回显输入，收到 exit 后正常结束。
// failTTY 为 true 时 TTY 交互 shell 立即以内部错误失败
func fakeKubeletExec(t *testing.T, failTTY bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/exec") {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		tty, stdin, stdout, stderr := q.Get("tty") == "true", q.Get("stdin") == "true", q.Get("stdout") == "true", q.Get("stderr") == "true"
		if _, err := httpstream.Handshake(r, w, []string{"v4.channel.k8s.io"}); err != nil {
			return
		}
		streamCh := make(chan httpstream.Stream, 8)
		conn := spdy.NewResponseUpgrader().UpgradeResponse(w, r, func(s httpstream.Stream, _ <-chan struct{}) error {
			streamCh <- s
			return nil
		})
		if conn == nil {
			return
		}
		defer conn.Close()

		expected := 1 // error
		for _, on := range []bool{stdin, stdout, stderr && !tty, tty} {
			if on {
				expected++
			}
		}
		streams := map[string]httpstream.Stream{}
		for len(streams) < expected {
			select {
			case s := <-streamCh:
				streams[s.Headers().Get(v1.StreamType)] = s
			case <-time.After(5 * time.Second):
				return
			}
		}
		if resize := streams[v1.StreamTypeResize]; resize != nil {
			go func() { _, _ = io.Copy(io.Discard, resize) }()
		}
		if tty && failTTY {
			_ = streams[v1.StreamTypeStdout].Close()
			status := `{"metadata":{},"status":"Failure","message":"` + internalExecDetail + `","reason":"InternalError","code":500}`
			_, _ = streams[v1.StreamTypeError].Write([]byte(status))
			_ = streams[v1.StreamTypeError].Close()
			select {
			case <-conn.CloseChan():
			case <-time.After(5 * time.Second):
			}
			return
		}
		if tty {
			in, out := streams[v1.StreamTypeStdin], streams[v1.StreamTypeStdout]
			var typed strings.Builder
			buf := make([]byte, 256)
			for !strings.Contains(typed.String(), "exit") {
				n, err := in.Read(buf)
				if n > 0 {
					typed.Write(buf[:n])
					_, _ = out.Write(buf[:n])
				}
				if err != nil {
					return
				}
			}
			_ = out.Close()
		} else if out := streams[v1.StreamTypeStdout]; out != nil {
			_ = out.Close()
		}
		errStream := streams[v1.StreamTypeError]
		_, _ = errStream.Write([]byte(`{"metadata":{},"status":"Success"}`))
		_ = errStream.Close()
		select {
		case <-conn.CloseChan():
		case <-time.After(5 * time.Second):
		}
	}))
}

func leakedTerminalGoroutines() []string {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]
	var leaked []string
	for _, g := range strings.Split(string(buf), "\n\n") {
		if strings.Contains(g, "leakedTerminalGoroutines") {
			continue // 正在执行检查的测试goroutine自身
		}
		for _, marker := range []string{"AI-CloudOps/pkg/terminal.", "client-go/tools/remotecommand", "gorilla/websocket", "httpstream/spdy"} {
			if strings.Contains(g, marker) {
				leaked = append(leaked, g)
				break
			}
		}
	}
	return leaked
}

// runTerminalSession 通过真实的 HandleSession 跑一次终端会话：发送 resize 与 exit 输入，
// 读到关闭为止，并在全部服务关闭后检查 goroutine 回到基线
func runTerminalSession(t *testing.T, failTTY bool) (string, *websocket.CloseError) {
	t.Helper()
	baseline := runtime.NumGoroutine()

	kubelet := fakeKubeletExec(t, failTTY)
	cfg := &rest.Config{Host: kubelet.URL}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	handlerDone := make(chan struct{})
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(handlerDone)
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close() // 与 internal/k8s/api/pod.go 的 PodExec 处理一致
		NewTerminalHandler(client, cfg, zap.NewNop()).HandleSession(r.Context(), "sh", "default", "pod", "app", conn)
	}))

	peer, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(wsServer.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := peer.WriteJSON(Message{Op: "resize", RowSize: 40, ColSize: 120}); err != nil {
		t.Fatal(err)
	}
	if err := peer.WriteJSON(Message{Op: "stdin", Data: "exit\r"}); err != nil {
		t.Fatal(err)
	}

	_ = peer.SetReadDeadline(time.Now().Add(30 * time.Second))
	var output strings.Builder
	var closeErr error
	for {
		var msg Message
		if err := peer.ReadJSON(&msg); err != nil {
			closeErr = err
			break
		}
		output.WriteString(msg.Data)
	}

	select {
	case <-handlerDone:
	case <-time.After(10 * time.Second):
		t.Fatal("服务端终端处理未退出")
	}
	_ = peer.Close()
	wsServer.Close()
	kubelet.Close()

	deadline := time.Now().Add(5 * time.Second)
	for {
		leaked := leakedTerminalGoroutines()
		if len(leaked) == 0 && runtime.NumGoroutine() <= baseline+2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("存在 goroutine 泄漏 (baseline=%d now=%d):\n%s", baseline, runtime.NumGoroutine(), strings.Join(leaked, "\n\n"))
		}
		time.Sleep(50 * time.Millisecond)
	}

	var ce *websocket.CloseError
	if !asCloseError(closeErr, &ce) {
		t.Fatalf("应收到 WebSocket 关闭帧, got %v", closeErr)
	}
	return output.String(), ce
}

func TestExitSendsNormalClosureWithoutGoroutineLeak(t *testing.T) {
	output, ce := runTerminalSession(t, false)
	if !strings.Contains(output, "exit") {
		t.Fatalf("未收到 shell 回显, output=%q", output)
	}
	if ce.Code != websocket.CloseNormalClosure {
		t.Fatalf("exit 后应收到关闭码 1000, got %v", ce)
	}
}

func TestExecErrorSendsInternalErrorClosureWithoutGoroutineLeak(t *testing.T) {
	_, ce := runTerminalSession(t, true)
	if ce.Code != websocket.CloseInternalServerErr {
		t.Fatalf("exec 出错后应收到关闭码 1011, got %v", ce)
	}
	if len(ce.Text) == 0 || len(ce.Text) > 123 {
		t.Fatalf("关闭帧 reason 长度应在 1..123 字节, got %d (%q)", len(ce.Text), ce.Text)
	}
	if ce.Text != terminalErrorCloseReason {
		t.Fatalf("关闭帧 reason 应为固定的用户可读说明, got %q", ce.Text)
	}
	for _, detail := range []string{"INTERNAL-DETAIL", "oci runtime", "/run/containerd", "所有Shell", "最后一个错误"} {
		if strings.Contains(ce.Text, detail) {
			t.Fatalf("关闭帧 reason 不应包含内部错误细节 %q: %q", detail, ce.Text)
		}
	}
}

func TestTruncateCloseReasonRespectsByteLimitAndUTF8(t *testing.T) {
	long := strings.Repeat("终", 60) // 180 字节
	got := truncateCloseReason(long)
	if len(got) > maxCloseReasonBytes || !utf8.ValidString(got) {
		t.Fatalf("截断结果非法: len=%d valid=%v", len(got), utf8.ValidString(got))
	}
	if truncateCloseReason(terminalErrorCloseReason) != terminalErrorCloseReason {
		t.Fatal("短 reason 不应被截断")
	}
	if len(websocket.FormatCloseMessage(websocket.CloseInternalServerErr, got)) > 125 {
		t.Fatal("关闭帧载荷超过控制帧上限")
	}
}

func asCloseError(err error, target **websocket.CloseError) bool {
	ce, ok := err.(*websocket.CloseError)
	if ok {
		*target = ce
	}
	return ok
}

// CloseNormally/CloseWithError 在没有其他读者时自行等待对端回应，且有超时上限
func TestCloseIsBoundedWhenPeerNeverReplies(t *testing.T) {
	cases := []struct {
		name   string
		code   int
		reason string
		close  func(*Session) error
	}{
		{"normal", websocket.CloseNormalClosure, "", (*Session).CloseNormally},
		{"error", websocket.CloseInternalServerErr, terminalErrorCloseReason, (*Session).CloseWithError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ready := make(chan *websocket.Conn, 1)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
				if err == nil {
					ready <- conn
				}
			}))
			defer srv.Close()
			peer, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer peer.Close()
			conn := <-ready
			_, cancel := context.WithCancel(context.Background())
			s := &Session{conn: conn, logger: zap.NewNop(), cancel: cancel}
			start := time.Now()
			if err := tc.close(s); err != nil {
				t.Fatal(err)
			}
			if elapsed := time.Since(start); elapsed > closeGracePeriod+500*time.Millisecond {
				t.Fatalf("关闭等待过久: %v", elapsed)
			}
			// 对端不读也不回应，但关闭帧已经写出，可以被读到
			_ = peer.SetReadDeadline(time.Now().Add(time.Second))
			_, _, err = peer.ReadMessage()
			var ce *websocket.CloseError
			if !asCloseError(err, &ce) || ce.Code != tc.code || ce.Text != tc.reason {
				t.Fatalf("应读到关闭码 %d reason %q, got %v", tc.code, tc.reason, err)
			}
		})
	}
}

// 对端持续发送 Pong（并夹杂输入），但从不回应关闭帧：Pong 不得把读期限延长回 pongWait，
// 无论是关闭方自己读取，还是 stdin 复制 goroutine 正在读取，关闭都必须在上限内结束
func TestCloseIsBoundedWhilePeerFloodsPongs(t *testing.T) {
	// readerMode: 0=无并发reader（关闭方自己读）；1=stdin reader 持续读；
	// 2=stdin reader 在关闭开始时停止读取（exec 结束后 stdin 复制随之停止的情形）
	for readerMode, name := range []string{"close-reads-itself", "stdin-reader-active", "stdin-reader-stops-on-close"} {
		readerMode := readerMode
		t.Run(name, func(t *testing.T) {
			ready := make(chan *websocket.Conn, 1)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
				if err == nil {
					ready <- conn
				}
			}))
			defer srv.Close()
			peer, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer peer.Close()
			conn := <-ready
			_, cancel := context.WithCancel(context.Background())
			s := &Session{conn: conn, logger: zap.NewNop(), cancel: cancel}
			(&terminaler{logger: zap.NewNop()}).setupPongHandler(s)

			// 对端：只写不读（因此永远不会回应关闭帧），不停发送 Pong/Ping，并周期性发送输入让 reader 释放读锁
			stopPeer := make(chan struct{})
			peerDone := make(chan struct{})
			go func() {
				defer close(peerDone)
				stdin, _ := json.Marshal(Message{Op: "stdin", Data: "x"})
				for i := 0; ; i++ {
					select {
					case <-stopPeer:
						return
					default:
					}
					if err := peer.WriteControl(websocket.PongMessage, nil, time.Now().Add(time.Second)); err != nil {
						return
					}
					if err := peer.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second)); err != nil {
						return
					}
					if i%16 == 0 {
						if err := peer.WriteMessage(websocket.TextMessage, stdin); err != nil {
							return
						}
					}
					time.Sleep(100 * time.Microsecond)
				}
			}()
			defer func() { close(stopPeer); <-peerDone }()

			readerDone := make(chan struct{})
			stopReading := make(chan struct{})
			if readerMode > 0 {
				go func() {
					defer close(readerDone)
					buf := make([]byte, 64)
					for {
						if _, err := s.Read(buf); err != nil {
							return
						}
						if readerMode == 2 {
							select {
							case <-stopReading:
								return
							default:
							}
						}
					}
				}()
			} else {
				close(readerDone)
			}

			time.Sleep(200 * time.Millisecond) // 让 Pong 洪流先跑起来
			close(stopReading)
			start := time.Now()
			if err := s.CloseNormally(); err != nil {
				t.Fatal(err)
			}
			if elapsed := time.Since(start); elapsed > closeGracePeriod+500*time.Millisecond {
				t.Fatalf("对端持续发送 Pong 时关闭等待过久: %v", elapsed)
			}
			select {
			case <-readerDone:
			case <-time.After(2 * time.Second):
				t.Fatal("stdin reader 未在关闭后退出")
			}
		})
	}
}

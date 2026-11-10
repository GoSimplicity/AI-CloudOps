package ssh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	pkgssh "github.com/GoSimplicity/AI-CloudOps/pkg/ssh"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const fixturePassword = "isolated-ssh-fixture-password"

type localSSH struct {
	Host      string
	Port      int
	KeyPEM    string
	Known     string
	commands  chan string
	stall     bool
	terminal  bool
	rejectPTY bool
	failExec  bool
	signer    ssh.Signer
	listener  net.Listener
	mu        sync.Mutex
	conns     map[net.Conn]bool
	wait      sync.WaitGroup
}

func startLocalSSH(t *testing.T, stall, terminal, rejectPTY, failExec bool) *localSSH {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatal(err)
	}
	private, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	host, portText, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(portText)
	known := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(known, []byte(knownhosts.Line([]string{listener.Addr().String()}, signer.PublicKey())+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := &localSSH{
		Host: host, Port: port,
		KeyPEM: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private})),
		Known:  known, commands: make(chan string, 8),
		stall: stall, terminal: terminal, rejectPTY: rejectPTY, failExec: failExec,
		signer: signer, listener: listener, conns: map[net.Conn]bool{},
	}
	t.Cleanup(func() {
		_ = listener.Close()
		server.mu.Lock()
		for conn := range server.conns {
			_ = conn.Close()
		}
		server.mu.Unlock()
		server.wait.Wait()
	})
	server.wait.Add(1)
	go func() {
		defer server.wait.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			server.mu.Lock()
			server.conns[conn] = true
			server.mu.Unlock()
			server.wait.Add(1)
			go func() {
				defer server.wait.Done()
				defer conn.Close()
				defer func() {
					server.mu.Lock()
					delete(server.conns, conn)
					server.mu.Unlock()
				}()
				server.serve(conn)
			}()
		}
	}()
	return server
}

func (s *localSSH) serve(conn net.Conn) {
	if s.stall {
		_, _ = io.Copy(io.Discard, conn)
		return
	}
	config := &ssh.ServerConfig{
		PasswordCallback: func(metadata ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if metadata.User() == "fixture" && string(password) == fixturePassword {
				return nil, nil
			}
			return nil, fmt.Errorf("fixture authentication rejected")
		},
		PublicKeyCallback: func(metadata ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if metadata.User() == "fixture" && string(key.Marshal()) == string(s.signer.PublicKey().Marshal()) {
				return nil, nil
			}
			return nil, fmt.Errorf("fixture key rejected")
		},
	}
	config.AddHostKey(s.signer)
	connection, channels, requests, err := ssh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	defer connection.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = connection.Wait(); cancel() }()
	go ssh.DiscardRequests(requests)
	var sessions sync.WaitGroup
	defer sessions.Wait()
	for channel := range channels {
		if channel.ChannelType() != "session" {
			_ = channel.Reject(ssh.UnknownChannelType, "session only")
			continue
		}
		accepted, channelRequests, err := channel.Accept()
		if err != nil {
			continue
		}
		sessions.Add(1)
		go func() {
			defer sessions.Done()
			defer accepted.Close()
			for request := range channelRequests {
				if s.terminal && request.Type == "pty-req" {
					if s.rejectPTY {
						_ = request.Reply(false, nil)
						continue
					}
					_ = request.Reply(true, nil)
					continue
				}
				if s.terminal && request.Type == "shell" {
					_ = request.Reply(true, nil)
					select {
					case s.commands <- "shell":
					default:
					}
					_, _ = io.Copy(accepted, accepted)
					_, _ = accepted.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
					return
				}
				if request.Type != "exec" {
					_ = request.Reply(false, nil)
					continue
				}
				var command struct{ Command string }
				if ssh.Unmarshal(request.Payload, &command) != nil {
					_ = request.Reply(false, nil)
					return
				}
				_ = request.Reply(true, nil)
				select {
				case s.commands <- command.Command:
				default:
				}
				if s.failExec || command.Command == "fail" {
					_, _ = accepted.Stderr().Write([]byte("partial"))
					_, _ = accepted.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{1}))
					return
				}
				switch command.Command {
				case "block":
					<-ctx.Done()
					return
				case "large":
					_, _ = accepted.Write([]byte(strings.Repeat("x", 100000)))
					_, _ = accepted.Stderr().Write([]byte(strings.Repeat("y", 100000)))
				default:
					_, _ = accepted.Write([]byte("executed"))
				}
				_, _ = accepted.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
				return
			}
		}()
	}
}

func trust(t *testing.T, known string) {
	t.Helper()
	t.Setenv("ENV", "development")
	t.Setenv("SSH_KNOWN_HOSTS", known)
}

func TestNewSSHNilLoggerAndConnectValidation(t *testing.T) {
	client := NewSSH(nil).(*ecsSSH)
	if client.logger == nil || client.Sessions == nil {
		t.Fatal("nil logger or session map was not initialized")
	}
	cases := []struct {
		name string
		ip   string
		user string
		pass string
		key  string
		mode int8
		want string
	}{
		{name: "empty ip", user: "fixture", pass: "x", mode: 1, want: "IP地址不能为空"},
		{name: "empty user", ip: "127.0.0.1", pass: "x", mode: 1, want: "用户名不能为空"},
		{name: "bad mode", ip: "127.0.0.1", user: "fixture", pass: "x", mode: 3, want: "认证方式必须是 1 或 2"},
		{name: "empty password", ip: "127.0.0.1", user: "fixture", mode: 1, want: "密码不能为空"},
		{name: "empty key", ip: "127.0.0.1", user: "fixture", mode: 2, want: "私钥不能为空"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := client.ConnectContext(context.Background(), tc.ip, 22, tc.user, tc.pass, tc.key, tc.mode, 7)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validation error = %v", err)
			}
		})
	}
}

func TestConnectRejectsMissingKnownHostsAndRewritesPort(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("SSH_KNOWN_HOSTS", "")
	client := NewSSH(zap.NewNop()).(*ecsSSH)
	err := client.ConnectContext(context.Background(), "127.0.0.1", 22, "fixture", fixturePassword, "", 1, 7)
	if err == nil || !strings.Contains(err.Error(), "SSH_KNOWN_HOSTS") {
		t.Fatalf("production host key policy = %v", err)
	}
	if client.Port != 22 || client.IP != "127.0.0.1" || client.Password != fixturePassword || client.UserID != 7 {
		t.Fatalf("credential/port state = %+v", client)
	}
	t.Setenv("ENV", "")
	err = client.Connect("127.0.0.1", 22, "fixture", fixturePassword, "", 1, 7)
	if err == nil || !strings.Contains(err.Error(), "SSH_KNOWN_HOSTS") || client.Port != 22 {
		t.Fatalf("unset ENV did not keep host checking: %v port=%d", err, client.Port)
	}
	fresh := NewSSH(zap.NewNop()).(*ecsSSH)
	for _, port := range []int{0, -1, 65536, 70000} {
		err = fresh.ConnectContext(context.Background(), "127.0.0.1", port, "fixture", fixturePassword, "", 1, 7)
		if err == nil || !strings.Contains(err.Error(), "1 到 65535") || fresh.Port == 22 {
			t.Fatalf("port %d = %v stored=%d", port, err, fresh.Port)
		}
	}
	err = fresh.ConnectContext(context.Background(), "127.0.0.1", 1, "fixture", fixturePassword, "", 1, 7)
	if err == nil || strings.Contains(err.Error(), "1 到 65535") {
		t.Fatalf("port 1 = %v", err)
	}
}

func TestConnectRejectsUnreadableKnownHostsAndBadKey(t *testing.T) {
	t.Setenv("ENV", "development")
	t.Setenv("SSH_KNOWN_HOSTS", filepath.Join(t.TempDir(), "missing-known-hosts"))
	client := NewSSH(zap.NewNop())
	if err := client.ConnectContext(context.Background(), "127.0.0.1", 22, "fixture", fixturePassword, "", 1, 9); err == nil {
		t.Fatal("missing known hosts file was accepted")
	}
	t.Setenv("SSH_KNOWN_HOSTS", "")
	err := client.ConnectContext(context.Background(), "127.0.0.1", 22, "fixture", "", "not-a-private-key", 2, 9)
	if err == nil || !strings.Contains(err.Error(), "解析私钥失败") {
		t.Fatalf("bad key = %v", err)
	}
}

func TestConnectPasswordKeyAndHandshakeTimeout(t *testing.T) {
	server := startLocalSSH(t, false, false, false, false)
	trust(t, server.Known)
	core, recorded := observer.New(zap.DebugLevel)
	client := NewSSH(zap.New(core)).(*ecsSSH)
	defer client.Close()

	err := client.ConnectContext(context.Background(), server.Host, server.Port, "fixture", "wrong-password", "", 1, 3)
	if err == nil || !strings.Contains(err.Error(), "SSH连接失败") {
		t.Fatalf("wrong password = %v", err)
	}
	for _, entry := range recorded.All() {
		blob := entry.Message
		for _, field := range entry.Context {
			blob += " " + field.Key + "=" + fmt.Sprint(field.Interface)
		}
		if strings.Contains(blob, "wrong-password") || strings.Contains(blob, fixturePassword) {
			t.Fatalf("credential leaked into logs: %s", blob)
		}
	}

	if err := client.ConnectContext(context.Background(), server.Host, server.Port, "fixture", fixturePassword, "", 1, 3); err != nil {
		t.Fatal(err)
	}
	if client.Client == nil || client.Sessions[3] == nil || client.Mode != 1 {
		t.Fatal("password session was not stored")
	}
	_ = client.Close()

	keyClient := NewSSH(zap.NewNop()).(*ecsSSH)
	defer keyClient.Close()
	if err := keyClient.ConnectContext(context.Background(), server.Host, server.Port, "fixture", "", server.KeyPEM, 2, 4); err != nil {
		t.Fatal(err)
	}
	if keyClient.Mode != 2 || keyClient.Sessions[4] == nil {
		t.Fatal("key session was not stored")
	}
	_, other, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	otherPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: mustPKCS8(t, other)})
	if err := keyClient.ConnectContext(context.Background(), server.Host, server.Port, "fixture", "", string(otherPEM), 2, 5); err == nil {
		t.Fatal("untrusted client key was accepted")
	}

	stall := startLocalSSH(t, true, false, false, false)
	trust(t, stall.Known)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	started := time.Now()
	err = NewSSH(zap.NewNop()).ConnectContext(ctx, stall.Host, stall.Port, "fixture", fixturePassword, "", 1, 1)
	if err == nil || time.Since(started) > 2*time.Second || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("handshake timeout = %v after %s", err, time.Since(started))
	}
	canceled, cancelNow := context.WithCancel(context.Background())
	cancelNow()
	err = NewSSH(zap.NewNop()).ConnectContext(canceled, server.Host, server.Port, "fixture", fixturePassword, "", 1, 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled dial = %v", err)
	}
}

func mustPKCS8(t *testing.T, key ed25519.PrivateKey) []byte {
	t.Helper()
	encoded, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestRunCommandIsLiteralAndBounded(t *testing.T) {
	server := startLocalSSH(t, false, false, false, false)
	trust(t, server.Known)
	client := NewSSH(zap.NewNop()).(*ecsSSH)
	defer client.Close()
	if _, err := client.RunContext(context.Background(), "echo"); err == nil || !strings.Contains(err.Error(), "IP地址不能为空") {
		t.Fatalf("disconnected run = %v", err)
	}
	if err := client.ConnectContext(context.Background(), server.Host, server.Port, "fixture", fixturePassword, "", 1, 8); err != nil {
		t.Fatal(err)
	}
	if _, err := client.RunContext(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "命令不能为空") {
		t.Fatalf("empty command = %v", err)
	}
	if _, err := client.RunContext(nil, "echo"); err == nil || !strings.Contains(err.Error(), "上下文不能为空") {
		t.Fatalf("nil context = %v", err)
	}
	command := "printf %s '$(id)'; rm -rf /tmp/owned"
	output, err := client.Run(command)
	if err != nil || output != "executed" {
		t.Fatalf("run = %q %v", output, err)
	}
	select {
	case got := <-server.commands:
		if got != command {
			t.Fatalf("command was rewritten: %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("command was not delivered")
	}
	if client.LastResult != "executed" {
		t.Fatalf("last result = %q", client.LastResult)
	}
	output, err = client.RunContext(context.Background(), "fail")
	if err == nil || !strings.Contains(err.Error(), "执行SSH命令失败") || !strings.Contains(output, "partial") {
		t.Fatalf("failed command = %q %v", output, err)
	}
	output, err = client.RunContext(context.Background(), "large")
	if err != nil || !strings.HasSuffix(output, "[输出已截断]") || len(output) > pkgssh.MaxCommandOutputBytes+len("[输出已截断]") {
		t.Fatalf("unbounded output len=%d err=%v", len(output), err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := client.RunContext(ctx, "block")
		done <- err
	}()
	select {
	case <-server.commands:
	case <-time.After(2 * time.Second):
		t.Fatal("blocking command did not start")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("command ignored cancellation")
	}
}

func TestRunFiveMinuteTimeoutNotInjectable(t *testing.T) {
	plain := NewSSH(zap.NewNop()).(*ecsSSH)
	if plain.effectiveRunTimeout() != 5*time.Minute {
		t.Fatalf("default timeout = %s", plain.effectiveRunTimeout())
	}
	server := startLocalSSH(t, false, false, false, false)
	trust(t, server.Known)
	client := NewSSH(zap.NewNop(), WithRunTimeout(200*time.Millisecond)).(*ecsSSH)
	defer client.Close()
	if client.effectiveRunTimeout() != 200*time.Millisecond {
		t.Fatalf("injected timeout = %s", client.effectiveRunTimeout())
	}
	if err := client.ConnectContext(context.Background(), server.Host, server.Port, "fixture", fixturePassword, "", 1, 8); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	_, err := client.Run("block")
	if err == nil || time.Since(started) > 2*time.Second || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("run timeout = %v after %s", err, time.Since(started))
	}
}

func TestAddPublicKeyQuotesPayloadAndSurfacesRemoteError(t *testing.T) {
	client := NewSSH(zap.NewNop())
	for _, key := range []string{"", "ssh-ed25519 not-a-key", "line1\nline2", strings.Repeat("a", 16*1024+1)} {
		if err := client.AddPublicKeyToRemoteHost(key); err == nil {
			t.Fatalf("invalid public key accepted: %q", key[:min(20, len(key))])
		}
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	authorized := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub))) + " owner';touch /tmp/owned;#"
	want, err := pkgssh.AuthorizedKeyCommand(authorized)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(want, "'\"'\"'") || strings.Contains(want, "owner';touch") {
		t.Fatalf("quote helper did not isolate the comment: %s", want)
	}

	server := startLocalSSH(t, false, false, false, false)
	trust(t, server.Known)
	connected := NewSSH(zap.NewNop())
	defer connected.Close()
	if err := connected.ConnectContext(context.Background(), server.Host, server.Port, "fixture", fixturePassword, "", 1, 2); err != nil {
		t.Fatal(err)
	}
	if err := connected.AddPublicKeyToRemoteHost(authorized); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-server.commands:
		if got != want {
			t.Fatalf("remote command\n got %s\nwant %s", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("authorized_keys command was not sent")
	}

	failing := startLocalSSH(t, false, false, false, true)
	trust(t, failing.Known)
	remote := NewSSH(zap.NewNop())
	defer remote.Close()
	if err := remote.ConnectContext(context.Background(), failing.Host, failing.Port, "fixture", fixturePassword, "", 1, 2); err != nil {
		t.Fatal(err)
	}
	if err := remote.AddPublicKeyToRemoteHost(authorized); err == nil || !strings.Contains(err.Error(), "添加公钥失败") {
		t.Fatalf("remote failure = %v", err)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	server := startLocalSSH(t, false, false, false, false)
	trust(t, server.Known)
	client := NewSSH(zap.NewNop()).(*ecsSSH)
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if err := client.ConnectContext(context.Background(), server.Host, server.Port, "fixture", fixturePassword, "", 1, 6); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if client.Client != nil || len(client.Sessions) != 0 {
		t.Fatal("close left the connection open")
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestWebTerminalValidationAndPTYFailure(t *testing.T) {
	client := NewSSH(zap.NewNop()).(*ecsSSH)
	if err := client.Web2SSHWithOptionsContext(context.Background(), nil, pkgssh.TerminalOptions{Mode: "nope"}); err == nil || !strings.Contains(err.Error(), "不支持的终端模式") {
		t.Fatalf("bad mode = %v", err)
	}
	if err := client.Web2SSHWithOptionsContext(nil, nil, pkgssh.TerminalOptions{}); err == nil {
		t.Fatal("nil terminal arguments accepted")
	}
	ws := websocketPair(t)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.Web2SSHContext(canceled, ws); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled parent = %v", err)
	}
	if err := client.Web2SSHContext(context.Background(), ws); err == nil || !strings.Contains(err.Error(), "SSH会话未建立") {
		t.Fatalf("missing session = %v", err)
	}
	client.Web2SSH(nil)

	server := startLocalSSH(t, false, true, true, false)
	trust(t, server.Known)
	if err := client.ConnectContext(context.Background(), server.Host, server.Port, "fixture", fixturePassword, "", 1, 1); err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	err := client.Web2SSHWithOptionsContext(context.Background(), websocketPair(t), pkgssh.TerminalOptions{Mode: pkgssh.TerminalModePTY, Rows: 0, Cols: 0})
	if err == nil || !strings.Contains(err.Error(), "请求SSH伪终端失败") {
		t.Fatalf("rejected pty = %v", err)
	}
	oversized := client.Web2SSHWithOptionsContext(context.Background(), websocketPair(t), pkgssh.TerminalOptions{Mode: pkgssh.TerminalModePTY, Rows: pkgssh.MaxTerminalRows + 1, Cols: 80})
	if oversized == nil || !strings.Contains(oversized.Error(), "终端尺寸") {
		t.Fatalf("oversized terminal = %v", oversized)
	}
}

func TestWebTerminalEchoAndPeerClose(t *testing.T) {
	server := startLocalSSH(t, false, true, false, false)
	trust(t, server.Known)
	client := NewSSH(zap.NewNop())
	defer client.Close()
	if err := client.ConnectContext(context.Background(), server.Host, server.Port, "fixture", fixturePassword, "", 1, 11); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			done <- err
			return
		}
		done <- client.Web2SSHContext(context.Background(), conn)
	}))
	defer web.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(web.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case command := <-server.commands:
		if command != "shell" {
			t.Fatalf("terminal command = %s", command)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shell did not start")
	}
	if err := conn.WriteMessage(websocket.TextMessage, []byte("fixture")); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil || string(data) != "fixture\n" {
		t.Fatalf("echo = %q %v", data, err)
	}
	_ = conn.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("terminal did not return after peer close")
	}
}

func websocketPair(t *testing.T) *websocket.Conn {
	t.Helper()
	accepted := make(chan *websocket.Conn, 1)
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		accepted <- conn
	}))
	t.Cleanup(web.Close)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(web.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	select {
	case serverConn := <-accepted:
		t.Cleanup(func() { _ = serverConn.Close() })
	case <-time.After(2 * time.Second):
		t.Fatal("websocket server did not accept")
	}
	return conn
}

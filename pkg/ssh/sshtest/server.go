//go:build smoke

// Package sshtest 提供仅绑定本机回环地址的SSH协议测试服务
package sshtest

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const Password = "isolated-ssh-fixture-password"

type TerminalSize struct {
	Rows uint32
	Cols uint32
	Echo uint32
}

type Server struct {
	PTY             chan TerminalSize
	Resize          chan TerminalSize
	Input           chan []byte
	captureTerminal bool
	Host            string
	Port            int
	Key             string
	KnownHosts      string
	Started         chan string
	StallSession    bool
	Terminal        bool
	signer          ssh.Signer
	listener        net.Listener
	mu              sync.Mutex
	connections     map[net.Conn]bool
	wait            sync.WaitGroup
}

func Start(t *testing.T, stallSession bool) *Server { return start(t, stallSession, false, false) }
func StartTerminal(t *testing.T) *Server            { return start(t, false, true, false) }
func StartPTY(t *testing.T) *Server                 { return start(t, false, true, true) }
func start(t *testing.T, stallSession, terminal, capture bool) *Server {
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
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	number, _ := strconv.Atoi(port)
	server := &Server{PTY: make(chan TerminalSize, 8), Resize: make(chan TerminalSize, 8), Input: make(chan []byte, 32), captureTerminal: capture, Host: host, Port: number, Key: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private})), signer: signer, listener: listener, connections: map[net.Conn]bool{}, Started: make(chan string, 8), StallSession: stallSession, Terminal: terminal}
	server.KnownHosts = filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(server.KnownHosts, []byte(knownhosts.Line([]string{listener.Addr().String()}, signer.PublicKey())+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
		server.mu.Lock()
		for conn := range server.connections {
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
			server.connections[conn] = true
			server.mu.Unlock()
			server.wait.Add(1)
			go func() {
				defer server.wait.Done()
				defer conn.Close()
				defer func() { server.mu.Lock(); delete(server.connections, conn); server.mu.Unlock() }()
				server.serve(conn)
			}()
		}
	}()
	return server
}

func (s *Server) serve(conn net.Conn) {
	config := &ssh.ServerConfig{
		PasswordCallback: func(metadata ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if metadata.User() == "fixture" && string(password) == Password {
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
			_ = channel.Reject(ssh.UnknownChannelType, "fixture session only")
			continue
		}
		if s.StallSession {
			select {
			case s.Started <- "session":
			default:
			}
			<-ctx.Done()
			return
		}
		accepted, channelRequests, err := channel.Accept()
		if err != nil {
			continue
		}
		sessions.Add(1)
		go func() {
			defer sessions.Done()
			var terminalIO sync.WaitGroup
			defer func() { _ = accepted.Close(); terminalIO.Wait() }()
			for request := range channelRequests {
				if s.Terminal && request.Type == "pty-req" {
					var payload struct {
						Term   string
						Cols   uint32
						Rows   uint32
						Width  uint32
						Height uint32
						Modes  string
					}
					if ssh.Unmarshal(request.Payload, &payload) != nil {
						_ = request.Reply(false, nil)
						continue
					}
					size := TerminalSize{Rows: payload.Rows, Cols: payload.Cols}
					data := []byte(payload.Modes)
					for len(data) >= 5 && data[0] != 0 {
						if data[0] == ssh.ECHO {
							size.Echo = binary.BigEndian.Uint32(data[1:5])
						}
						data = data[5:]
					}
					select {
					case s.PTY <- size:
					case <-ctx.Done():
						return
					}
					_ = request.Reply(true, nil)
					continue
				}
				if s.captureTerminal && request.Type == "window-change" {
					var size struct {
						Cols   uint32
						Rows   uint32
						Width  uint32
						Height uint32
					}
					if ssh.Unmarshal(request.Payload, &size) == nil {
						select {
						case s.Resize <- TerminalSize{Rows: size.Rows, Cols: size.Cols}:
						case <-ctx.Done():
							return
						}
					}
					continue
				}
				if s.Terminal && request.Type == "shell" {
					_ = request.Reply(true, nil)
					select {
					case s.Started <- "shell":
					default:
					}
					if s.captureTerminal {
						terminalIO.Add(3)
						go func() {
							defer terminalIO.Done()
							buffer := make([]byte, 8192)
							for {
								n, err := accepted.Read(buffer)
								if n > 0 {
									data := append([]byte(nil), buffer[:n]...)
									select {
									case s.Input <- data:
									case <-ctx.Done():
										return
									}
								}
								if err != nil {
									return
								}
							}
						}()
						// 只发送固定协议数据，不将echo当作远端执行证据
						go func() { defer terminalIO.Done(); _, _ = accepted.Write(bytes.Repeat([]byte{0, 255, 'A'}, 4096)) }()
						go func() { defer terminalIO.Done(); _, _ = accepted.Stderr().Write(bytes.Repeat([]byte{'B'}, 8192)) }()
						continue
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
				case s.Started <- command.Command:
				default:
				}
				switch command.Command {
				case "block", "(\nblock\n)":
					<-ctx.Done()
					return
				case "large":
					_, _ = accepted.Write([]byte(strings.Repeat("x", 100000)))
					_, _ = accepted.Stderr().Write([]byte(strings.Repeat("y", 100000)))
				case "fail":
					_, _ = accepted.Stderr().Write([]byte("partial"))
					_, _ = accepted.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{1}))
					return
				default:
					_, _ = accepted.Write([]byte("executed"))
				}
				_, _ = accepted.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
				return
			}
		}()
	}
}

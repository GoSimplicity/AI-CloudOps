//go:build smoke

package api

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/GoSimplicity/AI-CloudOps/pkg/ssh"
	wsprotocol "github.com/GoSimplicity/AI-CloudOps/pkg/websocket"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type terminalResourceService struct{ service.TreeLocalService }

func (terminalResourceService) GetTreeLocalForConnection(_ context.Context, req *model.GetTreeLocalResourceDetailReq) (*model.TreeLocalResource, error) {
	host := "host-A"
	if req.ID == 2 {
		host = "host-B"
	}
	if req.ID == 3 {
		host = "fail"
	}
	return &model.TreeLocalResource{IpAddr: host, Port: 22, Username: "user-" + host, Password: "fixture-password-" + host, AuthMode: model.AuthModePassword}, nil
}

type cloudTerminalResourceService struct{ service.TreeCloudService }

func (cloudTerminalResourceService) GetTreeCloudResourceForConnection(_ context.Context, req *model.GetTreeCloudResourceDetailReq) (*model.TreeCloudResource, error) {
	local, _ := (terminalResourceService{}).GetTreeLocalForConnection(context.Background(), &model.GetTreeLocalResourceDetailReq{ID: req.ID})
	return &model.TreeCloudResource{ResourceType: model.ResourceTypeECS, PublicIP: local.IpAddr, Port: local.Port, Username: local.Username, Password: local.Password, AuthMode: local.AuthMode}, nil
}

type isolatedTerminal struct {
	mu         sync.Mutex
	config     ssh.Config
	options    ssh.TerminalOptions
	closed     chan struct{}
	once       sync.Once
	closeCalls atomic.Int32
}

func (s *isolatedTerminal) ConnectContext(ctx context.Context, config *ssh.Config) error {
	s.mu.Lock()
	s.config = *config
	s.mu.Unlock()
	if config.Host == "fail" {
		return errors.New("fixture initialization failed")
	}
	return ctx.Err()
}
func (s *isolatedTerminal) WebTerminalContext(ctx context.Context, _ int, conn *websocket.Conn) error {
	done := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { defer close(done); _ = conn.Close() })
	defer func() {
		if !stop() {
			<-done
		}
	}()
	s.mu.Lock()
	host := s.config.Host
	s.mu.Unlock()
	if err := conn.WriteMessage(websocket.TextMessage, []byte("ready:"+host)); err != nil {
		return err
	}
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(host+":"+string(message))); err != nil {
			return err
		}
	}
}
func (s *isolatedTerminal) WebTerminalWithOptionsContext(ctx context.Context, id int, conn *websocket.Conn, options ssh.TerminalOptions) error {
	s.mu.Lock()
	s.options = options
	s.mu.Unlock()
	return s.WebTerminalContext(ctx, id, conn)
}
func (s *isolatedTerminal) Close() error {
	s.closeCalls.Add(1)
	s.once.Do(func() { close(s.closed) })
	return nil
}

func TestTerminalSessionIsolation(t *testing.T) {
	for _, kind := range []string{"local", "cloud"} {
		for _, endMode := range []string{"peer close", "request cancel"} {
			t.Run(kind+"/"+endMode, func(t *testing.T) {
				var mu sync.Mutex
				clients := []*isolatedTerminal{}
				factory := TerminalClientFactory(func() TerminalClient {
					client := &isolatedTerminal{closed: make(chan struct{})}
					mu.Lock()
					clients = append(clients, client)
					mu.Unlock()
					return client
				})
				cancels := map[string]context.CancelFunc{}
				engine := gin.New()
				engine.Use(func(c *gin.Context) {
					ctx, cancel := context.WithCancel(c.Request.Context())
					defer cancel()
					mu.Lock()
					cancels[c.Param("id")] = cancel
					mu.Unlock()
					c.Request = c.Request.WithContext(ctx)
					// 同一用户同时访问不同主机也必须是不同SSH客户端
					c.Set("user", jwt.UserClaims{Uid: 7, Username: "fixture-user"})
					c.Next()
				})
				upgrader := &websocket.Upgrader{Subprotocols: []string{wsprotocol.Protocol}}
				if kind == "local" {
					handler := NewTreeLocalHandler(terminalResourceService{}, factory, upgrader, zap.NewNop())
					engine.GET("/terminal/:id", handler.ConnectTerminal)
				} else {
					handler := NewTreeCloudHandler(cloudTerminalResourceService{}, factory, upgrader, zap.NewNop())
					engine.GET("/terminal/:id", handler.ConnectCloudResourceTerminal)
				}
				server := httptest.NewServer(engine)
				t.Cleanup(server.Close)
				dial := func(id, host string) *websocket.Conn {
					t.Helper()
					conn, _, err := (&websocket.Dialer{Subprotocols: []string{wsprotocol.Protocol}, HandshakeTimeout: time.Second}).Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/terminal/"+id, nil)
					if err != nil {
						t.Fatal("WebSocket handshake failed")
					}
					t.Cleanup(func() { _ = conn.Close() })
					_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
					_, ready, err := conn.ReadMessage()
					if err != nil || string(ready) != "ready:"+host {
						t.Fatal("owned session did not acknowledge its own host")
					}
					if conn.Subprotocol() != wsprotocol.Protocol {
						t.Fatal("fixed public subprotocol changed")
					}
					return conn
				}
				first := dial("1", "host-A")
				second := dial("2", "host-B")
				echo := func() {
					t.Helper()
					if err := second.WriteMessage(websocket.TextMessage, []byte("still-alive")); err != nil {
						t.Fatal("session B closed unexpectedly")
					}
					_, reply, err := second.ReadMessage()
					if err != nil || string(reply) != "host-B:still-alive" {
						t.Fatal("session B crossed hosts or lost its connection")
					}
				}
				// 初始化失败也只能释放该请求创建的客户端
				failed, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/terminal/3", nil)
				if failed != nil {
					_ = failed.Close()
				}
				if response != nil {
					_ = response.Body.Close()
				}
				if err == nil {
					t.Fatal("failed SSH initialization upgraded WebSocket")
				}
				mu.Lock()
				snapshot := append([]*isolatedTerminal(nil), clients...)
				cancelSecond := cancels["2"]
				cancelFirst := cancels["1"]
				mu.Unlock()
				if len(snapshot) != 3 {
					t.Fatal("factory did not allocate one client per request")
				}
				for index, host := range []string{"host-A", "host-B", "fail"} {
					client := snapshot[index]
					client.mu.Lock()
					config := client.config
					client.mu.Unlock()
					if config.Host != host || config.Username != "user-"+host || config.Password != "fixture-password-"+host {
						t.Fatal("credentials or host crossed session ownership")
					}
				}
				waitClosed := func(client *isolatedTerminal) {
					t.Helper()
					select {
					case <-client.closed:
					case <-time.After(time.Second):
						t.Fatal("owned SSH client was not reclaimed")
					}
					if client.closeCalls.Load() != 1 {
						t.Fatal("owned client closed more than once")
					}
				}
				waitClosed(snapshot[2])
				echo()
				if endMode == "peer close" {
					_ = first.Close()
				} else {
					cancelFirst()
				}
				waitClosed(snapshot[0])
				echo()
				select {
				case <-snapshot[1].closed:
					t.Fatal("A cleanup closed B")
				default:
				}
				cancelSecond()
				waitClosed(snapshot[1])
				if snapshot[0].closeCalls.Load() != 1 {
					t.Fatal("B cancellation closed A again")
				}
			})
		}
	}
}

func TestTerminalPTYOptionsAndInvalidQueryBeforeConnect(t *testing.T) {
	for _, kind := range []string{"local", "cloud"} {
		t.Run(kind, func(t *testing.T) {
			var calls atomic.Int32
			client := &isolatedTerminal{closed: make(chan struct{})}
			factory := TerminalClientFactory(func() TerminalClient { calls.Add(1); return client })
			engine := gin.New()
			engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7}); c.Next() })
			upgrader := &websocket.Upgrader{Subprotocols: []string{wsprotocol.Protocol}}
			if kind == "local" {
				handler := NewTreeLocalHandler(terminalResourceService{}, factory, upgrader, zap.NewNop())
				engine.GET("/terminal/:id", handler.ConnectTerminal)
			} else {
				handler := NewTreeCloudHandler(cloudTerminalResourceService{}, factory, upgrader, zap.NewNop())
				engine.GET("/terminal/:id", handler.ConnectCloudResourceTerminal)
			}
			invalid := httptest.NewRecorder()
			engine.ServeHTTP(invalid, httptest.NewRequest("GET", "/terminal/1?terminal_mode=pty-v1&rows=0", nil))
			if invalid.Code != 400 || calls.Load() != 0 {
				t.Fatal("invalid terminal options opened SSH")
			}
			server := httptest.NewServer(engine)
			defer server.Close()
			conn, _, err := (&websocket.Dialer{Subprotocols: []string{wsprotocol.Protocol}}).Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/terminal/1?terminal_mode=pty-v1&cols=96&rows=37", nil)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			_ = conn.SetReadDeadline(time.Now().Add(time.Second))
			if _, _, err := conn.ReadMessage(); err != nil {
				t.Fatal(err)
			}
			client.mu.Lock()
			options := client.options
			client.mu.Unlock()
			if options.Mode != ssh.TerminalModePTY || options.Rows != 37 || options.Cols != 96 || conn.Subprotocol() != wsprotocol.Protocol {
				t.Fatal("PTY options or auth subprotocol changed")
			}
		})
	}
}

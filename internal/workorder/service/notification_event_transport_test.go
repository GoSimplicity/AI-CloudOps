package service

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"go.uber.org/zap"
)

type outboxEmailConfig struct{ port int }

func (outboxEmailConfig) IsEnabled() bool                 { return true }
func (outboxEmailConfig) GetMaxRetries() int              { return 3 }
func (outboxEmailConfig) GetRetryInterval() time.Duration { return time.Second }
func (outboxEmailConfig) GetTimeout() time.Duration       { return 5 * time.Second }
func (outboxEmailConfig) GetChannelName() string          { return "email" }
func (outboxEmailConfig) Validate() error                 { return nil }
func (outboxEmailConfig) GetSMTPHost() string             { return "127.0.0.1" }
func (c outboxEmailConfig) GetSMTPPort() int              { return c.port }
func (outboxEmailConfig) GetUsername() string             { return "sender@example.test" }
func (outboxEmailConfig) GetPassword() string             { return "test-only" }
func (outboxEmailConfig) GetFromName() string             { return "test" }
func (outboxEmailConfig) GetUseTLS() bool                 { return false }

type outboxChannels struct {
	email  notification.EmailConfig
	feishu notification.FeishuConfig
}

func (c outboxChannels) GetEmail() notification.EmailConfig   { return c.email }
func (c outboxChannels) GetFeishu() notification.FeishuConfig { return c.feishu }

func TestUnknownTransportOutcomeIsNotRetried(t *testing.T) {
	for _, kind := range []string{"smtp-after-data", "feishu-before-response"} {
		t.Run(kind, func(t *testing.T) {
			var accepted atomic.Int32
			var config notification.NotificationConfig
			if kind == "feishu-before-response" {
				inbox := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/token" {
						w.Header().Set("Content-Type", "application/json")
						_, _ = io.WriteString(w, `{"code":0,"tenant_access_token":"local-token","expire":7200}`)
						return
					}
					accepted.Add(1)
					conn, _, err := w.(http.Hijacker).Hijack()
					if err == nil {
						_ = conn.Close()
					}
				}))
				defer inbox.Close()
				config = outboxChannels{feishu: inboxFeishu{url: inbox.URL}}
			} else {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				defer listener.Close()
				go func() {
					for {
						conn, err := listener.Accept()
						if err != nil {
							return
						}
						go func() {
							defer conn.Close()
							_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
							peer := textproto.NewConn(conn)
							_ = peer.PrintfLine("220 local ESMTP")
							for {
								line, err := peer.ReadLine()
								if err != nil {
									return
								}
								switch {
								case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
									_ = peer.PrintfLine("250 local")
								case strings.HasPrefix(line, "MAIL FROM"), strings.HasPrefix(line, "RCPT TO"):
									_ = peer.PrintfLine("250 accepted")
								case line == "DATA":
									_ = peer.PrintfLine("354 continue")
									_, _ = io.Copy(io.Discard, peer.DotReader())
									accepted.Add(1)
									return
								default:
									_ = peer.PrintfLine("500 unsupported")
								}
							}
						}()
					}
				}()
				config = outboxChannels{email: outboxEmailConfig{port: listener.Addr().(*net.TCPAddr).Port}}
			}
			manager, err := notification.NewManager(config, nil, zap.NewNop())
			if err != nil {
				t.Fatal(err)
			}
			db := eventTestDB(t)
			snapshot := model.WorkorderInstance{Model: model.Model{ID: 4}, ProcessID: 1, Title: "fixture"}
			payload, _ := json.Marshal(snapshot)
			event := model.WorkorderNotificationEvent{InstanceID: 4, EventType: model.EventTypeInstanceCreated, Snapshot: payload, Status: "pending", NextAttemptAt: time.Now().UTC().Add(-time.Second)}
			if err = db.Create(&event).Error; err != nil {
				t.Fatal(err)
			}
			channel, address := "email", "target@example.test"
			if kind == "feishu-before-response" {
				channel, address = "feishu", "oc_fixture"
			}
			cfg := &model.WorkorderNotification{Model: model.Model{ID: 5}, Status: 1, TriggerType: model.TriggerTypeImmediate, MaxRetries: 3, Channels: model.StringList{channel}, RecipientTypes: model.StringList{"custom"}, RecipientAddresses: model.NotificationAddresses{{Channel: channel, Address: address}}, MessageTemplate: "fixture"}
			worker := &NotificationEventWorker{db: db, service: &workorderNotificationService{dao: &eventTestConfigDAO{config: cfg}, notificationMgr: manager, logger: zap.NewNop()}, logger: zap.NewNop()}
			if err = worker.processOne(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err = db.First(&event, event.ID).Error; err != nil {
				t.Fatal(err)
			}
			if event.Status != "uncertain" || accepted.Load() != 1 {
				t.Fatalf("status=%s transport calls=%d", event.Status, accepted.Load())
			}
			if err = worker.processOne(context.Background()); err != errNoNotificationEvent {
				t.Fatalf("unknown event was eligible: %v", err)
			}
			if accepted.Load() != 1 {
				t.Fatal("unknown delivery repeated")
			}
		})
	}
}

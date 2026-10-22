package notification

import (
	"context"
	"encoding/json"
	"io"
	"mime/quotedprintable"
	"net"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"net/textproto"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

type localEmailConfig struct{ port int }

func (localEmailConfig) IsEnabled() bool                 { return true }
func (localEmailConfig) GetMaxRetries() int              { return 0 }
func (localEmailConfig) GetRetryInterval() time.Duration { return time.Millisecond }
func (localEmailConfig) GetTimeout() time.Duration       { return 5 * time.Second }
func (localEmailConfig) GetChannelName() string          { return "email" }
func (localEmailConfig) Validate() error                 { return nil }
func (localEmailConfig) GetSMTPHost() string             { return "127.0.0.1" }
func (c localEmailConfig) GetSMTPPort() int              { return c.port }
func (localEmailConfig) GetUsername() string             { return "sender@example.test" }
func (localEmailConfig) GetPassword() string             { return "local-test-only" }
func (localEmailConfig) GetFromName() string             { return "Local integration" }
func (localEmailConfig) GetUseTLS() bool                 { return false }

type localFeishuConfig struct{ base string }

func (localFeishuConfig) IsEnabled() bool                   { return true }
func (localFeishuConfig) GetMaxRetries() int                { return 0 }
func (localFeishuConfig) GetRetryInterval() time.Duration   { return time.Millisecond }
func (localFeishuConfig) GetTimeout() time.Duration         { return 5 * time.Second }
func (localFeishuConfig) GetChannelName() string            { return "feishu" }
func (localFeishuConfig) Validate() error                   { return nil }
func (localFeishuConfig) GetAppID() string                  { return "local-app" }
func (localFeishuConfig) GetAppSecret() string              { return "local-test-only" }
func (c localFeishuConfig) GetWebhookURL() string           { return c.base + "/wrong-webhook/" }
func (c localFeishuConfig) GetPrivateMessageAPI() string    { return c.base + "/messages" }
func (c localFeishuConfig) GetTenantAccessTokenAPI() string { return c.base + "/token" }

type localChannels struct {
	email  EmailConfig
	feishu FeishuConfig
}

func (c localChannels) GetEmail() EmailConfig   { return c.email }
func (c localChannels) GetFeishu() FeishuConfig { return c.feishu }

func TestRealLocalSMTPDeliveryAndExplicitChannel(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	received := make(chan string, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		defer connection.Close()
		_ = connection.SetDeadline(time.Now().Add(5 * time.Second))
		peer := textproto.NewConn(connection)
		_ = peer.PrintfLine("220 localhost ESMTP")
		for {
			line, err := peer.ReadLine()
			if err != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				_ = peer.PrintfLine("250 localhost")
			case strings.HasPrefix(line, "MAIL FROM"), strings.HasPrefix(line, "RCPT TO"):
				_ = peer.PrintfLine("250 accepted")
			case line == "DATA":
				_ = peer.PrintfLine("354 send data")
				data, err := io.ReadAll(peer.DotReader())
				if err != nil {
					return
				}
				received <- string(data)
				_ = peer.PrintfLine("250 queued")
			case line == "QUIT":
				_ = peer.PrintfLine("221 bye")
				return
			default:
				_ = peer.PrintfLine("500 unsupported")
			}
		}
	}()
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	number, _ := strconv.Atoi(port)
	manager, err := NewManager(localChannels{email: localEmailConfig{number}}, nil, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	result, err := manager.SendNotification(context.Background(), &SendRequest{Channel: "email", RecipientType: "feishu_user", RecipientAddr: "recipient@example.test", Subject: "Local verification", Content: "owned-local-delivery-marker"})
	if err != nil || result == nil || !result.Success {
		t.Fatalf("SMTP delivery failed: %v", err)
	}
	select {
	case body := <-received:
		message, parseErr := mail.ReadMessage(strings.NewReader(body))
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		decoded, readErr := io.ReadAll(quotedprintable.NewReader(message.Body))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !strings.Contains(string(decoded), "owned-local-delivery-marker") {
			t.Fatal("missing message body")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SMTP inbox empty")
	}
	if _, err = manager.SendNotification(context.Background(), &SendRequest{Channel: "unknown", RecipientAddr: "recipient@example.test"}); err == nil {
		t.Fatal("unknown channel silently became email")
	}
}

func TestRealLocalFeishuChatRoutingAndFailure(t *testing.T) {
	var deliveries atomic.Int32
	var rejected atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/token":
			_, _ = io.WriteString(w, `{"code":0,"tenant_access_token":"local-token","expire":7200}`)
		case "/messages":
			if r.URL.Query().Get("receive_id_type") != "chat_id" || r.Header.Get("Authorization") != "Bearer local-token" {
				t.Error("incorrect authenticated chat routing")
			}
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["receive_id"] != "oc_ownedlocal" {
				t.Error("incorrect recipient")
			}
			deliveries.Add(1)
			if rejected.Load() {
				_, _ = io.WriteString(w, `{"code":999,"msg":"test failure"}`)
			} else {
				_, _ = io.WriteString(w, `{"code":0,"data":{"message_id":"local-message"}}`)
			}
		default:
			t.Error("chat ID used as webhook token")
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	manager, err := NewManager(localChannels{feishu: localFeishuConfig{server.URL}}, nil, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	request := &SendRequest{Channel: "feishu", RecipientAddr: "oc_ownedlocal", Content: "Local delivery only"}
	result, err := manager.SendNotification(context.Background(), request)
	if err != nil || !result.Success {
		t.Fatalf("local HTTP delivery failed: %v", err)
	}
	rejected.Store(true)
	if _, err = manager.SendNotification(context.Background(), request); err == nil {
		t.Fatal("provider failure reported as success")
	}
	if deliveries.Load() != 2 {
		t.Fatal("unexpected dispatch count")
	}
}

func TestAddressValidation(t *testing.T) {
	for _, item := range []struct{ channel, address string }{{"email", "Name <a@example.test>"}, {"email", "a@example.test,b@example.test"}, {"email", "a@example.test\r\nBcc: victim@example.test"}, {"feishu", "https://example.test/hook"}, {"feishu", "user"}, {"webhook", "https://example.test"}} {
		if _, err := NormalizeAddress(item.channel, item.address); err == nil {
			t.Fatalf("invalid %s address accepted", item.channel)
		}
	}
	if address, err := NormalizeAddress("email", " Person@Example.Test "); err != nil || address != "person@example.test" {
		t.Fatal("normalization failed")
	}
}

type falseSuccessChannel struct{}

func (falseSuccessChannel) GetName() string { return "email" }
func (falseSuccessChannel) Send(context.Context, *SendRequest) (*SendResponse, error) {
	return &SendResponse{Success: false}, nil
}
func (falseSuccessChannel) Validate() error                 { return nil }
func (falseSuccessChannel) IsEnabled() bool                 { return true }
func (falseSuccessChannel) GetMaxRetries() int              { return 0 }
func (falseSuccessChannel) GetRetryInterval() time.Duration { return 0 }
func TestManagerRejectsUnconfirmedDelivery(t *testing.T) {
	manager := &Manager{channels: map[string]NotificationChannel{"email": falseSuccessChannel{}}, logger: zap.NewNop()}
	if _, err := manager.SendNotification(context.Background(), &SendRequest{Channel: "email", RecipientAddr: "a@example.test"}); err == nil {
		t.Fatal("false success ignored")
	}
}

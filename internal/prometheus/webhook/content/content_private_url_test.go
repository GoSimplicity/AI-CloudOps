package content

import (
	"context"
	"net/http"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	robotAPI "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/robot"
	"go.uber.org/zap"
)

type fixturePrivateRobot struct{ robotAPI.WebhookRobot }

func (fixturePrivateRobot) GetPrivateRobotToken() string { return "fixture-token" }

func TestPrivateMessageEndpointCanBeIsolated(t *testing.T) {
	cfg := &config.WebhookConfig{}
	cfg.ImFeishu.PrivateMessageAPI = "http://127.0.0.1:9/open-apis/im/v1/messages"
	transport := &recordTransport{}
	content := &webhookContent{frontDomain: cfg.FrontDomain, backendDomain: cfg.BackendDomain, groupMessageAPI: cfg.ImFeishu.GroupMessageAPI, privateMessageAPI: cfg.ImFeishu.PrivateMessageAPI, l: zap.NewNop(), robot: fixturePrivateRobot{}, client: &http.Client{Transport: transport}}
	if err := content.SentFeishuPrivate(context.Background(), "{}", map[string]string{"fixture-user": ""}); err != nil {
		t.Fatal(err)
	}
	if len(transport.urls) != 1 || transport.urls[0] != "http://127.0.0.1:9/open-apis/im/v1/messages?receive_id_type=user_id" {
		t.Fatal("private messaging ignored the configured isolated endpoint")
	}
}

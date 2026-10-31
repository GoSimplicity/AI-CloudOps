package robot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func TestRobotUsesTypedConfigurationSnapshot(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("webhook.im_feishu.private_robot_app_id", "wrong-global-id")
	viper.Set("webhook.im_feishu.private_robot_app_secret", "wrong-global-secret")
	viper.Set("webhook.im_feishu.tenant_access_token_api", "https://wrong.invalid/token")
	viper.Set("webhook.im_feishu.request_timeout_seconds", 1)
	cfg := &config.WebhookConfig{ImFeishu: config.ImFeishuConfig{PrivateRobotAppID: "fixture-id", PrivateRobotAppSecret: "fixture-secret", TenantAccessTokenAPI: "https://fixture.invalid/token", RequestTimeoutSeconds: 7}}
	client := NewWebhookRobot(cfg, zap.NewNop()).(*webhookRobot)
	*cfg = config.WebhookConfig{}
	if client.httpClient.Timeout != 7*time.Second {
		t.Fatal("typed token timeout was not captured")
	}
	calls := 0
	client.httpClient.Transport = tokenPrivacyTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.String() != "https://fixture.invalid/token" {
			t.Fatal("typed token endpoint drifted")
		}
		var body map[string]string
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["app_id"] != "fixture-id" || body["app_secret"] != "fixture-secret" {
			t.Fatal("typed credential snapshot drifted")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0,"tenant_access_token":"fixture-token","expire":7200}`)), Header: make(http.Header), Request: req}, nil
	})
	token, err := client.GetTenantAccessToken(context.Background())
	if err != nil || token != "fixture-token" {
		t.Fatal("typed token request failed")
	}
	client.RefreshPrivateRobotToken(context.Background())
	if client.GetPrivateRobotToken() != "fixture-token" || calls != 2 {
		t.Fatal("typed snapshot changed token refresh behavior")
	}
}

func TestNilRobotConfigurationRemainsDisabled(t *testing.T) {
	client := NewWebhookRobot(nil, zap.NewNop()).(*webhookRobot)
	transport := &forbiddenTokenTransport{}
	client.httpClient.Transport = transport
	client.RefreshPrivateRobotToken(context.Background())
	if transport.calls != 0 {
		t.Fatal("nil configuration attempted HTTP")
	}
}

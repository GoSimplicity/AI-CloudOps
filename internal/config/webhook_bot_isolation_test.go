package config_test

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/content"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/robot"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type loadedBotTransport func(*http.Request) (*http.Response, error)

func (transport loadedBotTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestLoadedWebhookBotIsolationAndCredentialGate(t *testing.T) {
	const tokenURL = "http://127.0.0.1:9/open-apis/auth/v3/tenant_access_token/internal"
	const privateURL = "http://127.0.0.1:9/open-apis/im/v1/messages"
	const groupURL = "http://127.0.0.1:9/open-apis/bot/v2/hook"
	for _, test := range []struct {
		name, id, secret string
		enabled          bool
	}{{"empty", "", "", false}, {"half ID", "fixture-id", "", false}, {"half secret", "", "fixture-secret", false}, {"whitespace", " \t", " \n", false}, {"enabled", "fixture-id", "fixture-secret", true}} {
		t.Run(test.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			config.ClearWebhookEnvForTest(t)
			t.Setenv("WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_ID", test.id)
			t.Setenv("WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_SECRET", test.secret)
			t.Setenv("WEBHOOK_IM_FEISHU_TENANT_ACCESS_TOKEN_API", tokenURL)
			t.Setenv("WEBHOOK_IM_FEISHU_PRIVATE_MESSAGE_API", privateURL)
			t.Setenv("WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API", groupURL)
			cfg, _, err := config.LoadWebhookFromFileForTest("../../config/webhook.yaml")
			if err != nil {
				t.Fatal("actual webhook loader rejected fixture settings")
			}
			if cfg.ImFeishu.TenantAccessTokenAPI != tokenURL || cfg.ImFeishu.PrivateMessageAPI != privateURL || cfg.ImFeishu.GroupMessageAPI != groupURL {
				t.Fatal("actual loader ignored one of the three isolated bot endpoints")
			}
			calls := 0
			original := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = original })
			http.DefaultTransport = loadedBotTransport(func(request *http.Request) (*http.Response, error) {
				calls++
				if !test.enabled {
					t.Fatal("disabled loaded configuration attempted HTTP")
				}
				if request.URL.String() != tokenURL {
					t.Fatal("token request escaped the isolated endpoint")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0,"tenant_access_token":"fixture-token","expire":7200}`)), Header: make(http.Header), Request: request}, nil
			})
			core, logs := observer.New(zap.DebugLevel)
			client := robot.NewWebhookRobot(cfg, zap.New(core))
			client.RefreshPrivateRobotToken(context.Background())
			client.RefreshPrivateRobotToken(context.Background())
			first, err := client.GetTenantAccessToken(context.Background())
			if !test.enabled {
				if !errors.Is(err, robot.ErrPrivateRobotDisabled) || calls != 0 || logs.Len() != 0 {
					t.Fatal("loaded disabled bot did not remain quiet and offline")
				}
				return
			}
			if err != nil || first != "fixture-token" || client.GetPrivateRobotToken() != "fixture-token" {
				t.Fatal("enabled token polling failed")
			}
			before := calls
			second, err := client.GetTenantAccessToken(context.Background())
			if err != nil || second != first || calls != before || calls != 3 {
				t.Fatal("valid credentials polling or tenant token caching contract changed")
			}
		})
	}
}

func TestFreshBotEndpointsOverrideFakeCredentialFile(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	config.ClearWebhookEnvForTest(t)
	endpoints := map[string]string{
		"WEBHOOK_IM_FEISHU_TENANT_ACCESS_TOKEN_API": "http://127.0.0.1:9/open-apis/auth/v3/tenant_access_token/internal",
		"WEBHOOK_IM_FEISHU_PRIVATE_MESSAGE_API":     "http://127.0.0.1:9/open-apis/im/v1/messages",
		"WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API":       "http://127.0.0.1:9/open-apis/bot/v2/hook",
	}
	for key, value := range endpoints {
		t.Setenv(key, value)
	}
	path := filepath.Join(t.TempDir(), "fixture.yaml")
	source := `webhook:
  port: "8889"
  im_feishu:
    private_robot_app_id: fixture-file-id
    private_robot_app_secret: fixture-file-secret
    tenant_access_token_api: https://file-provider.invalid/token
    private_message_api: https://file-provider.invalid/private
    group_message_api: https://file-provider.invalid/group
`
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := config.LoadWebhookFromFileForTest(path)
	if err != nil {
		t.Fatal("fake credential file failed to load")
	}
	if cfg.ImFeishu.PrivateRobotAppID != "fixture-file-id" || cfg.ImFeishu.PrivateRobotAppSecret != "fixture-file-secret" {
		t.Fatal("existing empty-env fallback contract changed")
	}
	if cfg.ImFeishu.TenantAccessTokenAPI != endpoints["WEBHOOK_IM_FEISHU_TENANT_ACCESS_TOKEN_API"] || cfg.ImFeishu.PrivateMessageAPI != endpoints["WEBHOOK_IM_FEISHU_PRIVATE_MESSAGE_API"] || cfg.ImFeishu.GroupMessageAPI != endpoints["WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API"] {
		t.Fatal("fresh endpoint override was lost")
	}
	observed := map[string]int{}
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	http.DefaultTransport = loadedBotTransport(func(request *http.Request) (*http.Response, error) {
		if request.URL.Scheme != "http" || request.URL.Host != "127.0.0.1:9" {
			t.Fatal("configured source file escaped loopback bot boundary")
		}
		observed[request.URL.Path]++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0,"tenant_access_token":"fixture-token","expire":7200}`)), Header: make(http.Header), Request: request}, nil
	})
	client := robot.NewWebhookRobot(cfg, zap.NewNop())
	client.RefreshPrivateRobotToken(context.Background())
	messages := content.NewWebhookContent(cfg, zap.NewNop(), nil, client)
	if err := messages.SentFeishuGroup(context.Background(), "{}", "fixture-group"); err != nil {
		t.Fatal("fake group message failed")
	}
	if err := messages.SentFeishuPrivate(context.Background(), "{}", map[string]string{"fixture-user": ""}); err != nil {
		t.Fatal("fake private message failed")
	}
	for _, path := range []string{"/open-apis/auth/v3/tenant_access_token/internal", "/open-apis/im/v1/messages", "/open-apis/bot/v2/hook/fixture-group"} {
		if observed[path] != 1 {
			t.Fatal("one of the three actual bot paths was not observed exactly once")
		}
	}
}

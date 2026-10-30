package utils_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	alertDAO "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/content"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/robot"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type confirmedTransport func(*http.Request) (*http.Response, error)

func (transport confirmedTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestAllFeishuEntrypointsRequireExplicitSuccess(t *testing.T) {
	const canary = "RESULT_PRIVATE"
	for _, entrypoint := range []string{"eventDAO", "content", "robot"} {
		t.Run(entrypoint, func(t *testing.T) {
			for _, test := range []struct {
				name, prefix string
				success      bool
			}{
				{"missing code", "", false}, {"null code", `"code":null,`, false}, {"bool code", `"code":false,`, false}, {"string code", `"code":"0",`, false},
				{"nonzero", `"code":42,`, false}, {"duplicate", `"code":42,"code":0,`, false}, {"StatusCode substitute", `"StatusCode":0,`, false},
				{"non JSON", "INVALID", false}, {"explicit zero", `"code":0,`, true},
			} {
				t.Run(test.name, func(t *testing.T) {
					viper.Reset()
					t.Cleanup(viper.Reset)
					viper.Set("webhook.im_feishu.private_robot_app_id", canary)
					viper.Set("webhook.im_feishu.private_robot_app_secret", canary)
					viper.Set("webhook.im_feishu.request_timeout_seconds", 7)
					endpoint := "https://" + canary + ":" + canary + "@fixture.invalid/path?private=" + canary
					viper.Set("webhook.im_feishu.tenant_access_token_api", endpoint)
					viper.Set("webhook.im_feishu.group_message_api", endpoint)
					body := "{" + test.prefix + `"tenant_access_token":"` + canary + `","expire":7200,"msg":"` + canary + `","data":{"unknown":true}}`
					if test.prefix == "INVALID" {
						body = canary
					}
					calls := 0
					original := http.DefaultTransport
					t.Cleanup(func() { http.DefaultTransport = original })
					http.DefaultTransport = confirmedTransport(func(request *http.Request) (*http.Response, error) {
						calls++
						if request.Method != "POST" {
							t.Fatal("entrypoint HTTP method changed")
						}
						return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: request}, nil
					})
					core, logs := observer.New(zap.DebugLevel)
					logger := zap.New(core)
					var err error
					switch entrypoint {
					case "eventDAO":
						err = alertDAO.NewAlertManagerEventDAO(nil, logger, nil).SendMessageToGroup(context.Background(), endpoint, canary)
					case "content":
						err = content.NewWebhookContent(logger, nil, nil).SentFeishuGroup(context.Background(), canary, canary)
					case "robot":
						client := robot.NewWebhookRobot(logger)
						var token string
						token, err = client.GetTenantAccessToken(context.Background())
						if test.success && token != canary {
							t.Fatal("standard success token changed")
						}
						if calls != 1 {
							t.Fatal("first token operation retried")
						}
						_, again := client.GetTenantAccessToken(context.Background())
						if (again == nil) != test.success {
							t.Fatal("explicit repeated operation changed protocol outcome")
						}
						expected := 2
						if test.success {
							expected = 1
						}
						if calls != expected {
							t.Fatal("unconfirmed token was cached, or confirmed token was not cached")
						}
					}
					if (err == nil) != test.success {
						t.Fatal("HTTP200 did not require explicit business code zero")
					}
					if entrypoint != "robot" && calls != 1 {
						t.Fatal("entrypoint retried")
					}
					if err != nil && strings.Contains(err.Error(), canary) {
						t.Fatal("provider message/body leaked in returned error")
					}
					for _, entry := range logs.All() {
						text := entry.Message + fmt.Sprint(entry.ContextMap())
						if strings.Contains(text, canary) {
							t.Fatal("entrypoint leaked private data")
						}
						if !test.success && strings.Contains(entry.Message, "成功") {
							t.Fatal("unconfirmed response logged success")
						}
					}
				})
			}
		})
	}
}

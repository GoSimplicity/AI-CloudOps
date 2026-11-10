package api

import (
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"go.uber.org/zap"
)

func TestProductionBackendDomain(t *testing.T) {
	const key = "fixture-action-hmac"
	const token = "fixture-receive-auth"
	reject := []struct {
		name   string
		env    string
		domain string
		reason string
	}{
		{"ipv4", "production", "http://127.0.0.1:8889/api/v1/alerts", "回环地址"},
		{"ipv4-other", "production", "http://127.1.2.3/api/v1/alerts", "回环地址"},
		{"localhost", "production", "http://localhost/api/v1/alerts", "回环地址"},
		{"LOCALHOST", "production", "https://LOCALHOST:8889/api/v1/alerts", "回环地址"},
		{"ipv6-port", "production", "http://[::1]:8889/api/v1/alerts", "回环地址"},
		{"localhost-suffix", "production", "http://card.localhost:8889/api/v1/alerts", "回环地址"},
		{"unset", "", "http://127.0.0.1:8889/api/v1/alerts", "回环地址"},
		{"unrecognized", "test", "http://localhost/api/v1/alerts", "回环地址"},
		{"no-scheme", "production", "127.0.0.1:8889/api/v1/alerts", "无法解析"},
		{"empty-host", "production", "http://", "无法解析"},
		{"bare-ipv6", "prd", "[::1]:8889", "无法解析"},
		{"unspecified-v4", "production", "http://0.0.0.0/api/v1/alerts", "未指定地址"},
		{"unspecified-v6", "production", "http://[::]/api/v1/alerts", "未指定地址"},
		{"unspecified-v6-brackets", "production", "https://[::]/api/v1/alerts", "未指定地址"},
		{"unspecified-v4-port", "production", "http://0.0.0.0:8889/api/v1/alerts", "未指定地址"},
	}
	for _, tt := range reject {
		t.Run("reject/"+tt.name, func(t *testing.T) {
			t.Setenv("ENV", tt.env)
			cfg := &config.WebhookConfig{
				BackendDomain:           tt.domain,
				ActionSigningKey:        key,
				AlertmanagerBearerToken: token,
				ImFeishu:                config.ImFeishuConfig{RequestTimeoutSeconds: 7},
			}
			handler, err := NewWebHookHandler(cfg, zap.NewNop(), nil, nil)
			if err == nil || handler != nil {
				t.Fatal("loopback or unparsed backend domain was accepted")
			}
			if !strings.Contains(err.Error(), "WEBHOOK_BACKEND_DOMAIN") || !strings.Contains(err.Error(), tt.reason) {
				t.Fatal(err)
			}
			if strings.Contains(err.Error(), key) || strings.Contains(err.Error(), token) || strings.Contains(err.Error(), tt.domain) {
				t.Fatal("error echoed the setting")
			}
		})
	}

	allow := []struct {
		name   string
		env    string
		domain string
	}{
		{"public", "production", "https://alerts.example/api/v1/alerts"},
		{"public-port", " Production ", "https://alerts.example:8443/api/v1/alerts"},
		{"development", "development", "http://127.0.0.1:8889/api/v1/alerts"},
		{"dev", "dev", "http://[::1]:8889/api/v1/alerts"},
		{"dev-blank", " Dev ", "http://LOCALHOST/api/v1/alerts"},
		{"dev-unspecified", "development", "http://0.0.0.0:8889/api/v1/alerts"},
		{"dev-unspecified-v6", "dev", "http://[::]/api/v1/alerts"},
	}
	for _, tt := range allow {
		t.Run("allow/"+tt.name, func(t *testing.T) {
			t.Setenv("ENV", tt.env)
			cfg := &config.WebhookConfig{
				BackendDomain: tt.domain,
				ImFeishu:      config.ImFeishuConfig{RequestTimeoutSeconds: 7},
			}
			if tt.env != "development" && tt.env != "dev" && tt.env != " Dev " {
				cfg.ActionSigningKey = key
				cfg.AlertmanagerBearerToken = token
			}
			handler, err := NewWebHookHandler(cfg, zap.NewNop(), nil, nil)
			if err != nil || handler == nil {
				t.Fatal(err)
			}
		})
	}
}

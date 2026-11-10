package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestHTTPServiceDiscoveryStartupLogOmitsToken(t *testing.T) {
	for _, tt := range []struct {
		name, token, want string
		warn              bool
	}{
		{name: "unset", want: "未配置", warn: true},
		{name: "invalid", token: "fixture token", want: "无效", warn: true},
		{name: "configured", token: "fixture-http-sd-token", want: "bearer token"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			core, recorded := observer.New(zap.DebugLevel)
			application := &App{logger: zap.New(core), cfg: &config.Config{Prometheus: config.PrometheusConfig{HttpSdToken: tt.token}}}
			application.logHTTPServiceDiscovery()
			if recorded.Len() != 1 {
				t.Fatalf("log count=%d", recorded.Len())
			}
			entry := recorded.All()[0]
			if tt.warn && entry.Level != zap.WarnLevel {
				t.Fatal("unsafe HTTP SD mode was not a warning")
			}
			encoded, err := json.Marshal(entry)
			if err != nil {
				t.Fatal(err)
			}
			text := string(encoded)
			if !strings.Contains(text, tt.want) || strings.Contains(text, "fixture") {
				t.Fatal("startup log missing the mode or contained the token")
			}
		})
	}
}

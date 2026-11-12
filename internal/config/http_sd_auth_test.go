package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestHTTPServiceDiscoveryTokenFromEnvironment(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	setDefaults()
	bindEnvVars()
	if got := viper.GetString("prometheus.httpSdToken"); got != "" {
		t.Fatal("HTTP SD token default must stay empty")
	}
	t.Setenv("PROMETHEUS_HTTP_SD_TOKEN", "fixture-http-sd-token")
	cfg := Config{}
	if err := viper.Unmarshal(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Prometheus.HttpSdToken != "fixture-http-sd-token" {
		t.Fatal("PROMETHEUS_HTTP_SD_TOKEN was not loaded")
	}
	if cfg.Prometheus.UsableHTTPServiceDiscoveryToken() != "fixture-http-sd-token" {
		t.Fatal("usable HTTP SD token did not keep the configured value")
	}
}

func TestValidateHTTPServiceDiscoveryToken(t *testing.T) {
	t.Setenv("ENV", "development")
	valid := validConfigForTest()
	valid.Prometheus.HttpSdToken = "fixture-http-sd-token"
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{" fixture-http-sd-token", "fixture-http-sd-token ", "fixture token", "fixture\nhttp", strings.Repeat("a", 513)} {
		cfg := validConfigForTest()
		cfg.Prometheus.HttpSdToken = value
		err := cfg.Validate()
		if err == nil || !strings.Contains(err.Error(), "PROMETHEUS_HTTP_SD_TOKEN") {
			t.Fatalf("invalid token accepted: %v", err)
		}
		if strings.Contains(err.Error(), "fixture") || strings.Contains(err.Error(), strings.Repeat("a", 16)) {
			t.Fatal("validation error revealed the token")
		}
		if cfg.Prometheus.UsableHTTPServiceDiscoveryToken() != "" {
			t.Fatal("invalid token remained usable")
		}
	}
}

func TestMissingHTTPServiceDiscoveryTokenFollowsEnvironment(t *testing.T) {
	for _, env := range []string{"development", "dev", " Dev ", "DEVELOPMENT"} {
		t.Run("allow/"+env, func(t *testing.T) {
			t.Setenv("ENV", env)
			cfg := validConfigForTest()
			cfg.Prometheus.HttpSdToken = ""
			if err := cfg.Validate(); err != nil {
				t.Fatalf("开发环境应允许留空: %v", err)
			}
		})
	}
	for _, env := range []string{"production", "prod", "prd", " Production ", "PROD", "", " ", "test", "testing", "TEST", "staging"} {
		t.Run("refuse/"+env, func(t *testing.T) {
			t.Setenv("ENV", env)
			cfg := prodTestConfig()
			cfg.Prometheus.HttpSdToken = ""
			err := cfg.Validate()
			if err == nil || !strings.Contains(err.Error(), "PROMETHEUS_HTTP_SD_TOKEN") || !strings.Contains(err.Error(), "拒绝启动") {
				t.Fatalf("ENV=%q 应拒绝启动, got %v", env, err)
			}
			if strings.Contains(err.Error(), "fixture") || strings.Contains(err.Error(), "fake-") {
				t.Fatal("错误信息泄露了配置值")
			}
		})
	}
	t.Setenv("ENV", "production")
	if err := prodTestConfig().Validate(); err != nil {
		t.Fatalf("生产环境配置了令牌后应通过: %v", err)
	}
}

func TestHTTPServiceDiscoveryStartupMessageOmitsToken(t *testing.T) {
	t.Setenv("ENV", "development")
	for _, token := range []string{"", "fixture-http-sd-token"} {
		warn, message := (PrometheusConfig{HttpSdToken: token}).HTTPServiceDiscoveryStartup()
		if strings.Contains(message, "fixture") || strings.Contains(message, "已按未配置处理") {
			t.Fatal("startup message contained the token or a stale fallback")
		}
		if token == "fixture-http-sd-token" {
			if warn || !strings.Contains(message, "bearer token") {
				t.Fatal("configured token did not report bearer mode")
			}
			continue
		}
		if !warn || !strings.Contains(message, "回环") {
			t.Fatal("unset token did not report loopback-only mode")
		}
	}
	t.Setenv("ENV", "production")
	warn, message := (PrometheusConfig{}).HTTPServiceDiscoveryStartup()
	if !warn || !strings.Contains(message, "拒绝启动") || strings.Contains(message, "当前为开发环境") {
		t.Fatalf("生产环境未配置提示不正确: warn=%v message=%s", warn, message)
	}
}

func TestHTTPServiceDiscoveryRequestPolicy(t *testing.T) {
	const token = "fixture-http-sd-token"
	for _, tt := range []struct {
		name, token, remote, header string
		allow                       bool
	}{
		{name: "missing token", token: token, remote: "203.0.113.8:4000", allow: false},
		{name: "wrong token", token: token, remote: "203.0.113.8:4000", header: "Bearer fixture-http-sd-wrong", allow: false},
		{name: "token prefix", token: token, remote: "203.0.113.8:4000", header: "Bearer fixture-http-sd-token-extra", allow: false},
		{name: "correct token", token: token, remote: "203.0.113.8:4000", header: "Bearer " + token, allow: true},
		{name: "bearer case", token: token, remote: "198.51.100.8:4000", header: "bearer " + token, allow: true},
		{name: "basic auth", token: token, remote: "127.0.0.1:4000", header: "Basic " + token, allow: false},
		{name: "configured loopback without token", token: token, remote: "127.0.0.1:4000", allow: false},
		{name: "unconfigured remote", remote: "203.0.113.8:4000", allow: false},
		{name: "unconfigured private", remote: "172.18.0.4:4000", allow: false},
		{name: "unconfigured empty peer", allow: false},
		{name: "unconfigured loopback", remote: "127.0.0.1:4000", allow: true},
		{name: "unconfigured ipv6 loopback", remote: "[::1]:4000", allow: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := HTTPServiceDiscoveryRequestAllowed(tt.token, tt.remote, tt.header); got != tt.allow {
				t.Fatalf("allowed=%v", got)
			}
		})
	}
}

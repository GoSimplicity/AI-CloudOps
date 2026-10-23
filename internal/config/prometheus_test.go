package config

import (
	"github.com/spf13/viper"
	"testing"
)

func TestHTTPDiscoveryDefaultAndEnvironmentOverride(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	setDefaults()
	bindEnvVars()
	if got := viper.GetString("prometheus.httpSdAPI"); got != "http://localhost:8889/api/not_auth/getTreeNodeBindIps" {
		t.Fatal("discovery default does not target backend")
	}
	t.Setenv("PROMETHEUS_HTTP_SD_API", "http://backend:8889/api/not_auth/getTreeNodeBindIps")
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Prometheus.HttpSdAPI != "http://backend:8889/api/not_auth/getTreeNodeBindIps" {
		t.Fatal("discovery environment override ignored")
	}
}

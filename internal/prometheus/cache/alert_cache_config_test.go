package cache

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertDAO "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type configSendGroups struct{ alertDAO.AlertManagerSendDAO }

func (configSendGroups) GetMonitorSendGroupByPoolID(context.Context, int) ([]*model.MonitorSendGroup, int64, error) {
	return []*model.MonitorSendGroup{{Model: model.Model{ID: 3}, Name: "fixture-group", RepeatInterval: "1h"}}, 1, nil
}

func TestAlertCacheUsesInjectedConfiguration(t *testing.T) {
	for _, test := range []struct{ name, address, port, wantURL string }{
		{"configured host", "typed-webhook:8890", "8891", "http://typed-webhook:8890/api/v1/alerts/receive?send_group_id=3"},
		{"configured full URL", "HTTPS://typed-webhook/api/v1/alerts/receive?existing=kept", "8891", "HTTPS://typed-webhook/api/v1/alerts/receive?existing=kept&send_group_id=3"},
		{"blank address typed port fallback", "", "8891", "http://localhost:8891/api/v1/alerts/receive?send_group_id=3"},
	} {
		t.Run(test.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("prometheus.alert_webhook_addr", "global-wrong:1")
			viper.Set("prometheus.alert_webhook_file_dir", "/global-wrong")
			viper.Set("server.port", "2")
			cfg := &config.Config{Server: config.ServerConfig{Port: test.port}, Prometheus: config.PrometheusConfig{AlertWebhookAddr: test.address, AlertWebhookFileDir: "/typed/hooks"}}
			cache, err := NewAlertManagerConfigCache(cfg, zap.NewNop(), nil, configSendGroups{}, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			implementation := cache.(*alertManagerConfigCache)
			if got := implementation.generateWebhookFileContent(&model.MonitorSendGroup{Model: model.Model{ID: 3}}); got != test.wantURL {
				t.Fatalf("typed configuration ignored: got %q want %q", got, test.wantURL)
			}
			// A constructed component is one configuration snapshot. Neither global
			// state nor later mutation of the caller's Config can redirect its output.
			viper.Set("server.port", "3")
			viper.Set("prometheus.alert_webhook_addr", "later-global:4")
			viper.Set("prometheus.alert_webhook_file_dir", "/later-global")
			cfg.Server.Port = "5"
			cfg.Prometheus.AlertWebhookAddr = "later-typed:6"
			cfg.Prometheus.AlertWebhookFileDir = "/later-typed"
			if got := implementation.generateWebhookFileContent(&model.MonitorSendGroup{Model: model.Model{ID: 3}}); got != test.wantURL {
				t.Fatalf("constructed cache changed with unrelated configuration: %q", got)
			}
			// Exercise the public route output. No AlertManager instances are supplied,
			// so this fixture does not contact Redis, a database, or an external service.
			routes, receivers := cache.GenerateRouteConfigForPool(context.Background(), &model.MonitorAlertManagerPool{Model: model.Model{ID: 7}, Name: "fixture-pool"})
			if len(routes) != 1 || len(receivers) != 1 || len(receivers[0].WebhookConfigs) != 1 {
				t.Fatal("route/receiver output was lost")
			}
			if got := receivers[0].WebhookConfigs[0].URLFile; got != "/typed/hooks/webhook_fixture-pool_3.yml" {
				t.Fatalf("receiver URLFile ignores injected directory: %q", got)
			}
		})
	}
}

func TestAlertCacheRejectsMissingConfiguration(t *testing.T) {
	for _, test := range []struct {
		name  string
		cfg   *config.Config
		valid bool
	}{{"missing", nil, false}, {"provided", &config.Config{}, true}} {
		t.Run(test.name, func(t *testing.T) {
			cache, err := NewAlertManagerConfigCache(test.cfg, zap.NewNop(), nil, nil, nil, nil, nil)
			if (err == nil) != test.valid {
				t.Fatalf("unexpected constructor error %v", err)
			}
			if !test.valid && cache != nil {
				t.Fatal("failed constructor returned usable cache")
			}
		})
	}
}

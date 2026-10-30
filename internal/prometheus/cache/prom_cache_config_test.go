package cache

import (
	"context"
	"net/url"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	httpSD "github.com/prometheus/prometheus/discovery/http"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func TestPrometheusCacheUsesInjectedDiscoveryEndpoint(t *testing.T) {
	for _, test := range []struct{ name, global string }{
		{"conflicting global endpoint", "http://global-wrong.invalid/discovery"},
		{"empty global endpoint", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("prometheus.httpSdAPI", test.global)
			cfg := &config.Config{Prometheus: config.PrometheusConfig{HttpSdAPI: "https://typed-discovery.invalid/api/targets?existing=kept&port=1&tree_node_ids=old"}}
			job := &model.MonitorScrapeJob{Name: "typed-http", ServiceDiscoveryType: model.ServiceDiscoveryTypeHttp, Scheme: "https", MetricsPath: "/metrics", ScrapeInterval: 30, ScrapeTimeout: 10, RefreshInterval: 20, Port: 9100, TreeNodeIDs: model.StringList{"7", "8"}}
			cache, err := NewPrometheusConfigCache(cfg, zap.NewNop(), nil, scrapeJobsStub{jobs: []*model.MonitorScrapeJob{job}}, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			for generation := 0; generation < 2; generation++ {
				if generation == 1 {
					viper.Set("prometheus.httpSdAPI", "http://later-global.invalid/targets")
					cfg.Prometheus.HttpSdAPI = "http://later-typed.invalid/targets"
				}
				scrapes := cache.GenerateScrapeConfigs(context.Background(), &model.MonitorScrapePool{})
				if len(scrapes) != 1 || len(scrapes[0].ServiceDiscoveryConfigs) != 1 {
					t.Fatal("typed endpoint did not produce the HTTP discovery job")
				}
				discovery, ok := scrapes[0].ServiceDiscoveryConfigs[0].(*httpSD.SDConfig)
				if !ok {
					t.Fatal("HTTP discovery type was lost")
				}
				endpoint, err := url.Parse(discovery.URL)
				if err != nil {
					t.Fatal(err)
				}
				if endpoint.Scheme != "https" || endpoint.Host != "typed-discovery.invalid" || endpoint.Path != "/api/targets" {
					t.Fatal("generated discovery URL did not retain typed endpoint")
				}
				query := endpoint.Query()
				if query.Get("existing") != "kept" || query.Get("port") != "9100" || query.Get("tree_node_ids") != "7,8" {
					t.Fatal("existing query or job-specific query override contract changed")
				}
			}
		})
	}
}

func TestPrometheusCacheKeepsInvalidDiscoveryBehavior(t *testing.T) {
	for _, test := range []struct{ name, endpoint string }{{"empty", ""}, {"missing scheme", "discovery.invalid/targets"}, {"unsupported scheme", "ftp://discovery.invalid/targets"}} {
		t.Run(test.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("prometheus.httpSdAPI", "https://global-valid.invalid/targets")
			cfg := &config.Config{Prometheus: config.PrometheusConfig{HttpSdAPI: test.endpoint}}
			job := &model.MonitorScrapeJob{Name: "invalid-endpoint", ServiceDiscoveryType: model.ServiceDiscoveryTypeHttp}
			cache, err := NewPrometheusConfigCache(cfg, zap.NewNop(), nil, scrapeJobsStub{jobs: []*model.MonitorScrapeJob{job}}, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if scrapes := cache.GenerateScrapeConfigs(context.Background(), &model.MonitorScrapePool{}); len(scrapes) != 0 {
				t.Fatal("invalid typed discovery endpoint was silently replaced by a global endpoint")
			}
		})
	}
}

func TestPrometheusCacheRequiresConfiguration(t *testing.T) {
	for _, test := range []struct {
		name  string
		cfg   *config.Config
		valid bool
	}{{"missing", nil, false}, {"provided", &config.Config{}, true}} {
		t.Run(test.name, func(t *testing.T) {
			cache, err := NewPrometheusConfigCache(test.cfg, zap.NewNop(), nil, nil, nil, nil, nil)
			if (err == nil) != test.valid {
				t.Fatalf("constructor result differs: %v", err)
			}
			if !test.valid && cache != nil {
				t.Fatal("failed constructor returned usable cache")
			}
		})
	}
}

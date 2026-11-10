package cache

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pcc "github.com/prometheus/common/config"
	pc "github.com/prometheus/prometheus/config"
	httpSD "github.com/prometheus/prometheus/discovery/http"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func TestGeneratedHTTPDiscoveryCarriesServiceToken(t *testing.T) {
	const discoveryToken = "fixture-http-sd-token"
	const targetToken = "fixture-target-token"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+discoveryToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	job := &model.MonitorScrapeJob{
		Name: "http-sd", ServiceDiscoveryType: model.ServiceDiscoveryTypeHttp,
		Scheme: "http", MetricsPath: "/metrics", ScrapeInterval: 30, ScrapeTimeout: 10, RefreshInterval: 30,
		Port: 9100, TreeNodeIDs: model.StringList{"7", "8"}, BearerToken: targetToken,
	}
	generator := &prometheusConfigCache{
		logger: zap.NewNop(), httpSdAPI: "http://backend:8889/api/not_auth/getTreeNodeBindIps?existing=kept",
		httpSdToken: discoveryToken, scrapeJobDAO: scrapeJobsStub{jobs: []*model.MonitorScrapeJob{job}},
	}
	pool := &model.MonitorScrapePool{ScrapeInterval: 30, ScrapeTimeout: 10}
	baseConfig, err := generator.CreateBaseConfig(pool)
	if err != nil {
		t.Fatal(err)
	}
	baseConfig.ScrapeConfigs = generator.GenerateScrapeConfigs(context.Background(), pool)
	if len(baseConfig.ScrapeConfigs) != 1 {
		t.Fatal("HTTP SD job missing")
	}
	plain, err := yaml.Marshal(baseConfig.ScrapeConfigs[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(plain), discoveryToken) || strings.Contains(string(plain), targetToken) {
		t.Fatal("plain SDK marshal revealed a credential")
	}
	data, err := marshalPrometheusConfig(baseConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), discoveryToken) || !strings.Contains(string(data), targetToken) {
		t.Fatal("operational export dropped a required credential")
	}
	loaded, err := pc.Load(string(data), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	scrape := loaded.ScrapeConfigs[0]
	if scrape.HTTPClientConfig.Authorization == nil || string(scrape.HTTPClientConfig.Authorization.Credentials) != targetToken {
		t.Fatal("target token was replaced by the HTTP SD token")
	}
	sd, ok := scrape.ServiceDiscoveryConfigs[0].(*httpSD.SDConfig)
	if !ok || sd.HTTPClientConfig.Authorization == nil || string(sd.HTTPClientConfig.Authorization.Credentials) != discoveryToken {
		t.Fatal("HTTP SD authorization did not survive the Prometheus parser")
	}
	if sd.URL == "" || strings.Contains(sd.URL, discoveryToken) || strings.Contains(sd.URL, targetToken) {
		t.Fatal("HTTP SD URL carried a credential")
	}
	client, err := pcc.NewClientFromConfig(sd.HTTPClientConfig, "http-sd-token")
	if err != nil {
		t.Fatal(err)
	}
	defer client.CloseIdleConnections()
	response, err := client.Get(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatal("generated HTTP SD client did not present its bearer token")
	}
}

func TestHTTPServiceDiscoveryTokenIsSnapshotted(t *testing.T) {
	cfg := &config.Config{Prometheus: config.PrometheusConfig{
		HttpSdAPI:   "http://backend:8889/api/not_auth/getTreeNodeBindIps",
		HttpSdToken: "fixture-http-sd-token",
	}}
	job := &model.MonitorScrapeJob{
		Name: "http-sd", ServiceDiscoveryType: model.ServiceDiscoveryTypeHttp,
		Scheme: "http", MetricsPath: "/metrics", ScrapeInterval: 30, ScrapeTimeout: 10, RefreshInterval: 30,
		Port: 9100, TreeNodeIDs: model.StringList{"7"},
	}
	cache, err := NewPrometheusConfigCache(cfg, zap.NewNop(), nil, scrapeJobsStub{jobs: []*model.MonitorScrapeJob{job}}, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Prometheus.HttpSdToken = "fixture-later-token"
	cfg.Prometheus.HttpSdAPI = "http://later.invalid/targets"
	scrapes := cache.GenerateScrapeConfigs(context.Background(), &model.MonitorScrapePool{})
	if len(scrapes) != 1 {
		t.Fatal("snapshot lost the HTTP SD job")
	}
	sd := scrapes[0].ServiceDiscoveryConfigs[0].(*httpSD.SDConfig)
	if string(sd.HTTPClientConfig.Authorization.Credentials) != "fixture-http-sd-token" || strings.Contains(sd.URL, "later.invalid") {
		t.Fatal("HTTP SD token or URL followed a later config change")
	}
	if scrapes[0].HTTPClientConfig.Authorization != nil {
		t.Fatal("HTTP SD token leaked into the target client")
	}
}

func TestUnusableHTTPServiceDiscoveryTokenIsNotExported(t *testing.T) {
	for _, token := range []string{"", "fixture token", "fixture\nhttp"} {
		cfg := &config.Config{Prometheus: config.PrometheusConfig{
			HttpSdAPI: "http://backend:8889/api/not_auth/getTreeNodeBindIps", HttpSdToken: token,
		}}
		job := &model.MonitorScrapeJob{
			Name: "http-sd", ServiceDiscoveryType: model.ServiceDiscoveryTypeHttp,
			Scheme: "http", MetricsPath: "/metrics", ScrapeInterval: 30, ScrapeTimeout: 10, RefreshInterval: 30,
			Port: 9100, TreeNodeIDs: model.StringList{"7"},
		}
		cache, err := NewPrometheusConfigCache(cfg, zap.NewNop(), nil, scrapeJobsStub{jobs: []*model.MonitorScrapeJob{job}}, nil, nil, nil, nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		scrapes := cache.GenerateScrapeConfigs(context.Background(), &model.MonitorScrapePool{})
		sd := scrapes[0].ServiceDiscoveryConfigs[0].(*httpSD.SDConfig)
		if sd.HTTPClientConfig.Authorization != nil || strings.Contains(sd.URL, "fixture") {
			t.Fatal("unusable token was written into HTTP SD config")
		}
	}
}

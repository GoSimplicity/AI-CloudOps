package cache

import (
	"context"
	"encoding/pem"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pcc "github.com/prometheus/common/config"
	pc "github.com/prometheus/prometheus/config"
	httpSD "github.com/prometheus/prometheus/discovery/http"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func TestGeneratedConfigRoundTripAuthenticatesWithTrustedCA(t *testing.T) {
	const token = "fixture-token-value"
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	ca := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: upstream.Certificate().Raw}))
	directory := t.TempDir()
	tokenPath, caPath := filepath.Join(directory, "token"), filepath.Join(directory, "ca.pem")
	if err := os.WriteFile(tokenPath, []byte(token), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caPath, []byte(ca), 0600); err != nil {
		t.Fatal(err)
	}
	for _, discoveryType := range []model.ServiceDiscoveryType{model.ServiceDiscoveryTypeStatic, model.ServiceDiscoveryTypeHttp, model.ServiceDiscoveryTypeK8s} {
		for _, files := range []bool{false, true} {
			name := "inline"
			if files {
				name = "file"
			}
			t.Run(string(rune('0'+discoveryType))+"/"+name, func(t *testing.T) {
				job := &model.MonitorScrapeJob{Name: "fixture", ServiceDiscoveryType: discoveryType, Scheme: "https", MetricsPath: "/metrics", ScrapeInterval: 30, ScrapeTimeout: 10, RefreshInterval: 30, KubernetesSdRole: "pod", IpAddress: strings.TrimPrefix(upstream.URL, "https://"), Port: 9100, TreeNodeIDs: model.StringList{"7", "8"}}
				if files {
					job.BearerTokenFile = tokenPath
					job.TlsCaFilePath = caPath
				} else {
					job.BearerToken = token
					job.TlsCaContent = ca
				}
				cache := &prometheusConfigCache{logger: zap.NewNop(), httpSdAPI: "http://backend:8889/api/not_auth/getTreeNodeBindIps?existing=kept", scrapeJobDAO: scrapeJobsStub{jobs: []*model.MonitorScrapeJob{job}}}
				scrapes := cache.GenerateScrapeConfigs(context.Background(), &model.MonitorScrapePool{})
				if len(scrapes) != 1 {
					t.Fatal("job missing")
				}
				config, err := cache.CreateBaseConfig(&model.MonitorScrapePool{ScrapeInterval: 30, ScrapeTimeout: 10})
				if err != nil {
					t.Fatal(err)
				}
				config.ScrapeConfigs = cache.ApplyHashMod(scrapes, 2, 0)
				data, err := marshalPrometheusConfig(config)
				if err != nil {
					t.Fatal(err)
				}
				loaded, err := pc.Load(string(data), slog.New(slog.NewTextHandler(io.Discard, nil)))
				if err != nil {
					t.Fatal("Prometheus rejected generated configuration", err)
				}
				client, err := pcc.NewClientFromConfig(loaded.ScrapeConfigs[0].HTTPClientConfig, "fixture")
				if err != nil {
					t.Fatal(err)
				}
				defer client.CloseIdleConnections()
				response, err := client.Get(upstream.URL)
				if err != nil {
					t.Fatal("trusted target request failed", err)
				}
				response.Body.Close()
				if response.StatusCode != http.StatusOK {
					t.Fatal("target authentication did not survive operational YAML roundtrip")
				}
				ordinary, err := yaml.Marshal(scrapes[0])
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(ordinary), token) {
					t.Fatal("operational export disabled normal SDK secret redaction")
				}
				if discoveryType == model.ServiceDiscoveryTypeHttp {
					sd := loaded.ScrapeConfigs[0].ServiceDiscoveryConfigs[0].(*httpSD.SDConfig)
					parsed, _ := url.Parse(sd.URL)
					if parsed.Query().Get("existing") != "kept" || parsed.Query().Get("port") != "9100" || parsed.Query().Get("tree_node_ids") != "7,8" {
						t.Fatal("HTTP SD query not preserved")
					}
					if sd.HTTPClientConfig.Authorization != nil {
						t.Fatal("target token leaked into service discovery identity")
					}
				}
			})
		}
	}
}

func TestConflictingScrapeCredentialsDoNotGenerateUnauthenticatedJob(t *testing.T) {
	for _, job := range []*model.MonitorScrapeJob{
		{BearerToken: "fixture", BearerTokenFile: "/fixture"}, {TlsCaContent: "fixture", TlsCaFilePath: "/fixture"},
	} {
		job.Name = "invalid"
		job.ServiceDiscoveryType = model.ServiceDiscoveryTypeStatic
		cache := &prometheusConfigCache{logger: zap.NewNop(), scrapeJobDAO: scrapeJobsStub{jobs: []*model.MonitorScrapeJob{job}}}
		if len(cache.GenerateScrapeConfigs(context.Background(), &model.MonitorScrapePool{})) != 0 {
			t.Fatal("conflicting credential sources accepted")
		}
	}
}

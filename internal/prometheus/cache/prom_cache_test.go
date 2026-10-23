package cache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	scrapeDAO "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	pcc "github.com/prometheus/common/config"
	"github.com/prometheus/prometheus/discovery/kubernetes"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type scrapeJobsStub struct {
	scrapeDAO.ScrapeJobDAO
	jobs []*model.MonitorScrapeJob
}

func (s scrapeJobsStub) GetMonitorScrapeJobsByPoolId(context.Context, int) ([]*model.MonitorScrapeJob, error) {
	return s.jobs, nil
}

func TestKubernetesDiscoveryConfigCanBeLoaded(t *testing.T) {
	for _, kubeconfig := range []string{"", "/etc/prometheus/kubeconfig"} {
		t.Run(kubeconfig, func(t *testing.T) {
			cache := &prometheusConfigCache{logger: zap.NewNop(), scrapeJobDAO: scrapeJobsStub{jobs: []*model.MonitorScrapeJob{{
				Name: "pods", ServiceDiscoveryType: model.ServiceDiscoveryTypeK8s,
				KubernetesSdRole: "pod", KubeConfigFilePath: kubeconfig,
				BearerTokenFile: "/etc/prometheus/target-token", TlsCaFilePath: "/etc/prometheus/target-ca.pem",
			}}}}
			configs := cache.GenerateScrapeConfigs(context.Background(), &model.MonitorScrapePool{})
			if len(configs) != 1 {
				t.Fatalf("expected one job, got %d", len(configs))
			}
			scrape := configs[0]
			if scrape.HTTPClientConfig.TLSConfig.InsecureSkipVerify {
				t.Fatal("target TLS verification disabled")
			}
			if scrape.HTTPClientConfig.TLSConfig.CAFile != "/etc/prometheus/target-ca.pem" {
				t.Fatal("target CA lost")
			}
			sd := scrape.ServiceDiscoveryConfigs[0].(*kubernetes.SDConfig)
			content, err := yaml.Marshal(sd)
			if err != nil {
				t.Fatal(err)
			}
			var loaded kubernetes.SDConfig
			if err := yaml.Unmarshal(content, &loaded); err != nil {
				t.Fatalf("Prometheus rejects generated discovery: %v", err)
			}
			if loaded.KubeConfig != kubeconfig {
				t.Fatal("kubeconfig lost")
			}
		})
	}
}

func TestKubernetesScrapeRejectsUntrustedCertificate(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer upstream.Close()
	cache := &prometheusConfigCache{logger: zap.NewNop(), scrapeJobDAO: scrapeJobsStub{jobs: []*model.MonitorScrapeJob{{Name: "pods", ServiceDiscoveryType: model.ServiceDiscoveryTypeK8s}}}}
	configs := cache.GenerateScrapeConfigs(context.Background(), &model.MonitorScrapePool{})
	client, err := pcc.NewClientFromConfig(configs[0].HTTPClientConfig, "tls-regression")
	if err != nil {
		t.Fatal(err)
	}
	defer client.CloseIdleConnections()
	response, err := client.Get(upstream.URL)
	if response != nil {
		response.Body.Close()
	}
	if err == nil {
		t.Fatal("scrape accepted an untrusted server certificate")
	}
}

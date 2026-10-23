//go:build smoke

package cache

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

// TestGeneratedConfigLoadsInReleasePrometheus 只创建无网络临时容器，不访问运行中的编排
func TestGeneratedConfigLoadsInReleasePrometheus(t *testing.T) {
	image := os.Getenv("CLOUDOPS_PROMTOOL_IMAGE")
	if image == "" {
		t.Skip("set CLOUDOPS_PROMTOOL_IMAGE to explicitly check the release image")
	}
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer upstream.Close()
	ca := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: upstream.Certificate().Raw}))
	jobs := []*model.MonitorScrapeJob{}
	for i, kind := range []model.ServiceDiscoveryType{model.ServiceDiscoveryTypeStatic, model.ServiceDiscoveryTypeHttp, model.ServiceDiscoveryTypeK8s} {
		jobs = append(jobs, &model.MonitorScrapeJob{Name: []string{"static", "http", "k8s"}[i], ServiceDiscoveryType: kind, Scheme: "https", MetricsPath: "/metrics", ScrapeInterval: 30, ScrapeTimeout: 10, RefreshInterval: 30, KubernetesSdRole: "pod", IpAddress: "127.0.0.1:9100", Port: 9100, TreeNodeIDs: model.StringList{"7"}, BearerToken: "synthetic-release-token", TlsCaContent: ca})
	}
	generator := &prometheusConfigCache{logger: zap.NewNop(), httpSdAPI: "http://backend:8889/api/not_auth/getTreeNodeBindIps", scrapeJobDAO: scrapeJobsStub{jobs: jobs}}
	config, err := generator.CreateBaseConfig(&model.MonitorScrapePool{ScrapeInterval: 30, ScrapeTimeout: 10})
	if err != nil {
		t.Fatal(err)
	}
	config.ScrapeConfigs = generator.GenerateScrapeConfigs(context.Background(), &model.MonitorScrapePool{})
	data, err := marshalPrometheusConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err := os.Chmod(directory, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "prometheus.yml"), data, 0644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "docker", "run", "--rm", "--network=none", "--read-only", "--entrypoint=/bin/promtool", "-v", directory+":/fixture:ro", image, "check", "config", "/fixture/prometheus.yml")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("release promtool rejected generated configuration: %v: %s", err, strings.ReplaceAll(string(output), "synthetic-release-token", "[redacted]"))
	}
}

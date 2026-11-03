package cache

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pm "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"go.uber.org/zap"
)

func TestStaticTargetsHonorSeparatePortAndLegacyAddress(t *testing.T) {
	for _, tc := range []struct {
		name, address, want string
		port                int
	}{
		{"bare-ip", "127.0.0.1", "127.0.0.1:9100", 9100},
		{"bare-host", "metrics.internal", "metrics.internal:9200", 9200},
		{"legacy-explicit-port", "127.0.0.1:53047", "127.0.0.1:53047", 9100},
		{"legacy-port-precedes-unused-field", "metrics.internal:53047", "metrics.internal:53047", -1},
		{"ipv6", "2001:db8::1", "[2001:db8::1]:9100", 9100},
		{"bracketed-ipv6", "[2001:db8::1]", "[2001:db8::1]:9100", 9100},
		{"legacy-ipv6-port", "[2001:db8::1]:53047", "[2001:db8::1]:53047", 9100},
		{"zero-keeps-protocol-default", "metrics.internal", "metrics.internal", 0},
		{"zero-ipv6", "2001:db8::1", "[2001:db8::1]", 0},
		{"zero-bracketed-ipv6", "[2001:db8::1]", "[2001:db8::1]", 0},
		{"legacy-explicit-zero", "metrics.internal:0", "metrics.internal:0", 9100},
		{"invalid-negative-port", "127.0.0.1", "", -1},
		{"invalid-large-port", "127.0.0.1", "", 65536},
		{"missing-host", "", "", 9100},
		{"empty-bracketed-host", "[]", "", 9100},
		{"whitespace-host", " 127.0.0.1", "", 9100},
		{"invalid-ip-literal", "[not-an-ip]", "", 9100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			job := &model.MonitorScrapeJob{
				Name: "fixture", ServiceDiscoveryType: model.ServiceDiscoveryTypeStatic,
				IpAddress: tc.address, Port: tc.port,
				Scheme: "http", MetricsPath: "/metrics", ScrapeInterval: 30, ScrapeTimeout: 10,
			}
			cache := &prometheusConfigCache{logger: zap.NewNop(), scrapeJobDAO: scrapeJobsStub{jobs: []*model.MonitorScrapeJob{job}}}
			configs := cache.GenerateScrapeConfigs(context.Background(), &model.MonitorScrapePool{})
			if tc.want == "" {
				if len(configs) != 0 {
					t.Fatal("invalid separate port must not publish an unusable static target")
				}
				return
			}
			if len(configs) != 1 || len(configs[0].ServiceDiscoveryConfigs) != 1 {
				t.Fatal("valid static job must produce one discovery configuration")
			}
			groups, ok := configs[0].ServiceDiscoveryConfigs[0].(discovery.StaticConfig)
			if !ok || len(groups) != 1 || len(groups[0].Targets) != 1 {
				t.Fatal("valid static job must produce one target")
			}
			if got := string(groups[0].Targets[0][pm.AddressLabel]); got != tc.want {
				t.Fatalf("generated address=%q, expected %q", got, tc.want)
			}
		})
	}
}

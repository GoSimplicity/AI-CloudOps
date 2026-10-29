package app

import (
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
)

func TestPredictionBacktestHasExplicitPostPermission(t *testing.T) {
	routes, err := parseAIOpsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	permission.SetProxyCatalog(permission.AIOpsProxyTemplate, routes)
	t.Cleanup(func() { permission.SetProxyCatalog(permission.AIOpsProxyTemplate, nil) })
	const path = "/api/v1/predict/backtest"
	if got := permission.ResolveProxyTemplate("POST", permission.AIOpsProxyTemplate, path); got != path {
		t.Fatalf("POST backtest permission resolved to %q", got)
	}
	if got := permission.ResolveProxyTemplate("GET", permission.AIOpsProxyTemplate, path); got == path {
		t.Fatal("POST grant must not authorize GET")
	}
}

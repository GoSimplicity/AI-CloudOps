package app

import (
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
)

func TestAutofixHistoryRoutesHaveSeparateReadPermissions(t *testing.T) {
	routes, err := parseAIOpsCatalog()
	if err != nil {
		t.Fatal(err)
	}
	permission.SetProxyCatalog(permission.AIOpsProxyTemplate, routes)
	t.Cleanup(func() { permission.SetProxyCatalog(permission.AIOpsProxyTemplate, nil) })
	for path, want := range map[string]string{
		"/api/v1/autofix/workflow/history":       "/api/v1/autofix/workflow/history",
		"/api/v1/autofix/workflow/operations/id": "/api/v1/autofix/workflow/operations/:operation_id",
		"/api/v1/autofix/workflow/plans/id":      "/api/v1/autofix/workflow/plans/:plan_id",
	} {
		if got := permission.ResolveProxyTemplate("GET", permission.AIOpsProxyTemplate, path); got != want {
			t.Errorf("GET %s resolved to %q; want %q", path, got, want)
		}
		if got := permission.ResolveProxyTemplate("POST", permission.AIOpsProxyTemplate, path); got == want {
			t.Errorf("read permission for %s cannot authorize POST", path)
		}
	}
}

package permission

import (
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/gin-gonic/gin"
)

func TestCoversComparesTemplatesNotRequestPaths(t *testing.T) {
	for _, tt := range []struct {
		granted, template string
		want              bool
	}{
		{"/api/tree/cloud/:id/status", "/api/tree/cloud/:id/status", true},
		{"/api/tree/cloud/:id/status", "/api/tree/cloud/batch/status", false},
		{"/api/tree/cloud/:id/status", "/api/tree/cloud/9/status", false},
		{"/api/k8s/clusters/:cluster_id/events/:namespace/:name", "/api/k8s/clusters/:cluster_id/events/nodes/:node_name", false},
		{"/*", "/api/anything/:id", true},
		{"*", "/api/anything/:id", true}, // 与改动前一致：单个 * 视为全局前缀
		{"/api/tree/*", "/api/tree/cloud/batch/status", true},
		{"/api/tree/*", "/api/treex/list", false},
		{"/api/tree*", "/api/treex/list", true}, // 前缀按字符串比较，与改动前一致
		{"/api/tree/cloud/:id/*", "/api/tree/cloud/:id/status", true},
		{"/api/tree/cloud/:id/*", "/api/tree/cloud/batch/status", false},
		{"/api/v1/*path", "/api/v1/*path", true},
		{"/api/v1/*", "/api/v1/*path", true},
		{"*/status", "/api/tree/cloud/:id/status", false},
		{"/api/*/status", "/api/tree/cloud/:id/status", false},
		{"/api/**", "/api/tree/list", false},
		{"/api/*/*", "/api/tree/list", false},
		{"", "/api/tree/list", false},
		{"/*", "", false},
	} {
		if got := Covers(tt.granted, tt.template); got != tt.want {
			t.Errorf("Covers(%q, %q)=%v want %v", tt.granted, tt.template, got, tt.want)
		}
	}
}

func TestAllowedRequiresSameMethodAndIgnoresDisabledRoles(t *testing.T) {
	grants := GrantsFromRoles([]*model.Role{
		nil,
		{Status: 1, Apis: []*model.Api{nil, {Method: 1, Path: "/api/x/:id"}}},
		{Status: 0, Apis: []*model.Api{{Method: 3, Path: "/*"}}},
	})
	if len(grants) != 1 {
		t.Fatalf("grants=%v", grants)
	}
	if !Allowed(grants, 1, "/api/x/:id") || Allowed(grants, 3, "/api/x/:id") || Allowed(grants, 1, "", "") {
		t.Fatal("method/empty-template handling broken")
	}
	if !Allowed(grants, 1, "/api/v1/*path", "/api/x/:id") {
		t.Fatal("any candidate template should be enough")
	}
}

func TestMethodCodes(t *testing.T) {
	for name, code := range map[string]int8{"GET": 1, "POST": 2, "PUT": 3, "DELETE": 4} {
		if got, ok := MethodCode(name); !ok || got != code || MethodName(code) != name {
			t.Fatalf("%s", name)
		}
	}
	if _, ok := MethodCode("PATCH"); ok {
		t.Fatal("PATCH has no grant code")
	}
	if MethodName(9) != "" {
		t.Fatal("unknown code")
	}
}

func withProxyCatalog(t *testing.T, routes []Route) {
	t.Helper()
	previous := ProxyCatalog(AIOpsProxyTemplate)
	SetProxyCatalog(AIOpsProxyTemplate, routes)
	t.Cleanup(func() { SetProxyCatalog(AIOpsProxyTemplate, previous) })
}

func TestResolveProxyTemplatePicksMostSpecific(t *testing.T) {
	withProxyCatalog(t, []Route{
		{"GET", "/api/v1/report/:id"},
		{"GET", "/api/v1/report/latest"},
		{"GET", "/api/v1/:kind/latest"},
		{"GET", "/api/v1/report/:id/markdown"},
		{"POST", "/api/v1/autofix"},
		{"POST", "/api/v1/autofix/"},
	})
	for _, tt := range []struct {
		method, fullPath, path, want string
	}{
		{"GET", AIOpsProxyTemplate, "/api/v1/report/r1", "/api/v1/report/:id"},
		{"GET", AIOpsProxyTemplate, "/api/v1/report/latest", "/api/v1/report/latest"},
		{"GET", AIOpsProxyTemplate, "/api/v1/task/latest", "/api/v1/:kind/latest"},
		{"GET", AIOpsProxyTemplate, "/api/v1/report/r1/markdown", "/api/v1/report/:id/markdown"},
		{"GET", AIOpsProxyTemplate, "/api/v1/report/", ""},
		{"GET", AIOpsProxyTemplate, "/api/v1/report/..", ""},
		{"GET", AIOpsProxyTemplate, "/api/v1/report/.", ""},
		{"GET", AIOpsProxyTemplate, "/api/v1/report/r1/extra", ""},
		{"POST", AIOpsProxyTemplate, "/api/v1/report/r1", ""},
		{"POST", AIOpsProxyTemplate, "/api/v1/autofix", "/api/v1/autofix"},
		{"POST", AIOpsProxyTemplate, "/api/v1/autofix/", "/api/v1/autofix/"},
		{"GET", "/api/other/*path", "/api/v1/report/r1", ""},
		{"GET", "", "/api/v1/report/r1", ""},
	} {
		if got := ResolveProxyTemplate(tt.method, tt.fullPath, tt.path); got != tt.want {
			t.Errorf("Resolve(%s %s)=%q want %q", tt.method, tt.path, got, tt.want)
		}
	}
}

func TestProxyCatalogIsCopiedAndClearable(t *testing.T) {
	routes := []Route{{"GET", "/api/v1/a"}}
	withProxyCatalog(t, routes)
	routes[0].Path = "/api/v1/mutated"
	got := ProxyCatalog(AIOpsProxyTemplate)
	got[0].Path = "/api/v1/mutated-again"
	if ProxyCatalog(AIOpsProxyTemplate)[0].Path != "/api/v1/a" {
		t.Fatal("catalog must be copied on set and get")
	}
	SetProxyCatalog(AIOpsProxyTemplate, nil)
	if len(ProxyCatalog(AIOpsProxyTemplate)) != 0 || ResolveProxyTemplate("GET", AIOpsProxyTemplate, "/api/v1/a") != "" {
		t.Fatal("empty catalog should clear registration")
	}
}

func TestBuildCatalogAndExpand(t *testing.T) {
	withProxyCatalog(t, []Route{{"GET", "/api/v1/a"}, {"POST", "/api/v1/b"}, {"PATCH", "/api/v1/c"}})
	routes := gin.RoutesInfo{
		{Method: "GET", Path: "/api/item/:id"},
		{Method: "GET", Path: "/api/item/:id"}, // 重复
		{Method: "PATCH", Path: "/api/item/:id"},
		{Method: "POST", Path: "/api/user/signup"},
		{Method: "GET", Path: "/api/department/list"},
		{Method: "GET", Path: "/api/user/codes"},
		{Method: "GET", Path: "/api/user/permissions"},
		{Method: "GET", Path: "/health"},
		{Method: "GET", Path: AIOpsProxyTemplate}, // 代理只注册了 GET
	}
	var codes []string
	for _, entry := range BuildCatalog(routes) {
		codes = append(codes, entry.Code()+"|"+entry.Via)
	}
	want := "GET /api/item/:id|,GET /api/v1/*path|,GET /api/v1/a|/api/v1/*path"
	if strings.Join(codes, ",") != want {
		t.Fatalf("catalog=%v", codes)
	}

	catalog := BuildCatalog(routes)
	if got := Expand(nil, catalog); got == nil || len(got) != 0 {
		t.Fatalf("empty grants must expand to [], got %#v", got)
	}
	if got := strings.Join(Expand([]Grant{{1, AIOpsProxyTemplate}}, catalog), ","); got != "GET /api/v1/*path,GET /api/v1/a" {
		t.Fatalf("whole proxy: %s", got)
	}
	if got := strings.Join(Expand([]Grant{{1, "/api/v1/a"}, {1, "/api/v1/a"}, {1, "/api/item/:id"}}, catalog), ","); got != "GET /api/item/:id,GET /api/v1/a" {
		t.Fatalf("exact: %s", got)
	}
	if got := Expand([]Grant{{2, "/*"}}, catalog); len(got) != 0 {
		t.Fatalf("super-only and unregistered-method entries must not appear: %v", got)
	}
}

func TestRequiresSuperAdminAndSkips(t *testing.T) {
	if !RequiresSuperAdmin("POST", "/api/user/signup") || RequiresSuperAdmin("GET", "/api/user/signup") {
		t.Fatal("signup")
	}
	if !RequiresSuperAdmin("GET", "/api/department") || !RequiresSuperAdmin("DELETE", "/api/department/:id") || RequiresSuperAdmin("GET", "/api/departments") {
		t.Fatal("department")
	}
	if !SkipsAuthorization("/api/user/permissions") || SkipsAuthorization("/api/user/list") {
		t.Fatal("skip")
	}
}

package permission

import (
	"reflect"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/gin-gonic/gin"
)

func TestSensitivePermissionsRequireExactMethodAndTemplate(t *testing.T) {
	for _, endpoint := range []CatalogEntry{
		{Method: 1, Path: model.K8sSecretRevealRoute},
		{Method: 1, Path: model.K8sServiceAccountTokenRoute},
		{Method: 2, Path: model.K8sServiceAccountTokenRoute},
		{Method: 2, Path: model.WorkorderNotificationCancelScheduledRoute},
	} {
		t.Run(endpoint.Code(), func(t *testing.T) {
			if !Allowed([]Grant{{Method: endpoint.Method, Path: endpoint.Path}}, endpoint.Method, endpoint.Path) {
				t.Fatal("exact method and template must authorize the sensitive endpoint")
			}
			for _, path := range []string{"*", "/*", "/api/*", "/api/k8s/*", "/api/workorder/*", "/api/workorder/notification/*"} {
				if Covers(path, endpoint.Path) || Allowed([]Grant{{Method: endpoint.Method, Path: path}}, endpoint.Method, endpoint.Path) {
					t.Errorf("wildcard %q authorized sensitive endpoint", path)
				}
			}
			otherMethod := int8(1)
			if endpoint.Method == otherMethod {
				otherMethod = 2
			}
			if Allowed([]Grant{{Method: otherMethod, Path: endpoint.Path}}, endpoint.Method, endpoint.Path) {
				t.Fatal("exact template with a different method must not authorize the endpoint")
			}
		})
	}
}

func TestExpandKeepsSensitivePermissionsExplicit(t *testing.T) {
	catalog := BuildCatalog(gin.RoutesInfo{
		{Method: "GET", Path: model.K8sSecretRevealRoute},
		{Method: "GET", Path: model.K8sServiceAccountTokenRoute},
		{Method: "POST", Path: model.K8sServiceAccountTokenRoute},
		{Method: "GET", Path: "/api/k8s/secret/:cluster_id/:namespace/:name/detail"},
	})
	for _, wildcard := range []string{"*", "/*", "/api/*", "/api/k8s/*"} {
		t.Run(wildcard, func(t *testing.T) {
			grants := []Grant{{Method: 1, Path: wildcard}, {Method: 2, Path: wildcard}}
			want := []string{"GET /api/k8s/secret/:cluster_id/:namespace/:name/detail"}
			if got := Expand(grants, catalog); !reflect.DeepEqual(got, want) {
				t.Fatalf("wildcard expansion=%v want=%v", got, want)
			}
		})
	}
	for _, endpoint := range catalog {
		if endpoint.Path != model.K8sSecretRevealRoute && endpoint.Path != model.K8sServiceAccountTokenRoute {
			continue
		}
		t.Run(endpoint.Code(), func(t *testing.T) {
			grants := []Grant{{Method: endpoint.Method, Path: endpoint.Path}}
			want := []string{endpoint.Code()}
			if got := Expand(grants, catalog); !reflect.DeepEqual(got, want) {
				t.Fatalf("exact grant expansion=%v want=%v", got, want)
			}
		})
	}
}

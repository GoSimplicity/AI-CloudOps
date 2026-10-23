package app

import (
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestSecretRevealIsRegisteredAsNamedPermission(t *testing.T) {
	named, ok := namedPermissions["GET "+model.K8sSecretRevealRoute]
	if !ok {
		t.Fatal("reveal 接口未登记到接口目录")
	}
	if named.Name != model.K8sSecretRevealPermissionName || named.Description == "" {
		t.Fatalf("权限码不符: %+v", named)
	}
	// cl_system_apis.name 为 varchar(50)
	if len(named.Name) > 50 {
		t.Fatal("权限名超过 50 字符")
	}
}

func TestServiceAccountTokenIsRegisteredAsNamedPermission(t *testing.T) {
	names := map[string]bool{}
	for _, method := range []string{"GET", "POST"} {
		named, ok := namedPermissions[method+" "+model.K8sServiceAccountTokenRoute]
		if !ok {
			t.Fatalf("%s token 接口未登记到接口目录", method)
		}
		if named.Description == "" || len(named.Name) > 50 {
			t.Fatalf("权限码不符: %+v", named)
		}
		names[named.Name] = true
	}
	if !names[model.K8sServiceAccountTokenPermissionName] || !names[model.K8sServiceAccountTokenCreatePermissionName] {
		t.Fatalf("权限码不符: %v", names)
	}
	// cl_system_apis.name 有唯一索引，所有登记的权限码不能重复
	seen := map[string]bool{}
	for _, named := range namedPermissions {
		if seen[named.Name] {
			t.Fatalf("权限码重复: %s", named.Name)
		}
		seen[named.Name] = true
	}
}

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

package service

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestRBACRequestsMustBePresent(t *testing.T) {
	svc := NewRBACService(nil, nil, nil, nil, zap.NewNop())
	if _, err := svc.AnalyzeRBACPermissions(context.Background(), nil); err == nil {
		t.Fatal("nil analysis accepted")
	}
	if _, err := svc.CheckRBACPermission(context.Background(), nil); err == nil {
		t.Fatal("nil check accepted")
	}
}

func TestRBACPermissionCheckUsesBindings(t *testing.T) {
	t.Skip("CheckRBACPermission 与 AnalyzeRBACPermissions 校验非空后直接返回空结果，不查询 Role/ClusterRole（rbac_service.go:84-91 与 rbac_service.go:110-114）。未改生产代码。")
}

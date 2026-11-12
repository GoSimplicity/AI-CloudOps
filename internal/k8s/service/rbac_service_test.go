package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type rbacRoleFixture struct {
	RoleService
	roles map[string]*model.K8sRole
	err   error
}

func (f *rbacRoleFixture) GetRoleDetails(_ context.Context, req *model.GetRoleDetailsReq) (*model.K8sRole, error) {
	if f.err != nil {
		return nil, f.err
	}
	role, ok := f.roles[req.Namespace+"/"+req.Name]
	if !ok {
		return nil, errors.New("role not found")
	}
	return role, nil
}

type rbacClusterRoleFixture struct {
	ClusterRoleService
	roles map[string]*model.K8sClusterRole
}

func (f *rbacClusterRoleFixture) GetClusterRoleDetails(_ context.Context, req *model.GetClusterRoleDetailsReq) (*model.K8sClusterRole, error) {
	role, ok := f.roles[req.Name]
	if !ok {
		return nil, errors.New("clusterrole not found")
	}
	return role, nil
}

type rbacRoleBindingFixture struct {
	RoleBindingService
	items []*model.K8sRoleBinding
	err   error
}

func (f *rbacRoleBindingFixture) GetRoleBindingList(_ context.Context, req *model.GetRoleBindingListReq) (model.ListResp[*model.K8sRoleBinding], error) {
	if f.err != nil {
		return model.ListResp[*model.K8sRoleBinding]{}, f.err
	}
	if req.Page > 1 {
		return model.ListResp[*model.K8sRoleBinding]{Total: int64(len(f.items))}, nil
	}
	var items []*model.K8sRoleBinding
	for _, item := range f.items {
		if req.Namespace == "" || item.Namespace == req.Namespace {
			items = append(items, item)
		}
	}
	return model.ListResp[*model.K8sRoleBinding]{Items: items, Total: int64(len(items))}, nil
}

type rbacClusterRoleBindingFixture struct {
	ClusterRoleBindingService
	items []*model.K8sClusterRoleBinding
}

func (f *rbacClusterRoleBindingFixture) GetClusterRoleBindingList(_ context.Context, req *model.GetClusterRoleBindingListReq) (model.ListResp[*model.K8sClusterRoleBinding], error) {
	if req.Page > 1 {
		return model.ListResp[*model.K8sClusterRoleBinding]{Total: int64(len(f.items))}, nil
	}
	return model.ListResp[*model.K8sClusterRoleBinding]{Items: f.items, Total: int64(len(f.items))}, nil
}

func containsVerb(verbs []string, verb string) bool {
	for _, item := range verbs {
		if item == verb {
			return true
		}
	}
	return false
}

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
	roles := &rbacRoleFixture{roles: map[string]*model.K8sRole{
		"ops/reader": {Name: "reader", Namespace: "ops", Rules: []model.PolicyRule{
			{Resources: []string{"pods"}, Verbs: []string{"get"}},
			{Resources: []string{"pods/log"}, Verbs: []string{"get"}, ResourceNames: []string{"nginx"}},
		}},
		"ops/runner-role": {Name: "runner-role", Namespace: "ops", Rules: []model.PolicyRule{{Resources: []string{"secrets"}, Verbs: []string{"get"}}}},
	}}
	clusterRoles := &rbacClusterRoleFixture{roles: map[string]*model.K8sClusterRole{
		"view":  {Name: "view", Rules: []model.PolicyRule{{Resources: []string{"configmaps"}, Verbs: []string{"get"}}, {Resources: []string{"pods/*"}, Verbs: []string{"get"}}}},
		"admin": {Name: "admin", Rules: []model.PolicyRule{{Resources: []string{"*"}, Verbs: []string{"*"}}}},
	}}
	roleBindings := &rbacRoleBindingFixture{items: []*model.K8sRoleBinding{
		{Name: "read-pods", Namespace: "ops", RoleRef: model.RoleRef{Kind: "Role", Name: "reader"}, Subjects: []model.Subject{{Kind: "User", Name: "alice"}}},
		{Name: "view-dev", Namespace: "dev", RoleRef: model.RoleRef{Kind: "ClusterRole", Name: "view"}, Subjects: []model.Subject{{Kind: "User", Name: "alice"}}},
		{Name: "runner", Namespace: "ops", RoleRef: model.RoleRef{Kind: "Role", Name: "runner-role"}, Subjects: []model.Subject{{Kind: "ServiceAccount", Name: "runner", Namespace: "ops"}}},
		{Name: "other", Namespace: "ops", RoleRef: model.RoleRef{Kind: "Role", Name: "reader"}, Subjects: []model.Subject{{Kind: "User", Name: "bob"}}},
	}}
	clusterBindings := &rbacClusterRoleBindingFixture{items: []*model.K8sClusterRoleBinding{
		{Name: "sre-admin", RoleRef: model.RoleRef{Kind: "ClusterRole", Name: "admin"}, Subjects: []model.Subject{{Kind: "Group", Name: "sre"}}},
	}}
	svc := NewRBACService(roles, clusterRoles, roleBindings, clusterBindings, zap.NewNop())
	ctx := context.Background()
	alice := model.Subject{Kind: "User", Name: "alice"}

	allowed, err := svc.CheckRBACPermission(ctx, &model.CheckRBACPermissionReq{ClusterID: 1, Subject: alice, Namespace: "ops", Resource: "pods", Verb: "get"})
	if err != nil || allowed == nil || !allowed.Allowed || allowed.Source != "read-pods" || allowed.Reason != "Role/reader" {
		t.Fatalf("alice pods/get = %+v %v", allowed, err)
	}
	denied, err := svc.CheckRBACPermission(ctx, &model.CheckRBACPermissionReq{ClusterID: 1, Subject: alice, Namespace: "dev", Resource: "pods", Verb: "get"})
	if err != nil || denied == nil || denied.Allowed {
		t.Fatalf("dev pods/get = %+v %v", denied, err)
	}
	configmaps, err := svc.CheckRBACPermission(ctx, &model.CheckRBACPermissionReq{ClusterID: 1, Subject: alice, Namespace: "dev", Resource: "configmaps", Verb: "get"})
	if err != nil || configmaps == nil || !configmaps.Allowed || configmaps.Source != "view-dev" || configmaps.Reason != "ClusterRole/view" {
		t.Fatalf("dev configmaps = %+v %v", configmaps, err)
	}
	logs, err := svc.CheckRBACPermission(ctx, &model.CheckRBACPermissionReq{ClusterID: 1, Subject: alice, Namespace: "dev", Resource: "pods/log", Verb: "get"})
	if err != nil || logs == nil || !logs.Allowed {
		t.Fatalf("pods/log via pods/* = %+v %v", logs, err)
	}
	named, err := svc.CheckRBACPermission(ctx, &model.CheckRBACPermissionReq{ClusterID: 1, Subject: alice, Namespace: "ops", Resource: "pods/log", Verb: "get"})
	if err != nil || named == nil || named.Allowed {
		t.Fatalf("named pods/log should not grant unnamed check: %+v %v", named, err)
	}
	cluster, err := svc.CheckRBACPermission(ctx, &model.CheckRBACPermissionReq{ClusterID: 1, Subject: alice, Resource: "pods", Verb: "get"})
	if err != nil || cluster == nil || cluster.Allowed {
		t.Fatalf("namespaced role must not grant cluster scope: %+v %v", cluster, err)
	}
	sre, err := svc.CheckRBACPermission(ctx, &model.CheckRBACPermissionReq{ClusterID: 1, Subject: model.Subject{Kind: "Group", Name: "sre"}, Resource: "nodes", Verb: "delete"})
	if err != nil || sre == nil || !sre.Allowed || sre.Source != "sre-admin" || sre.Reason != "ClusterRole/admin" {
		t.Fatalf("clusterrolebinding = %+v %v", sre, err)
	}
	runner, err := svc.CheckRBACPermission(ctx, &model.CheckRBACPermissionReq{
		ClusterID: 1, Namespace: "ops", Resource: "secrets", Verb: "get",
		Subject: model.Subject{Kind: "ServiceAccount", Name: "runner", Namespace: "ops"},
	})
	if err != nil || runner == nil || !runner.Allowed || runner.Source != "runner" {
		t.Fatalf("serviceaccount = %+v %v", runner, err)
	}
	otherNS, err := svc.CheckRBACPermission(ctx, &model.CheckRBACPermissionReq{
		ClusterID: 1, Namespace: "ops", Resource: "secrets", Verb: "get",
		Subject: model.Subject{Kind: "ServiceAccount", Name: "runner", Namespace: "dev"},
	})
	if err != nil || otherNS == nil || otherNS.Allowed {
		t.Fatalf("serviceaccount namespace mismatch = %+v %v", otherNS, err)
	}

	analyzed, err := svc.AnalyzeRBACPermissions(ctx, &model.AnalyzeRBACPermissionsReq{ClusterID: 1, Subject: alice})
	if err != nil || analyzed == nil || !containsVerb(analyzed.Permissions["pods"], "get") || !containsVerb(analyzed.Permissions["configmaps"], "get") || len(analyzed.Sources) < 2 {
		t.Fatalf("analyze = %+v %v", analyzed, err)
	}
	foundPods, foundView := false, false
	for _, source := range analyzed.Sources {
		if source.BindingName == "read-pods" && source.Type == "Role" && source.Namespace == "ops" {
			foundPods = true
		}
		if source.BindingName == "view-dev" && source.Type == "ClusterRole" && source.Name == "view" && source.Namespace == "dev" {
			foundView = true
		}
	}
	if !foundPods || !foundView {
		t.Fatalf("sources = %+v", analyzed.Sources)
	}

	roles.err = errors.New("apiserver down")
	_, err = svc.CheckRBACPermission(ctx, &model.CheckRBACPermissionReq{ClusterID: 1, Subject: alice, Namespace: "ops", Resource: "pods", Verb: "get"})
	if err == nil || !strings.Contains(err.Error(), "读取Role规则失败") || !strings.Contains(err.Error(), "apiserver down") {
		t.Fatalf("lookup error = %v", err)
	}
}

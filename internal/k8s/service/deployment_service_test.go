package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type deploymentManagerFixture struct {
	manager.DeploymentManager
	items     []*model.K8sDeployment
	err       error
	cluster   int
	namespace string
	selector  string
	replicas  int32
	scaled    bool
	deleted   string
	revision  int64
}

func (m *deploymentManagerFixture) GetDeploymentList(_ context.Context, cluster int, namespace string, options metav1.ListOptions) ([]*model.K8sDeployment, error) {
	m.cluster, m.namespace, m.selector = cluster, namespace, options.LabelSelector
	return m.items, m.err
}

func (m *deploymentManagerFixture) ScaleDeployment(_ context.Context, _ int, _, name string, replicas int32) error {
	m.deleted, m.replicas, m.scaled = name, replicas, true
	return m.err
}

func (m *deploymentManagerFixture) DeleteDeployment(_ context.Context, _ int, _, name string, _ metav1.DeleteOptions) error {
	m.deleted = name
	return m.err
}

func (m *deploymentManagerFixture) RollbackDeployment(_ context.Context, _ int, _, name string, revision int64) error {
	m.deleted, m.revision = name, revision
	return m.err
}

func TestDeploymentListFiltersBeforePagination(t *testing.T) {
	fixture := &deploymentManagerFixture{items: []*model.K8sDeployment{
		{Name: "api-old", Status: model.K8sDeploymentStatusRunning, CreatedAt: time.Unix(1, 0)},
		{Name: "api-paused", Status: model.K8sDeploymentStatusPaused, CreatedAt: time.Unix(4, 0)},
		{Name: "batch", Status: model.K8sDeploymentStatusRunning, CreatedAt: time.Unix(3, 0)},
		{Name: "API-new", Status: model.K8sDeploymentStatusRunning, CreatedAt: time.Unix(2, 0)},
	}}
	svc := NewDeploymentService(fixture, zap.NewNop())
	result, err := svc.GetDeploymentList(context.Background(), &model.GetDeploymentListReq{
		ListReq: model.ListReq{Page: 2, Size: 1, Search: "api"}, ClusterID: 5, Namespace: "ops", Status: "RUNNING", Labels: map[string]string{"app": "api"},
	})
	if err != nil || fixture.cluster != 5 || fixture.namespace != "ops" || fixture.selector != "app=api" || result.Total != 2 || len(result.Items) != 1 || result.Items[0].Name != "api-old" {
		t.Fatalf("deployment page = %+v selector=%s err=%v", result, fixture.selector, err)
	}
	if _, err := svc.GetDeploymentList(context.Background(), nil); err == nil {
		t.Fatal("nil list accepted")
	}
	if _, err := svc.GetDeploymentList(context.Background(), &model.GetDeploymentListReq{Namespace: "ops"}); err == nil {
		t.Fatal("missing cluster accepted")
	}
	fixture.err = errors.New("forbidden")
	if _, err := svc.GetDeploymentList(context.Background(), &model.GetDeploymentListReq{ClusterID: 5, Namespace: "ops"}); err == nil || !strings.Contains(err.Error(), "获取Deployment列表失败") {
		t.Fatalf("list error = %v", err)
	}
}

func TestDeploymentScaleRollbackAndDeleteErrors(t *testing.T) {
	fixture := &deploymentManagerFixture{}
	svc := NewDeploymentService(fixture, zap.NewNop())
	if err := svc.ScaleDeployment(context.Background(), nil); err == nil {
		t.Fatal("nil scale accepted")
	}
	if err := svc.ScaleDeployment(context.Background(), &model.ScaleDeploymentReq{ClusterID: 5, Namespace: "ops", Name: "api", Replicas: -1}); err == nil || fixture.scaled {
		t.Fatal("negative replicas reached the manager")
	}
	if err := svc.ScaleDeployment(context.Background(), &model.ScaleDeploymentReq{ClusterID: 5, Namespace: "ops", Name: "api", Replicas: 0}); err != nil || !fixture.scaled || fixture.replicas != 0 {
		t.Fatalf("zero scale = %v replicas=%d", err, fixture.replicas)
	}
	fixture.err = errors.New("conflict")
	if err := svc.ScaleDeployment(context.Background(), &model.ScaleDeploymentReq{ClusterID: 5, Namespace: "ops", Name: "api", Replicas: 2}); err == nil || !strings.Contains(err.Error(), "扩缩容Deployment失败") {
		t.Fatalf("scale error = %v", err)
	}
	if err := svc.RollbackDeployment(context.Background(), &model.RollbackDeploymentReq{ClusterID: 5, Namespace: "ops", Name: "api", Revision: 0}); err == nil {
		t.Fatal("revision 0 accepted")
	}
	fixture.err = nil
	if err := svc.RollbackDeployment(context.Background(), &model.RollbackDeploymentReq{ClusterID: 5, Namespace: "ops", Name: "api", Revision: 3}); err != nil || fixture.revision != 3 {
		t.Fatalf("rollback = %v revision=%d", err, fixture.revision)
	}
	fixture.err = errors.New("denied")
	if err := svc.DeleteDeployment(context.Background(), &model.DeleteDeploymentReq{ClusterID: 5, Namespace: "ops", Name: "api"}); err == nil || !strings.Contains(err.Error(), "删除Deployment失败") || fixture.deleted != "api" {
		t.Fatalf("delete = %v", err)
	}
	if _, err := svc.GetDeploymentPods(context.Background(), &model.GetDeploymentPodsReq{ClusterID: 5, Namespace: "ops"}); err == nil {
		t.Fatal("missing deployment name accepted")
	}
}

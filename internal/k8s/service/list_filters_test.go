package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type serviceListFixture struct {
	manager.ServiceManager
	items     []corev1.Service
	err       error
	cluster   int
	namespace string
}

func (m *serviceListFixture) ListServices(_ context.Context, cluster int, namespace string) (*corev1.ServiceList, error) {
	m.cluster, m.namespace = cluster, namespace
	return &corev1.ServiceList{Items: m.items}, m.err
}

type configMapListFixture struct {
	manager.ConfigMapManager
	items               []corev1.ConfigMap
	namespace, selector string
}

func (m *configMapListFixture) ListConfigMaps(_ context.Context, _ int, namespace string) (*corev1.ConfigMapList, error) {
	m.namespace = namespace
	return &corev1.ConfigMapList{Items: m.items}, nil
}

func (m *configMapListFixture) ListConfigMapsBySelector(ctx context.Context, cluster int, namespace, selector string) (*corev1.ConfigMapList, error) {
	m.selector = selector
	return m.ListConfigMaps(ctx, cluster, namespace)
}

func TestServiceFiltersBeforePagination(t *testing.T) {
	fixture := &serviceListFixture{}
	for index, ip := range []string{"", "10.0.0.1", "", "10.0.0.2"} {
		fixture.items = append(fixture.items, corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "api-" + string(rune('a'+index)), CreationTimestamp: metav1.NewTime(time.Unix(int64(10-index), 0)), Labels: map[string]string{"app": "api"}},
			Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP, ClusterIP: ip},
		})
	}
	svc := NewSvcService(fixture, zap.NewNop())
	req := &model.GetServiceListReq{ListReq: model.ListReq{Page: 2, Size: 1, Search: "API"}, ClusterID: 7, Namespace: "ops", Status: model.K8sSvcStatusRunning, Type: "ClusterIP", Labels: map[string]string{"app": "api"}}
	result, err := svc.GetServiceList(context.Background(), req)
	if err != nil || result.Total != 2 || len(result.Items) != 1 || result.Items[0].Name != "api-d" || fixture.cluster != 7 || fixture.namespace != "ops" {
		t.Fatalf("filter/page/namespace mismatch: result=%+v err=%v", result, err)
	}
	req.Labels = map[string]string{"missing": ""}
	result, err = svc.GetServiceList(context.Background(), req)
	if err != nil || result.Total != 0 || len(result.Items) != 0 {
		t.Fatalf("missing label matched: %+v %v", result, err)
	}
	req.Status = 4
	if _, err := svc.GetServiceList(context.Background(), req); err == nil {
		t.Fatal("invalid status accepted")
	}
	req.Status = 0
	fixture.err = errors.New("cluster unavailable")
	if _, err := svc.GetServiceList(context.Background(), req); err == nil {
		t.Fatal("upstream failure hidden")
	}
}

func TestConfigMapDataKeysBeforePagination(t *testing.T) {
	fixture := &configMapListFixture{items: []corev1.ConfigMap{
		{ObjectMeta: metav1.ObjectMeta{Name: "api-decoy", CreationTimestamp: metav1.NewTime(time.Unix(3, 0))}, Data: map[string]string{"other": "MATCH-KEY"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "api-text", CreationTimestamp: metav1.NewTime(time.Unix(2, 0))}, Data: map[string]string{"MATCH-key": "private content"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "api-binary", CreationTimestamp: metav1.NewTime(time.Unix(1, 0))}, BinaryData: map[string][]byte{"match-KEY.bin": {1, 2}}},
	}}
	svc := NewConfigMapService(nil, fixture, zap.NewNop())
	req := &model.GetConfigMapListReq{ListReq: model.ListReq{Page: 2, Size: 1, Search: "API"}, ClusterID: 7, Namespace: "ops", DataKey: "MaTcH-kEy", Labels: map[string]string{"app": "api"}}
	result, err := svc.GetConfigMapList(context.Background(), req)
	if err != nil || result.Total != 2 || len(result.Items) != 1 || result.Items[0].Name != "api-binary" || fixture.namespace != "ops" || fixture.selector != "app=api" {
		t.Fatalf("key/page/namespace mismatch: result=%+v err=%v", result, err)
	}
	req.DataKey = "missing"
	result, err = svc.GetConfigMapList(context.Background(), req)
	if err != nil || result.Total != 0 || len(result.Items) != 0 {
		t.Fatalf("empty filter mismatch: %+v %v", result, err)
	}
}

func TestResourceFilterQueryBinding(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/?status=2&data_key=MiXeD&namespace=ops", nil)
	var serviceReq model.GetServiceListReq
	var configMapReq model.GetConfigMapListReq
	if err := ctx.ShouldBindQuery(&serviceReq); err != nil || serviceReq.Status != model.K8sSvcStatusStopped {
		t.Fatalf("status query not bound: %v", err)
	}
	if err := ctx.ShouldBindQuery(&configMapReq); err != nil || configMapReq.DataKey != "MiXeD" || configMapReq.Namespace != "ops" {
		t.Fatalf("data_key query not bound: %v", err)
	}
	ctx.Request = httptest.NewRequest("GET", "/?status=4", nil)
	if err := ctx.ShouldBindQuery(&serviceReq); err == nil {
		t.Fatal("invalid HTTP status filter accepted")
	}
}

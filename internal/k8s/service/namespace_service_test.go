package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type namespaceManagerFixture struct {
	manager.NamespaceManager
	items      []corev1.Namespace
	err        error
	cluster    int
	name       string
	created    *corev1.Namespace
	updated    *corev1.Namespace
	deleteOpts metav1.DeleteOptions
	selector   string
}

func (m *namespaceManagerFixture) GetNamespaceList(_ context.Context, cluster int, options metav1.ListOptions) (*corev1.NamespaceList, error) {
	m.cluster = cluster
	m.selector = options.LabelSelector
	return &corev1.NamespaceList{Items: m.items}, m.err
}

func (m *namespaceManagerFixture) CreateNamespace(_ context.Context, cluster int, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	m.cluster = cluster
	m.created = namespace
	return namespace, m.err
}

func (m *namespaceManagerFixture) GetNamespace(_ context.Context, cluster int, name string) (*corev1.Namespace, error) {
	m.cluster, m.name = cluster, name
	if m.err != nil {
		return nil, m.err
	}
	for i := range m.items {
		if m.items[i].Name == name {
			return m.items[i].DeepCopy(), nil
		}
	}
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name, UID: "uid-1"}}, nil
}

func (m *namespaceManagerFixture) UpdateNamespace(_ context.Context, cluster int, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	m.cluster = cluster
	m.updated = namespace
	return namespace, m.err
}

func (m *namespaceManagerFixture) DeleteNamespace(_ context.Context, cluster int, name string, options metav1.DeleteOptions) error {
	m.cluster, m.name, m.deleteOpts = cluster, name, options
	return m.err
}

func TestNamespaceCreateRejectsReservedNamesAndBadMetadata(t *testing.T) {
	fixture := &namespaceManagerFixture{}
	svc := NewNamespaceService(nil, fixture, zap.NewNop())
	if err := svc.CreateNamespace(context.Background(), nil); err == nil {
		t.Fatal("nil create accepted")
	}
	if err := svc.CreateNamespace(context.Background(), &model.K8sNamespaceCreateReq{ClusterID: 1}); err == nil {
		t.Fatal("empty name accepted")
	}
	if err := svc.CreateNamespace(context.Background(), &model.K8sNamespaceCreateReq{Name: "team-a"}); err == nil {
		t.Fatal("missing cluster accepted")
	}
	for _, name := range []string{"kube-system", "default", "Not_Valid"} {
		if err := svc.CreateNamespace(context.Background(), &model.K8sNamespaceCreateReq{ClusterID: 2, Name: name}); err == nil || !strings.Contains(err.Error(), "命名空间名称验证失败") {
			t.Fatalf("name %s = %v", name, err)
		}
	}
	if err := svc.CreateNamespace(context.Background(), &model.K8sNamespaceCreateReq{
		ClusterID: 2, Name: "team-a", Labels: model.KeyValueList{{Key: "owner", Value: strings.Repeat("v", 64)}},
	}); err == nil || !strings.Contains(err.Error(), "标签验证失败") {
		t.Fatalf("bad label = %v", err)
	}
	if err := svc.CreateNamespace(context.Background(), &model.K8sNamespaceCreateReq{
		ClusterID: 2, Name: "team-a", Labels: model.KeyValueList{{Key: "owner", Value: "team-a"}},
	}); err != nil {
		t.Fatal(err)
	}
	if fixture.cluster != 2 || fixture.created == nil || fixture.created.Labels["owner"] != "team-a" || fixture.created.Name != "team-a" {
		t.Fatalf("created = %+v", fixture.created)
	}
	fixture.err = errors.New("already exists")
	if err := svc.CreateNamespace(context.Background(), &model.K8sNamespaceCreateReq{
		ClusterID: 2, Name: "team-a", Labels: model.KeyValueList{{Key: "owner", Value: "team-a"}},
	}); err == nil || !strings.Contains(err.Error(), "创建命名空间失败") {
		t.Fatalf("create error = %v", err)
	}
}

func TestNamespaceOptionalLabelsAreRejected(t *testing.T) {
	fixture := &namespaceManagerFixture{items: []corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{Name: "team-a", Labels: map[string]string{"old": "1"}, UID: "uid-1"},
	}}}
	svc := NewNamespaceService(nil, fixture, zap.NewNop())
	if err := svc.CreateNamespace(context.Background(), &model.K8sNamespaceCreateReq{ClusterID: 2, Name: "team-a"}); err != nil {
		t.Fatal(err)
	}
	if fixture.created == nil || fixture.created.Name != "team-a" || len(fixture.created.Labels) != 0 {
		t.Fatalf("created = %+v", fixture.created)
	}
	if err := svc.UpdateNamespace(context.Background(), &model.K8sNamespaceUpdateReq{ClusterID: 2, Name: "team-a"}); err != nil {
		t.Fatal(err)
	}
	if fixture.updated == nil || fixture.updated.Labels["old"] != "1" {
		t.Fatalf("omitted labels were not kept: %+v", fixture.updated)
	}
}

func TestNamespaceDeleteForceAndListFilters(t *testing.T) {
	grace := int64(9)
	fixture := &namespaceManagerFixture{items: []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "api-old", Labels: map[string]string{"team": "a"}, CreationTimestamp: metav1.NewTime(time.Unix(1, 0))}, Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive}},
		{ObjectMeta: metav1.ObjectMeta{Name: "api-new", Labels: map[string]string{"team": "a"}, CreationTimestamp: metav1.NewTime(time.Unix(3, 0))}, Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive}},
		{ObjectMeta: metav1.ObjectMeta{Name: "api-ending", Labels: map[string]string{"team": "a"}, CreationTimestamp: metav1.NewTime(time.Unix(4, 0))}, Status: corev1.NamespaceStatus{Phase: corev1.NamespaceTerminating}},
		{ObjectMeta: metav1.ObjectMeta{Name: "other", Labels: map[string]string{"team": "b"}, CreationTimestamp: metav1.NewTime(time.Unix(2, 0))}, Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive}},
	}}
	svc := NewNamespaceService(nil, fixture, zap.NewNop())
	if err := svc.DeleteNamespace(context.Background(), &model.K8sNamespaceDeleteReq{ClusterID: 2, Name: "team-a", Force: 1, GracePeriodSeconds: &grace}); err != nil {
		t.Fatal(err)
	}
	if fixture.deleteOpts.GracePeriodSeconds == nil || *fixture.deleteOpts.GracePeriodSeconds != 0 || fixture.name != "team-a" {
		t.Fatalf("force delete = %+v", fixture.deleteOpts)
	}
	fixture.err = errors.New("forbidden")
	if err := svc.DeleteNamespace(context.Background(), &model.K8sNamespaceDeleteReq{ClusterID: 2, Name: "team-a", Force: 2}); err == nil || !strings.Contains(err.Error(), "删除命名空间失败") {
		t.Fatalf("delete error = %v", err)
	}

	fixture.err = nil
	result, err := svc.ListNamespaces(context.Background(), &model.K8sNamespaceListReq{
		ListReq: model.ListReq{Page: 2, Size: 1}, ClusterID: 2, Status: "Active", Search: "API",
		Labels: model.KeyValueList{{Key: "team", Value: "a"}},
	})
	if err != nil || result.Total != 2 || len(result.Items) != 1 || result.Items[0].Name != "api-old" || result.Items[0].Status != "活跃" || fixture.cluster != 2 {
		t.Fatalf("namespace page = %+v err=%v", result, err)
	}
	if _, err := svc.ListNamespaces(context.Background(), &model.K8sNamespaceListReq{ClusterID: 2, LabelSelector: "app=.."}); err == nil || !strings.Contains(err.Error(), "过滤参数验证失败") {
		t.Fatalf("selector = %v", err)
	}
	fixture.err = errors.New("timeout")
	if _, err := svc.ListNamespaces(context.Background(), &model.K8sNamespaceListReq{ClusterID: 2}); err == nil || !strings.Contains(err.Error(), "获取命名空间列表失败") {
		t.Fatalf("list error = %v", err)
	}
	if _, err := svc.GetNamespaceDetails(context.Background(), &model.K8sNamespaceGetDetailsReq{ClusterID: 2, Name: "team-a"}); err == nil || !strings.Contains(err.Error(), "获取命名空间详情失败") {
		t.Fatalf("details = %v", err)
	}
}

func TestNamespaceUpdateReplacesLabels(t *testing.T) {
	fixture := &namespaceManagerFixture{items: []corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{Name: "team-a", Labels: map[string]string{"old": "1"}, UID: "uid-1"},
		Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
	}}}
	svc := NewNamespaceService(nil, fixture, zap.NewNop())
	if err := svc.UpdateNamespace(context.Background(), &model.K8sNamespaceUpdateReq{
		ClusterID: 2, Name: "team-a", Labels: model.KeyValueList{{Key: "owner", Value: "team-a"}},
	}); err != nil {
		t.Fatal(err)
	}
	if fixture.updated == nil || fixture.updated.Labels["owner"] != "team-a" || fixture.updated.Labels["old"] != "" {
		t.Fatalf("labels were merged: %+v", fixture.updated)
	}
	if err := svc.UpdateNamespace(context.Background(), &model.K8sNamespaceUpdateReq{
		ClusterID: 2, Name: "team-a", Labels: model.KeyValueList{{Key: "owner", Value: "team-a"}}, Annotations: model.KeyValueList{{Key: "", Value: "x"}},
	}); err == nil || !strings.Contains(err.Error(), "注解验证失败") {
		t.Fatalf("empty annotation key = %v", err)
	}
	fixture.err = errors.New("not found")
	if err := svc.UpdateNamespace(context.Background(), &model.K8sNamespaceUpdateReq{
		ClusterID: 2, Name: "missing", Labels: model.KeyValueList{{Key: "owner", Value: "team-a"}},
	}); err == nil || !strings.Contains(err.Error(), "获取命名空间失败") {
		t.Fatalf("update lookup = %v", err)
	}
}

func namespaceWithSystemMetadata() *namespaceManagerFixture {
	return &namespaceManagerFixture{items: []corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "team-a",
			UID:  "uid-1",
			Labels: map[string]string{
				"kubernetes.io/metadata.name":        "team-a",
				"pod-security.kubernetes.io/enforce": "restricted",
				"team":                               "ops",
				"notkubernetes.io/custom":            "no",
			},
			Annotations: map[string]string{
				"kubernetes.io/metadata.name": "team-a",
				"example.com/note":            "keep-or-drop",
			},
		},
	}}}
}

func TestNamespaceUpdateJSONAbsentKeepsEmptyClears(t *testing.T) {
	svcFor := func() (*namespaceService, *namespaceManagerFixture) {
		fixture := namespaceWithSystemMetadata()
		return NewNamespaceService(nil, fixture, zap.NewNop()).(*namespaceService), fixture
	}

	absentBody := []byte(`{"cluster_id":2,"name":"team-a"}`)
	var absent model.K8sNamespaceUpdateReq
	if err := json.Unmarshal(absentBody, &absent); err != nil {
		t.Fatal(err)
	}
	if absent.Labels != nil || absent.Annotations != nil {
		t.Fatalf("absent fields bound as labels=%#v annotations=%#v", absent.Labels, absent.Annotations)
	}
	svc, fixture := svcFor()
	if err := svc.UpdateNamespace(context.Background(), &absent); err != nil {
		t.Fatal(err)
	}
	if fixture.updated.Labels["team"] != "ops" || fixture.updated.Labels["kubernetes.io/metadata.name"] != "team-a" || fixture.updated.Annotations["example.com/note"] != "keep-or-drop" {
		t.Fatalf("absent JSON changed stored metadata: %+v", fixture.updated.ObjectMeta)
	}

	emptyBody := []byte(`{"cluster_id":2,"name":"team-a","labels":[],"annotations":[]}`)
	var empty model.K8sNamespaceUpdateReq
	if err := json.Unmarshal(emptyBody, &empty); err != nil {
		t.Fatal(err)
	}
	if empty.Labels == nil || len(empty.Labels) != 0 || empty.Annotations == nil || len(empty.Annotations) != 0 {
		t.Fatalf("[] bound as labels=%#v annotations=%#v", empty.Labels, empty.Annotations)
	}
	svc, fixture = svcFor()
	if err := svc.UpdateNamespace(context.Background(), &empty); err != nil {
		t.Fatal(err)
	}
	if fixture.updated.Labels["kubernetes.io/metadata.name"] != "team-a" || fixture.updated.Labels["pod-security.kubernetes.io/enforce"] != "restricted" {
		t.Fatalf("system labels dropped: %#v", fixture.updated.Labels)
	}
	if _, ok := fixture.updated.Labels["team"]; ok {
		t.Fatal("user label team survived []")
	}
	if _, ok := fixture.updated.Labels["notkubernetes.io/custom"]; ok {
		t.Fatal("notkubernetes.io was treated as reserved")
	}
	if fixture.updated.Annotations["kubernetes.io/metadata.name"] != "team-a" {
		t.Fatalf("system annotation dropped: %#v", fixture.updated.Annotations)
	}
	if _, ok := fixture.updated.Annotations["example.com/note"]; ok {
		t.Fatal("user annotation survived []")
	}

	replaceBody := []byte(`{"cluster_id":2,"name":"team-a","labels":[{"key":"owner","value":"team-a"},{"key":"pod-security.kubernetes.io/enforce","value":"baseline"}],"annotations":[{"key":"note","value":"fresh"}]}`)
	var replace model.K8sNamespaceUpdateReq
	if err := json.Unmarshal(replaceBody, &replace); err != nil {
		t.Fatal(err)
	}
	svc, fixture = svcFor()
	if err := svc.UpdateNamespace(context.Background(), &replace); err != nil {
		t.Fatal(err)
	}
	if fixture.updated.Labels["owner"] != "team-a" || fixture.updated.Labels["pod-security.kubernetes.io/enforce"] != "baseline" || fixture.updated.Labels["kubernetes.io/metadata.name"] != "team-a" {
		t.Fatalf("replacement labels = %#v", fixture.updated.Labels)
	}
	if _, ok := fixture.updated.Labels["team"]; ok {
		t.Fatal("previous user label survived replacement")
	}
	if fixture.updated.Annotations["note"] != "fresh" {
		t.Fatalf("replacement annotations = %#v", fixture.updated.Annotations)
	}
	if _, ok := fixture.updated.Annotations["example.com/note"]; ok || fixture.updated.Annotations["kubernetes.io/metadata.name"] != "team-a" {
		t.Fatalf("annotation merge = %#v", fixture.updated.Annotations)
	}
}

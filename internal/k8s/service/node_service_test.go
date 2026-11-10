package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type nodeManagerFixture struct {
	manager.NodeManager
	nodes    []corev1.Node
	err      error
	cluster  int
	selector string
	drain    *k8sutils.DrainOptions
	drained  string
	cordoned string
	labels   map[string]string
	buildErr string
	taints   []*model.NodeTaint
}

func (m *nodeManagerFixture) GetNodeList(_ context.Context, cluster int, options metav1.ListOptions) (*corev1.NodeList, int64, error) {
	m.cluster = cluster
	m.selector = options.LabelSelector
	return &corev1.NodeList{Items: m.nodes}, int64(len(m.nodes)), m.err
}

func (m *nodeManagerFixture) BuildK8sNode(_ context.Context, _ int, node corev1.Node) (*model.K8sNode, error) {
	if node.Name == m.buildErr {
		return nil, errors.New("metrics unavailable")
	}
	return &model.K8sNode{Name: node.Name}, nil
}

func (m *nodeManagerFixture) GetNode(_ context.Context, _ int, name string) (*corev1.Node, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: name}}, nil
}

func (m *nodeManagerFixture) DrainNode(_ context.Context, _ int, name string, options *k8sutils.DrainOptions) error {
	m.drained = name
	m.drain = options
	return m.err
}

func (m *nodeManagerFixture) CordonNode(_ context.Context, _ int, name string) error {
	m.cordoned = name
	return m.err
}

func (m *nodeManagerFixture) UncordonNode(_ context.Context, _ int, name string) error {
	m.cordoned = name
	return m.err
}

func (m *nodeManagerFixture) UpdateNodeLabels(_ context.Context, _ int, name string, labels map[string]string) error {
	m.drained = name
	m.labels = labels
	return m.err
}

func (m *nodeManagerFixture) GetNodeTaints(context.Context, int, string) ([]*model.NodeTaint, int64, error) {
	return m.taints, int64(len(m.taints)), m.err
}

func readyNode(name string, at int64, unschedulable bool) corev1.Node {
	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name, CreationTimestamp: metav1.NewTime(time.Unix(at, 0))},
		Spec:       corev1.NodeSpec{Unschedulable: unschedulable},
		Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{{
			Type: corev1.NodeReady, Status: corev1.ConditionTrue,
		}}},
	}
}

func TestNodeListFiltersBeforePagination(t *testing.T) {
	fixture := &nodeManagerFixture{nodes: []corev1.Node{
		readyNode("api-old", 1, false),
		readyNode("api-held", 3, true),
		readyNode("batch", 4, false),
		readyNode("API-new", 2, false),
	}}
	svc := NewNodeService(nil, nil, fixture, zap.NewNop())
	result, err := svc.GetNodeList(context.Background(), &model.GetNodeListReq{
		ListReq: model.ListReq{Page: 2, Size: 1, Search: "api"}, ClusterID: 6, Status: []model.NodeStatus{model.NodeStatusReady}, LabelSelector: "role=worker",
	})
	if err != nil || fixture.cluster != 6 || fixture.selector != "role=worker" || result.Total != 2 || len(result.Items) != 1 || result.Items[0].Name != "api-old" {
		t.Fatalf("node page = %+v selector=%s err=%v", result, fixture.selector, err)
	}
	if _, err := svc.GetNodeList(context.Background(), nil); err == nil {
		t.Fatal("nil list accepted")
	}
	if _, err := svc.GetNodeList(context.Background(), &model.GetNodeListReq{}); err == nil {
		t.Fatal("missing cluster accepted")
	}
	fixture.err = errors.New("apiserver down")
	if _, err := svc.GetNodeList(context.Background(), &model.GetNodeListReq{ClusterID: 6}); err == nil || !strings.Contains(err.Error(), "获取节点列表失败") {
		t.Fatalf("list error = %v", err)
	}
}

func TestNodeListPageZeroIsUnsafe(t *testing.T) {
	fixture := &nodeManagerFixture{nodes: []corev1.Node{
		readyNode("n1", 1, false),
		readyNode("n2", 2, false),
	}}
	svc := NewNodeService(nil, nil, fixture, zap.NewNop())
	for _, page := range []int{0, -1} {
		result, err := svc.GetNodeList(context.Background(), &model.GetNodeListReq{
			ListReq: model.ListReq{Page: page, Size: 1}, ClusterID: 6,
		})
		if err != nil || result.Total != 2 || len(result.Items) != 1 || result.Items[0].Name != "n2" {
			t.Fatalf("page %d = %+v err=%v", page, result, err)
		}
	}
}

func TestNodeDrainCordonAndLabels(t *testing.T) {
	fixture := &nodeManagerFixture{}
	svc := NewNodeService(dao.ClusterDAO(nil), client.K8sClient(nil), fixture, zap.NewNop())
	if err := svc.DrainNode(context.Background(), nil); err == nil {
		t.Fatal("nil drain accepted")
	}
	if err := svc.DrainNode(context.Background(), &model.DrainNodeReq{ClusterID: 0, NodeName: "worker-a"}); err == nil {
		t.Fatal("invalid drain target accepted")
	}
	req := &model.DrainNodeReq{ClusterID: 6, NodeName: "worker-a", Force: 1, IgnoreDaemonSets: 2, DeleteLocalData: 1, GracePeriodSeconds: 20, TimeoutSeconds: 40}
	if err := svc.DrainNode(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if fixture.drained != "worker-a" || fixture.drain == nil || fixture.drain.Force != 1 || fixture.drain.IgnoreDaemonSets != 2 || fixture.drain.GracePeriodSeconds != 20 || fixture.drain.TimeoutSeconds != 40 {
		t.Fatalf("drain options = %+v", fixture.drain)
	}
	fixture.err = errors.New("pdb blocked")
	if err := svc.DrainNode(context.Background(), req); err == nil || !strings.Contains(err.Error(), "驱逐节点失败") {
		t.Fatalf("drain error = %v", err)
	}
	fixture.err = errors.New("denied")
	if err := svc.CordonNode(context.Background(), &model.NodeCordonReq{ClusterID: 6, NodeName: "worker-a"}); err == nil || !strings.Contains(err.Error(), "worker-a") {
		t.Fatalf("cordon = %v", err)
	}
	fixture.err = nil
	if err := svc.UncordonNode(context.Background(), &model.NodeUncordonReq{ClusterID: 6, NodeName: "worker-a"}); err != nil || fixture.cordoned != "worker-a" {
		t.Fatalf("uncordon = %v", err)
	}
	if err := svc.UpdateNodeLabels(context.Background(), &model.UpdateNodeLabelsReq{ClusterID: 6, NodeName: "worker-a", Labels: map[string]string{"": "x"}}); err == nil || !strings.Contains(err.Error(), "标签验证失败") {
		t.Fatalf("bad label = %v", err)
	}
	if err := svc.UpdateNodeLabels(context.Background(), &model.UpdateNodeLabelsReq{ClusterID: 6, NodeName: "worker-a"}); err != nil || fixture.labels != nil {
		t.Fatalf("nil labels = %#v %v", fixture.labels, err)
	}
	fixture.err = errors.New("conflict")
	if err := svc.UpdateNodeLabels(context.Background(), &model.UpdateNodeLabelsReq{ClusterID: 6, NodeName: "worker-a", Labels: map[string]string{"role": "worker"}}); err == nil || !strings.Contains(err.Error(), "更新节点标签失败") || fixture.labels["role"] != "worker" {
		t.Fatalf("label update = %v", err)
	}
	if _, err := svc.GetNodeDetail(context.Background(), &model.GetNodeDetailReq{ClusterID: 6, NodeName: "worker-a"}); err == nil || !strings.Contains(err.Error(), "获取节点失败") {
		t.Fatalf("detail error = %v", err)
	}
	fixture.err = nil
	fixture.taints = []*model.NodeTaint{{Key: "dedicated", Value: "ops", Effect: "NoSchedule"}}
	taints, err := svc.GetNodeTaints(context.Background(), &model.GetNodeTaintsReq{ClusterID: 6, NodeName: "worker-a"})
	if err != nil || taints.Total != 1 || taints.Items[0].Key != "dedicated" {
		t.Fatalf("taints = %+v %v", taints, err)
	}
}

func TestNodeEmptyLabelMapConflictsWithComment(t *testing.T) {
	fixture := &nodeManagerFixture{}
	svc := NewNodeService(nil, nil, fixture, zap.NewNop())
	err := svc.UpdateNodeLabels(context.Background(), &model.UpdateNodeLabelsReq{
		ClusterID: 6, NodeName: "worker-a", Labels: map[string]string{},
	})
	if err != nil || fixture.labels == nil || len(fixture.labels) != 0 {
		t.Fatalf("empty labels = %#v err=%v", fixture.labels, err)
	}
}

func TestNodeBuildFailureDropsItemButKeepsTotal(t *testing.T) {
	fixture := &nodeManagerFixture{nodes: []corev1.Node{
		readyNode("ok-a", 1, false),
		readyNode("bad", 3, false),
		readyNode("ok-b", 2, false),
	}, buildErr: "bad"}
	svc := NewNodeService(nil, nil, fixture, zap.NewNop())
	result, err := svc.GetNodeList(context.Background(), &model.GetNodeListReq{
		ListReq: model.ListReq{Page: 1, Size: 10}, ClusterID: 6,
	})
	if err != nil || result.Total != 2 || len(result.Items) != 2 {
		t.Fatalf("list = %+v err=%v", result, err)
	}
	for _, item := range result.Items {
		if item.Name == "bad" {
			t.Fatalf("dropped node still listed: %+v", result.Items)
		}
	}
}

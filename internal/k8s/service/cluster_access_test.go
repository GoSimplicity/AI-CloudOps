package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	kclient "github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
)

const otherClusterKubeconfig = `apiVersion: v1
kind: Config
current-context: fixture
clusters:
- name: fixture
  cluster: {server: 'https://127.0.0.1:6444'}
users:
- name: fixture
  user: {token: fixture-token}
contexts:
- name: fixture
  context: {cluster: fixture, user: fixture}
`

type clusterDAOFixture struct {
	dao.ClusterDAO
	byID       *model.K8sCluster
	byIDErr    error
	byName     *model.K8sCluster
	byNameErr  error
	list       []*model.K8sCluster
	listTotal  int64
	listErr    error
	created    *model.K8sCluster
	updated    *model.K8sCluster
	deletedID  int
	deleted    bool
	status     model.ClusterStatus
	statusSeen bool
}

func (d *clusterDAOFixture) GetClusterList(context.Context, *model.ListClustersReq) ([]*model.K8sCluster, int64, error) {
	return d.list, d.listTotal, d.listErr
}

func (d *clusterDAOFixture) GetClusterByID(context.Context, int) (*model.K8sCluster, error) {
	if d.byIDErr != nil {
		return nil, d.byIDErr
	}
	return d.byID, nil
}

func (d *clusterDAOFixture) GetClusterByName(context.Context, string) (*model.K8sCluster, error) {
	if d.byNameErr != nil {
		return nil, d.byNameErr
	}
	if d.byName == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return d.byName, nil
}

func (d *clusterDAOFixture) CreateCluster(_ context.Context, cluster *model.K8sCluster) error {
	cluster.ID = 9
	copied := *cluster
	d.created = &copied
	return nil
}

func (d *clusterDAOFixture) UpdateCluster(_ context.Context, cluster *model.K8sCluster) error {
	copied := *cluster
	d.updated = &copied
	return nil
}

func (d *clusterDAOFixture) UpdateClusterStatus(_ context.Context, _ int, status model.ClusterStatus) error {
	d.status, d.statusSeen = status, true
	return nil
}

func (d *clusterDAOFixture) DeleteCluster(_ context.Context, id int) error {
	d.deletedID, d.deleted = id, true
	return nil
}

type clusterManagerFixture struct {
	manager.ClusterManager
	err       error
	created   *model.K8sCluster
	refreshed int
}

func (m *clusterManagerFixture) CreateCluster(_ context.Context, cluster *model.K8sCluster) error {
	m.created = cluster
	return m.err
}

func (m *clusterManagerFixture) RefreshCluster(_ context.Context, id int) error {
	m.refreshed = id
	return m.err
}

type clusterClientFixture struct {
	kclient.K8sClient
	kubeErr error
	removed []int
}

func (c *clusterClientFixture) GetKubeClient(int) (*kubernetes.Clientset, error) {
	if c.kubeErr == nil {
		c.kubeErr = errors.New("kube client unavailable")
	}
	return nil, c.kubeErr
}

func (c *clusterClientFixture) RemoveCluster(id int) { c.removed = append(c.removed, id) }

func TestClusterCreateValidatesOwnerAndQuota(t *testing.T) {
	daoFixture := &clusterDAOFixture{}
	mgr := &clusterManagerFixture{}
	svc := NewClusterService(daoFixture, &clusterClientFixture{}, mgr, zap.NewNop())
	if err := svc.CreateCluster(context.Background(), nil); err == nil {
		t.Fatal("nil create accepted")
	}
	if err := svc.CreateCluster(context.Background(), &model.CreateClusterReq{Name: "owned"}); err == nil {
		t.Fatal("incomplete create accepted")
	}
	req := &model.CreateClusterReq{
		Name: "owned", ApiServerAddr: "https://127.0.0.1:6443", KubeConfigContent: quotaKubeconfig,
		CreateUserID: 7, CreateUserName: "alice",
	}
	if err := svc.CreateCluster(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if daoFixture.created == nil || daoFixture.created.CreateUserID != 7 || daoFixture.created.CreateUserName != "alice" || daoFixture.created.ID != 9 || mgr.created == nil {
		t.Fatalf("owner was not stored: %+v", daoFixture.created)
	}

	daoFixture.byName = &model.K8sCluster{Name: "owned"}
	daoFixture.created = nil
	if err := svc.CreateCluster(context.Background(), req); err == nil || !strings.Contains(err.Error(), "已存在") || daoFixture.created != nil {
		t.Fatalf("duplicate = %v", err)
	}
	daoFixture.byName = nil
	daoFixture.byNameErr = errors.New("db down")
	if err := svc.CreateCluster(context.Background(), req); err == nil || !strings.Contains(err.Error(), "查询集群失败") {
		t.Fatalf("lookup error = %v", err)
	}
	daoFixture.byNameErr = nil
	req.KubeConfigContent = "not-a-kubeconfig"
	if err := svc.CreateCluster(context.Background(), req); err == nil || !strings.Contains(err.Error(), "kubeconfig") {
		t.Fatalf("bad kubeconfig = %v", err)
	}
	req.KubeConfigContent = quotaKubeconfig
	req.CpuRequest, req.CpuLimit = "500", "100"
	if err := svc.CreateCluster(context.Background(), req); err == nil || !strings.Contains(err.Error(), "资源配置验证失败") {
		t.Fatalf("quota order = %v", err)
	}
	req.CpuRequest, req.CpuLimit = "", ""
	req.RestrictNamespace = model.StringList{"team-a", "team-a"}
	if err := svc.CreateCluster(context.Background(), req); err == nil || !strings.Contains(err.Error(), "配额命名空间重复") {
		t.Fatalf("duplicate namespace = %v", err)
	}
	req.RestrictNamespace = model.StringList{"BAD"}
	if err := svc.CreateCluster(context.Background(), req); err == nil || !strings.Contains(err.Error(), "无效的配额命名空间") {
		t.Fatalf("bad namespace = %v", err)
	}
}

func TestClusterReadDeleteAndRefreshErrors(t *testing.T) {
	stored := &model.K8sCluster{
		Model: model.Model{ID: 9}, Name: "owned", KubeConfigContent: quotaKubeconfig,
		CpuRequest: "100", RestrictNamespace: model.StringList{"team-a"},
	}
	daoFixture := &clusterDAOFixture{byID: stored, list: []*model.K8sCluster{stored}, listTotal: 1}
	client := &clusterClientFixture{kubeErr: errors.New("api unreachable")}
	mgr := &clusterManagerFixture{}
	svc := NewClusterService(daoFixture, client, mgr, zap.NewNop())
	if _, err := svc.ListClusters(context.Background(), nil); err == nil {
		t.Fatal("nil list accepted")
	}
	listed, err := svc.ListClusters(context.Background(), &model.ListClustersReq{ListReq: model.ListReq{Page: 1, Size: 10}})
	if err != nil || listed.Total != 1 || listed.Items[0].Name != "owned" {
		t.Fatalf("list = %+v %v", listed, err)
	}
	daoFixture.listErr = errors.New("db down")
	if _, err := svc.ListClusters(context.Background(), &model.ListClustersReq{}); err == nil || !strings.Contains(err.Error(), "查询所有集群失败") {
		t.Fatalf("list error = %v", err)
	}
	if _, err := svc.GetClusterByID(context.Background(), &model.GetClusterReq{}); err == nil {
		t.Fatal("id 0 accepted")
	}
	got, err := svc.GetClusterByID(context.Background(), &model.GetClusterReq{ID: 9})
	if err != nil || got.CreateUserID != stored.CreateUserID || got.Name != "owned" {
		t.Fatalf("get = %+v %v", got, err)
	}
	daoFixture.byIDErr = gorm.ErrRecordNotFound
	if _, err := svc.GetClusterByID(context.Background(), &model.GetClusterReq{ID: 9}); err == nil || !strings.Contains(err.Error(), "查询集群失败") {
		t.Fatalf("missing get = %v", err)
	}
	daoFixture.byID, daoFixture.byIDErr = nil, nil
	if _, err := svc.GetClusterByID(context.Background(), &model.GetClusterReq{ID: 9}); err == nil || !strings.Contains(err.Error(), "集群不存在") {
		t.Fatalf("nil cluster = %v", err)
	}

	daoFixture.byID, daoFixture.byIDErr = stored, nil
	if err := svc.DeleteCluster(context.Background(), &model.DeleteClusterReq{ID: 9}); err == nil || !strings.Contains(err.Error(), "清理平台配额失败") || daoFixture.deleted || len(client.removed) != 0 {
		t.Fatalf("quota delete = %v deleted=%v removed=%v", err, daoFixture.deleted, client.removed)
	}
	stored.RestrictNamespace = nil
	stored.CpuRequest = ""
	if err := svc.DeleteCluster(context.Background(), &model.DeleteClusterReq{ID: 9}); err != nil || !daoFixture.deleted || len(client.removed) != 1 || client.removed[0] != 9 {
		t.Fatalf("plain delete = %v removed=%v", err, client.removed)
	}
	daoFixture.byIDErr = gorm.ErrRecordNotFound
	if err := svc.DeleteCluster(context.Background(), &model.DeleteClusterReq{ID: 9}); err == nil || !strings.Contains(err.Error(), "集群不存在") {
		t.Fatalf("missing delete = %v", err)
	}
	daoFixture.byIDErr = nil
	mgr.err = errors.New("refresh denied")
	if err := svc.RefreshClusterStatus(context.Background(), &model.RefreshClusterReq{ID: 9}); err == nil || !strings.Contains(err.Error(), "刷新集群状态失败") || mgr.refreshed != 9 {
		t.Fatalf("refresh = %v", err)
	}
}

func TestClusterUpdateRejectsNameConflictAndIdentityChange(t *testing.T) {
	existing := &model.K8sCluster{
		Model: model.Model{ID: 9}, Name: "owned", Env: model.EnvDev,
		ApiServerAddr: "https://127.0.0.1:6443", KubeConfigContent: quotaKubeconfig,
		CpuRequest: "100", RestrictNamespace: model.StringList{"team-a"},
	}
	daoFixture := &clusterDAOFixture{byID: existing, byName: &model.K8sCluster{Model: model.Model{ID: 10}, Name: "other"}}
	svc := NewClusterService(daoFixture, &clusterClientFixture{}, &clusterManagerFixture{}, zap.NewNop())
	err := svc.UpdateCluster(context.Background(), &model.UpdateClusterReq{ID: 9, Name: "other", CpuRequest: "100", RestrictNamespace: model.StringList{"team-a"}})
	if err == nil || !strings.Contains(err.Error(), "已存在") || daoFixture.updated != nil {
		t.Fatalf("rename conflict = %v", err)
	}
	daoFixture.byName = nil
	err = svc.UpdateCluster(context.Background(), &model.UpdateClusterReq{
		ID: 9, Name: "owned", KubeConfigContent: otherClusterKubeconfig,
		CpuRequest: "100", RestrictNamespace: model.StringList{"team-a"},
	})
	if err == nil || !strings.Contains(err.Error(), "更换集群连接") || daoFixture.updated != nil {
		t.Fatalf("identity change = %v", err)
	}
	if err := svc.UpdateCluster(context.Background(), nil); err == nil {
		t.Fatal("nil update accepted")
	}
	if err := svc.RefreshClusterStatus(context.Background(), nil); err == nil {
		t.Fatal("nil refresh accepted")
	}
}

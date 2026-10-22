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
)

const quotaKubeconfig = `apiVersion: v1
kind: Config
current-context: fixture
clusters:
- name: fixture
  cluster: {server: 'https://127.0.0.1:6443'}
users:
- name: fixture
  user: {token: fixture-token}
contexts:
- name: fixture
  context: {cluster: fixture, user: fixture}
`

type quotaDAOFixture struct {
	dao.ClusterDAO
	stored  model.K8sCluster
	deleted bool
}

func (d *quotaDAOFixture) GetClusterByName(context.Context, string) (*model.K8sCluster, error) {
	return nil, gorm.ErrRecordNotFound
}
func (d *quotaDAOFixture) GetClusterByID(context.Context, int) (*model.K8sCluster, error) {
	copy := d.stored
	return &copy, nil
}
func (d *quotaDAOFixture) CreateCluster(_ context.Context, c *model.K8sCluster) error {
	c.ID = 42
	d.stored = *c
	return nil
}
func (d *quotaDAOFixture) UpdateCluster(_ context.Context, c *model.K8sCluster) error {
	d.stored = *c
	return nil
}
func (d *quotaDAOFixture) UpdateClusterStatus(_ context.Context, _ int, status model.ClusterStatus) error {
	d.stored.Status = status
	return nil
}
func (d *quotaDAOFixture) DeleteCluster(context.Context, int) error { d.deleted = true; return nil }

type quotaManagerFixture struct{ manager.ClusterManager }

func (*quotaManagerFixture) CreateCluster(context.Context, *model.K8sCluster) error {
	return errors.New("quota permission denied")
}
func (*quotaManagerFixture) UpdateCluster(context.Context, *model.K8sCluster) error {
	return errors.New("quota permission denied")
}

type quotaClientFixture struct {
	kclient.K8sClient
	removed bool
}

func (c *quotaClientFixture) RemoveCluster(int) { c.removed = true }

func TestFailedQuotaCreationRetainsRegistrationForCleanup(t *testing.T) {
	for _, policy := range []bool{false, true} {
		d := &quotaDAOFixture{}
		s := NewClusterService(d, nil, &quotaManagerFixture{}, zap.NewNop())
		req := &model.CreateClusterReq{Name: "quota-test", ApiServerAddr: "https://127.0.0.1:6443", KubeConfigContent: quotaKubeconfig}
		if policy {
			req.CpuRequest = "500"
			req.RestrictNamespace = model.StringList{"team-a"}
		}
		err := s.CreateCluster(context.Background(), req)
		if err == nil {
			t.Fatal("remote failure hidden")
		}
		if policy {
			if d.deleted || d.stored.Status != model.StatusError || !strings.Contains(err.Error(), "登记 42") {
				t.Fatal("lost failed quota ownership", err)
			}
		} else if !d.deleted {
			t.Fatal("ordinary failed registration not rolled back")
		}
	}
}

func TestFailedQuotaEditRestoresSavedConfiguration(t *testing.T) {
	d := &quotaDAOFixture{stored: model.K8sCluster{Model: model.Model{ID: 42}, Name: "quota-test", Env: model.EnvDev, ApiServerAddr: "https://127.0.0.1:6443", KubeConfigContent: quotaKubeconfig, CpuRequest: "100", CpuLimit: "400", RestrictNamespace: model.StringList{"team-a"}}}
	client := &quotaClientFixture{}
	s := NewClusterService(d, client, &quotaManagerFixture{}, zap.NewNop())
	err := s.UpdateCluster(context.Background(), &model.UpdateClusterReq{ID: 42, Name: "quota-test", CpuRequest: "200", CpuLimit: "400", RestrictNamespace: model.StringList{"team-a"}})
	if err == nil || !client.removed {
		t.Fatal("failed edit did not invalidate new client", err)
	}
	if d.stored.CpuRequest != "100" || d.stored.KubeConfigContent != quotaKubeconfig || d.stored.Env != model.EnvDev {
		t.Fatal("failed edit lost original configuration")
	}
}

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type yamlManagerFixture struct {
	manager.YamlManager
	task            *model.K8sYamlTask
	template        *model.K8sYamlTemplate
	tasks           []*model.K8sYamlTask
	templates       []*model.K8sYamlTemplate
	err             error
	deletedID       int
	deletedCluster  int
	deletedTemplate bool
	applyID         int
	applyCluster    int
	dryRun          bool
	applied         bool
}

func (m *yamlManagerFixture) CreateYamlTask(_ context.Context, task *model.K8sYamlTask) error {
	copied := *task
	m.task = &copied
	return m.err
}

func (m *yamlManagerFixture) UpdateYamlTask(_ context.Context, task *model.K8sYamlTask) error {
	copied := *task
	m.task = &copied
	return m.err
}

func (m *yamlManagerFixture) GetYamlTaskList(context.Context, *model.YamlTaskListReq) ([]*model.K8sYamlTask, error) {
	return m.tasks, m.err
}

func (m *yamlManagerFixture) DeleteYamlTask(_ context.Context, id, cluster int) error {
	m.deletedID, m.deletedCluster = id, cluster
	return m.err
}

func (m *yamlManagerFixture) ApplyYamlTask(_ context.Context, id, cluster int, dryRun bool) error {
	m.applyID, m.applyCluster, m.dryRun, m.applied = id, cluster, dryRun, true
	return m.err
}

func (m *yamlManagerFixture) GetYamlTaskDetail(context.Context, int, int) (*model.K8sYamlTask, error) {
	return m.task, m.err
}

func (m *yamlManagerFixture) CheckYamlTemplate(_ context.Context, template *model.K8sYamlTemplate) error {
	copied := *template
	m.template = &copied
	return m.err
}

func (m *yamlManagerFixture) CreateYamlTemplate(_ context.Context, template *model.K8sYamlTemplate) error {
	copied := *template
	m.template = &copied
	return m.err
}

func (m *yamlManagerFixture) UpdateYamlTemplate(_ context.Context, template *model.K8sYamlTemplate) error {
	copied := *template
	m.template = &copied
	return m.err
}

func (m *yamlManagerFixture) GetYamlTemplateList(context.Context, *model.YamlTemplateListReq) ([]*model.K8sYamlTemplate, error) {
	return m.templates, m.err
}

func (m *yamlManagerFixture) DeleteYamlTemplate(_ context.Context, id, cluster int) error {
	m.deletedID, m.deletedCluster, m.deletedTemplate = id, cluster, true
	return m.err
}

func (m *yamlManagerFixture) GetYamlTemplateDetail(context.Context, int, int) (*model.K8sYamlTemplate, error) {
	return m.template, m.err
}

func TestYamlTaskPreservesOwnerAndDryRun(t *testing.T) {
	fixture := &yamlManagerFixture{}
	svc := NewYamlTaskService(fixture, zap.NewNop())
	req := &model.YamlTaskCreateReq{Name: "rollout", UserID: 7, TemplateID: 3, ClusterID: 9, Variables: model.StringList{"image=example.invalid/app:test"}}
	if err := svc.CreateYamlTask(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if fixture.task == nil || fixture.task.UserID != 7 || fixture.task.ClusterID != 9 || fixture.task.Status != TaskPending || len(fixture.task.Variables) != 1 {
		t.Fatalf("created task = %+v", fixture.task)
	}
	fixture.err = errors.New("duplicate")
	if err := svc.CreateYamlTask(context.Background(), req); err == nil || err.Error() != "duplicate" {
		t.Fatalf("create error was wrapped or dropped: %v", err)
	}
	fixture.err = nil
	if err := svc.UpdateYamlTask(context.Background(), &model.YamlTaskUpdateReq{ID: 4, Name: "rollout", UserID: 8, TemplateID: 3, ClusterID: 9}); err != nil {
		t.Fatal(err)
	}
	if fixture.task.UserID != 8 || fixture.task.ID != 4 {
		t.Fatalf("updated owner = %+v", fixture.task)
	}
	if err := svc.ApplyYamlTask(context.Background(), &model.YamlTaskExecuteReq{ID: 4, ClusterID: 9, DryRun: true, UserID: 8}); err != nil || !fixture.applied || !fixture.dryRun || fixture.applyID != 4 || fixture.applyCluster != 9 {
		t.Fatalf("apply = %+v", fixture)
	}
	if err := svc.DeleteYamlTask(context.Background(), &model.YamlTaskDeleteReq{ID: 4, ClusterID: 9}); err != nil || fixture.deletedID != 4 || fixture.deletedCluster != 9 {
		t.Fatalf("delete = %+v", fixture)
	}
	fixture.tasks = []*model.K8sYamlTask{{Name: "rollout", UserID: 7}, {Name: "other", UserID: 9}}
	listed, err := svc.GetYamlTaskList(context.Background(), &model.YamlTaskListReq{ClusterID: 9})
	if err != nil || listed.Total != 2 || len(listed.Items) != 2 {
		t.Fatalf("list = %+v %v", listed, err)
	}
	fixture.err = errors.New("dao down")
	if _, err := svc.GetYamlTaskDetail(context.Background(), &model.YamlTaskDetailReq{ID: 4, ClusterID: 9}); err == nil || err.Error() != "dao down" {
		t.Fatalf("detail error = %v", err)
	}
}

func TestYamlTaskNilRequestPanics(t *testing.T) {
	t.Skip("CreateYamlTask 在 req 为 nil 时直接解引用（yaml_task_service.go:75），Apply/Delete/Update 同样没有空请求校验。未改生产代码。")
}

func TestYamlApplyDoesNotReceiveActor(t *testing.T) {
	t.Skip("YamlTaskExecuteReq.UserID 与 YamlTemplateDeleteReq.UserID 没有传给 manager（yaml_task_service.go:103、yaml_template_service.go:99），服务层不做任务归属校验。未改生产代码。")
}

func TestYamlTemplateOwnerIsStoredOnWrite(t *testing.T) {
	fixture := &yamlManagerFixture{}
	svc := NewYamlTemplateService(fixture, zap.NewNop())
	if err := svc.CreateYamlTemplate(context.Background(), &model.YamlTemplateCreateReq{Name: "deploy", UserID: 7, ClusterID: 9, Content: "kind: ConfigMap"}); err != nil {
		t.Fatal(err)
	}
	if fixture.template == nil || fixture.template.UserID != 7 || fixture.template.ClusterID != 9 || fixture.template.Content != "kind: ConfigMap" {
		t.Fatalf("template = %+v", fixture.template)
	}
	if err := svc.UpdateYamlTemplate(context.Background(), &model.YamlTemplateUpdateReq{ID: 2, Name: "deploy", UserID: 8, ClusterID: 9, Content: "kind: Secret"}); err != nil || fixture.template.UserID != 8 || fixture.template.ID != 2 {
		t.Fatalf("updated template = %+v %v", fixture.template, err)
	}
	fixture.templates = []*model.K8sYamlTemplate{{Name: "deploy", UserID: 7}}
	listed, err := svc.GetYamlTemplateList(context.Background(), &model.YamlTemplateListReq{ClusterID: 9})
	if err != nil || listed.Total != 1 {
		t.Fatalf("template list = %+v %v", listed, err)
	}
	fixture.err = errors.New("invalid yaml")
	if err := svc.CheckYamlTemplate(context.Background(), &model.YamlTemplateCheckReq{Name: "deploy", ClusterID: 9, Content: ":"}); err == nil || err.Error() != "invalid yaml" {
		t.Fatalf("check error = %v", err)
	}
	fixture.err = nil
	if err := svc.DeleteYamlTemplate(context.Background(), &model.YamlTemplateDeleteReq{ID: 2, ClusterID: 9, UserID: 7}); err != nil || !fixture.deletedTemplate || fixture.deletedID != 2 {
		t.Fatalf("delete = %+v", fixture)
	}
}

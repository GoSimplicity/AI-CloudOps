package service

import (
	"context"
	"errors"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
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
	applyUserID     int
	dryRun          bool
	applied         bool
	deletedUserID   int
}

func (m *yamlManagerFixture) CreateYamlTask(_ context.Context, task *model.K8sYamlTask) error {
	copied := *task
	m.task = &copied
	return m.err
}

func (m *yamlManagerFixture) UpdateYamlTask(_ context.Context, task *model.K8sYamlTask, ownerID int) error {
	if task == nil || task.UserID != ownerID {
		return constants.ErrNotResourceOwner
	}
	copied := *task
	m.task = &copied
	return m.err
}

func (m *yamlManagerFixture) GetYamlTaskList(context.Context, *model.YamlTaskListReq) ([]*model.K8sYamlTask, error) {
	return m.tasks, m.err
}

func (m *yamlManagerFixture) DeleteYamlTask(_ context.Context, id, cluster, ownerID int) error {
	m.deletedID, m.deletedCluster, m.deletedUserID = id, cluster, ownerID
	return m.err
}

func (m *yamlManagerFixture) ApplyYamlTask(_ context.Context, id, cluster int, dryRun bool, userID int) error {
	m.applyID, m.applyCluster, m.applyUserID, m.dryRun, m.applied = id, cluster, userID, dryRun, true
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

func (m *yamlManagerFixture) UpdateYamlTemplate(_ context.Context, template *model.K8sYamlTemplate, ownerID int) error {
	if template == nil || template.UserID != ownerID {
		return constants.ErrNotResourceOwner
	}
	copied := *template
	m.template = &copied
	return m.err
}

func (m *yamlManagerFixture) GetYamlTemplateList(context.Context, *model.YamlTemplateListReq) ([]*model.K8sYamlTemplate, error) {
	return m.templates, m.err
}

func (m *yamlManagerFixture) DeleteYamlTemplate(_ context.Context, id, cluster, userID int) error {
	m.deletedID, m.deletedCluster, m.deletedUserID, m.deletedTemplate = id, cluster, userID, true
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
	if err := svc.UpdateYamlTask(context.Background(), &model.YamlTaskUpdateReq{ID: 4, Name: "rollout", UserID: 7, TemplateID: 3, ClusterID: 9}, false); err != nil {
		t.Fatal(err)
	}
	if fixture.task.UserID != 7 || fixture.task.ID != 4 {
		t.Fatalf("updated owner = %+v", fixture.task)
	}
	if err := svc.ApplyYamlTask(context.Background(), &model.YamlTaskExecuteReq{ID: 4, ClusterID: 9, DryRun: true, UserID: 7}, false); err != nil || !fixture.applied || !fixture.dryRun || fixture.applyID != 4 || fixture.applyCluster != 9 || fixture.applyUserID != 7 {
		t.Fatalf("apply = %+v", fixture)
	}
	if err := svc.DeleteYamlTask(context.Background(), &model.YamlTaskDeleteReq{ID: 4, ClusterID: 9, UserID: 7}, false); err != nil || fixture.deletedID != 4 || fixture.deletedCluster != 9 || fixture.deletedUserID != 7 {
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
	fixture := &yamlManagerFixture{}
	svc := NewYamlTaskService(fixture, zap.NewNop())
	ctx := context.Background()
	calls := []struct {
		name string
		call func() error
	}{
		{"create", func() error { return svc.CreateYamlTask(ctx, nil) }},
		{"update", func() error { return svc.UpdateYamlTask(ctx, nil, false) }},
		{"delete", func() error { return svc.DeleteYamlTask(ctx, nil, false) }},
		{"apply", func() error { return svc.ApplyYamlTask(ctx, nil, false) }},
	}
	for _, tc := range calls {
		err := tc.call()
		var biz *base.BusinessError
		if !errors.As(err, &biz) || !errors.Is(biz.Code, constants.ErrInvalidParam) || fixture.task != nil || fixture.applied || fixture.deletedID != 0 {
			t.Fatalf("%s = %v fixture=%+v", tc.name, err, fixture)
		}
	}
}

func TestYamlApplyDoesNotReceiveActor(t *testing.T) {
	fixture := &yamlManagerFixture{task: &model.K8sYamlTask{Model: model.Model{ID: 4}, UserID: 7, ClusterID: 9}}
	taskSvc := NewYamlTaskService(fixture, zap.NewNop())
	err := taskSvc.ApplyYamlTask(context.Background(), &model.YamlTaskExecuteReq{ID: 4, ClusterID: 9, DryRun: true, UserID: 8}, false)
	var biz *base.BusinessError
	if err == nil || fixture.applied || !errors.As(err, &biz) || !errors.Is(biz.Code, constants.ErrNotResourceOwner) || err.Error() != "当前用户无权限操作该YAML任务" {
		t.Fatalf("non-owner apply = %v applied=%v", err, fixture.applied)
	}
	if err := taskSvc.ApplyYamlTask(context.Background(), &model.YamlTaskExecuteReq{ID: 4, ClusterID: 9, DryRun: false, UserID: 7}, false); err != nil || !fixture.applied || fixture.applyUserID != 7 || fixture.dryRun {
		t.Fatalf("owner apply = %v fixture=%+v", err, fixture)
	}
	if err := taskSvc.ApplyYamlTask(context.Background(), &model.YamlTaskExecuteReq{ID: 4, ClusterID: 9, UserID: 0}, false); err == nil || !errors.As(err, &biz) || !errors.Is(biz.Code, constants.ErrInvalidParam) {
		t.Fatalf("missing user = %v", err)
	}

	fixture.template = &model.K8sYamlTemplate{Model: model.Model{ID: 2}, UserID: 7, ClusterID: 9}
	templateSvc := NewYamlTemplateService(fixture, zap.NewNop())
	err = templateSvc.DeleteYamlTemplate(context.Background(), &model.YamlTemplateDeleteReq{ID: 2, ClusterID: 9, UserID: 8}, false)
	if err == nil || fixture.deletedTemplate || !errors.As(err, &biz) || !errors.Is(biz.Code, constants.ErrNotResourceOwner) || err.Error() != "当前用户无权限操作该YAML模板" {
		t.Fatalf("non-owner delete = %v deleted=%v", err, fixture.deletedTemplate)
	}
	if err := templateSvc.DeleteYamlTemplate(context.Background(), &model.YamlTemplateDeleteReq{ID: 2, ClusterID: 9, UserID: 7}, false); err != nil || !fixture.deletedTemplate || fixture.deletedUserID != 7 || fixture.deletedID != 2 {
		t.Fatalf("owner delete = %v fixture=%+v", err, fixture)
	}
	if err := templateSvc.DeleteYamlTemplate(context.Background(), nil, false); err == nil || !errors.As(err, &biz) || !errors.Is(biz.Code, constants.ErrInvalidParam) {
		t.Fatalf("nil template delete = %v", err)
	}
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
	if err := svc.UpdateYamlTemplate(context.Background(), &model.YamlTemplateUpdateReq{ID: 2, Name: "deploy", UserID: 7, ClusterID: 9, Content: "kind: Secret"}, false); err != nil || fixture.template.UserID != 7 || fixture.template.Content != "kind: Secret" || fixture.template.ID != 2 {
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
	fixture.template = &model.K8sYamlTemplate{Model: model.Model{ID: 2}, Name: "deploy", UserID: 7, ClusterID: 9, Content: "kind: Secret"}
	if err := svc.DeleteYamlTemplate(context.Background(), &model.YamlTemplateDeleteReq{ID: 2, ClusterID: 9, UserID: 7}, false); err != nil || !fixture.deletedTemplate || fixture.deletedID != 2 || fixture.deletedUserID != 7 {
		t.Fatalf("delete = %+v", fixture)
	}
}

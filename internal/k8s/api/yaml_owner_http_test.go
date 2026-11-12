package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	yamlOwnerA = 7
	yamlOwnerB = 8
	yamlAdmin  = 99
)

type yamlClusterStub struct {
	dao.ClusterDAO
}

func (yamlClusterStub) GetClusterByID(context.Context, int) (*model.K8sCluster, error) {
	return &model.K8sCluster{Model: model.Model{ID: 9}, Name: "owned"}, nil
}

type yamlOwnerHarness struct {
	db        *gorm.DB
	tasks     *K8sYamlTaskHandler
	templates *K8sYamlTemplateHandler
	logs      *observer.ObservedLogs
}

func newYAMLOwnerHarness(t *testing.T) *yamlOwnerHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.K8sYamlTask{}, &model.K8sYamlTemplate{}); err != nil {
		t.Fatal(err)
	}
	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	taskDAO := dao.NewYamlTaskDAO(db, logger)
	templateDAO := dao.NewYamlTemplateDAO(db, logger)
	yamlManager := manager.NewYamlManager(templateDAO, taskDAO, yamlClusterStub{}, nil, logger)
	return &yamlOwnerHarness{
		db:        db,
		tasks:     NewK8sYamlTaskHandler(service.NewYamlTaskService(yamlManager, logger)),
		templates: NewK8sYamlTemplateHandler(service.NewYamlTemplateService(yamlManager, logger)),
		logs:      logs,
	}
}

func (h *yamlOwnerHarness) seed(t *testing.T) (*model.K8sYamlTemplate, *model.K8sYamlTask) {
	t.Helper()
	template := &model.K8sYamlTemplate{
		Name: "keep", Content: "kind: ConfigMap\nmetadata:\n  name: keep\n", ClusterID: 9, UserID: yamlOwnerA,
	}
	if err := h.db.Create(template).Error; err != nil {
		t.Fatal(err)
	}
	task := &model.K8sYamlTask{
		Name: "rollout", UserID: yamlOwnerA, TemplateID: template.ID, ClusterID: 9,
		Variables: model.StringList{"image=example.invalid/app:test"}, Status: "Pending",
	}
	if err := h.db.Create(task).Error; err != nil {
		t.Fatal(err)
	}
	return template, task
}

func invokeYAML(t *testing.T, handler gin.HandlerFunc, method, target, body string, uid int, super bool, params gin.Params) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = params
	ctx.Set("user", jwt.UserClaims{Uid: uid})
	permission.MarkSuperAdmin(ctx, super)
	handler(ctx)
	return recorder
}

func assertForbidden(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	var response base.ApiResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("response %s: %v", recorder.Body.String(), err)
	}
	if recorder.Code != http.StatusForbidden || response.Code != http.StatusForbidden || !strings.Contains(response.Message, "无权限") {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func (h *yamlOwnerHarness) assertTaskUnchanged(t *testing.T, task *model.K8sYamlTask) {
	t.Helper()
	var got model.K8sYamlTask
	if err := h.db.First(&got, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.UserID != yamlOwnerA || got.Name != "rollout" || len(got.Variables) != 1 || got.Variables[0] != "image=example.invalid/app:test" {
		t.Fatalf("task changed: %+v", got)
	}
}

func (h *yamlOwnerHarness) assertTemplateUnchanged(t *testing.T, template *model.K8sYamlTemplate) {
	t.Helper()
	var got model.K8sYamlTemplate
	if err := h.db.First(&got, template.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.UserID != yamlOwnerA || got.Content != template.Content || got.Name != "keep" {
		t.Fatalf("template changed: %+v", got)
	}
}

func TestNonOwnerYamlTaskUpdateIsForbidden(t *testing.T) {
	h := newYAMLOwnerHarness(t)
	_, task := h.seed(t)
	recorder := invokeYAML(t, h.tasks.UpdateYamlTask, http.MethodPost, "/api/k8s/yaml_task/9/1/update",
		`{"name":"stolen","template_id":1,"user_id":7,"variables":["image=evil"]}`, yamlOwnerB, false,
		gin.Params{{Key: "cluster_id", Value: "9"}, {Key: "id", Value: strconv.Itoa(task.ID)}})
	assertForbidden(t, recorder)
	h.assertTaskUnchanged(t, task)
}

func TestNonOwnerYamlTemplateUpdateIsForbidden(t *testing.T) {
	h := newYAMLOwnerHarness(t)
	template, _ := h.seed(t)
	recorder := invokeYAML(t, h.templates.UpdateYamlTemplate, http.MethodPost, "/api/k8s/yaml_template/9/1/update",
		`{"name":"stolen","user_id":7,"content":"kind: Secret\nmetadata:\n  name: stolen\n"}`, yamlOwnerB, false,
		gin.Params{{Key: "cluster_id", Value: "9"}, {Key: "id", Value: strconv.Itoa(template.ID)}})
	assertForbidden(t, recorder)
	h.assertTemplateUnchanged(t, template)
}

func TestNonOwnerYamlTaskDeleteIsForbidden(t *testing.T) {
	h := newYAMLOwnerHarness(t)
	_, task := h.seed(t)
	recorder := invokeYAML(t, h.tasks.DeleteYamlTask, http.MethodDelete, "/api/k8s/yaml_task/9/1/delete",
		`{"user_id":7}`, yamlOwnerB, false,
		gin.Params{{Key: "cluster_id", Value: "9"}, {Key: "id", Value: strconv.Itoa(task.ID)}})
	assertForbidden(t, recorder)
	h.assertTaskUnchanged(t, task)
}

func TestNonOwnerYamlTaskUpdateThenApplyIsForbidden(t *testing.T) {
	h := newYAMLOwnerHarness(t)
	_, task := h.seed(t)
	params := gin.Params{{Key: "cluster_id", Value: "9"}, {Key: "id", Value: strconv.Itoa(task.ID)}}
	updated := invokeYAML(t, h.tasks.UpdateYamlTask, http.MethodPost, "/api/k8s/yaml_task/9/1/update",
		`{"name":"stolen","template_id":1,"user_id":7,"variables":["image=evil"]}`, yamlOwnerB, false, params)
	assertForbidden(t, updated)
	applied := invokeYAML(t, h.tasks.ApplyYamlTask, http.MethodPost, "/api/k8s/yaml_task/9/1/apply",
		`{"dry_run":true,"user_id":7}`, yamlOwnerB, false, params)
	assertForbidden(t, applied)
	h.assertTaskUnchanged(t, task)
}

func TestSuperAdminUpdateKeepsOriginalOwner(t *testing.T) {
	h := newYAMLOwnerHarness(t)
	template, task := h.seed(t)
	before := h.logs.Len()
	taskResp := invokeYAML(t, h.tasks.UpdateYamlTask, http.MethodPost, "/api/k8s/yaml_task/9/1/update",
		`{"name":"admin-edit","template_id":`+strconv.Itoa(template.ID)+`,"user_id":`+strconv.Itoa(yamlAdmin)+`,"variables":["image=admin"]}`,
		yamlAdmin, true, gin.Params{{Key: "cluster_id", Value: "9"}, {Key: "id", Value: strconv.Itoa(task.ID)}})
	if taskResp.Code != http.StatusOK {
		t.Fatalf("admin task update status=%d body=%s", taskResp.Code, taskResp.Body.String())
	}
	var gotTask model.K8sYamlTask
	if err := h.db.First(&gotTask, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotTask.UserID != yamlOwnerA || gotTask.Name != "admin-edit" || len(gotTask.Variables) != 1 || gotTask.Variables[0] != "image=admin" {
		t.Fatalf("admin task update changed owner or missed the edit: %+v", gotTask)
	}
	templateResp := invokeYAML(t, h.templates.UpdateYamlTemplate, http.MethodPost, "/api/k8s/yaml_template/9/1/update",
		`{"name":"admin-template","user_id":`+strconv.Itoa(yamlAdmin)+`,"content":"kind: ConfigMap\nmetadata:\n  name: admin\n"}`,
		yamlAdmin, true, gin.Params{{Key: "cluster_id", Value: "9"}, {Key: "id", Value: strconv.Itoa(template.ID)}})
	if templateResp.Code != http.StatusOK {
		t.Fatalf("admin template update status=%d body=%s", templateResp.Code, templateResp.Body.String())
	}
	var gotTemplate model.K8sYamlTemplate
	if err := h.db.First(&gotTemplate, template.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotTemplate.UserID != yamlOwnerA || gotTemplate.Name != "admin-template" || !strings.Contains(gotTemplate.Content, "name: admin") {
		t.Fatalf("admin template update changed owner or missed the edit: %+v", gotTemplate)
	}
	if !auditHas(h.logs.All()[before:], "yaml_task", task.ID) || !auditHas(h.logs.All()[before:], "yaml_template", template.ID) {
		t.Fatalf("admin update audit missing: %+v", h.logs.All()[before:])
	}
}

func TestSuperAdminCanDeleteOwnedYamlTaskAndLogs(t *testing.T) {
	h := newYAMLOwnerHarness(t)
	_, task := h.seed(t)
	before := h.logs.Len()
	recorder := invokeYAML(t, h.tasks.DeleteYamlTask, http.MethodDelete, "/api/k8s/yaml_task/9/1/delete",
		`{}`, yamlAdmin, true,
		gin.Params{{Key: "cluster_id", Value: "9"}, {Key: "id", Value: strconv.Itoa(task.ID)}})
	if recorder.Code != http.StatusOK {
		t.Fatalf("admin delete status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got model.K8sYamlTask
	if err := h.db.First(&got, task.ID).Error; err == nil {
		t.Fatalf("admin delete left task %+v", got)
	}
	if !auditHas(h.logs.All()[before:], "yaml_task", task.ID) {
		t.Fatalf("audit log missing: %+v", h.logs.All()[before:])
	}
}

func auditHas(entries []observer.LoggedEntry, objectType string, objectID int) bool {
	for _, entry := range entries {
		if entry.Message != "超级管理员操作他人YAML资源" {
			continue
		}
		fields := entry.ContextMap()
		if fields["operator_user_id"] == int64(yamlAdmin) && fields["owner_user_id"] == int64(yamlOwnerA) && fields["object_type"] == objectType && fields["object_id"] == int64(objectID) {
			return true
		}
	}
	return false
}

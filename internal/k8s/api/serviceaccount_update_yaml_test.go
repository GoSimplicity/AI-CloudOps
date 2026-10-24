package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// yamlUpdateSAManager 模拟 manager 层的两种 YAML 更新路径
type yamlUpdateSAManager struct {
	*handlerSAManager
	strictErr            error
	strictCalls, retries int
	strictRV             string
}

func (m *yamlUpdateSAManager) UpdateServiceAccountWithResourceVersion(_ context.Context, _ int, _, _, rv string, _ k8sutils.ServiceAccountMutateFunc) error {
	m.strictCalls++
	m.strictRV = rv
	return m.strictErr
}

func (m *yamlUpdateSAManager) UpdateServiceAccountWithRetry(context.Context, int, string, string, k8sutils.ServiceAccountMutateFunc) error {
	m.retries++
	return nil
}

func putSAYaml(engine *gin.Engine, yaml string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]interface{}{"cluster_id": 3, "namespace": "ops", "name": "app", "yaml_content": yaml})
	req := httptest.NewRequest("PUT", "/api/k8s/serviceaccount/3/ops/app/update/yaml", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

// SEC-29(A)：YAML 带过期 resourceVersion → HTTP 409 + code=1（respondBusinessError 映射 BusinessError{ErrK8sResourceConflict}）；
// 版本正确 → 200；不带版本 → 走 RetryOnConflict 路径
func TestUpdateServiceAccountYamlConflictMapsTo409(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withRV := "apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: app\n  resourceVersion: \"4\"\n"
	conflict := apierrors.NewConflict(schema.GroupResource{Resource: "serviceaccounts"}, "app", errors.New("the object has been modified"))

	mgr := &yamlUpdateSAManager{handlerSAManager: &handlerSAManager{}, strictErr: conflict}
	rec := putSAYaml(newSATokenEngine(service.NewServiceAccountService(mgr, &recordingAudit{}, zap.NewNop())), withRV)
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if rec.Code != 409 || resp.Code != 1 || !strings.Contains(resp.Message, "resourceVersion") {
		t.Fatalf("版本冲突应返回 HTTP 409 + code=1: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if mgr.strictCalls != 1 || mgr.retries != 0 || mgr.strictRV != "4" {
		t.Fatalf("应只走一次乐观锁更新、不重试: strict=%d retry=%d rv=%q", mgr.strictCalls, mgr.retries, mgr.strictRV)
	}

	mgr = &yamlUpdateSAManager{handlerSAManager: &handlerSAManager{}}
	if rec := putSAYaml(newSATokenEngine(service.NewServiceAccountService(mgr, &recordingAudit{}, zap.NewNop())), withRV); rec.Code != 200 || mgr.strictCalls != 1 {
		t.Fatalf("版本正确应返回 200: status=%d body=%s", rec.Code, rec.Body.String())
	}

	mgr = &yamlUpdateSAManager{handlerSAManager: &handlerSAManager{}}
	noRV := "apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: app\n"
	if rec := putSAYaml(newSATokenEngine(service.NewServiceAccountService(mgr, &recordingAudit{}, zap.NewNop())), noRV); rec.Code != 200 || mgr.retries != 1 || mgr.strictCalls != 0 {
		t.Fatalf("不带 resourceVersion 应走重试路径: status=%d strict=%d retry=%d", rec.Code, mgr.strictCalls, mgr.retries)
	}

	// 其他错误保持项目约定：HTTP 200 + code=1
	mgr = &yamlUpdateSAManager{handlerSAManager: &handlerSAManager{}, strictErr: errors.New("boom")}
	rec = putSAYaml(newSATokenEngine(service.NewServiceAccountService(mgr, &recordingAudit{}, zap.NewNop())), withRV)
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if rec.Code != 200 || resp.Code != 1 {
		t.Fatalf("非冲突错误应保持 200 + code=1: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

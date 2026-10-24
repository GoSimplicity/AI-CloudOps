package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/connection"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// 测试用的假私钥/证书内容，仅用于断言响应中不会出现，不是任何真实凭据
const (
	fakeKeyMarker  = "FIXTURE-PRIVATE-KEY-MATERIAL-DO-NOT-LEAK"
	fakeCertMarker = "FIXTURE-CLIENT-CERT-MATERIAL-DO-NOT-LEAK"
	fakeToken      = "fixture-token-do-not-leak"
)

func fixturePEM(kind, marker string) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: kind, Bytes: []byte(marker)})
}

func kubeconfigWithCredentials(server string, caPEM []byte) string {
	b64 := base64.StdEncoding.EncodeToString
	return fmt.Sprintf(`apiVersion: v1
kind: Config
current-context: fixture
clusters:
- name: fixture
  cluster:
    server: %s
    certificate-authority-data: %s
users:
- name: fixture
  user:
    token: %s
- name: cert-user
  user:
    client-certificate-data: %s
    client-key-data: %s
contexts:
- name: fixture
  context: {cluster: fixture, user: fixture}
`, server, b64(caPEM), fakeToken, b64(fixturePEM("CERTIFICATE", fakeCertMarker)), b64(fixturePEM("RSA PRIVATE KEY", fakeKeyMarker)))
}

type connectingManager struct {
	manager.ClusterManager
	updated []*model.K8sCluster
	connErr error
}

// UpdateCluster 模拟真实的客户端初始化：解析 kubeconfig 并请求 API Server
func (m *connectingManager) UpdateCluster(ctx context.Context, cluster *model.K8sCluster) error {
	m.updated = append(m.updated, cluster)
	cfg, err := connection.Parse(cluster.KubeConfigContent)
	if err != nil {
		m.connErr = err
		return err
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err == nil {
		_, err = cs.Discovery().ServerVersion()
	}
	m.connErr = err
	return err
}

func setupClusterAPI(t *testing.T) (*gin.Engine, *gorm.DB, *connectingManager, *httptest.Server) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	apiServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+fakeToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"major":"1","minor":"30","gitVersion":"v1.30.0"}`))
	}))
	t.Cleanup(apiServer.Close)

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.K8sCluster{}); err != nil {
		t.Fatal(err)
	}

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: apiServer.Certificate().Raw})
	cluster := &model.K8sCluster{
		Name:              "kubeconfig-fixture",
		ApiServerAddr:     apiServer.URL,
		KubeConfigContent: kubeconfigWithCredentials(apiServer.URL, caPEM),
		Status:            model.StatusRunning,
		Env:               model.EnvDev,
	}
	if err := db.Create(cluster).Error; err != nil {
		t.Fatal(err)
	}
	empty := &model.K8sCluster{Name: "no-kubeconfig", ApiServerAddr: "https://127.0.0.1:6443", Status: model.StatusStopped}
	if err := db.Create(empty).Error; err != nil {
		t.Fatal(err)
	}

	mgr := &connectingManager{}
	svc := service.NewClusterService(dao.NewClusterDAO(db, zap.NewNop()), nil, mgr, zap.NewNop())
	r := gin.New()
	NewK8sClusterHandler(svc).RegisterRouters(r)
	return r, db, mgr, apiServer
}

func assertNoKubeconfigLeak(t *testing.T, body string) {
	t.Helper()
	b64 := base64.StdEncoding.EncodeToString
	forbidden := []string{
		"kube_config_content", "client-key-data", "client-certificate-data", "certificate-authority-data",
		"PRIVATE KEY", "BEGIN CERTIFICATE", fakeKeyMarker, fakeCertMarker, fakeToken,
		b64(fixturePEM("RSA PRIVATE KEY", fakeKeyMarker)), b64(fixturePEM("CERTIFICATE", fakeCertMarker)),
	}
	for _, f := range forbidden {
		if strings.Contains(body, f) {
			t.Fatalf("响应泄露了 kubeconfig 敏感内容 %q: %s", f, body)
		}
	}
}

func doRequest(r *gin.Engine, method, path string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestClusterDetailAndListDoNotExposeKubeconfig(t *testing.T) {
	r, _, _, _ := setupClusterAPI(t)

	detail := doRequest(r, http.MethodGet, "/api/k8s/cluster/1/detail", nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status %d: %s", detail.Code, detail.Body.String())
	}
	assertNoKubeconfigLeak(t, detail.Body.String())
	var detailResp struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(detail.Body.Bytes(), &detailResp); err != nil {
		t.Fatal(err)
	}
	if detailResp.Data["has_kube_config"] != true || detailResp.Data["name"] != "kubeconfig-fixture" {
		t.Fatalf("详情应返回 has_kube_config=true 及其他字段: %s", detail.Body.String())
	}

	list := doRequest(r, http.MethodGet, "/api/k8s/cluster/list?page=1&size=10", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list status %d: %s", list.Code, list.Body.String())
	}
	assertNoKubeconfigLeak(t, list.Body.String())
	var listResp struct {
		Data struct {
			Items []map[string]any `json:"items"`
			Total int              `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &listResp); err != nil {
		t.Fatal(err)
	}
	if listResp.Data.Total != 2 || len(listResp.Data.Items) != 2 {
		t.Fatalf("列表数量不符: %s", list.Body.String())
	}
	got := map[string]any{}
	for _, item := range listResp.Data.Items {
		got[item["name"].(string)] = item["has_kube_config"]
	}
	if got["kubeconfig-fixture"] != true || got["no-kubeconfig"] != false {
		t.Fatalf("列表 has_kube_config 不正确: %v", got)
	}
}

func TestUpdateClusterWithoutKubeconfigKeepsStoredValueAndConnects(t *testing.T) {
	r, db, mgr, _ := setupClusterAPI(t)
	var before model.K8sCluster
	if err := db.First(&before, 1).Error; err != nil {
		t.Fatal(err)
	}

	for _, body := range []string{
		`{"name":"renamed-fixture","version":"v1.30.0"}`,                        // 不传
		`{"name":"renamed-again","version":"v1.30.0","kube_config_content":""}`, // 空字符串
	} {
		w := doRequest(r, http.MethodPut, "/api/k8s/cluster/1/update", []byte(body))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"code":0`) {
			t.Fatalf("更新失败 %d: %s", w.Code, w.Body.String())
		}
		var after model.K8sCluster
		if err := db.First(&after, 1).Error; err != nil {
			t.Fatal(err)
		}
		if after.KubeConfigContent != before.KubeConfigContent {
			t.Fatal("未传 kubeconfig 时数据库中的原值被修改")
		}
		if mgr.connErr != nil {
			t.Fatalf("保留原 kubeconfig 后应仍能连接集群: %v", mgr.connErr)
		}
		last := mgr.updated[len(mgr.updated)-1]
		if last.KubeConfigContent != before.KubeConfigContent {
			t.Fatal("客户端初始化未使用保留的 kubeconfig")
		}
	}

	var after model.K8sCluster
	if err := db.First(&after, 1).Error; err != nil {
		t.Fatal(err)
	}
	if after.Name != "renamed-again" {
		t.Fatalf("其他字段应正常更新, got %q", after.Name)
	}
}

func TestUpdateClusterRejectsInvalidReplacementKubeconfig(t *testing.T) {
	r, db, mgr, _ := setupClusterAPI(t)
	var before model.K8sCluster
	if err := db.First(&before, 1).Error; err != nil {
		t.Fatal(err)
	}
	w := doRequest(r, http.MethodPut, "/api/k8s/cluster/1/update", []byte(`{"name":"x","kube_config_content":"not: [valid"}`))
	if strings.Contains(w.Body.String(), `"code":0`) {
		t.Fatalf("非法 kubeconfig 应被拒绝: %s", w.Body.String())
	}
	var after model.K8sCluster
	if err := db.First(&after, 1).Error; err != nil {
		t.Fatal(err)
	}
	if after.KubeConfigContent != before.KubeConfigContent || len(mgr.updated) != 0 {
		t.Fatal("校验失败时不应修改已存 kubeconfig")
	}
}

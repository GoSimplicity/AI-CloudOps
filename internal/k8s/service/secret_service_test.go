package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	systemService "github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// 测试用的明文（非真实凭据），用于断言任何脱敏输出中都不出现
const (
	fixtureAlpha   = "plain-alpha-fixture-001"
	fixtureBeta    = "plain-beta-fixture-0002"
	fixtureApplied = "plain-applied-fixture-03"
)

type secretManagerFixture struct {
	manager.SecretManager
	secret  *corev1.Secret
	getErr  error
	updated *corev1.Secret
	created *corev1.Secret
}

func (m *secretManagerFixture) GetSecret(_ context.Context, _ int, _, _ string) (*corev1.Secret, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.secret.DeepCopy(), nil
}

func (m *secretManagerFixture) ListSecrets(_ context.Context, _ int, _ string) (*corev1.SecretList, error) {
	return &corev1.SecretList{Items: []corev1.Secret{*m.secret.DeepCopy()}}, nil
}

func (m *secretManagerFixture) UpdateSecret(_ context.Context, _ int, secret *corev1.Secret) (*corev1.Secret, error) {
	m.updated = secret.DeepCopy()
	return secret, nil
}

func (m *secretManagerFixture) CreateSecret(_ context.Context, _ int, secret *corev1.Secret) (*corev1.Secret, error) {
	m.created = secret.DeepCopy()
	return secret, nil
}

type auditFixture struct {
	systemService.AuditService
	logs []*model.CreateAuditLogRequest
	err  error
}

func (a *auditFixture) CreateAuditLog(_ context.Context, req *model.CreateAuditLogRequest) error {
	if a.err != nil {
		return a.err
	}
	cp := *req
	a.logs = append(a.logs, &cp)
	return nil
}

func newSecretFixture() *corev1.Secret {
	applied, _ := json.Marshal(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata":   map[string]interface{}{"name": "app", "namespace": "ops"},
		"data":       map[string]string{"alpha": base64.StdEncoding.EncodeToString([]byte(fixtureApplied))},
		"stringData": map[string]string{"beta": fixtureApplied},
	})
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "app",
			Namespace:         "ops",
			UID:               "uid-1",
			ResourceVersion:   "42",
			CreationTimestamp: metav1.NewTime(time.Unix(100, 0)),
			Labels:            map[string]string{"app": "demo"},
			Annotations: map[string]string{
				k8sutils.LastAppliedConfigAnnotation: string(applied),
				"owner":                              "team-a",
			},
		},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"alpha": []byte(fixtureAlpha), "beta": []byte(fixtureBeta)},
		StringData: map[string]string{"gamma": fixtureBeta},
	}
}

// assertNoPlaintext 断言输出中不含任何明文及其 base64 形式
func assertNoPlaintext(t *testing.T, where, out string) {
	t.Helper()
	for _, v := range []string{fixtureAlpha, fixtureBeta, fixtureApplied} {
		if strings.Contains(out, v) || strings.Contains(out, base64.StdEncoding.EncodeToString([]byte(v))) {
			t.Fatalf("%s 泄露了明文值", where)
		}
	}
	if strings.Contains(out, k8sutils.LastAppliedConfigAnnotation) {
		t.Fatalf("%s 仍包含 last-applied-configuration 注解", where)
	}
}

func newTestSecretService(m *secretManagerFixture, a *auditFixture) SecretService {
	return NewSecretService(m, a, zap.NewNop())
}

func TestSecretListAndDetailAreRedacted(t *testing.T) {
	m := &secretManagerFixture{secret: newSecretFixture()}
	svc := newTestSecretService(m, &auditFixture{})

	list, err := svc.GetSecretList(context.Background(), &model.GetSecretListReq{ClusterID: 1, Namespace: "ops"})
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("list err=%v items=%d", err, len(list.Items))
	}
	detail, err := svc.GetSecret(context.Background(), &model.GetSecretDetailsReq{ClusterID: 1, Namespace: "ops", Name: "app"})
	if err != nil {
		t.Fatal(err)
	}

	for where, item := range map[string]*model.K8sSecret{"list": list.Items[0], "detail": detail} {
		raw, err := json.Marshal(item)
		if err != nil {
			t.Fatal(err)
		}
		assertNoPlaintext(t, where, string(raw))

		if !item.Redacted {
			t.Fatalf("%s 未标记 redacted", where)
		}
		if len(item.Data) != 2 || len(item.Data["alpha"]) != 0 || len(item.Data["beta"]) != 0 {
			t.Fatalf("%s data 应只保留键且值为空: %v", where, item.Data)
		}
		if v, ok := item.StringData["gamma"]; !ok || v != "" {
			t.Fatalf("%s string_data 应只保留键", where)
		}
		wantKeys := []model.SecretKeyInfo{{Key: "alpha", Size: len(fixtureAlpha)}, {Key: "beta", Size: len(fixtureBeta)}}
		if len(item.DataKeys) != 2 || item.DataKeys[0] != wantKeys[0] || item.DataKeys[1] != wantKeys[1] {
			t.Fatalf("%s data_keys 不符: %+v", where, item.DataKeys)
		}
		if len(item.StringDataKeys) != 1 || item.StringDataKeys[0] != (model.SecretKeyInfo{Key: "gamma", Size: len(fixtureBeta)}) {
			t.Fatalf("%s string_data_keys 不符: %+v", where, item.StringDataKeys)
		}
		if item.Annotations["owner"] != "team-a" {
			t.Fatalf("%s 普通注解应保留", where)
		}
		if item.DataCount != 3 {
			t.Fatalf("%s data_count=%d", where, item.DataCount)
		}
	}

	// 原始对象不被修改
	if string(m.secret.Data["alpha"]) != fixtureAlpha {
		t.Fatal("脱敏不应修改原始对象")
	}
}

func TestSecretYAMLIsRedacted(t *testing.T) {
	m := &secretManagerFixture{secret: newSecretFixture()}
	svc := newTestSecretService(m, &auditFixture{})
	out, err := svc.GetSecretYAML(context.Background(), &model.GetSecretYamlReq{ClusterID: 1, Namespace: "ops", Name: "app"})
	if err != nil {
		t.Fatal(err)
	}
	assertNoPlaintext(t, "yaml", out.YAML)

	var obj struct {
		Data       map[string]string `json:"data"`
		StringData map[string]string `json:"stringData"`
		Metadata   struct {
			Annotations map[string]string `json:"annotations"`
		} `json:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(out.YAML), &obj); err != nil {
		t.Fatal(err)
	}
	if obj.Data["alpha"] != k8sutils.SecretRedactedPlaceholder || obj.Data["beta"] != k8sutils.SecretRedactedPlaceholder || obj.StringData["gamma"] != k8sutils.SecretRedactedPlaceholder {
		t.Fatalf("data/stringData 应替换为占位符: %+v", obj)
	}
	if obj.Metadata.Annotations["owner"] != "team-a" {
		t.Fatal("普通注解应保留")
	}
}

func TestRevealSecretReturnsValuesAndAuditsWithoutValues(t *testing.T) {
	m := &secretManagerFixture{secret: newSecretFixture()}
	a := &auditFixture{}
	svc := newTestSecretService(m, a)
	actor := model.SecretRevealActor{UserID: 7, Username: "alice", IP: "10.0.0.1", UserAgent: "ua", Endpoint: "/api/k8s/secret/1/ops/app/reveal"}

	out, err := svc.RevealSecret(context.Background(), &model.RevealSecretReq{ClusterID: 1, Namespace: "ops", Name: "app"}, actor)
	if err != nil {
		t.Fatal(err)
	}
	if string(out.Data["alpha"]) != fixtureAlpha || string(out.Data["beta"]) != fixtureBeta || out.StringData["gamma"] != fixtureBeta {
		t.Fatal("reveal 未返回明文")
	}
	raw, _ := json.Marshal(out)
	if !strings.Contains(string(raw), base64.StdEncoding.EncodeToString([]byte(fixtureAlpha))) {
		t.Fatal("reveal data 应保持 base64 编码")
	}
	if strings.Join(out.Keys, ",") != "alpha,beta,gamma" {
		t.Fatalf("keys=%v", out.Keys)
	}

	if len(a.logs) != 1 {
		t.Fatalf("应写 1 条审计，实际 %d", len(a.logs))
	}
	log := a.logs[0]
	auditRaw, _ := json.Marshal(log)
	assertNoPlaintext(t, "audit", string(auditRaw))
	if log.ResponseBody != nil {
		t.Fatal("审计不应记录响应体")
	}
	if log.UserID != 7 || log.OperationType != "REVEAL" || log.TargetType != "k8s_secret" || log.TargetID != "1/ops/app" || log.ErrorMsg != "" || log.Endpoint != actor.Endpoint {
		t.Fatalf("审计元数据不符: %+v", log)
	}
	var rec model.SecretRevealAuditRecord
	if err := json.Unmarshal(log.RequestBody, &rec); err != nil {
		t.Fatal(err)
	}
	if rec.Username != "alice" || rec.ClusterID != 1 || rec.Namespace != "ops" || rec.Name != "app" || strings.Join(rec.Keys, ",") != "alpha,beta,gamma" || !rec.Success || rec.RevealedAt == "" {
		t.Fatalf("审计内容不符: %+v", rec)
	}
}

func TestRevealSecretSelectedKeys(t *testing.T) {
	m := &secretManagerFixture{secret: newSecretFixture()}
	a := &auditFixture{}
	svc := newTestSecretService(m, a)

	out, err := svc.RevealSecret(context.Background(), &model.RevealSecretReq{ClusterID: 1, Namespace: "ops", Name: "app", Keys: []string{"beta", "beta, alpha"}}, model.SecretRevealActor{UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Data) != 2 || out.StringData != nil || strings.Join(out.Keys, ",") != "alpha,beta" {
		t.Fatalf("按键过滤失败: %+v", out.Keys)
	}

	if _, err := svc.RevealSecret(context.Background(), &model.RevealSecretReq{ClusterID: 1, Namespace: "ops", Name: "app", Keys: []string{"missing"}}, model.SecretRevealActor{UserID: 1}); err == nil {
		t.Fatal("不存在的键应报错")
	}
	if len(a.logs) != 2 || a.logs[1].ErrorMsg == "" {
		t.Fatal("失败的 reveal 也应写审计")
	}
}

func TestRevealSecretFailsClosedWhenAuditFails(t *testing.T) {
	m := &secretManagerFixture{secret: newSecretFixture()}
	svc := newTestSecretService(m, &auditFixture{err: errors.New("db down")})
	out, err := svc.RevealSecret(context.Background(), &model.RevealSecretReq{ClusterID: 1, Namespace: "ops", Name: "app"}, model.SecretRevealActor{UserID: 1})
	if err == nil || out != nil {
		t.Fatal("审计写入失败时不应返回明文")
	}
	if strings.Contains(err.Error(), fixtureAlpha) {
		t.Fatal("错误信息泄露明文")
	}
}

func TestRevealSecretAuditsGetFailure(t *testing.T) {
	m := &secretManagerFixture{secret: newSecretFixture(), getErr: errors.New("forbidden")}
	a := &auditFixture{}
	svc := newTestSecretService(m, a)
	if _, err := svc.RevealSecret(context.Background(), &model.RevealSecretReq{ClusterID: 1, Namespace: "ops", Name: "app"}, model.SecretRevealActor{UserID: 1}); err == nil {
		t.Fatal("应返回错误")
	}
	if len(a.logs) != 1 || a.logs[0].ErrorMsg == "" {
		t.Fatal("读取失败也应写审计")
	}
}

func TestUpdateSecretKeepsValuesForRedactedRoundTrip(t *testing.T) {
	m := &secretManagerFixture{secret: newSecretFixture()}
	svc := newTestSecretService(m, &auditFixture{})
	// 模拟前端把脱敏详情原样回填，只改了 beta 与标签
	err := svc.UpdateSecret(context.Background(), &model.UpdateSecretReq{
		ClusterID: 1, Namespace: "ops", Name: "app",
		Data:        model.BinaryDataMap{"alpha": []byte{}, "beta": []byte("new-beta"), "delta": []byte{}},
		Labels:      map[string]string{"app": "demo2"},
		Annotations: map[string]string{"owner": "team-a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(m.updated.Data["alpha"]) != fixtureAlpha || string(m.updated.Data["beta"]) != "new-beta" {
		t.Fatal("空值回填应保留原值，非空值应覆盖")
	}
	if v, ok := m.updated.Data["delta"]; !ok || len(v) != 0 {
		t.Fatal("新键空值应原样写入")
	}
	if m.updated.Annotations[k8sutils.LastAppliedConfigAnnotation] == "" {
		t.Fatal("未携带的 last-applied 注解应保留")
	}
}

func TestUpdateSecretByYamlRestoresPlaceholders(t *testing.T) {
	m := &secretManagerFixture{secret: newSecretFixture()}
	svc := newTestSecretService(m, &auditFixture{})
	y, err := svc.GetSecretYAML(context.Background(), &model.GetSecretYamlReq{ClusterID: 1, Namespace: "ops", Name: "app"})
	if err != nil {
		t.Fatal(err)
	}
	// 只改 beta
	edited := strings.Replace(y.YAML, "beta: "+k8sutils.SecretRedactedPlaceholder, "beta: "+base64.StdEncoding.EncodeToString([]byte("new-beta")), 1)
	if edited == y.YAML {
		t.Fatal("测试前提失败：未找到 beta 占位符")
	}
	if err := svc.UpdateSecretByYaml(context.Background(), &model.UpdateSecretByYamlReq{ClusterID: 1, Namespace: "ops", Name: "app", YAML: edited}); err != nil {
		t.Fatal(err)
	}
	if string(m.updated.Data["alpha"]) != fixtureAlpha || string(m.updated.Data["beta"]) != "new-beta" {
		t.Fatalf("占位符应恢复原值: alpha=%d beta=%q", len(m.updated.Data["alpha"]), m.updated.Data["beta"])
	}
	if m.updated.StringData["gamma"] != fixtureBeta {
		t.Fatal("stringData 占位符应恢复原值")
	}

	// 占位符对应的键在现有 Secret 中不存在 → 报错，不会写入占位符
	m.updated = nil
	withUnknown := strings.Replace(edited, "data:\n", "data:\n  omega: "+k8sutils.SecretRedactedPlaceholder+"\n", 1)
	if err := svc.UpdateSecretByYaml(context.Background(), &model.UpdateSecretByYamlReq{ClusterID: 1, Namespace: "ops", Name: "app", YAML: withUnknown}); err == nil || m.updated != nil {
		t.Fatal("未知键的占位符应拒绝更新")
	}
}

func TestCreateSecretByYamlRejectsPlaceholder(t *testing.T) {
	m := &secretManagerFixture{secret: newSecretFixture()}
	svc := newTestSecretService(m, &auditFixture{})
	y := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: x\n  namespace: ops\ndata:\n  alpha: " + k8sutils.SecretRedactedPlaceholder + "\n"
	if err := svc.CreateSecretByYaml(context.Background(), &model.CreateSecretByYamlReq{ClusterID: 1, YAML: y}); err == nil || m.created != nil {
		t.Fatal("含占位符的 YAML 不应被创建")
	}
}

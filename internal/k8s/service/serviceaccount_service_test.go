package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// 测试用的 token（非真实凭据），用于断言审计与错误中都不出现
const (
	fixtureOwnToken    = "fake-sa-token-own-0001"
	fixtureVictimToken = "fake-sa-token-victim-0002"
	fixtureIssuedToken = "fake-sa-token-issued-0003"
)

type saManagerFixture struct {
	manager.ServiceAccountManager
	sa          *corev1.ServiceAccount
	secrets     []corev1.Secret
	getErr      error
	created     *corev1.ServiceAccount
	updated     *corev1.ServiceAccount
	tokenReq    *authv1.TokenRequest
	tokenErr    error
	tokenTarget string
}

func (m *saManagerFixture) GetServiceAccount(_ context.Context, _ int, _, _ string) (*corev1.ServiceAccount, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.sa.DeepCopy(), nil
}

func (m *saManagerFixture) GetServiceAccountSecrets(_ context.Context, _ int, _, _ string) ([]corev1.Secret, error) {
	out := make([]corev1.Secret, 0, len(m.secrets))
	for i := range m.secrets {
		out = append(out, *m.secrets[i].DeepCopy())
	}
	return out, nil
}

func (m *saManagerFixture) CreateServiceAccount(_ context.Context, _ int, _ string, sa *corev1.ServiceAccount) error {
	m.created = sa.DeepCopy()
	return nil
}

func (m *saManagerFixture) UpdateServiceAccount(_ context.Context, _ int, _ string, sa *corev1.ServiceAccount) error {
	m.updated = sa.DeepCopy()
	return nil
}

func (m *saManagerFixture) CreateServiceAccountToken(_ context.Context, _ int, _, name string, tr *authv1.TokenRequest) (*authv1.TokenRequest, error) {
	m.tokenTarget = name
	if m.tokenErr != nil {
		return nil, m.tokenErr
	}
	m.tokenReq = tr.DeepCopy()
	out := tr.DeepCopy()
	out.Status.Token = fixtureIssuedToken
	out.Status.ExpirationTimestamp = metav1.Now()
	return out, nil
}

func newSAFixture() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ops", UID: types.UID("uid-app")},
		Secrets:    []corev1.ObjectReference{{Name: "app-token"}},
	}
}

func tokenSecret(name, saName, saUID, token string) corev1.Secret {
	ann := map[string]string{corev1.ServiceAccountNameKey: saName}
	if saUID != "" {
		ann[corev1.ServiceAccountUIDKey] = saUID
	}
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ops", Annotations: ann},
		Type:       corev1.SecretTypeServiceAccountToken,
		Data:       map[string][]byte{corev1.ServiceAccountTokenKey: []byte(token)},
	}
}

var saActor = model.SecretRevealActor{UserID: 7, Username: "operator", IP: "192.0.2.10", Endpoint: "/api/k8s/serviceaccount/1/ops/app/token"}

func assertNoToken(t *testing.T, v interface{}, tokens ...string) {
	t.Helper()
	raw, _ := json.Marshal(v)
	for _, tok := range tokens {
		if strings.Contains(string(raw), tok) {
			t.Fatalf("出现了 token: %s", raw)
		}
	}
}

func TestGetServiceAccountTokenRejectsBorrowedSecret(t *testing.T) {
	for _, tt := range []struct {
		name    string
		secrets []corev1.Secret
	}{
		{"借挂其他 SA 的 token Secret", []corev1.Secret{tokenSecret("admin-token", "admin", "uid-admin", fixtureVictimToken)}},
		{"名称相同但 UID 不同（同名 SA 被重建）", []corev1.Secret{tokenSecret("app-token-old", "app", "uid-old", fixtureVictimToken)}},
		{"UID 注解为空串", []corev1.Secret{tokenSecret("app-token-empty", "app", "", fixtureVictimToken)}},
		{"缺少 name 注解", []corev1.Secret{{
			ObjectMeta: metav1.ObjectMeta{Name: "no-ann", Namespace: "ops"},
			Type:       corev1.SecretTypeServiceAccountToken,
			Data:       map[string][]byte{corev1.ServiceAccountTokenKey: []byte(fixtureVictimToken)},
		}}},
		{"非 service-account-token 类型", []corev1.Secret{func() corev1.Secret {
			s := tokenSecret("opaque", "app", "uid-app", fixtureVictimToken)
			s.Type = corev1.SecretTypeOpaque
			return s
		}()}},
		{"其他命名空间", []corev1.Secret{func() corev1.Secret {
			s := tokenSecret("other-ns", "app", "uid-app", fixtureVictimToken)
			s.Namespace = "kube-system"
			return s
		}()}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "UID 注解为空串" {
				tt.secrets[0].Annotations[corev1.ServiceAccountUIDKey] = ""
			}
			audit := &auditFixture{}
			svc := NewServiceAccountService(&saManagerFixture{sa: newSAFixture(), secrets: tt.secrets}, audit, zap.NewNop())
			got, err := svc.GetServiceAccountToken(context.Background(), &model.GetServiceAccountTokenReq{ClusterID: 1, Namespace: "ops", Name: "app"}, saActor)
			if err == nil || got != nil {
				t.Fatalf("借挂的 token 不应返回: %+v", got)
			}
			assertNoToken(t, err.Error(), fixtureVictimToken)
			if len(audit.logs) != 1 || audit.logs[0].StatusCode != 404 || audit.logs[0].OperationType != model.ServiceAccountTokenRevealAuditOperation {
				t.Fatalf("拒绝也要写审计: %+v", audit.logs)
			}
			assertNoToken(t, audit.logs, fixtureVictimToken)
		})
	}
}

func TestGetServiceAccountTokenReturnsOwnSecretOnly(t *testing.T) {
	audit := &auditFixture{}
	fx := &saManagerFixture{sa: newSAFixture(), secrets: []corev1.Secret{
		tokenSecret("admin-token", "admin", "uid-admin", fixtureVictimToken),
		tokenSecret("app-token", "app", "uid-app", fixtureOwnToken),
	}}
	svc := NewServiceAccountService(fx, audit, zap.NewNop())
	got, err := svc.GetServiceAccountToken(context.Background(), &model.GetServiceAccountTokenReq{ClusterID: 1, Namespace: "ops", Name: "app"}, saActor)
	if err != nil || got == nil || got.Token != fixtureOwnToken {
		t.Fatalf("应返回 SA 自己的 token: %+v %v", got, err)
	}

	// 没有 uid 注解时只校验名称
	fx.secrets = []corev1.Secret{tokenSecret("app-token", "app", "", fixtureOwnToken)}
	got, err = svc.GetServiceAccountToken(context.Background(), &model.GetServiceAccountTokenReq{ClusterID: 1, Namespace: "ops", Name: "app"}, saActor)
	if err != nil || got.Token != fixtureOwnToken {
		t.Fatalf("无 uid 注解时名称匹配应放行: %+v %v", got, err)
	}
}

func TestGetServiceAccountTokenAuditHasMetadataOnly(t *testing.T) {
	audit := &auditFixture{}
	svc := NewServiceAccountService(&saManagerFixture{sa: newSAFixture(), secrets: []corev1.Secret{tokenSecret("app-token", "app", "uid-app", fixtureOwnToken)}}, audit, zap.NewNop())
	if _, err := svc.GetServiceAccountToken(context.Background(), &model.GetServiceAccountTokenReq{ClusterID: 1, Namespace: "ops", Name: "app"}, saActor); err != nil {
		t.Fatal(err)
	}
	if len(audit.logs) != 1 {
		t.Fatalf("应同步写 1 条审计，实际 %d", len(audit.logs))
	}
	l := audit.logs[0]
	if l.OperationType != "TOKEN_REVEAL" || l.TargetType != model.ServiceAccountTokenAuditTargetType || l.TargetID != "1/ops/app" ||
		l.StatusCode != 200 || l.HttpMethod != "GET" || l.UserID != 7 || l.IPAddress != "192.0.2.10" || len(l.ResponseBody) != 0 {
		t.Fatalf("审计元数据不符: %+v", l)
	}
	var rec model.ServiceAccountTokenAuditRecord
	if err := json.Unmarshal(l.RequestBody, &rec); err != nil {
		t.Fatal(err)
	}
	if rec.SecretName != "app-token" || rec.ServiceAccountUID != "uid-app" || !rec.Success || rec.Username != "operator" {
		t.Fatalf("审计内容不符: %s", l.RequestBody)
	}
	assertNoToken(t, audit.logs, fixtureOwnToken)
}

func TestGetServiceAccountTokenAuditFailureReturnsNoToken(t *testing.T) {
	svc := NewServiceAccountService(&saManagerFixture{sa: newSAFixture(), secrets: []corev1.Secret{tokenSecret("app-token", "app", "uid-app", fixtureOwnToken)}},
		&auditFixture{err: errors.New("audit db unavailable")}, zap.NewNop())
	got, err := svc.GetServiceAccountToken(context.Background(), &model.GetServiceAccountTokenReq{ClusterID: 1, Namespace: "ops", Name: "app"}, saActor)
	if got != nil || !errors.Is(err, ErrAuditWriteFailed) {
		t.Fatalf("审计失败不应返回 token: %+v %v", got, err)
	}
	assertNoToken(t, err.Error(), fixtureOwnToken)

	// 审计服务缺失同样 fail closed
	svc = NewServiceAccountService(&saManagerFixture{sa: newSAFixture(), secrets: []corev1.Secret{tokenSecret("app-token", "app", "uid-app", fixtureOwnToken)}}, nil, zap.NewNop())
	if got, err := svc.GetServiceAccountToken(context.Background(), &model.GetServiceAccountTokenReq{ClusterID: 1, Namespace: "ops", Name: "app"}, saActor); got != nil || !errors.Is(err, ErrAuditWriteFailed) {
		t.Fatalf("未注入审计服务时不应返回 token: %+v %v", got, err)
	}
}

func TestGetServiceAccountTokenSANotFoundAudited(t *testing.T) {
	audit := &auditFixture{}
	svc := NewServiceAccountService(&saManagerFixture{getErr: apierrors.NewNotFound(schema.GroupResource{Resource: "serviceaccounts"}, "app")}, audit, zap.NewNop())
	if _, err := svc.GetServiceAccountToken(context.Background(), &model.GetServiceAccountTokenReq{ClusterID: 1, Namespace: "ops", Name: "app"}, saActor); err == nil {
		t.Fatal("SA 不存在应报错")
	}
	if len(audit.logs) != 1 || audit.logs[0].StatusCode != 404 {
		t.Fatalf("审计状态码应为 404: %+v", audit.logs)
	}
}

func TestCreateServiceAccountTokenAudited(t *testing.T) {
	exp := int64(600)
	req := &model.CreateServiceAccountTokenReq{ClusterID: 1, Namespace: "ops", ServiceAccountName: "app", ExpirationSeconds: &exp}

	audit := &auditFixture{}
	fx := &saManagerFixture{}
	svc := NewServiceAccountService(fx, audit, zap.NewNop())
	got, err := svc.CreateServiceAccountToken(context.Background(), req, saActor)
	if err != nil || got.Token != fixtureIssuedToken || fx.tokenTarget != "app" || *fx.tokenReq.Spec.ExpirationSeconds != 600 {
		t.Fatalf("TokenRequest 结果不符: %+v %v", got, err)
	}
	if len(audit.logs) != 1 || audit.logs[0].OperationType != "TOKEN_CREATE" || audit.logs[0].HttpMethod != "POST" || audit.logs[0].StatusCode != 200 {
		t.Fatalf("TokenRequest 审计不符: %+v", audit.logs)
	}
	var rec model.ServiceAccountTokenAuditRecord
	if err := json.Unmarshal(audit.logs[0].RequestBody, &rec); err != nil || rec.ExpirationSeconds == nil || *rec.ExpirationSeconds != 600 {
		t.Fatalf("审计内容不符: %s", audit.logs[0].RequestBody)
	}
	assertNoToken(t, audit.logs, fixtureIssuedToken)

	// 审计失败：已签发也不返回
	svc = NewServiceAccountService(&saManagerFixture{}, &auditFixture{err: errors.New("audit db unavailable")}, zap.NewNop())
	got, err = svc.CreateServiceAccountToken(context.Background(), req, saActor)
	if got != nil || !errors.Is(err, ErrAuditWriteFailed) {
		t.Fatalf("审计失败不应返回签发的 token: %+v %v", got, err)
	}
	assertNoToken(t, err.Error(), fixtureIssuedToken)

	// K8s 拒绝：审计记录真实状态码
	audit = &auditFixture{}
	svc = NewServiceAccountService(&saManagerFixture{tokenErr: apierrors.NewForbidden(schema.GroupResource{Resource: "serviceaccounts/token"}, "app", errors.New("denied"))}, audit, zap.NewNop())
	if _, err := svc.CreateServiceAccountToken(context.Background(), req, saActor); err == nil {
		t.Fatal("应返回错误")
	}
	if len(audit.logs) != 1 || audit.logs[0].StatusCode != 403 {
		t.Fatalf("审计状态码应为 403: %+v", audit.logs)
	}
}

func TestCreateServiceAccountIgnoresSecretRefs(t *testing.T) {
	fx := &saManagerFixture{}
	svc := NewServiceAccountService(fx, &auditFixture{}, zap.NewNop())
	err := svc.CreateServiceAccount(context.Background(), &model.CreateServiceAccountReq{
		ClusterID: 1, Namespace: "ops", Name: "app", Secrets: []string{"admin-token"}, ImagePullSecrets: []string{"registry"},
	})
	if err != nil || fx.created == nil || len(fx.created.Secrets) != 0 || len(fx.created.ImagePullSecrets) != 1 {
		t.Fatalf("创建时应忽略 secrets、保留 imagePullSecrets: %+v %v", fx.created, err)
	}

	fx = &saManagerFixture{}
	svc = NewServiceAccountService(fx, &auditFixture{}, zap.NewNop())
	yamlContent := "apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: app\n  namespace: ops\nsecrets:\n- name: admin-token\n"
	if err := svc.CreateServiceAccountByYaml(context.Background(), &model.CreateServiceAccountByYamlReq{ClusterID: 1, YamlContent: yamlContent}); err != nil {
		t.Fatal(err)
	}
	if fx.created == nil || len(fx.created.Secrets) != 0 {
		t.Fatalf("YAML 创建时应忽略 secrets: %+v", fx.created)
	}
}

func TestUpdateServiceAccountKeepsExistingSecretRefs(t *testing.T) {
	fx := &saManagerFixture{sa: newSAFixture()}
	svc := NewServiceAccountService(fx, &auditFixture{}, zap.NewNop())
	err := svc.UpdateServiceAccount(context.Background(), &model.UpdateServiceAccountReq{
		ClusterID: 1, Namespace: "ops", Name: "app", Secrets: []string{"admin-token"}, Labels: map[string]string{"team": "ops"},
	})
	if err != nil || len(fx.updated.Secrets) != 1 || fx.updated.Secrets[0].Name != "app-token" || fx.updated.Labels["team"] != "ops" {
		t.Fatalf("更新时应忽略请求 secrets、保留集群现有引用: %+v %v", fx.updated, err)
	}

	// 不传 secrets 也不能清空集群维护的引用
	fx.updated = nil
	if err := svc.UpdateServiceAccount(context.Background(), &model.UpdateServiceAccountReq{ClusterID: 1, Namespace: "ops", Name: "app"}); err != nil {
		t.Fatal(err)
	}
	if len(fx.updated.Secrets) != 1 || fx.updated.Secrets[0].Name != "app-token" {
		t.Fatalf("未传 secrets 时应保留现有引用: %+v", fx.updated.Secrets)
	}

	// YAML 更新：secrets 被忽略，name/namespace 以 URL 为准
	fx.updated = nil
	yamlContent := "apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: other\n  namespace: kube-system\nsecrets:\n- name: admin-token\n"
	if err := svc.UpdateServiceAccountYaml(context.Background(), &model.UpdateServiceAccountByYamlReq{ClusterID: 1, Namespace: "ops", Name: "app", YamlContent: yamlContent}); err != nil {
		t.Fatal(err)
	}
	if fx.updated.Name != "app" || fx.updated.Namespace != "ops" || len(fx.updated.Secrets) != 1 || fx.updated.Secrets[0].Name != "app-token" {
		t.Fatalf("YAML 更新应忽略 secrets 并以 URL 为准: %+v", fx.updated)
	}

	// SA 不存在时不更新
	fx = &saManagerFixture{getErr: apierrors.NewNotFound(schema.GroupResource{Resource: "serviceaccounts"}, "app")}
	svc = NewServiceAccountService(fx, &auditFixture{}, zap.NewNop())
	if err := svc.UpdateServiceAccount(context.Background(), &model.UpdateServiceAccountReq{ClusterID: 1, Namespace: "ops", Name: "app"}); err == nil || fx.updated != nil {
		t.Fatalf("SA 不存在应报错且不更新: %v", err)
	}
}

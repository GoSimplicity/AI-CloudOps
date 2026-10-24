package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"go.uber.org/zap"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/util/retry"
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
	// kube 为 UpdateServiceAccountWithRetry 使用的 fake clientset，首次使用时以 sa 为初始数据创建；
	// 测试可预先设置并挂 reactor 模拟并发冲突
	kube *fake.Clientset
	// getCalls 记录 GetServiceAccount 调用次数；secretsSA 记录传给 GetServiceAccountSecrets 的 SA
	getCalls  int
	secretsSA *corev1.ServiceAccount
	// tokenExpiry 非零时作为 TokenRequest 返回的过期时间（默认 metav1.Now()）
	tokenExpiry time.Time
}

func (m *saManagerFixture) GetServiceAccount(_ context.Context, _ int, _, _ string) (*corev1.ServiceAccount, error) {
	m.getCalls++
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.sa.DeepCopy(), nil
}

func (m *saManagerFixture) GetServiceAccountSecrets(_ context.Context, _ int, sa *corev1.ServiceAccount) ([]corev1.Secret, error) {
	m.secretsSA = sa
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

func (m *saManagerFixture) ensureKube() {
	if m.kube == nil {
		var objs []runtime.Object
		if m.sa != nil {
			objs = append(objs, m.sa.DeepCopy())
		}
		m.kube = fake.NewSimpleClientset(objs...)
	}
}

func (m *saManagerFixture) UpdateServiceAccountWithResourceVersion(ctx context.Context, _ int, namespace, name, resourceVersion string, mutate k8sutils.ServiceAccountMutateFunc) error {
	m.ensureKube()
	updated, err := k8sutils.UpdateServiceAccountWithResourceVersion(ctx, m.kube.CoreV1().ServiceAccounts(namespace), name, resourceVersion, mutate)
	if err != nil {
		return err
	}
	m.updated = updated.DeepCopy()
	return nil
}

func (m *saManagerFixture) UpdateServiceAccountWithRetry(ctx context.Context, _ int, namespace, name string, mutate k8sutils.ServiceAccountMutateFunc) error {
	m.ensureKube()
	updated, err := k8sutils.UpdateServiceAccountWithRetry(ctx, m.kube.CoreV1().ServiceAccounts(namespace), name, mutate)
	if err != nil {
		return err
	}
	m.updated = updated.DeepCopy()
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
	if !m.tokenExpiry.IsZero() {
		out.Status.ExpirationTimestamp = metav1.NewTime(m.tokenExpiry)
	}
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

func TestCreateServiceAccountTokenExpirationBounds(t *testing.T) {
	i64 := func(v int64) *int64 { return &v }
	for _, tt := range []struct {
		name      string
		requested *int64
		want      int64 // 期望传给 K8s 的值；0 表示应被拒绝
	}{
		{"不传默认3600", nil, 3600},
		{"传0默认3600", i64(0), 3600},
		{"599低于下限", i64(599), 0},
		{"600下限", i64(600), 600},
		{"3600上限", i64(3600), 3600},
		{"3601高于上限", i64(3601), 0},
		{"负数", i64(-1), 0},
		{"负的600", i64(-600), 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			audit := &auditFixture{}
			fx := &saManagerFixture{}
			svc := NewServiceAccountService(fx, audit, zap.NewNop())
			got, err := svc.CreateServiceAccountToken(context.Background(), &model.CreateServiceAccountTokenReq{
				ClusterID: 1, Namespace: "ops", ServiceAccountName: "app", ExpirationSeconds: tt.requested,
			}, saActor)
			if len(audit.logs) != 1 {
				t.Fatalf("应写且只写一条审计: %+v", audit.logs)
			}
			var rec model.ServiceAccountTokenAuditRecord
			if err := json.Unmarshal(audit.logs[0].RequestBody, &rec); err != nil {
				t.Fatal(err)
			}
			if tt.want == 0 {
				var bizErr *base.BusinessError
				if got != nil || !errors.As(err, &bizErr) || !errors.Is(bizErr.Code, constants.ErrInvalidParam) {
					t.Fatalf("越界应返回参数错误且不返回 token: %+v %v", got, err)
				}
				if fx.tokenTarget != "" || fx.tokenReq != nil {
					t.Fatalf("越界时不应调用 K8s TokenRequest: target=%q", fx.tokenTarget)
				}
				if audit.logs[0].StatusCode != 400 || rec.Success || rec.ExpirationSeconds == nil || *rec.ExpirationSeconds != *tt.requested {
					t.Fatalf("越界审计应为 400、success=false、记录请求值: %+v %s", audit.logs[0], audit.logs[0].RequestBody)
				}
				assertNoToken(t, audit.logs, fixtureIssuedToken)
				return
			}
			if err != nil || got == nil || got.Token != fixtureIssuedToken {
				t.Fatalf("应签发成功: %+v %v", got, err)
			}
			if fx.tokenReq == nil || fx.tokenReq.Spec.ExpirationSeconds == nil || *fx.tokenReq.Spec.ExpirationSeconds != tt.want {
				t.Fatalf("传给 K8s 的有效期应为 %d: %+v", tt.want, fx.tokenReq)
			}
			if got.ExpirationSeconds == nil || *got.ExpirationSeconds != tt.want {
				t.Fatalf("响应有效期应为 %d: %+v", tt.want, got.ExpirationSeconds)
			}
			if audit.logs[0].StatusCode != 200 || rec.ExpirationSeconds == nil || *rec.ExpirationSeconds != tt.want {
				t.Fatalf("审计应记录生效有效期 %d: %s", tt.want, audit.logs[0].RequestBody)
			}
		})
	}
}

func countSAUpdates(c *fake.Clientset) int {
	n := 0
	for _, a := range c.Actions() {
		if a.GetVerb() == "update" && a.GetResource().Resource == "serviceaccounts" {
			n++
		}
	}
	return n
}

// 第一次 Update 前集群中的 SA 被并发修改并返回 Conflict（YAML 不带 resourceVersion，走 RetryOnConflict），第二次成功：JSON 与 YAML 两条路径都应基于最新版本重试、保留最新 secrets
func TestUpdateServiceAccountRetriesOnConflict(t *testing.T) {
	yamlContent := "apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: other\n  namespace: kube-system\n  labels:\n    team: ops\nsecrets:\n- name: admin-token\n"
	for _, tt := range []struct {
		name   string
		update func(svc ServiceAccountService) error
	}{
		{"json", func(svc ServiceAccountService) error {
			return svc.UpdateServiceAccount(context.Background(), &model.UpdateServiceAccountReq{
				ClusterID: 1, Namespace: "ops", Name: "app", Secrets: []string{"admin-token"}, Labels: map[string]string{"team": "ops"},
			})
		}},
		{"yaml", func(svc ServiceAccountService) error {
			return svc.UpdateServiceAccountYaml(context.Background(), &model.UpdateServiceAccountByYamlReq{ClusterID: 1, Namespace: "ops", Name: "app", YamlContent: yamlContent})
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			seed := newSAFixture()
			seed.ResourceVersion = "1"
			kube := fake.NewSimpleClientset(seed)
			calls := 0
			kube.PrependReactor("update", "serviceaccounts", func(k8stesting.Action) (bool, runtime.Object, error) {
				calls++
				if calls > 1 {
					return false, nil, nil
				}
				concurrent := newSAFixture()
				concurrent.ResourceVersion = "2"
				concurrent.Secrets = append(concurrent.Secrets, corev1.ObjectReference{Name: "app-token-rotated"})
				if err := kube.Tracker().Update(corev1.SchemeGroupVersion.WithResource("serviceaccounts"), concurrent, "ops"); err != nil {
					t.Errorf("模拟并发修改失败: %v", err)
				}
				return true, nil, apierrors.NewConflict(schema.GroupResource{Resource: "serviceaccounts"}, "app", errors.New("object has been modified"))
			})
			fx := &saManagerFixture{sa: newSAFixture(), kube: kube}
			if err := tt.update(NewServiceAccountService(fx, &auditFixture{}, zap.NewNop())); err != nil {
				t.Fatal(err)
			}
			if calls != 2 {
				t.Fatalf("应冲突一次后重试成功, update 调用 %d 次", calls)
			}
			var last *corev1.ServiceAccount
			for _, a := range kube.Actions() {
				if ua, ok := a.(k8stesting.UpdateAction); ok && a.GetResource().Resource == "serviceaccounts" {
					last = ua.GetObject().(*corev1.ServiceAccount)
				}
			}
			if last == nil || last.ResourceVersion != "2" {
				t.Fatalf("最后一次 Update 应基于最新版本 resourceVersion=2: %+v", last)
			}
			if last.Name != "app" || last.Namespace != "ops" || last.Labels["team"] != "ops" {
				t.Fatalf("name/namespace 应以 URL 为准、字段应被应用: %+v", last.ObjectMeta)
			}
			if got := k8sutils.ServiceAccountSecretRefNames(last.Secrets); len(got) != 2 || got[0] != "app-token" || got[1] != "app-token-rotated" {
				t.Fatalf("应保留集群最新的 secrets 引用、忽略请求中的 admin-token: %v", got)
			}
		})
	}
}

func TestUpdateServiceAccountConflictExhaustedReturnsError(t *testing.T) {
	kube := fake.NewSimpleClientset(newSAFixture())
	kube.PrependReactor("update", "serviceaccounts", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewConflict(schema.GroupResource{Resource: "serviceaccounts"}, "app", errors.New("object has been modified"))
	})
	fx := &saManagerFixture{sa: newSAFixture(), kube: kube}
	svc := NewServiceAccountService(fx, &auditFixture{}, zap.NewNop())
	err := svc.UpdateServiceAccount(context.Background(), &model.UpdateServiceAccountReq{ClusterID: 1, Namespace: "ops", Name: "app"})
	if err == nil || !apierrors.IsConflict(err) || fx.updated != nil {
		t.Fatalf("连续冲突超过重试次数应返回 Conflict 错误: %v", err)
	}
	if n := countSAUpdates(kube); n != retry.DefaultRetry.Steps {
		t.Fatalf("应尝试 %d 次, got %d", retry.DefaultRetry.Steps, n)
	}
	if err := svc.UpdateServiceAccountYaml(context.Background(), &model.UpdateServiceAccountByYamlReq{
		ClusterID: 1, Namespace: "ops", Name: "app", YamlContent: "apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: app\n",
	}); !apierrors.IsConflict(err) {
		t.Fatalf("YAML 路径连续冲突同样应返回错误: %v", err)
	}
}

func TestUpdateServiceAccountNotFoundSkipsUpdate(t *testing.T) {
	for _, op := range []string{"json", "yaml"} {
		t.Run(op, func(t *testing.T) {
			fx := &saManagerFixture{}
			svc := NewServiceAccountService(fx, &auditFixture{}, zap.NewNop())
			var err error
			if op == "json" {
				err = svc.UpdateServiceAccount(context.Background(), &model.UpdateServiceAccountReq{ClusterID: 1, Namespace: "ops", Name: "app"})
			} else {
				err = svc.UpdateServiceAccountYaml(context.Background(), &model.UpdateServiceAccountByYamlReq{
					ClusterID: 1, Namespace: "ops", Name: "app", YamlContent: "apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: app\n",
				})
			}
			if err == nil || !strings.Contains(err.Error(), "ServiceAccount不存在") {
				t.Fatalf("SA 不存在应报错: %v", err)
			}
			if n := countSAUpdates(fx.kube); n != 0 || fx.updated != nil {
				t.Fatalf("SA 不存在时不应调用 Update, got %d", n)
			}
		})
	}
}

// optimisticLockReactor 模拟 API Server 的乐观锁：Update 提交的 resourceVersion 与当前存储不一致时返回 Conflict，
// 一致时把版本号加 1 后交给默认 tracker（fake clientset 本身不校验 resourceVersion）
func optimisticLockReactor(t *testing.T, kube *fake.Clientset) {
	gvr := corev1.SchemeGroupVersion.WithResource("serviceaccounts")
	kube.PrependReactor("update", "serviceaccounts", func(a k8stesting.Action) (bool, runtime.Object, error) {
		obj := a.(k8stesting.UpdateAction).GetObject().(*corev1.ServiceAccount)
		cur, err := kube.Tracker().Get(gvr, obj.Namespace, obj.Name)
		if err != nil {
			return true, nil, err
		}
		curRV := cur.(*corev1.ServiceAccount).ResourceVersion
		if obj.ResourceVersion != curRV {
			return true, nil, apierrors.NewConflict(schema.GroupResource{Resource: "serviceaccounts"}, obj.Name, errors.New("the object has been modified"))
		}
		next := obj.DeepCopy()
		n, _ := strconv.Atoi(curRV)
		next.ResourceVersion = strconv.Itoa(n + 1)
		if err := kube.Tracker().Update(gvr, next, obj.Namespace); err != nil {
			t.Errorf("tracker 更新失败: %v", err)
		}
		return true, next, nil
	})
}

func saYamlWithRV(rv string) string {
	y := "apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: app\n  namespace: ops\n"
	if rv != "" {
		y += "  resourceVersion: \"" + rv + "\"\n"
	}
	return y + "  labels:\n    team: yaml\nsecrets:\n- name: admin-token\n"
}

// SEC-29(A)：YAML 带 resourceVersion 时严格乐观锁——版本过期只 Update 一次并返回 409 业务错误，不重试
func TestUpdateServiceAccountYamlStaleResourceVersionReturnsConflict(t *testing.T) {
	seed := newSAFixture()
	seed.ResourceVersion = "5"
	kube := fake.NewSimpleClientset(seed)
	optimisticLockReactor(t, kube)
	fx := &saManagerFixture{sa: newSAFixture(), kube: kube}
	svc := NewServiceAccountService(fx, &auditFixture{}, zap.NewNop())

	err := svc.UpdateServiceAccountYaml(context.Background(), &model.UpdateServiceAccountByYamlReq{ClusterID: 1, Namespace: "ops", Name: "app", YamlContent: saYamlWithRV("4")})
	var bizErr *base.BusinessError
	if !errors.As(err, &bizErr) || !errors.Is(bizErr.Code, constants.ErrK8sResourceConflict) {
		t.Fatalf("过期 resourceVersion 应返回 ErrK8sResourceConflict 业务错误: %v", err)
	}
	if n := countSAUpdates(kube); n != 1 {
		t.Fatalf("严格乐观锁只应 Update 一次（不重试）, got %d", n)
	}
	if fx.updated != nil {
		t.Fatalf("冲突时不应视为更新成功: %+v", fx.updated)
	}
	cur, _ := kube.CoreV1().ServiceAccounts("ops").Get(context.Background(), "app", metav1.GetOptions{})
	if cur.ResourceVersion != "5" || cur.Labels["team"] != "" {
		t.Fatalf("冲突时集群对象不应被修改: %+v", cur.ObjectMeta)
	}
}

func TestUpdateServiceAccountYamlCurrentResourceVersionSucceeds(t *testing.T) {
	seed := newSAFixture()
	seed.ResourceVersion = "5"
	kube := fake.NewSimpleClientset(seed)
	optimisticLockReactor(t, kube)
	fx := &saManagerFixture{sa: newSAFixture(), kube: kube}
	svc := NewServiceAccountService(fx, &auditFixture{}, zap.NewNop())

	if err := svc.UpdateServiceAccountYaml(context.Background(), &model.UpdateServiceAccountByYamlReq{ClusterID: 1, Namespace: "ops", Name: "app", YamlContent: saYamlWithRV("5")}); err != nil {
		t.Fatal(err)
	}
	if n := countSAUpdates(kube); n != 1 {
		t.Fatalf("应只 Update 一次, got %d", n)
	}
	cur, _ := kube.CoreV1().ServiceAccounts("ops").Get(context.Background(), "app", metav1.GetOptions{})
	if cur.ResourceVersion != "6" || cur.Labels["team"] != "yaml" {
		t.Fatalf("应以提交的版本更新成功并应用字段: %+v", cur.ObjectMeta)
	}
	if got := k8sutils.ServiceAccountSecretRefNames(cur.Secrets); len(got) != 1 || got[0] != "app-token" {
		t.Fatalf("乐观锁路径同样应忽略 YAML 中的 secrets、保留集群引用: %v", got)
	}
}

// 不带 resourceVersion 时保持 RetryOnConflict：即使存储中的版本与 YAML 无关，也基于最新版本重试成功
func TestUpdateServiceAccountYamlWithoutResourceVersionUsesRetry(t *testing.T) {
	seed := newSAFixture()
	seed.ResourceVersion = "5"
	kube := fake.NewSimpleClientset(seed)
	optimisticLockReactor(t, kube)
	conflicts := 0
	kube.PrependReactor("update", "serviceaccounts", func(k8stesting.Action) (bool, runtime.Object, error) {
		if conflicts == 0 {
			conflicts++
			return true, nil, apierrors.NewConflict(schema.GroupResource{Resource: "serviceaccounts"}, "app", errors.New("the object has been modified"))
		}
		return false, nil, nil
	})
	fx := &saManagerFixture{sa: newSAFixture(), kube: kube}
	svc := NewServiceAccountService(fx, &auditFixture{}, zap.NewNop())
	if err := svc.UpdateServiceAccountYaml(context.Background(), &model.UpdateServiceAccountByYamlReq{ClusterID: 1, Namespace: "ops", Name: "app", YamlContent: saYamlWithRV("")}); err != nil {
		t.Fatal(err)
	}
	if n := countSAUpdates(kube); n != 2 {
		t.Fatalf("未带 resourceVersion 应走重试路径（冲突 1 次后成功）, update %d 次", n)
	}
	cur, _ := kube.CoreV1().ServiceAccounts("ops").Get(context.Background(), "app", metav1.GetOptions{})
	if cur.Labels["team"] != "yaml" || cur.ResourceVersion != "6" {
		t.Fatalf("重试后应基于最新版本更新成功: %+v", cur.ObjectMeta)
	}
}

// SEC-29(B)：K8s 返回非 UTC 时区的过期时间时，响应中的 expiration_time 仍是正确的 UTC 字符串
func TestCreateServiceAccountTokenExpirationTimeIsUTC(t *testing.T) {
	cst := time.FixedZone("CST", 8*3600)
	fx := &saManagerFixture{tokenExpiry: time.Date(2026, 10, 24, 9, 30, 15, 0, cst)}
	svc := NewServiceAccountService(fx, &auditFixture{}, zap.NewNop())
	got, err := svc.CreateServiceAccountToken(context.Background(), &model.CreateServiceAccountTokenReq{ClusterID: 1, Namespace: "ops", ServiceAccountName: "app"}, saActor)
	if err != nil {
		t.Fatal(err)
	}
	if got.ExpirationTime != "2026-10-24T01:30:15Z" {
		t.Fatalf("expiration_time 应转换为 UTC: %q", got.ExpirationTime)
	}
}

// SEC-29 P3-3：GetServiceAccountToken 只读取一次 SA，并把同一个对象传给 GetServiceAccountSecrets
func TestGetServiceAccountTokenReadsServiceAccountOnce(t *testing.T) {
	fx := &saManagerFixture{sa: newSAFixture(), secrets: []corev1.Secret{tokenSecret("app-token", "app", "uid-app", fixtureOwnToken)}}
	svc := NewServiceAccountService(fx, &auditFixture{}, zap.NewNop())
	if _, err := svc.GetServiceAccountToken(context.Background(), &model.GetServiceAccountTokenReq{ClusterID: 1, Namespace: "ops", Name: "app"}, saActor); err != nil {
		t.Fatal(err)
	}
	if fx.getCalls != 1 {
		t.Fatalf("GetServiceAccount 应只调用一次, got %d", fx.getCalls)
	}
	if fx.secretsSA == nil || fx.secretsSA.Name != "app" || fx.secretsSA.UID != "uid-app" {
		t.Fatalf("应把已读取的 SA 传给 GetServiceAccountSecrets: %+v", fx.secretsSA)
	}
}

package utils

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/util/retry"
)

var saGVR = corev1.SchemeGroupVersion.WithResource("serviceaccounts")

func newRetrySA() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ops", ResourceVersion: "1", Labels: map[string]string{"v": "1"}},
		Secrets:    []corev1.ObjectReference{{Name: "app-token"}},
	}
}

func saUpdateActions(c *fake.Clientset) []k8stesting.UpdateAction {
	var out []k8stesting.UpdateAction
	for _, a := range c.Actions() {
		if a.GetVerb() == "update" && a.GetResource().Resource == "serviceaccounts" {
			out = append(out, a.(k8stesting.UpdateAction))
		}
	}
	return out
}

func conflictErr() error {
	return apierrors.NewConflict(schema.GroupResource{Resource: "serviceaccounts"}, "app", nil)
}

// 第一次 Update 前集群里的 SA 被并发修改（resourceVersion 变为 2、多了一个 secrets 引用），Update 返回 Conflict；
// 重试时应重新 Get 最新版本，在其上应用字段后再 Update，且保留最新的 secrets 引用。
func TestUpdateServiceAccountWithRetryConflictThenSuccess(t *testing.T) {
	client := fake.NewSimpleClientset(newRetrySA())
	calls := 0
	client.PrependReactor("update", "serviceaccounts", func(k8stesting.Action) (bool, runtime.Object, error) {
		calls++
		if calls > 1 {
			return false, nil, nil // 交给默认 tracker 处理
		}
		concurrent := newRetrySA()
		concurrent.ResourceVersion = "2"
		concurrent.Labels = map[string]string{"v": "2"}
		concurrent.Secrets = append(concurrent.Secrets, corev1.ObjectReference{Name: "app-token-rotated"})
		if err := client.Tracker().Update(saGVR, concurrent, "ops"); err != nil {
			t.Errorf("模拟并发修改失败: %v", err)
		}
		return true, nil, conflictErr()
	})

	mutations := 0
	updated, err := UpdateServiceAccountWithRetry(context.Background(), client.CoreV1().ServiceAccounts("ops"), "app", func(latest *corev1.ServiceAccount) error {
		mutations++
		latest.Labels = map[string]string{"team": "ops"}
		latest.Secrets = []corev1.ObjectReference{{Name: "admin-token"}} // 企图篡改 secrets，必须被还原
		latest.Name = "admin"                                            // 企图改名，必须被还原
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	actions := saUpdateActions(client)
	if calls != 2 || len(actions) != 2 || mutations != 2 {
		t.Fatalf("应冲突一次后重试成功: updateCalls=%d actions=%d mutations=%d", calls, len(actions), mutations)
	}
	last := actions[1].GetObject().(*corev1.ServiceAccount)
	if last.ResourceVersion != "2" {
		t.Fatalf("重试应基于最新版本（resourceVersion=2），got %q", last.ResourceVersion)
	}
	if last.Name != "app" || last.Labels["team"] != "ops" {
		t.Fatalf("name 应保持 URL 值、字段应被应用: %+v", last.ObjectMeta)
	}
	if len(last.Secrets) != 2 || last.Secrets[0].Name != "app-token" || last.Secrets[1].Name != "app-token-rotated" {
		t.Fatalf("应保留集群最新的 secrets 引用: %+v", last.Secrets)
	}
	if updated == nil || len(updated.Secrets) != 2 {
		t.Fatalf("返回值应为更新后的对象: %+v", updated)
	}
}

func TestUpdateServiceAccountWithRetryExhausted(t *testing.T) {
	client := fake.NewSimpleClientset(newRetrySA())
	client.PrependReactor("update", "serviceaccounts", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, conflictErr()
	})
	_, err := UpdateServiceAccountWithRetry(context.Background(), client.CoreV1().ServiceAccounts("ops"), "app", func(*corev1.ServiceAccount) error { return nil })
	if !apierrors.IsConflict(err) {
		t.Fatalf("连续冲突超过重试次数应返回 Conflict 错误, got %v", err)
	}
	if n := len(saUpdateActions(client)); n != retry.DefaultRetry.Steps {
		t.Fatalf("应尝试 %d 次 Update, got %d", retry.DefaultRetry.Steps, n)
	}
}

func TestUpdateServiceAccountWithRetryNotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	called := false
	_, err := UpdateServiceAccountWithRetry(context.Background(), client.CoreV1().ServiceAccounts("ops"), "app", func(*corev1.ServiceAccount) error {
		called = true
		return nil
	})
	if !apierrors.IsNotFound(err) || called {
		t.Fatalf("SA 不存在应返回 NotFound 且不调用 mutate: %v called=%v", err, called)
	}
	if n := len(saUpdateActions(client)); n != 0 {
		t.Fatalf("SA 不存在时不应调用 Update, got %d", n)
	}
}

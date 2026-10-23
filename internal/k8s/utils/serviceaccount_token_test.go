package utils

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func saTokenSecret(name string, ann map[string]string) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ops", Annotations: ann},
		Type:       corev1.SecretTypeServiceAccountToken,
		Data:       map[string][]byte{corev1.ServiceAccountTokenKey: []byte("fake-utils-token")},
	}
}

func TestIsServiceAccountTokenSecretFor(t *testing.T) {
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ops", UID: types.UID("uid-app")}}
	own := map[string]string{corev1.ServiceAccountNameKey: "app", corev1.ServiceAccountUIDKey: "uid-app"}
	for _, tt := range []struct {
		name   string
		mutate func(*corev1.Secret)
		want   bool
	}{
		{"own secret", func(*corev1.Secret) {}, true},
		{"no uid annotation", func(s *corev1.Secret) { delete(s.Annotations, corev1.ServiceAccountUIDKey) }, true},
		{"name mismatch", func(s *corev1.Secret) { s.Annotations[corev1.ServiceAccountNameKey] = "admin" }, false},
		{"uid mismatch", func(s *corev1.Secret) { s.Annotations[corev1.ServiceAccountUIDKey] = "uid-old" }, false},
		{"empty uid annotation", func(s *corev1.Secret) { s.Annotations[corev1.ServiceAccountUIDKey] = "" }, false},
		{"no annotations", func(s *corev1.Secret) { s.Annotations = nil }, false},
		{"opaque type", func(s *corev1.Secret) { s.Type = corev1.SecretTypeOpaque }, false},
		{"other namespace", func(s *corev1.Secret) { s.Namespace = "kube-system" }, false},
		{"empty token", func(s *corev1.Secret) { s.Data = map[string][]byte{corev1.ServiceAccountTokenKey: nil} }, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ann := map[string]string{}
			for k, v := range own {
				ann[k] = v
			}
			s := saTokenSecret("s", ann)
			tt.mutate(&s)
			if got := IsServiceAccountTokenSecretFor(&s, sa); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
	if IsServiceAccountTokenSecretFor(nil, sa) || IsServiceAccountTokenSecretFor(&corev1.Secret{}, nil) {
		t.Fatal("nil 输入应返回 false")
	}
}

func TestSelectServiceAccountTokenSecret(t *testing.T) {
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ops", UID: types.UID("uid-app")}}
	opaque := saTokenSecret("cfg", nil)
	opaque.Type = corev1.SecretTypeOpaque
	secrets := []corev1.Secret{
		saTokenSecret("admin-token", map[string]string{corev1.ServiceAccountNameKey: "admin"}),
		opaque,
		saTokenSecret("app-token", map[string]string{corev1.ServiceAccountNameKey: "app", corev1.ServiceAccountUIDKey: "uid-app"}),
	}
	selected, rejected := SelectServiceAccountTokenSecret(sa, secrets)
	if selected == nil || selected.Name != "app-token" {
		t.Fatalf("应选中 SA 自己的 token Secret: %+v", selected)
	}
	if len(rejected) != 1 || rejected[0] != "admin-token" {
		t.Fatalf("rejected 应只含借挂的 token Secret: %v", rejected)
	}
	if s, _ := SelectServiceAccountTokenSecret(sa, secrets[:2]); s != nil {
		t.Fatal("没有属于自己的 token Secret 时应返回 nil")
	}
	if names := ServiceAccountSecretRefNames([]corev1.ObjectReference{{Name: "a"}, {Name: ""}, {Name: "b"}}); len(names) < 2 {
		t.Fatalf("names=%v", names)
	}
}

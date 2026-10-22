package utils

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func TestQuotaAdmissionOnDedicatedKind(t *testing.T) {
	path := os.Getenv("CLOUDOPS_QUOTA_TEST_KUBECONFIG")
	if path == "" {
		t.Skip("dedicated kind integration disabled")
	}
	raw, err := clientcmd.LoadFromFile(path)
	if err != nil {
		t.Fatal("cannot load dedicated kubeconfig")
	}
	if raw.CurrentContext != "kind-cloudops-release" {
		t.Fatal("refusing non-dedicated context")
	}
	config, err := clientcmd.NewDefaultClientConfig(*raw, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		t.Fatal("invalid dedicated config")
	}
	target, _ := url.Parse(config.Host)
	if target.Hostname() != "127.0.0.1" && target.Hostname() != "localhost" {
		t.Fatal("refusing non-loopback cluster")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes.Items) == 0 {
		t.Fatal("no dedicated nodes")
	}
	for _, node := range nodes.Items {
		if !strings.HasPrefix(node.Name, "cloudops-release-") {
			t.Fatal("refusing foreign cluster")
		}
	}
	ns := "cloudops-quota-" + uuid.NewString()[:8]
	created, err := client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns, Labels: map[string]string{"cloudops.io/quota-test": "true"}}}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		uid := created.UID
		if err := client.CoreV1().Namespaces().Delete(cleanup, ns, metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &uid}}); err != nil {
			t.Error("owned namespace cleanup failed", err)
		}
	})
	cluster := &model.K8sCluster{Model: model.Model{ID: 424242}, CpuRequest: "100", CpuLimit: "200", RestrictNamespace: model.StringList{ns}}
	if err := AddClusterResourceLimit(ctx, client, cluster); err != nil {
		t.Fatal(err)
	}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "bounded", Namespace: ns}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "web", Image: "busybox:1.37.0", Command: []string{"sleep", "120"}, Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("200m")}, Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("200m")}}}}}}
	if _, err := client.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{}); !apierrors.IsForbidden(err) {
		t.Fatalf("quota did not reject over-budget Pod: %v", err)
	}
	cluster.CpuRequest = "300"
	cluster.CpuLimit = "400"
	if err := AddClusterResourceLimit(ctx, client, cluster); err != nil {
		t.Fatal(err)
	}
	// Quota status and admission caches converge asynchronously after a spec update.
	deadline := time.Now().Add(15 * time.Second)
	for {
		_, err := client.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{})
		if err == nil {
			break
		}
		if !apierrors.IsForbidden(err) || !strings.Contains(err.Error(), "exceeded quota") || time.Now().After(deadline) {
			t.Fatal("updated quota failed to admit valid Pod", err)
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}
	if err := RemoveClusterResourceLimits(ctx, client, cluster.ID, []string{ns}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CoreV1().ResourceQuotas(ns).Get(ctx, clusterQuotaName(cluster.ID), metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("quota cleanup did not persist: %v", err)
	}
}

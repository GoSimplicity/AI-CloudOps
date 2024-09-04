package di

import (
	"context"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func TestListPodsInNamespace(t *testing.T) {
	cli := InitK8sClient()
	namespaceName := "kube-system"
	var podList corev1.PodList
	err := cli.List(context.Background(), &podList, client.InNamespace(namespaceName))
	assert.NoError(t, err, "failed to list pods in namespace")
	t.Logf("Found %d pods in namespace %s", len(podList.Items), namespaceName)
	assert.Greater(t, len(podList.Items), 0, "No pods found in namespace")
}

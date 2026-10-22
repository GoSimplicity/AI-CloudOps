package utils

import (
	"context"
	"errors"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestQuotaValidationAndExistingUnits(t *testing.T) {
	c := &model.K8sCluster{CpuRequest: "500", CpuLimit: "2000", MemoryRequest: "1024", MemoryLimit: "2Gi", RestrictNamespace: model.StringList{"team-a"}}
	hard, err := clusterQuotaResources(c)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[corev1.ResourceName]string{corev1.ResourceRequestsCPU: "500m", corev1.ResourceLimitsCPU: "2", corev1.ResourceRequestsMemory: "1Gi", corev1.ResourceLimitsMemory: "2Gi"} {
		got := hard[name]
		expected := resource.MustParse(want)
		if got.Cmp(expected) != 0 {
			t.Fatalf("%s: got %s want %s", name, got.String(), want)
		}
	}
	for name, c := range map[string]*model.K8sCluster{
		"nil": nil, "negative": {CpuRequest: "-1"}, "malformed": {MemoryLimit: "invalid"}, "request above limit": {CpuRequest: "500", CpuLimit: "100"}, "namespace": {RestrictNamespace: model.StringList{"UPPER"}}, "duplicate": {RestrictNamespace: model.StringList{"team-a", "team-a"}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateResourceQuantities(c); err == nil {
				t.Fatal("invalid quota accepted")
			}
		})
	}
}

func TestQuotaOwnershipScopeUpdateAndRemoval(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	c := &model.K8sCluster{Model: model.Model{ID: 42}, CpuRequest: "500", RestrictNamespace: model.StringList{"team-a"}}
	if err := AddClusterResourceLimit(ctx, client, c); err != nil {
		t.Fatal(err)
	}
	c.CpuRequest = "700"
	if err := AddClusterResourceLimit(ctx, client, c); err != nil {
		t.Fatal(err)
	}
	q, err := client.CoreV1().ResourceQuotas("team-a").Get(ctx, clusterQuotaName(42), metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	got := q.Spec.Hard[corev1.ResourceRequestsCPU]
	if got.MilliValue() != 700 {
		t.Fatalf("quota update not persisted: %s", got.String())
	}
	if q.Labels[quotaOwnerLabel] != "42" {
		t.Fatal("ownership missing")
	}
	if err := RemoveClusterResourceLimits(ctx, client, 42, []string{"team-a"}); err != nil {
		t.Fatal(err)
	}
	if err := RemoveClusterResourceLimits(ctx, client, 42, []string{"team-a"}); err != nil {
		t.Fatal("cleanup not idempotent", err)
	}
	_, err = client.CoreV1().ResourceQuotas("team-a").Create(ctx, &corev1.ResourceQuota{ObjectMeta: metav1.ObjectMeta{Name: clusterQuotaName(42)}}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := AddClusterResourceLimit(ctx, client, c); err == nil {
		t.Fatal("overwrote manual quota")
	}
	if err := RemoveClusterResourceLimits(ctx, client, 42, []string{"team-a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CoreV1().ResourceQuotas("team-a").Get(ctx, clusterQuotaName(42), metav1.GetOptions{}); err != nil {
		t.Fatal("deleted manual quota", err)
	}
	untouched := fake.NewSimpleClientset()
	c.RestrictNamespace = nil
	if err := AddClusterResourceLimit(ctx, untouched, c); err != nil {
		t.Fatal(err)
	}
	if len(untouched.Actions()) != 0 {
		t.Fatal("empty namespace selection must not read or modify cluster")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := AddClusterResourceLimit(cancelled, untouched, c); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled operation not rejected", err)
	}
	if len(untouched.Actions()) != 0 {
		t.Fatal("cancelled operation reached API")
	}
}

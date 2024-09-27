package k8s

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
)

// EnsureNamespace 确保指定的命名空间存在，如果不存在则创建
func EnsureNamespace(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string) error {
	_, err := kubeClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// 创建命名空间
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_, createErr := kubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
			if createErr != nil {
				log.Printf("EnsureNamespace: 创建命名空间失败 %s: %v", namespace, createErr)
				return fmt.Errorf("创建命名空间 %s 失败: %w", namespace, createErr)
			}
			log.Printf("EnsureNamespace: 命名空间创建成功 %s", namespace)
			return nil
		}
		log.Printf("EnsureNamespace: 获取命名空间失败 %s: %v", namespace, err)
		return fmt.Errorf("获取命名空间 %s 失败: %w", namespace, err)
	}
	// 命名空间已存在
	log.Printf("EnsureNamespace: 命名空间已存在 %s", namespace)
	return nil
}

// ApplyLimitRange 应用 LimitRange 到指定命名空间
func ApplyLimitRange(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string, cluster *model.K8sCluster) error {
	limitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "resource-limits",
			Namespace: namespace,
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type: corev1.LimitTypeContainer,
					Default: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(cluster.CpuLimit),
						corev1.ResourceMemory: resource.MustParse(cluster.MemoryLimit),
					},
					DefaultRequest: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(cluster.CpuRequest),
						corev1.ResourceMemory: resource.MustParse(cluster.MemoryRequest),
					},
				},
			},
		},
	}

	_, err := kubeClient.CoreV1().LimitRanges(namespace).Create(ctx, limitRange, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			log.Printf("ApplyLimitRange: LimitRange 已存在 %s", namespace)
			return nil
		}
		log.Printf("ApplyLimitRange: 创建 LimitRange 失败 %s: %v", namespace, err)
		return fmt.Errorf("创建 LimitRange 失败 (namespace: %s): %w", namespace, err)
	}

	log.Printf("ApplyLimitRange: LimitRange 创建成功 %s", namespace)
	return nil
}

// ApplyResourceQuota 应用 ResourceQuota 到指定命名空间
func ApplyResourceQuota(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string, cluster *model.K8sCluster) error {
	resourceQuota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "compute-quota",
			Namespace: namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceRequestsCPU:    resource.MustParse(cluster.CpuRequest),
				corev1.ResourceRequestsMemory: resource.MustParse(cluster.MemoryRequest),
				corev1.ResourceLimitsCPU:      resource.MustParse(cluster.CpuLimit),
				corev1.ResourceLimitsMemory:   resource.MustParse(cluster.MemoryLimit),
			},
		},
	}

	_, err := kubeClient.CoreV1().ResourceQuotas(namespace).Create(ctx, resourceQuota, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			log.Printf("ApplyResourceQuota: ResourceQuota 已存在 %s", namespace)
			return nil
		}
		log.Printf("ApplyResourceQuota: 创建 ResourceQuota 失败 %s: %v", namespace, err)
		return fmt.Errorf("创建 ResourceQuota 失败 (namespace: %s): %w", namespace, err)
	}

	log.Printf("ApplyResourceQuota: ResourceQuota 创建成功 %s", namespace)
	return nil
}

// GetTaintsMapFromTaints 将 taints 转换为键为 "Key:Value:Effect" 的 map
func GetTaintsMapFromTaints(taints []corev1.Taint) map[string]corev1.Taint {
	taintsMap := make(map[string]corev1.Taint)
	for _, taint := range taints {
		key := fmt.Sprintf("%s:%s:%s", taint.Key, taint.Value, taint.Effect)
		taintsMap[key] = taint
	}
	return taintsMap
}

// MergeTaints 合并新的 taints，避免重复
func MergeTaints(existingTaints []corev1.Taint, newTaints []corev1.Taint) []corev1.Taint {
	taintsMap := GetTaintsMapFromTaints(existingTaints)

	for _, newTaint := range newTaints {
		key := fmt.Sprintf("%s:%s:%s", newTaint.Key, newTaint.Value, newTaint.Effect)
		if _, exists := taintsMap[key]; !exists {
			existingTaints = append(existingTaints, newTaint)
		}
	}
	return existingTaints
}

// RemoveTaints 从现有的 taints 中删除指定的 taints
func RemoveTaints(existingTaints []corev1.Taint, taintsToDelete []corev1.Taint) []corev1.Taint {
	taintsMap := GetTaintsMapFromTaints(taintsToDelete)

	var updatedTaints []corev1.Taint
	for _, existingTaint := range existingTaints {
		key := fmt.Sprintf("%s:%s:%s", existingTaint.Key, existingTaint.Value, existingTaint.Effect)
		if _, shouldDelete := taintsMap[key]; !shouldDelete {
			updatedTaints = append(updatedTaints, existingTaint)
		}
	}
	return updatedTaints
}

/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package manager

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatefulSetManager 定义 StatefulSet 资源管理接口
type StatefulSetManager interface {
	// StatefulSet CRUD 操作
	CreateStatefulSet(ctx context.Context, clusterID int, namespace string, statefulSet *appsv1.StatefulSet) error
	GetStatefulSet(ctx context.Context, clusterID int, namespace, name string) (*appsv1.StatefulSet, error)
	GetStatefulSetList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*appsv1.StatefulSetList, error)
	UpdateStatefulSet(ctx context.Context, clusterID int, namespace string, statefulSet *appsv1.StatefulSet) error
	DeleteStatefulSet(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error

	// StatefulSet 操作
	RestartStatefulSet(ctx context.Context, clusterID int, namespace, name string) error
	ScaleStatefulSet(ctx context.Context, clusterID int, namespace, name string, replicas int32) error

	// 批量操作
	BatchDeleteStatefulSets(ctx context.Context, clusterID int, namespace string, statefulSetNames []string) error
	BatchRestartStatefulSets(ctx context.Context, clusterID int, namespace string, statefulSetNames []string) error
}

type statefulSetManager struct {
	clientFactory client.K8sClient
}

// NewStatefulSetManager 创建新的 StatefulSet 管理器
func NewStatefulSetManager(clientFactory client.K8sClient) StatefulSetManager {
	return &statefulSetManager{
		clientFactory: clientFactory,
	}
}

// CreateStatefulSet 创建 StatefulSet
func (s *statefulSetManager) CreateStatefulSet(ctx context.Context, clusterID int, namespace string, statefulSet *appsv1.StatefulSet) error {
	clientset, err := s.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = clientset.AppsV1().StatefulSets(namespace).Create(ctx, statefulSet, metav1.CreateOptions{})
	return err
}

// GetStatefulSet 获取单个 StatefulSet
func (s *statefulSetManager) GetStatefulSet(ctx context.Context, clusterID int, namespace, name string) (*appsv1.StatefulSet, error) {
	clientset, err := s.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	return clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
}

// GetStatefulSetList 获取 StatefulSet 列表
func (s *statefulSetManager) GetStatefulSetList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*appsv1.StatefulSetList, error) {
	clientset, err := s.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	return clientset.AppsV1().StatefulSets(namespace).List(ctx, listOptions)
}

// UpdateStatefulSet 更新 StatefulSet
func (s *statefulSetManager) UpdateStatefulSet(ctx context.Context, clusterID int, namespace string, statefulSet *appsv1.StatefulSet) error {
	clientset, err := s.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = clientset.AppsV1().StatefulSets(namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
	return err
}

// DeleteStatefulSet 删除 StatefulSet
func (s *statefulSetManager) DeleteStatefulSet(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	clientset, err := s.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	return clientset.AppsV1().StatefulSets(namespace).Delete(ctx, name, deleteOptions)
}

// RestartStatefulSet 重启 StatefulSet
func (s *statefulSetManager) RestartStatefulSet(ctx context.Context, clusterID int, namespace, name string) error {
	clientset, err := s.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 获取当前 StatefulSet
	statefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// 添加重启注解触发重启
	if statefulSet.Spec.Template.Annotations == nil {
		statefulSet.Spec.Template.Annotations = make(map[string]string)
	}
	statefulSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().Format("2006-01-02T15:04:05Z")

	_, err = clientset.AppsV1().StatefulSets(namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
	return err
}

// ScaleStatefulSet 扩缩容 StatefulSet
func (s *statefulSetManager) ScaleStatefulSet(ctx context.Context, clusterID int, namespace, name string, replicas int32) error {
	clientset, err := s.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 获取当前 StatefulSet
	statefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// 更新副本数
	statefulSet.Spec.Replicas = &replicas
	_, err = clientset.AppsV1().StatefulSets(namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
	return err
}

// BatchDeleteStatefulSets 批量删除 StatefulSet
func (s *statefulSetManager) BatchDeleteStatefulSets(ctx context.Context, clusterID int, namespace string, statefulSetNames []string) error {
	for _, name := range statefulSetNames {
		if err := s.DeleteStatefulSet(ctx, clusterID, namespace, name, metav1.DeleteOptions{}); err != nil {
			return err
		}
	}
	return nil
}

// BatchRestartStatefulSets 批量重启 StatefulSet
func (s *statefulSetManager) BatchRestartStatefulSets(ctx context.Context, clusterID int, namespace string, statefulSetNames []string) error {
	for _, name := range statefulSetNames {
		if err := s.RestartStatefulSet(ctx, clusterID, namespace, name); err != nil {
			return err
		}
	}
	return nil
}

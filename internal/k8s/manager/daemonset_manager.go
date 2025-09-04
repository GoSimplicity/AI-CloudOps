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

// DaemonSetManager 定义 DaemonSet 资源管理接口
type DaemonSetManager interface {
	// DaemonSet CRUD 操作
	CreateDaemonSet(ctx context.Context, clusterID int, namespace string, daemonSet *appsv1.DaemonSet) error
	GetDaemonSet(ctx context.Context, clusterID int, namespace, name string) (*appsv1.DaemonSet, error)
	GetDaemonSetList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*appsv1.DaemonSetList, error)
	UpdateDaemonSet(ctx context.Context, clusterID int, namespace string, daemonSet *appsv1.DaemonSet) error
	DeleteDaemonSet(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error

	// DaemonSet 操作
	RestartDaemonSet(ctx context.Context, clusterID int, namespace, name string) error

	// 批量操作
	BatchDeleteDaemonSets(ctx context.Context, clusterID int, namespace string, daemonSetNames []string) error
	BatchRestartDaemonSets(ctx context.Context, clusterID int, namespace string, daemonSetNames []string) error
}

type daemonSetManager struct {
	clientFactory client.K8sClient
}

// NewDaemonSetManager 创建新的 DaemonSet 管理器
func NewDaemonSetManager(clientFactory client.K8sClient) DaemonSetManager {
	return &daemonSetManager{
		clientFactory: clientFactory,
	}
}

// CreateDaemonSet 创建 DaemonSet
func (d *daemonSetManager) CreateDaemonSet(ctx context.Context, clusterID int, namespace string, daemonSet *appsv1.DaemonSet) error {
	clientset, err := d.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = clientset.AppsV1().DaemonSets(namespace).Create(ctx, daemonSet, metav1.CreateOptions{})
	return err
}

// GetDaemonSet 获取单个 DaemonSet
func (d *daemonSetManager) GetDaemonSet(ctx context.Context, clusterID int, namespace, name string) (*appsv1.DaemonSet, error) {
	clientset, err := d.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	return clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
}

// GetDaemonSetList 获取 DaemonSet 列表
func (d *daemonSetManager) GetDaemonSetList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*appsv1.DaemonSetList, error) {
	clientset, err := d.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	return clientset.AppsV1().DaemonSets(namespace).List(ctx, listOptions)
}

// UpdateDaemonSet 更新 DaemonSet
func (d *daemonSetManager) UpdateDaemonSet(ctx context.Context, clusterID int, namespace string, daemonSet *appsv1.DaemonSet) error {
	clientset, err := d.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = clientset.AppsV1().DaemonSets(namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
	return err
}

// DeleteDaemonSet 删除 DaemonSet
func (d *daemonSetManager) DeleteDaemonSet(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	clientset, err := d.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	return clientset.AppsV1().DaemonSets(namespace).Delete(ctx, name, deleteOptions)
}

// RestartDaemonSet 重启 DaemonSet
func (d *daemonSetManager) RestartDaemonSet(ctx context.Context, clusterID int, namespace, name string) error {
	clientset, err := d.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 获取当前 DaemonSet
	daemonSet, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// 添加重启注解触发重启
	if daemonSet.Spec.Template.Annotations == nil {
		daemonSet.Spec.Template.Annotations = make(map[string]string)
	}
	daemonSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().Format("2006-01-02T15:04:05Z")

	_, err = clientset.AppsV1().DaemonSets(namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
	return err
}

// BatchDeleteDaemonSets 批量删除 DaemonSet
func (d *daemonSetManager) BatchDeleteDaemonSets(ctx context.Context, clusterID int, namespace string, daemonSetNames []string) error {
	for _, name := range daemonSetNames {
		if err := d.DeleteDaemonSet(ctx, clusterID, namespace, name, metav1.DeleteOptions{}); err != nil {
			return err
		}
	}
	return nil
}

// BatchRestartDaemonSets 批量重启 DaemonSet
func (d *daemonSetManager) BatchRestartDaemonSets(ctx context.Context, clusterID int, namespace string, daemonSetNames []string) error {
	for _, name := range daemonSetNames {
		if err := d.RestartDaemonSet(ctx, clusterID, namespace, name); err != nil {
			return err
		}
	}
	return nil
}

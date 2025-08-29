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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// DeploymentManager Deployment 资源管理器，负责 Deployment 相关的业务逻辑
// 通过依赖注入接收客户端工厂，实现业务逻辑与客户端创建的解耦
type DeploymentManager interface {
	// Deployment CRUD 操作
	CreateDeployment(ctx context.Context, clusterID int, namespace string, deployment *appsv1.Deployment) error
	GetDeployment(ctx context.Context, clusterID int, namespace, name string) (*appsv1.Deployment, error)
	GetDeploymentList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*appsv1.DeploymentList, error)
	UpdateDeployment(ctx context.Context, clusterID int, namespace string, deployment *appsv1.Deployment) error
	DeleteDeployment(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error

	// Deployment 操作
	RestartDeployment(ctx context.Context, clusterID int, namespace, name string) error
	ScaleDeployment(ctx context.Context, clusterID int, namespace, name string, replicas int32) error

	// 批量操作
	BatchDeleteDeployments(ctx context.Context, clusterID int, namespace string, deploymentNames []string) error
	BatchRestartDeployments(ctx context.Context, clusterID int, namespace string, deploymentNames []string) error
}

type deploymentManager struct {
	clientFactory client.K8sClientFactory
	logger        *zap.Logger
}

// NewDeploymentManager 创建新的 Deployment 管理器实例
// 通过构造函数注入客户端工厂依赖
func NewDeploymentManager(clientFactory client.K8sClientFactory, logger *zap.Logger) DeploymentManager {
	return &deploymentManager{
		clientFactory: clientFactory,
		logger:        logger,
	}
}

// getKubeClient 私有方法：获取 Kubernetes 客户端
// 封装客户端获取逻辑，统一错误处理
func (d *deploymentManager) getKubeClient(clusterID int) (*kubernetes.Clientset, error) {
	kubeClient, err := d.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		d.logger.Error("获取 Kubernetes 客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}
	return kubeClient, nil
}

// CreateDeployment 创建 Deployment
func (d *deploymentManager) CreateDeployment(ctx context.Context, clusterID int, namespace string, deployment *appsv1.Deployment) error {
	if deployment == nil {
		return fmt.Errorf("deployment 不能为空")
	}

	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = kubeClient.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		d.logger.Error("创建 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", deployment.Name),
			zap.Error(err))
		return fmt.Errorf("创建 Deployment 失败: %w", err)
	}

	d.logger.Info("成功创建 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", deployment.Name))
	return nil
}

// GetDeployment 获取单个 Deployment
func (d *deploymentManager) GetDeployment(ctx context.Context, clusterID int, namespace, name string) (*appsv1.Deployment, error) {
	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		d.logger.Error("获取 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, fmt.Errorf("获取 Deployment 失败: %w", err)
	}

	d.logger.Debug("成功获取 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return deployment, nil
}

// GetDeploymentList 获取 Deployment 列表
func (d *deploymentManager) GetDeploymentList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*appsv1.DeploymentList, error) {
	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	deploymentList, err := kubeClient.AppsV1().Deployments(namespace).List(ctx, listOptions)
	if err != nil {
		d.logger.Error("获取 Deployment 列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, fmt.Errorf("获取 Deployment 列表失败: %w", err)
	}

	d.logger.Debug("成功获取 Deployment 列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(deploymentList.Items)))
	return deploymentList, nil
}

// UpdateDeployment 更新 Deployment
func (d *deploymentManager) UpdateDeployment(ctx context.Context, clusterID int, namespace string, deployment *appsv1.Deployment) error {
	if deployment == nil {
		return fmt.Errorf("deployment 不能为空")
	}

	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = kubeClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		d.logger.Error("更新 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", deployment.Name),
			zap.Error(err))
		return fmt.Errorf("更新 Deployment 失败: %w", err)
	}

	d.logger.Info("成功更新 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", deployment.Name))
	return nil
}

// DeleteDeployment 删除 Deployment
func (d *deploymentManager) DeleteDeployment(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = kubeClient.AppsV1().Deployments(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		d.logger.Error("删除 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("删除 Deployment 失败: %w", err)
	}

	d.logger.Info("成功删除 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return nil
}

// RestartDeployment 重启 Deployment
func (d *deploymentManager) RestartDeployment(ctx context.Context, clusterID int, namespace, name string) error {
	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 使用 Patch 方式重启
	patchData := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`,
		time.Now().Format(time.RFC3339))

	_, err = kubeClient.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
	if err != nil {
		d.logger.Error("重启 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("重启 Deployment 失败: %w", err)
	}

	d.logger.Info("成功重启 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return nil
}

// ScaleDeployment 扩缩容 Deployment
func (d *deploymentManager) ScaleDeployment(ctx context.Context, clusterID int, namespace, name string, replicas int32) error {
	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 获取当前 Deployment
	deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		d.logger.Error("获取 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("获取 Deployment 失败: %w", err)
	}

	// 更新副本数
	deployment.Spec.Replicas = &replicas
	_, err = kubeClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		d.logger.Error("扩缩容 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Int32("replicas", replicas),
			zap.Error(err))
		return fmt.Errorf("扩缩容 Deployment 失败: %w", err)
	}

	d.logger.Info("成功扩缩容 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name),
		zap.Int32("replicas", replicas))
	return nil
}

// BatchDeleteDeployments 批量删除 Deployment
func (d *deploymentManager) BatchDeleteDeployments(ctx context.Context, clusterID int, namespace string, deploymentNames []string) error {
	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(deploymentNames))

	for _, deploymentName := range deploymentNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				errors <- fmt.Errorf("删除 Deployment %s 失败: %w", name, err)
			}
		}(deploymentName)
	}

	wg.Wait()
	close(errors)

	var errorMessages []string
	for err := range errors {
		errorMessages = append(errorMessages, err.Error())
		d.logger.Error("批量删除中的单个 Deployment 失败", zap.Error(err))
	}

	if len(errorMessages) > 0 {
		return fmt.Errorf("批量删除失败，详情: %s", strings.Join(errorMessages, "; "))
	}

	d.logger.Info("成功批量删除 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(deploymentNames)))
	return nil
}

// BatchRestartDeployments 批量重启 Deployment
func (d *deploymentManager) BatchRestartDeployments(ctx context.Context, clusterID int, namespace string, deploymentNames []string) error {
	var wg sync.WaitGroup
	errors := make(chan error, len(deploymentNames))

	for _, deploymentName := range deploymentNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := d.RestartDeployment(ctx, clusterID, namespace, name); err != nil {
				errors <- fmt.Errorf("重启 Deployment %s 失败: %w", name, err)
			}
		}(deploymentName)
	}

	wg.Wait()
	close(errors)

	var errorMessages []string
	for err := range errors {
		errorMessages = append(errorMessages, err.Error())
		d.logger.Error("批量重启中的单个 Deployment 失败", zap.Error(err))
	}

	if len(errorMessages) > 0 {
		return fmt.Errorf("批量重启失败，详情: %s", strings.Join(errorMessages, "; "))
	}

	d.logger.Info("成功批量重启 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(deploymentNames)))
	return nil
}

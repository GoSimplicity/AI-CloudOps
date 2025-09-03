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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type DeploymentManager interface {
	CreateDeployment(ctx context.Context, clusterID int, namespace string, deployment *appsv1.Deployment) error
	GetDeployment(ctx context.Context, clusterID int, namespace, name string) (*appsv1.Deployment, error)
	GetDeploymentList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sDeployment, error)
	UpdateDeployment(ctx context.Context, clusterID int, namespace string, deployment *appsv1.Deployment) error
	DeleteDeployment(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error
	RestartDeployment(ctx context.Context, clusterID int, namespace, name string) error
	ScaleDeployment(ctx context.Context, clusterID int, namespace, name string, replicas int32) error
	BatchDeleteDeployments(ctx context.Context, clusterID int, namespace string, deploymentNames []string) error
	BatchRestartDeployments(ctx context.Context, clusterID int, namespace string, deploymentNames []string) error
	GetDeploymentEvents(ctx context.Context, clusterID int, namespace, deploymentName string, limit int) ([]*model.K8sDeploymentEvent, int64, error)
	GetDeploymentHistory(ctx context.Context, clusterID int, namespace, deploymentName string) ([]*model.K8sDeploymentHistory, int64, error)
	GetDeploymentPods(ctx context.Context, clusterID int, namespace, deploymentName string) ([]*model.K8sPod, int64, error)

	RollbackDeployment(ctx context.Context, clusterID int, namespace, name string, revision int64) error
	PauseDeployment(ctx context.Context, clusterID int, namespace, name string) error
	ResumeDeployment(ctx context.Context, clusterID int, namespace, name string) error
}
type deploymentManager struct {
	clientFactory client.K8sClient
	logger        *zap.Logger
}

func NewDeploymentManager(clientFactory client.K8sClient, logger *zap.Logger) DeploymentManager {
	return &deploymentManager{
		clientFactory: clientFactory,
		logger:        logger,
	}
}

// getKubeClient 获取客户端
func (d *deploymentManager) getKubeClient(clusterID int) (*kubernetes.Clientset, error) {
	kubeClient, err := d.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		d.logger.Error("获取 Kubernetes 客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}
	return kubeClient, nil
}

// CreateDeployment 创建deployment
func (d *deploymentManager) CreateDeployment(ctx context.Context, clusterID int, namespace string, deployment *appsv1.Deployment) error {
	if deployment == nil {
		return fmt.Errorf("deployment 不能为空")
	}

	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 如果deployment对象中没有指定namespace，使用参数中的namespace
	targetNamespace := deployment.Namespace
	if targetNamespace == "" {
		targetNamespace = namespace
		deployment.Namespace = namespace
	}

	_, err = kubeClient.AppsV1().Deployments(targetNamespace).Create(ctx, deployment, metav1.CreateOptions{})
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

// GetDeployment 获取deployment
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

// GetDeploymentList 获取deployment列表
func (d *deploymentManager) GetDeploymentList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sDeployment, error) {
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

	// 转换为model结构
	var k8sDeployments []*model.K8sDeployment
	for _, deployment := range deploymentList.Items {
		k8sDeployment := utils.ConvertToK8sDeployment(&deployment)
		k8sDeployments = append(k8sDeployments, k8sDeployment)
	}

	d.logger.Debug("成功获取 Deployment 列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(k8sDeployments)))
	return k8sDeployments, nil
}

// UpdateDeployment 更新deployment
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

// DeleteDeployment 删除deployment
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

// RestartDeployment 重启deployment
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

// ScaleDeployment 扩缩容deployment
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

// BatchDeleteDeployments 批量删除deployment
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

// BatchRestartDeployments 批量重启deployment
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

// GetDeploymentEvents 获取deployment事件
func (d *deploymentManager) GetDeploymentEvents(ctx context.Context, clusterID int, namespace, deploymentName string, limit int) ([]*model.K8sDeploymentEvent, int64, error) {
	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return nil, 0, err
	}

	events, total, err := utils.GetDeploymentEvents(ctx, kubeClient, namespace, deploymentName, limit)
	if err != nil {
		d.logger.Error("获取 Deployment 事件失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("deploymentName", deploymentName),
			zap.Error(err))
		return nil, 0, fmt.Errorf("获取 Deployment 事件失败: %w", err)
	}

	d.logger.Debug("成功获取 Deployment 事件",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("deploymentName", deploymentName),
		zap.Int("count", len(events)),
		zap.Int64("total", total))
	return events, total, nil
}

// GetDeploymentHistory 获取deployment历史
func (d *deploymentManager) GetDeploymentHistory(ctx context.Context, clusterID int, namespace, deploymentName string) ([]*model.K8sDeploymentHistory, int64, error) {
	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return nil, 0, err
	}

	history, total, err := utils.GetDeploymentHistory(ctx, kubeClient, namespace, deploymentName)
	if err != nil {
		d.logger.Error("获取 Deployment 历史失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("deploymentName", deploymentName),
			zap.Error(err))
		return nil, 0, fmt.Errorf("获取 Deployment 历史失败: %w", err)
	}

	d.logger.Debug("成功获取 Deployment 历史",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("deploymentName", deploymentName),
		zap.Int("count", len(history)),
		zap.Int64("total", total))
	return history, total, nil
}

// GetDeploymentPods 获取deployment的pod列表
func (d *deploymentManager) GetDeploymentPods(ctx context.Context, clusterID int, namespace, deploymentName string) ([]*model.K8sPod, int64, error) {
	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return nil, 0, err
	}

	pods, total, err := utils.GetDeploymentPods(ctx, kubeClient, namespace, deploymentName)
	if err != nil {
		d.logger.Error("获取 Deployment Pods 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("deploymentName", deploymentName),
			zap.Error(err))
		return nil, 0, fmt.Errorf("获取 Deployment Pods 失败: %w", err)
	}

	d.logger.Debug("成功获取 Deployment Pods",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("deploymentName", deploymentName),
		zap.Int("count", len(pods)),
		zap.Int64("total", total))
	return pods, total, nil
}

// RollbackDeployment 回滚 Deployment 到指定版本
func (d *deploymentManager) RollbackDeployment(ctx context.Context, clusterID int, namespace, name string, revision int64) error {
	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 获取 ReplicaSet 列表，找到指定版本的 ReplicaSet
	replicaSets, err := kubeClient.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		d.logger.Error("获取 ReplicaSet 列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("获取 ReplicaSet 列表失败: %w", err)
	}

	var targetReplicaSet *appsv1.ReplicaSet
	for _, rs := range replicaSets.Items {
		// 检查 ReplicaSet 是否属于该 Deployment
		for _, ownerRef := range rs.OwnerReferences {
			if ownerRef.Kind == "Deployment" && ownerRef.Name == name {
				if revisionStr, ok := rs.Annotations["deployment.kubernetes.io/revision"]; ok {
					if revisionStr == fmt.Sprintf("%d", revision) {
						targetReplicaSet = &rs
						break
					}
				}
			}
		}
		if targetReplicaSet != nil {
			break
		}
	}

	if targetReplicaSet == nil {
		return fmt.Errorf("未找到版本 %d 的 ReplicaSet", revision)
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

	// 使用目标 ReplicaSet 的 PodTemplateSpec 更新 Deployment
	deployment.Spec.Template = targetReplicaSet.Spec.Template

	// 更新 Deployment
	_, err = kubeClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		d.logger.Error("回滚 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Int64("revision", revision),
			zap.Error(err))
		return fmt.Errorf("回滚 Deployment 失败: %w", err)
	}

	d.logger.Info("成功回滚 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name),
		zap.Int64("revision", revision))
	return nil
}

// PauseDeployment 暂停 Deployment
func (d *deploymentManager) PauseDeployment(ctx context.Context, clusterID int, namespace, name string) error {
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

	// 检查是否已经暂停
	if deployment.Spec.Paused {
		d.logger.Info("Deployment 已经处于暂停状态",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name))
		return nil
	}

	// 设置暂停状态
	deployment.Spec.Paused = true

	// 更新 Deployment
	_, err = kubeClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		d.logger.Error("暂停 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("暂停 Deployment 失败: %w", err)
	}

	d.logger.Info("成功暂停 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return nil
}

// ResumeDeployment 恢复 Deployment
func (d *deploymentManager) ResumeDeployment(ctx context.Context, clusterID int, namespace, name string) error {
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

	// 检查是否已经恢复
	if !deployment.Spec.Paused {
		d.logger.Info("Deployment 已经处于运行状态",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name))
		return nil
	}

	// 设置恢复状态
	deployment.Spec.Paused = false

	// 更新 Deployment
	_, err = kubeClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		d.logger.Error("恢复 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("恢复 Deployment 失败: %w", err)
	}

	d.logger.Info("成功恢复 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return nil
}

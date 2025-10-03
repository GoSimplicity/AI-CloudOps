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

// DeploymentManager Deployment资源管理器接口
type DeploymentManager interface {
	CreateDeployment(ctx context.Context, clusterID int, namespace string, deployment *appsv1.Deployment) error
	GetDeployment(ctx context.Context, clusterID int, namespace, name string) (*appsv1.Deployment, error)
	GetDeploymentList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sDeployment, error)
	UpdateDeployment(ctx context.Context, clusterID int, namespace string, deployment *appsv1.Deployment) error
	DeleteDeployment(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error
	RestartDeployment(ctx context.Context, clusterID int, namespace, name string) error
	ScaleDeployment(ctx context.Context, clusterID int, namespace, name string, replicas int32) error
	RollbackDeployment(ctx context.Context, clusterID int, namespace, name string, revision int64) error
	PauseDeployment(ctx context.Context, clusterID int, namespace, name string) error
	ResumeDeployment(ctx context.Context, clusterID int, namespace, name string) error
	GetDeploymentHistory(ctx context.Context, clusterID int, namespace, deploymentName string) ([]*model.K8sDeploymentHistory, int64, error)
	GetDeploymentPods(ctx context.Context, clusterID int, namespace, deploymentName string) ([]*model.K8sPod, int64, error)
}

// deploymentManager Deployment资源管理器实现
type deploymentManager struct {
	clientFactory client.K8sClient
	logger        *zap.Logger
}

// NewDeploymentManager 创建新的Deployment管理器实例
func NewDeploymentManager(clientFactory client.K8sClient, logger *zap.Logger) DeploymentManager {
	return &deploymentManager{
		clientFactory: clientFactory,
		logger:        logger,
	}
}

// getKubeClient 私有方法：获取Kubernetes客户端
func (m *deploymentManager) getKubeClient(clusterID int) (*kubernetes.Clientset, error) {
	kubeClient, err := m.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}
	return kubeClient, nil
}

// CreateDeployment 创建deployment
func (m *deploymentManager) CreateDeployment(ctx context.Context, clusterID int, namespace string, deployment *appsv1.Deployment) error {
	if deployment == nil {
		return fmt.Errorf("deployment 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
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
		m.logger.Error("创建 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", deployment.Name),
			zap.Error(err))
		return fmt.Errorf("创建 Deployment 失败: %w", err)
	}

	m.logger.Info("成功创建 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", deployment.Name))
	return nil
}

// GetDeployment 获取deployment
func (m *deploymentManager) GetDeployment(ctx context.Context, clusterID int, namespace, name string) (*appsv1.Deployment, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, fmt.Errorf("获取 Deployment 失败: %w", err)
	}

	m.logger.Debug("成功获取 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return deployment, nil
}

// GetDeploymentList 获取deployment列表
func (m *deploymentManager) GetDeploymentList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sDeployment, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	deploymentList, err := kubeClient.AppsV1().Deployments(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取 Deployment 列表失败",
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

	m.logger.Debug("成功获取 Deployment 列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(k8sDeployments)))
	return k8sDeployments, nil
}

// UpdateDeployment 更新deployment
func (m *deploymentManager) UpdateDeployment(ctx context.Context, clusterID int, namespace string, deployment *appsv1.Deployment) error {
	if deployment == nil {
		return fmt.Errorf("deployment 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = kubeClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", deployment.Name),
			zap.Error(err))
		return fmt.Errorf("更新 Deployment 失败: %w", err)
	}

	m.logger.Info("成功更新 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", deployment.Name))
	return nil
}

// DeleteDeployment 删除deployment
func (m *deploymentManager) DeleteDeployment(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = kubeClient.AppsV1().Deployments(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		m.logger.Error("删除 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("删除 Deployment 失败: %w", err)
	}

	m.logger.Info("成功删除 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return nil
}

// RestartDeployment 重启deployment的所有pod
func (m *deploymentManager) RestartDeployment(ctx context.Context, clusterID int, namespace, name string) error {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	patchData := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`,
		time.Now().Format(time.RFC3339))

	_, err = kubeClient.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
	if err != nil {
		m.logger.Error("重启 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("重启 Deployment 失败: %w", err)
	}

	m.logger.Info("成功触发 Deployment 滚动重启，将逐个重启所有 Pod",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return nil
}

// ScaleDeployment 扩缩容deployment
func (m *deploymentManager) ScaleDeployment(ctx context.Context, clusterID int, namespace, name string, replicas int32) error {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	scale, err := kubeClient.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取 Deployment Scale 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("获取 Deployment Scale 失败: %w", err)
	}

	// 更新副本数
	scale.Spec.Replicas = replicas
	_, err = kubeClient.AppsV1().Deployments(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("扩缩容 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Int32("replicas", replicas),
			zap.Error(err))
		return fmt.Errorf("扩缩容 Deployment 失败: %w", err)
	}

	m.logger.Info("成功扩缩容 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name),
		zap.Int32("replicas", replicas))
	return nil
}

// GetDeploymentHistory 获取deployment历史
func (m *deploymentManager) GetDeploymentHistory(ctx context.Context, clusterID int, namespace, deploymentName string) ([]*model.K8sDeploymentHistory, int64, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, 0, err
	}

	history, total, err := utils.GetDeploymentHistory(ctx, kubeClient, namespace, deploymentName)
	if err != nil {
		m.logger.Error("获取 Deployment 历史失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("deploymentName", deploymentName),
			zap.Error(err))
		return nil, 0, fmt.Errorf("获取 Deployment 历史失败: %w", err)
	}

	m.logger.Debug("成功获取 Deployment 历史",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("deploymentName", deploymentName),
		zap.Int("count", len(history)),
		zap.Int64("total", total))
	return history, total, nil
}

// GetDeploymentPods 获取deployment的pod列表
func (m *deploymentManager) GetDeploymentPods(ctx context.Context, clusterID int, namespace, deploymentName string) ([]*model.K8sPod, int64, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, 0, err
	}

	pods, total, err := utils.GetDeploymentPods(ctx, kubeClient, namespace, deploymentName)
	if err != nil {
		m.logger.Error("获取 Deployment Pods 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("deploymentName", deploymentName),
			zap.Error(err))
		return nil, 0, fmt.Errorf("获取 Deployment Pods 失败: %w", err)
	}

	m.logger.Debug("成功获取 Deployment Pods",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("deploymentName", deploymentName),
		zap.Int("count", len(pods)),
		zap.Int64("total", total))
	return pods, total, nil
}

// RollbackDeployment 回滚 Deployment 到指定版本
// 通过查找对应版本的 ReplicaSet，并将其 PodTemplateSpec 应用到 Deployment
func (m *deploymentManager) RollbackDeployment(ctx context.Context, clusterID int, namespace, name string, revision int64) error {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 首先获取当前 Deployment 以获取其 selector
	deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("获取 Deployment 失败: %w", err)
	}

	// 使用 selector 过滤，只获取属于该 Deployment 的 ReplicaSet
	var labelSelectors []string
	if deployment.Spec.Selector != nil && deployment.Spec.Selector.MatchLabels != nil {
		for key, value := range deployment.Spec.Selector.MatchLabels {
			labelSelectors = append(labelSelectors, fmt.Sprintf("%s=%s", key, value))
		}
	}

	listOptions := metav1.ListOptions{}
	if len(labelSelectors) > 0 {
		listOptions.LabelSelector = strings.Join(labelSelectors, ",")
	}

	// 获取 ReplicaSet 列表，找到指定版本的 ReplicaSet
	replicaSets, err := kubeClient.AppsV1().ReplicaSets(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取 ReplicaSet 列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("获取 ReplicaSet 列表失败: %w", err)
	}

	var targetReplicaSet *appsv1.ReplicaSet
	for i := range replicaSets.Items {
		rs := &replicaSets.Items[i]

		// 检查 ReplicaSet 是否属于该 Deployment
		isOwned := false
		for _, ownerRef := range rs.OwnerReferences {
			if ownerRef.Kind == "Deployment" && ownerRef.Name == name {
				isOwned = true
				break
			}
		}

		// 如果这个 ReplicaSet 属于该 Deployment，检查版本
		if isOwned {
			if revisionStr, ok := rs.Annotations["deployment.kubernetes.io/revision"]; ok {
				if revisionStr == fmt.Sprintf("%d", revision) {
					targetReplicaSet = rs
					break
				}
			}
		}
	}

	if targetReplicaSet == nil {
		m.logger.Error("未找到指定版本的 ReplicaSet",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Int64("revision", revision))
		return fmt.Errorf("未找到版本 %d 的 ReplicaSet，请检查该版本是否存在", revision)
	}

	// 使用目标 ReplicaSet 的 PodTemplateSpec 更新 Deployment
	// 这会触发滚动更新，逐步替换为旧版本的 Pod
	deployment.Spec.Template = targetReplicaSet.Spec.Template

	// 添加回滚注解，方便追踪
	if deployment.Annotations == nil {
		deployment.Annotations = make(map[string]string)
	}
	deployment.Annotations["kubernetes.io/change-cause"] = fmt.Sprintf("Rollback to revision %d", revision)

	// 更新 Deployment
	_, err = kubeClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("回滚 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Int64("revision", revision),
			zap.Error(err))
		return fmt.Errorf("回滚 Deployment 失败: %w", err)
	}

	m.logger.Info("成功回滚 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name),
		zap.Int64("revision", revision),
		zap.String("targetReplicaSet", targetReplicaSet.Name))
	return nil
}

// PauseDeployment 暂停 Deployment
func (m *deploymentManager) PauseDeployment(ctx context.Context, clusterID int, namespace, name string) error {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 获取当前 Deployment
	deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("获取 Deployment 失败: %w", err)
	}

	// 检查是否已经暂停
	if deployment.Spec.Paused {
		m.logger.Info("Deployment 已经处于暂停状态",
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
		m.logger.Error("暂停 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("暂停 Deployment 失败: %w", err)
	}

	m.logger.Info("成功暂停 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return nil
}

// ResumeDeployment 恢复 Deployment
func (m *deploymentManager) ResumeDeployment(ctx context.Context, clusterID int, namespace, name string) error {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 获取当前 Deployment
	deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("获取 Deployment 失败: %w", err)
	}

	// 检查是否已经恢复
	if !deployment.Spec.Paused {
		m.logger.Info("Deployment 已经处于运行状态",
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
		m.logger.Error("恢复 Deployment 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("恢复 Deployment 失败: %w", err)
	}

	m.logger.Info("成功恢复 Deployment",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))
	return nil
}

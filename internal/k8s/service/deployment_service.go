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

package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type DeploymentService interface {
	// 获取Deployment列表
	GetDeploymentsByNamespace(ctx context.Context, id int, namespace string) ([]*appsv1.Deployment, error)
	GetDeploymentList(ctx context.Context, req *model.K8sGetResourceListReq) ([]*model.K8sDeployment, error)

	// 获取Deployment详情
	GetDeployment(ctx context.Context, req *model.K8sGetResourceReq) (*model.K8sDeployment, error)
	GetDeploymentYaml(ctx context.Context, id int, namespace, deploymentName string) (string, error)

	// Deployment操作
	UpdateDeployment(ctx context.Context, deployment *model.K8sDeploymentReq) error
	DeleteDeployment(ctx context.Context, id int, namespace, deploymentName string) error
	RestartDeployment(ctx context.Context, id int, namespace, deploymentName string) error

	// 批量操作
	BatchDeleteDeployment(ctx context.Context, id int, namespace string, deploymentNames []string) error
	BatchDeleteDeployments(ctx context.Context, req *model.DeploymentBatchDeleteReq) error
	BatchRestartDeployments(ctx context.Context, req *model.DeploymentBatchRestartReq) error
}

type deploymentService struct {
	dao               dao.ClusterDAO            // 保持对DAO的依赖
	client            client.K8sClient          // 保持向后兼容
	deploymentManager manager.DeploymentManager // 新的依赖注入
	logger            *zap.Logger
}

// NewDeploymentService 创建新的 DeploymentService 实例
func NewDeploymentService(dao dao.ClusterDAO, client client.K8sClient, deploymentManager manager.DeploymentManager, logger *zap.Logger) DeploymentService {
	return &deploymentService{
		dao:               dao,
		client:            client,
		deploymentManager: deploymentManager,
		logger:            logger,
	}
}

// GetDeploymentsByNamespace 获取指定命名空间下的所有 Deployment（保持向后兼容）
func (d *deploymentService) GetDeploymentsByNamespace(ctx context.Context, id int, namespace string) ([]*appsv1.Deployment, error) {
	// 使用 DeploymentManager 获取 Deployment 列表
	deploymentList, err := d.deploymentManager.GetDeploymentList(ctx, id, namespace, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Deployment list: %w", err)
	}

	var result []*appsv1.Deployment
	for i := range deploymentList.Items {
		result = append(result, &deploymentList.Items[i])
	}

	return result, nil
}

// GetDeploymentList 获取Deployment列表（使用新的请求结构体）
func (d *deploymentService) GetDeploymentList(ctx context.Context, req *model.K8sGetResourceListReq) ([]*model.K8sDeployment, error) {
	kubeClient, err := d.client.GetKubeClient(req.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	listOptions := utils.ConvertToMetaV1ListOptions(req)
	deploymentList, err := kubeClient.AppsV1().Deployments(req.Namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get Deployment list: %w", err)
	}

	deployments := make([]*model.K8sDeployment, 0, len(deploymentList.Items))
	for _, deploy := range deploymentList.Items {
		deploymentResponse := d.convertDeploymentToResponse(&deploy)
		deployments = append(deployments, deploymentResponse)
	}

	return deployments, nil
}

// GetDeployment 获取单个Deployment详情
func (d *deploymentService) GetDeployment(ctx context.Context, req *model.K8sGetResourceReq) (*model.K8sDeployment, error) {
	kubeClient, err := d.client.GetKubeClient(req.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	deployment, err := kubeClient.AppsV1().Deployments(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Deployment '%s': %w", req.ResourceName, err)
	}

	return d.convertDeploymentToResponse(deployment), nil
}

// UpdateDeployment 更新 Deployment 配置
func (d *deploymentService) UpdateDeployment(ctx context.Context, deployment *model.K8sDeploymentReq) error {
	kubeClient, err := d.client.GetKubeClient(deployment.ClusterId)
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	if deployment.DeploymentYaml == nil {
		return fmt.Errorf("deployment YAML cannot be nil")
	}

	if _, err := kubeClient.AppsV1().Deployments(deployment.Namespace).Update(ctx, deployment.DeploymentYaml, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update Deployment: %w", err)
	}

	return nil
}

// GetDeploymentYaml 获取 Deployment 的 YAML 配置
func (d *deploymentService) GetDeploymentYaml(ctx context.Context, id int, namespace, deploymentName string) (string, error) {
	kubeClient, err := d.client.GetKubeClient(id)
	if err != nil {
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get Deployment '%s': %w", deploymentName, err)
	}

	yamlBytes, err := yaml.Marshal(deployment)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Deployment to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

// BatchDeleteDeployment 批量删除 Deployment（保持向后兼容）
func (d *deploymentService) BatchDeleteDeployment(ctx context.Context, id int, namespace string, deploymentNames []string) error {
	kubeClient, err := d.client.GetKubeClient(id)
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(deploymentNames))

	for _, deploymentName := range deploymentNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				errors <- fmt.Errorf("failed to delete Deployment '%s': %w", name, err)
			}
		}(deploymentName)
	}

	wg.Wait()
	close(errors)

	var errorMessages []string
	for err := range errors {
		errorMessages = append(errorMessages, err.Error())
	}

	if len(errorMessages) > 0 {
		return fmt.Errorf("batch delete failed: %v", errorMessages)
	}

	return nil
}

// DeleteDeployment 删除指定的 Deployment
func (d *deploymentService) DeleteDeployment(ctx context.Context, id int, namespace, deploymentName string) error {
	// 使用 DeploymentManager 删除 Deployment
	err := d.deploymentManager.DeleteDeployment(ctx, id, namespace, deploymentName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete Deployment '%s': %w", deploymentName, err)
	}

	return nil
}

// RestartDeployment 重启指定的 Deployment
func (d *deploymentService) RestartDeployment(ctx context.Context, id int, namespace, deploymentName string) error {
	// 使用 DeploymentManager 重启 Deployment
	err := d.deploymentManager.RestartDeployment(ctx, id, namespace, deploymentName)
	if err != nil {
		return fmt.Errorf("failed to restart Deployment '%s': %w", deploymentName, err)
	}

	return nil
}

// BatchDeleteDeployments 批量删除Deployment（使用新的请求结构体）
func (d *deploymentService) BatchDeleteDeployments(ctx context.Context, req *model.DeploymentBatchDeleteReq) error {
	kubeClient, err := d.client.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(req.DeploymentNames))

	for _, deploymentName := range req.DeploymentNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.AppsV1().Deployments(req.Namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				errors <- fmt.Errorf("failed to delete Deployment '%s': %w", name, err)
			}
		}(deploymentName)
	}

	wg.Wait()
	close(errors)

	var errorMessages []string
	for err := range errors {
		errorMessages = append(errorMessages, err.Error())
	}

	if len(errorMessages) > 0 {
		return fmt.Errorf("batch delete failed: %v", errorMessages)
	}

	d.logger.Info("Batch deleted deployments successfully",
		zap.String("namespace", req.Namespace),
		zap.Int("count", len(req.DeploymentNames)))
	return nil
}

// BatchRestartDeployments 批量重启Deployment（使用新的请求结构体）
func (d *deploymentService) BatchRestartDeployments(ctx context.Context, req *model.DeploymentBatchRestartReq) error {
	kubeClient, err := d.client.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(req.DeploymentNames))

	for _, deploymentName := range req.DeploymentNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := d.restartSingleDeployment(ctx, kubeClient, req.Namespace, name); err != nil {
				errors <- fmt.Errorf("failed to restart Deployment '%s': %w", name, err)
			}
		}(deploymentName)
	}

	wg.Wait()
	close(errors)

	var errorMessages []string
	for err := range errors {
		errorMessages = append(errorMessages, err.Error())
	}

	if len(errorMessages) > 0 {
		return fmt.Errorf("batch restart failed: %v", errorMessages)
	}

	d.logger.Info("Batch restarted deployments successfully",
		zap.String("namespace", req.Namespace),
		zap.Int("count", len(req.DeploymentNames)))
	return nil
}

// convertDeploymentToResponse 将Kubernetes Deployment对象转换为响应模型
func (d *deploymentService) convertDeploymentToResponse(deployment *appsv1.Deployment) *model.K8sDeployment {
	// 获取镜像列表
	var images []string
	for _, container := range deployment.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	return &model.K8sDeployment{
		Name:              deployment.Name,
		UID:               string(deployment.UID),
		Namespace:         deployment.Namespace,
		Replicas:          *deployment.Spec.Replicas,
		ReadyReplicas:     deployment.Status.ReadyReplicas,
		AvailableReplicas: deployment.Status.AvailableReplicas,
		UpdatedReplicas:   deployment.Status.UpdatedReplicas,
		Strategy:          string(deployment.Spec.Strategy.Type),
		Labels:            deployment.Labels,
		Annotations:       deployment.Annotations,
		CreationTimestamp: deployment.CreationTimestamp.Time,
		Images:            images,
		Age:               pkg.GetAge(deployment.CreationTimestamp.Time),
	}
}

// restartSingleDeployment 重启单个Deployment的辅助方法
func (d *deploymentService) restartSingleDeployment(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, deploymentName string) error {
	deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}

	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = kubeClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

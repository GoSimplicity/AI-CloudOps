package admin

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

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeploymentService interface {
	GetDeploymentsByNamespace(ctx context.Context, id int, namespace string) ([]*appsv1.Deployment, error)
	UpdateDeployment(ctx context.Context, deployment *model.K8sDeploymentRequest) error
	BatchDeleteDeployment(ctx context.Context, id int, namespace string, deploymentNames []string) error
	BatchRestartDeployments(ctx context.Context, req *model.K8sDeploymentRequest) error
	GetDeploymentYaml(ctx context.Context, id int, namespace, deploymentName string) (string, error)
}

type deploymentService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

// NewDeploymentService 创建新的 DeploymentService 实例
func NewDeploymentService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) DeploymentService {
	return &deploymentService{dao: dao, client: client, logger: logger}
}

// GetDeploymentsByNamespace 获取指定命名空间下的所有 Deployment
func (d *deploymentService) GetDeploymentsByNamespace(ctx context.Context, id int, namespace string) ([]*appsv1.Deployment, error) {
	kubeClient, err := pkg.GetKubeClient(id, d.client, d.logger)
	if err != nil {
		d.logger.Error("Failed to get Kubernetes client", zap.Error(err))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	deployments, err := kubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		d.logger.Error("Failed to get Deployment list", zap.Error(err))
		return nil, fmt.Errorf("failed to get Deployment list: %w", err)
	}

	result := make([]*appsv1.Deployment, len(deployments.Items))
	for i := range deployments.Items {
		result[i] = &deployments.Items[i]
	}

	return result, nil
}

// GetDeploymentYaml 获取指定 Deployment 的 YAML 定义
func (d *deploymentService) GetDeploymentYaml(ctx context.Context, id int, namespace string, deploymentName string) (string, error) {
	kubeClient, err := pkg.GetKubeClient(id, d.client, d.logger)
	if err != nil {
		d.logger.Error("Failed to get Kubernetes client", zap.Error(err))
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		d.logger.Error("Failed to get Deployment", zap.Error(err))
		return "", fmt.Errorf("failed to get Deployment: %w", err)
	}

	yamlData, err := yaml.Marshal(deployment)
	if err != nil {
		d.logger.Error("Failed to serialize Deployment YAML", zap.Error(err))
		return "", fmt.Errorf("failed to serialize Deployment YAML: %w", err)
	}

	return string(yamlData), nil
}

// UpdateDeployment 更新 Deployment
func (d *deploymentService) UpdateDeployment(ctx context.Context, deploymentResource *model.K8sDeploymentRequest) error {
	kubeClient, err := pkg.GetKubeClient(deploymentResource.ClusterId, d.client, d.logger)
	if err != nil {
		d.logger.Error("Failed to get Kubernetes client", zap.Error(err))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	existingDeployment, err := kubeClient.AppsV1().Deployments(deploymentResource.Namespace).Get(ctx, deploymentResource.DeploymentYaml.Name, metav1.GetOptions{})
	if err != nil {
		d.logger.Error("Failed to get existing Deployment", zap.Error(err))
		return fmt.Errorf("failed to get existing Deployment: %w", err)
	}

	existingDeployment.Spec = deploymentResource.DeploymentYaml.Spec

	_, err = kubeClient.AppsV1().Deployments(deploymentResource.Namespace).Update(ctx, existingDeployment, metav1.UpdateOptions{})
	if err != nil {
		d.logger.Error("Failed to update Deployment", zap.Error(err))
		return fmt.Errorf("failed to update Deployment: %w", err)
	}

	return nil
}

// BatchDeleteDeployment 批量删除 Deployment
func (d *deploymentService) BatchDeleteDeployment(ctx context.Context, id int, namespace string, deploymentNames []string) error {
	kubeClient, err := pkg.GetKubeClient(id, d.client, d.logger)
	if err != nil {
		d.logger.Error("Failed to get Kubernetes client", zap.Error(err))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(deploymentNames))

	for _, name := range deploymentNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				d.logger.Error("Failed to delete Deployment", zap.String("DeploymentName", name), zap.Error(err))
				errChan <- fmt.Errorf("failed to delete Deployment '%s': %w", name, err)
			}
		}(name)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred while deleting Deployments: %v", errs)
	}

	return nil
}

// BatchRestartDeployments 批量重启 Deployment
func (d *deploymentService) BatchRestartDeployments(ctx context.Context, deploymentResource *model.K8sDeploymentRequest) error {
	kubeClient, err := pkg.GetKubeClient(deploymentResource.ClusterId, d.client, d.logger)
	if err != nil {
		d.logger.Error("Failed to get Kubernetes client", zap.Error(err))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(deploymentResource.DeploymentNames))

	for _, deploy := range deploymentResource.DeploymentNames {
		wg.Add(1)
		go func(deploy string) {
			defer wg.Done()
			deployment, err := kubeClient.AppsV1().Deployments(deploymentResource.Namespace).Get(ctx, deploy, metav1.GetOptions{})
			if err != nil {
				d.logger.Error("Failed to get Deployment", zap.String("DeploymentName", deploy), zap.Error(err))
				errChan <- fmt.Errorf("failed to get Deployment '%s': %w", deploy, err)
				return
			}

			if deployment.Spec.Template.Annotations == nil {
				deployment.Spec.Template.Annotations = make(map[string]string)
			}
			deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

			if _, err := kubeClient.AppsV1().Deployments(deploymentResource.Namespace).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
				d.logger.Error("Failed to update Deployment", zap.String("DeploymentName", deploy), zap.Error(err))
				errChan <- fmt.Errorf("failed to update Deployment '%s': %w", deploy, err)
			}
		}(deploy)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred while restarting Deployments: %v", errs)
	}

	return nil
}

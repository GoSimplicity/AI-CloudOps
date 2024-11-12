package admin

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sync"
	"time"
)

type DeploymentService interface {
	GetDeploymentsByNamespace(ctx context.Context, id int, namespace string) ([]*appsv1.Deployment, error)
	CreateDeployment(ctx context.Context, deployment *model.K8sDeploymentRequest) error
	UpdateDeployment(ctx context.Context, deployment *model.K8sDeploymentRequest) error
	BatchDeleteDeployment(ctx context.Context, id int, namespace string, deploymentNames []string) error
	BatchRestartDeployments(ctx context.Context, req *model.K8sDeploymentRequest) error
	GetDeploymentYaml(ctx context.Context, id int, namespace string, deploymentName string) (string, error)
}

type deploymentService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	l      *zap.Logger
}

// NewDeploymentService 创建新的 DeploymentService 实例
func NewDeploymentService(dao admin.ClusterDAO, client client.K8sClient, l *zap.Logger) DeploymentService {
	return &deploymentService{dao: dao, client: client, l: l}
}

// GetDeploymentsByNamespace 获取指定命名空间下的所有 Deployment
func (d *deploymentService) GetDeploymentsByNamespace(ctx context.Context, id int, namespace string) ([]*appsv1.Deployment, error) {
	kubeClient, err := pkg.GetKubeClient(id, d.client, d.l)
	if err != nil {
		d.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	deployments, err := kubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		d.l.Error("获取 Deployment 列表失败", zap.Error(err))
		return nil, err
	}

	result := make([]*appsv1.Deployment, len(deployments.Items))
	for i, deployment := range deployments.Items {
		result[i] = &deployment
	}

	return result, nil
}

// GetDeploymentYaml 获取指定 Deployment 的 YAML 定义
func (d *deploymentService) GetDeploymentYaml(ctx context.Context, id int, namespace string, deploymentName string) (string, error) {
	kubeClient, err := pkg.GetKubeClient(id, d.client, d.l)
	if err != nil {
		d.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return "", err
	}

	deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		d.l.Error("获取 Deployment 失败", zap.Error(err))
		return "", err
	}

	yamlData, err := yaml.Marshal(deployment)
	if err != nil {
		d.l.Error("序列化 Deployment YAML 失败", zap.Error(err))
		return "", err
	}

	return string(yamlData), nil
}

// CreateDeployment 创建新的 Deployment
func (d *deploymentService) CreateDeployment(ctx context.Context, deploymentResource *model.K8sDeploymentRequest) error {
	kubeClient, err := pkg.GetKubeClient(deploymentResource.ClusterId, d.client, d.l)
	if err != nil {
		d.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	_, err = kubeClient.AppsV1().Deployments(deploymentResource.Namespace).Create(ctx, deploymentResource.DeploymentYaml, metav1.CreateOptions{})
	if err != nil {
		d.l.Error("创建 Deployment 失败", zap.Error(err))
		return err
	}

	d.l.Info("创建 Deployment 成功", zap.String("deploymentName", deploymentResource.DeploymentYaml.Name))
	return nil
}

// UpdateDeployment 更新 Deployment
func (d *deploymentService) UpdateDeployment(ctx context.Context, deploymentResource *model.K8sDeploymentRequest) error {
	// 获取 Kubernetes 客户端
	kubeClient, err := pkg.GetKubeClient(deploymentResource.ClusterId, d.client, d.l)
	if err != nil {
		d.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 获取现有的 Deployment
	existingDeployment, err := kubeClient.AppsV1().Deployments(deploymentResource.Namespace).Get(ctx, deploymentResource.DeploymentYaml.Name, metav1.GetOptions{})
	if err != nil {
		d.l.Error("获取现有 Deployment 失败", zap.Error(err))
		return err
	}

	existingDeployment.Spec = deploymentResource.DeploymentYaml.Spec

	// 更新 Deployment
	_, err = kubeClient.AppsV1().Deployments(deploymentResource.Namespace).Update(ctx, existingDeployment, metav1.UpdateOptions{})
	if err != nil {
		d.l.Error("更新 Deployment 失败", zap.Error(err))
		return err
	}

	return nil
}

// BatchDeleteDeployment 批量删除 Deployment
func (d *deploymentService) BatchDeleteDeployment(ctx context.Context, id int, namespace string, deploymentNames []string) error {
	kubeClient, err := pkg.GetKubeClient(id, d.client, d.l)
	if err != nil {
		d.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(deploymentNames))
	for _, name := range deploymentNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				errChan <- err
				d.l.Error("删除 Deployment 失败", zap.String("DeploymentName", name), zap.Error(err))
			}
		}(name)
	}

	wg.Wait()
	close(errChan)
	if len(errChan) > 0 {
		return <-errChan
	}

	return nil
}

// BatchRestartDeployments 批量重启 Deployment
func (d *deploymentService) BatchRestartDeployments(ctx context.Context, deploymentResource *model.K8sDeploymentRequest) error {
	kubeClient, err := pkg.GetKubeClient(deploymentResource.ClusterId, d.client, d.l)
	if err != nil {
		d.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(deploymentResource.DeploymentNames))
	for _, deploy := range deploymentResource.DeploymentNames {
		wg.Add(1)
		go func(deploy string) {
			defer wg.Done()
			deployment, err := kubeClient.AppsV1().Deployments(deploymentResource.Namespace).Get(ctx, deploy, metav1.GetOptions{})
			if err != nil {
				errChan <- err
				d.l.Error("获取 Deployment 失败", zap.Error(err))
				return
			}

			if deployment.Spec.Template.Annotations == nil {
				deployment.Spec.Template.Annotations = make(map[string]string)
			}
			deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

			if _, err := kubeClient.AppsV1().Deployments(deployment.Namespace).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
				errChan <- err
				d.l.Error("更新 Deployment 失败", zap.Error(err))
				return
			}
		}(deploy)
	}

	wg.Wait()
	close(errChan)
	if len(errChan) > 0 {
		return <-errChan
	}

	return nil
}

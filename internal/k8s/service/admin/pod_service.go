package admin

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	"go.uber.org/zap"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

type PodService interface {
	// GetPodsByNamespace 获取指定命名空间的 Pod 列表
	GetPodsByNamespace(ctx context.Context, clusterID int, namespace string, podName string) ([]*model.K8sPod, error)
	// GetContainersByPod 获取指定 Pod 的容器列表
	GetContainersByPod(ctx context.Context, clusterID int, namespace string, podName string) ([]*model.K8sPodContainer, error)
	// GetContainerLogs 获取指定 Pod 的容器日志
	GetContainerLogs(ctx context.Context, clusterID int, namespace, podName, containerName string) (string, error)
	// GetPodYaml 获取指定 Pod 的 YAML 配置
	GetPodYaml(ctx context.Context, clusterID int, namespace, podName string) (*corev1.Pod, error)
	// GetPodsByNodeName 获取指定节点的 Pod 列表
	GetPodsByNodeName(ctx context.Context, id int, name string) ([]*model.K8sPod, error)
	// CreatePod 创建 Pod
	CreatePod(ctx context.Context, pod *model.K8sPodRequest) error
	// UpdatePod 更新 Pod
	UpdatePod(ctx context.Context, pod *model.K8sPodRequest) error
	// DeletePod 删除 Pod
	DeletePod(ctx context.Context, clusterName, namespace, podName string) error
}

type podService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	l      *zap.Logger
}

// NewPodService 用于创建 PodService 实例
func NewPodService(dao admin.ClusterDAO, client client.K8sClient, l *zap.Logger) PodService {
	return &podService{
		dao:    dao,
		client: client,
		l:      l,
	}
}

// GetPodsByNamespace 获取指定命名空间中的 Pod 列表
func (p *podService) GetPodsByNamespace(ctx context.Context, clusterID int, namespace string, podName string) ([]*model.K8sPod, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	listOptions := metav1.ListOptions{}
	if podName != "" {
		listOptions.FieldSelector = fmt.Sprintf("metadata.name=%s", podName)
	}

	pods, err := kubeClient.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		p.l.Error("获取 Pod 列表失败", zap.Error(err))
		return nil, err
	}

	// 使用帮助函数转换成 model.K8sPod 格式
	return pkg.BuildK8sPods(pods), nil
}

// GetContainersByPod 获取指定 Pod 中的容器列表
func (p *podService) GetContainersByPod(ctx context.Context, clusterID int, namespace string, podName string) ([]*model.K8sPodContainer, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		p.l.Error("获取 Pod 失败", zap.Error(err))
		return nil, err
	}

	// 转换 Pod 中容器信息
	containers := pkg.BuildK8sContainers(pod.Spec.Containers)
	return pkg.BuildK8sContainersWithPointer(containers), nil
}

// GetContainerLogs 获取指定容器的日志
func (p *podService) GetContainerLogs(ctx context.Context, clusterID int, namespace, podName, containerName string) (string, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return "", err
	}

	req := kubeClient.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
		Follow:    false, // 不跟随日志
		Previous:  false, // 不使用 previous
	})

	podLogs, err := req.Stream(ctx)
	if err != nil {
		p.l.Error("获取 Pod 日志失败", zap.Error(err))
		return "", err
	}
	defer podLogs.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		p.l.Error("读取 Pod 日志失败", zap.Error(err))
		return "", err
	}

	return buf.String(), nil
}

// GetPodYaml 获取指定 Pod 的 YAML 配置
func (p *podService) GetPodYaml(ctx context.Context, clusterID int, namespace, podName string) (*corev1.Pod, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		p.l.Error("获取 Pod 信息失败", zap.Error(err))
		return nil, err
	}

	return pod, nil
}

func (p *podService) GetPodsByNodeName(ctx context.Context, id int, name string) ([]*model.K8sPod, error) {
	kubeClient, err := p.client.GetKubeClient(id)
	if err != nil {
		p.l.Error("CreateCluster: 获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, constants.ErrorK8sClientNotReady
	}

	nodes, err := pkg.GetNodesByClusterID(ctx, kubeClient, name)
	if err != nil || len(nodes.Items) == 0 {
		return nil, constants.ErrorNodeNotFound
	}

	pods, err := pkg.GetPodsByNodeName(ctx, kubeClient, name)
	if err != nil {
		return nil, err
	}

	return pkg.BuildK8sPods(pods), nil
}

// CreatePod 创建 Pod
func (p *podService) CreatePod(ctx context.Context, podResource *model.K8sPodRequest) error {
	kubeClient, err := pkg.GetKubeClient(ctx, podResource.ClusterName, p.dao, p.client, p.l)
	if err != nil {
		p.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 创建 Pod
	_, err = kubeClient.CoreV1().Pods(podResource.Pod.Namespace).Create(ctx, podResource.Pod, metav1.CreateOptions{})
	if err != nil {
		p.l.Error("创建 Pod 失败", zap.Error(err))
		return err
	}

	p.l.Info("创建 Pod 成功", zap.String("podName", podResource.Pod.Name))
	return nil
}

// UpdatePod 更新 Pod
func (p *podService) UpdatePod(ctx context.Context, podResource *model.K8sPodRequest) error {
	kubeClient, err := pkg.GetKubeClient(ctx, podResource.ClusterName, p.dao, p.client, p.l)
	if err != nil {
		p.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 更新 Pod
	_, err = kubeClient.CoreV1().Pods(podResource.Pod.Namespace).Update(ctx, podResource.Pod, metav1.UpdateOptions{})
	if err != nil {
		p.l.Error("更新 Pod 失败", zap.Error(err))
		return err
	}

	p.l.Info("更新 Pod 成功", zap.String("podName", podResource.Pod.Name))
	return nil
}

// DeletePod 删除 Pod
func (p *podService) DeletePod(ctx context.Context, clusterName, namespace, podName string) error {
	kubeClient, err := pkg.GetKubeClient(ctx, clusterName, p.dao, p.client, p.l)
	if err != nil {
		p.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 删除 Pod
	err = kubeClient.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		p.l.Error("删除 Pod 失败", zap.Error(err))
		return err
	}

	p.l.Info("删除 Pod 成功", zap.String("podName", podName))
	return nil
}

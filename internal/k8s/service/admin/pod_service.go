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
	GetPodsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sPod, error)
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
	// DeletePod 删除 Pod
	DeletePod(ctx context.Context, clusterId int, namespace, podName string) error
}

type podService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

func NewPodService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) PodService {
	return &podService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetPodsByNamespace 获取指定命名空间中的 Pod 列表
func (p *podService) GetPodsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sPod, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	pods, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		p.logger.Error("获取 Pod 列表失败", zap.Error(err))
		return nil, err
	}

	return pkg.BuildK8sPods(pods), nil
}

// GetContainersByPod 获取指定 Pod 中的容器列表
func (p *podService) GetContainersByPod(ctx context.Context, clusterID int, namespace, podName string) ([]*model.K8sPodContainer, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("获取 Pod 失败", zap.Error(err))
		return nil, err
	}

	containers := pkg.BuildK8sContainers(pod.Spec.Containers)

	return pkg.BuildK8sContainersWithPointer(containers), nil
}

// GetContainerLogs 获取指定容器的日志
func (p *podService) GetContainerLogs(ctx context.Context, clusterID int, namespace, podName, containerName string) (string, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return "", err
	}

	req := kubeClient.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{Container: containerName})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		p.logger.Error("获取 Pod 日志失败", zap.Error(err))
		return "", err
	}

	defer podLogs.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		p.logger.Error("读取 Pod 日志失败", zap.Error(err))
		return "", err
	}

	return buf.String(), nil
}

// GetPodYaml 获取指定 Pod 的 YAML 配置
func (p *podService) GetPodYaml(ctx context.Context, clusterID int, namespace, podName string) (*corev1.Pod, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("获取 Pod 信息失败", zap.Error(err))
		return nil, err
	}

	return pod, nil
}

// GetPodsByNodeName 获取指定节点的 Pod 列表
func (p *podService) GetPodsByNodeName(ctx context.Context, id int, name string) ([]*model.K8sPod, error) {
	kubeClient, err := p.client.GetKubeClient(id)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, constants.ErrorK8sClientNotReady
	}

	nodes, err := pkg.GetNodesByName(ctx, kubeClient, name)
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
	kubeClient, err := p.client.GetKubeClient(podResource.ClusterId)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	_, err = kubeClient.CoreV1().Pods(podResource.Pod.Namespace).Create(ctx, podResource.Pod, metav1.CreateOptions{})
	if err != nil {
		p.logger.Error("创建 Pod 失败", zap.Error(err))
		return err
	}

	return nil
}

// DeletePod 删除 Pod
func (p *podService) DeletePod(ctx context.Context, clusterId int, namespace, podName string) error {
	kubeClient, err := p.client.GetKubeClient(clusterId)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	err = kubeClient.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		p.logger.Error("删除 Pod 失败", zap.Error(err))
		return err
	}

	return nil
}

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
	"io"
	"strings"

	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodService interface {
	GetPodsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sPod, error)
	GetContainersByPod(ctx context.Context, clusterID int, namespace string, podName string) ([]*model.K8sPodContainer, error)
	GetContainerLogs(ctx context.Context, clusterID int, namespace, podName, containerName string) (string, error)
	GetPodYaml(ctx context.Context, clusterID int, namespace, podName string) (*corev1.Pod, error)
	GetPodsByNodeName(ctx context.Context, id int, name string) ([]*model.K8sPod, error)
	DeletePod(ctx context.Context, clusterId int, namespace, podName string) error
}

type podService struct {
	dao    dao.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

func NewPodService(dao dao.ClusterDAO, client client.K8sClient, logger *zap.Logger) PodService {
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
		p.logger.Error("Failed to get Kubernetes client", zap.Error(err))
		return nil, err
	}

	podList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		p.logger.Error("Failed to list Pods", zap.String("Namespace", namespace), zap.Error(err))
		return nil, err
	}

	return pkg.BuildK8sPods(podList), nil
}

// GetContainersByPod 获取指定 Pod 中的容器列表
func (p *podService) GetContainersByPod(ctx context.Context, clusterID int, namespace, podName string) ([]*model.K8sPodContainer, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.logger.Error("Failed to get Kubernetes client", zap.Error(err))
		return nil, err
	}

	pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("Failed to get Pod", zap.String("PodName", podName), zap.Error(err))
		return nil, err
	}

	return pkg.BuildK8sContainersWithPointer(pkg.BuildK8sContainers(pod.Spec.Containers)), nil
}

// GetContainerLogs 获取指定容器的日志
func (p *podService) GetContainerLogs(ctx context.Context, clusterID int, namespace, podName, containerName string) (string, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.logger.Error("Failed to get Kubernetes client", zap.Error(err))
		return "", err
	}

	logStream, err := kubeClient.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{Container: containerName}).Stream(ctx)
	if err != nil {
		p.logger.Error("Failed to stream Pod logs", zap.String("PodName", podName), zap.String("ContainerName", containerName), zap.Error(err))
		return "", err
	}
	defer logStream.Close()

	var logs strings.Builder
	if _, err := io.Copy(&logs, logStream); err != nil {
		p.logger.Error("Failed to read Pod logs", zap.String("PodName", podName), zap.String("ContainerName", containerName), zap.Error(err))
		return "", err
	}

	return logs.String(), nil
}

// GetPodYaml 获取指定 Pod 的 YAML 配置
func (p *podService) GetPodYaml(ctx context.Context, clusterID int, namespace, podName string) (*corev1.Pod, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.logger.Error("Failed to get Kubernetes client", zap.Error(err))
		return nil, err
	}

	pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("Failed to get Pod YAML", zap.String("PodName", podName), zap.Error(err))
		return nil, err
	}

	return pod, nil
}

// GetPodsByNodeName 获取指定节点的 Pod 列表
func (p *podService) GetPodsByNodeName(ctx context.Context, id int, name string) ([]*model.K8sPod, error) {
	kubeClient, err := p.client.GetKubeClient(id)
	if err != nil {
		p.logger.Error("Failed to get Kubernetes client", zap.Error(err))
		return nil, constants.ErrorK8sClientNotReady
	}

	pods, err := pkg.GetPodsByNodeName(ctx, kubeClient, name)
	if err != nil {
		p.logger.Error("Failed to get Pods by Node", zap.String("NodeName", name), zap.Error(err))
		return nil, err
	}

	return pkg.BuildK8sPods(pods), nil
}

// DeletePod 删除 Pod
func (p *podService) DeletePod(ctx context.Context, clusterId int, namespace, podName string) error {
	kubeClient, err := p.client.GetKubeClient(clusterId)
	if err != nil {
		p.logger.Error("Failed to get Kubernetes client", zap.Error(err))
		return err
	}

	if err := kubeClient.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{}); err != nil {
		p.logger.Error("Failed to delete Pod", zap.String("PodName", podName), zap.Error(err))
		return err
	}

	return nil
}

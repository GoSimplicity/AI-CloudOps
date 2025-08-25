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
	"io"
	"strings"
	"time"

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
	// 获取Pod列表
	GetPodsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sPod, error)
	GetPodList(ctx context.Context, req *model.K8sGetResourceListRequest) ([]*model.K8sPodResponse, error)
	GetPodsByNodeName(ctx context.Context, clusterID int, nodeName string) ([]*model.K8sPod, error)
	
	// 获取Pod详情
	GetPod(ctx context.Context, req *model.K8sGetResourceRequest) (*model.K8sPodResponse, error)
	GetPodYaml(ctx context.Context, clusterID int, namespace, podName string) (*corev1.Pod, error)
	
	// 获取容器相关信息
	GetContainersByPod(ctx context.Context, clusterID int, namespace string, podName string) ([]*model.K8sPodContainer, error)
	GetContainerLogs(ctx context.Context, clusterID int, namespace, podName, containerName string) (string, error)
	GetPodLogs(ctx context.Context, req *model.PodLogRequest) (string, error)
	
	// Pod操作
	DeletePod(ctx context.Context, clusterId int, namespace, podName string) error
	DeletePodWithOptions(ctx context.Context, req *model.K8sDeleteResourceRequest) error
	
	// 批量操作
	BatchDeletePods(ctx context.Context, req *model.K8sBatchDeleteRequest) error
	
	// 高级功能
	ExecInPod(ctx context.Context, req *model.PodExecRequest) error
	PortForward(ctx context.Context, req *model.PodPortForwardRequest) error
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

// ==================== 新增的标准化Service方法 ====================

// GetPodList 获取Pod列表（使用新的请求结构体）
func (p *podService) GetPodList(ctx context.Context, req *model.K8sGetResourceListRequest) ([]*model.K8sPodResponse, error) {
	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	listOptions := req.ToMetaV1ListOptions()
	podList, err := kubeClient.CoreV1().Pods(req.Namespace).List(ctx, listOptions)
	if err != nil {
		p.logger.Error("获取Pod列表失败", 
			zap.String("Namespace", req.Namespace), 
			zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceList, "获取Pod列表失败")
	}

	pods := make([]*model.K8sPodResponse, 0, len(podList.Items))
	for _, pod := range podList.Items {
		podResponse := p.convertPodToResponse(&pod)
		pods = append(pods, podResponse)
	}

	return pods, nil
}

// GetPod 获取单个Pod详情
func (p *podService) GetPod(ctx context.Context, req *model.K8sGetResourceRequest) (*model.K8sPodResponse, error) {
	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	pod, err := kubeClient.CoreV1().Pods(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("获取Pod详情失败", 
			zap.String("Namespace", req.Namespace),
			zap.String("PodName", req.ResourceName),
			zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取Pod详情失败")
	}

	return p.convertPodToResponse(pod), nil
}

// GetPodLogs 获取Pod日志（使用新的请求结构体）
func (p *podService) GetPodLogs(ctx context.Context, req *model.PodLogRequest) (string, error) {
	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	logOptions := &corev1.PodLogOptions{
		Container:    req.Container,
		Follow:       req.Follow,
		Previous:     req.Previous,
		SinceSeconds: req.SinceSeconds,
		Timestamps:   req.Timestamps,
		TailLines:    req.TailLines,
		LimitBytes:   req.LimitBytes,
	}

	if req.SinceTime != "" {
		sinceTime, err := time.Parse(time.RFC3339, req.SinceTime)
		if err != nil {
			p.logger.Error("解析时间参数失败", zap.String("SinceTime", req.SinceTime), zap.Error(err))
			return "", pkg.NewBusinessError(constants.ErrInvalidParam, "时间参数格式错误")
		}
		metaTime := metav1.NewTime(sinceTime)
		logOptions.SinceTime = &metaTime
	}

	podLogRequest := kubeClient.CoreV1().Pods(req.Namespace).GetLogs(req.ResourceName, logOptions)
	podLogs, err := podLogRequest.Stream(ctx)
	if err != nil {
		p.logger.Error("获取Pod日志失败", 
			zap.String("Namespace", req.Namespace),
			zap.String("PodName", req.ResourceName),
			zap.String("Container", req.Container),
			zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sResourceOperation, "获取Pod日志失败")
	}
	defer podLogs.Close()

	logData, err := io.ReadAll(podLogs)
	if err != nil {
		p.logger.Error("读取日志数据失败", zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sResourceOperation, "读取日志数据失败")
	}

	return string(logData), nil
}

// DeletePodWithOptions 删除Pod（使用新的请求结构体）
func (p *podService) DeletePodWithOptions(ctx context.Context, req *model.K8sDeleteResourceRequest) error {
	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	deleteOptions := metav1.DeleteOptions{}
	if req.GracePeriodSeconds != nil {
		deleteOptions.GracePeriodSeconds = req.GracePeriodSeconds
	}

	if req.Force {
		// 强制删除需要设置GracePeriodSeconds为0
		zero := int64(0)
		deleteOptions.GracePeriodSeconds = &zero
	}

	err = kubeClient.CoreV1().Pods(req.Namespace).Delete(ctx, req.ResourceName, deleteOptions)
	if err != nil {
		p.logger.Error("删除Pod失败", 
			zap.String("Namespace", req.Namespace),
			zap.String("PodName", req.ResourceName),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, "删除Pod失败")
	}

	p.logger.Info("成功删除Pod", 
		zap.String("Namespace", req.Namespace),
		zap.String("PodName", req.ResourceName))
	return nil
}

// BatchDeletePods 批量删除Pod
func (p *podService) BatchDeletePods(ctx context.Context, req *model.K8sBatchDeleteRequest) error {
	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	var errors []string
	for _, podName := range req.ResourceNames {
		err := kubeClient.CoreV1().Pods(req.Namespace).Delete(ctx, podName, metav1.DeleteOptions{})
		if err != nil {
			errorMsg := fmt.Sprintf("删除Pod %s 失败: %v", podName, err)
			errors = append(errors, errorMsg)
			p.logger.Error("批量删除Pod中的单个Pod失败", 
				zap.String("PodName", podName),
				zap.Error(err))
		}
	}

	if len(errors) > 0 {
		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, 
			fmt.Sprintf("批量删除失败，详情: %s", strings.Join(errors, "; ")))
	}

	p.logger.Info("成功批量删除Pod", 
		zap.String("Namespace", req.Namespace),
		zap.Int("Count", len(req.ResourceNames)))
	return nil
}

// convertPodToResponse 将Kubernetes Pod对象转换为响应模型
func (p *podService) convertPodToResponse(pod *corev1.Pod) *model.K8sPodResponse {
	// 计算重启次数
	var totalRestartCount int32
	containers := make([]model.ContainerInfo, 0, len(pod.Spec.Containers))
	
	for _, container := range pod.Spec.Containers {
		containerInfo := model.ContainerInfo{
			Name:  container.Name,
			Image: container.Image,
			Resources: model.ContainerResources{
				CpuRequest:    container.Resources.Requests.Cpu().String(),
				CpuLimit:      container.Resources.Limits.Cpu().String(),
				MemoryRequest: container.Resources.Requests.Memory().String(),
				MemoryLimit:   container.Resources.Limits.Memory().String(),
			},
			Ports:        container.Ports,
			Env:          container.Env,
			VolumeMounts: container.VolumeMounts,
		}
		
		// 从容器状态获取重启次数和状态
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name == container.Name {
				containerInfo.RestartCount = containerStatus.RestartCount
				containerInfo.Ready = containerStatus.Ready
				totalRestartCount += containerStatus.RestartCount
				
				if containerStatus.State.Running != nil {
					containerInfo.Status = "Running"
				} else if containerStatus.State.Waiting != nil {
					containerInfo.Status = "Waiting"
				} else if containerStatus.State.Terminated != nil {
					containerInfo.Status = "Terminated"
				}
				break
			}
		}
		
		containers = append(containers, containerInfo)
	}

	return &model.K8sPodResponse{
		Name:              pod.Name,
		UID:               string(pod.UID),
		Namespace:         pod.Namespace,
		Status:            string(pod.Status.Phase),
		Phase:             string(pod.Status.Phase),
		NodeName:          pod.Spec.NodeName,
		PodIP:             pod.Status.PodIP,
		HostIP:            pod.Status.HostIP,
		RestartCount:      totalRestartCount,
		Age:               pkg.GetAge(pod.CreationTimestamp.Time),
		Labels:            pod.Labels,
		Annotations:       pod.Annotations,
		OwnerReferences:   pod.OwnerReferences,
		CreationTimestamp: pod.CreationTimestamp.Time,
		Containers:        containers,
	}
}

// ExecInPod Pod命令执行（占位实现）
func (p *podService) ExecInPod(ctx context.Context, req *model.PodExecRequest) error {
	// TODO: 实现Pod命令执行功能
	return pkg.NewBusinessError(constants.ErrNotImplemented, "Pod命令执行功能尚未实现")
}

// PortForward Pod端口转发（占位实现）
func (p *podService) PortForward(ctx context.Context, req *model.PodPortForwardRequest) error {
	// TODO: 实现Pod端口转发功能
	return pkg.NewBusinessError(constants.ErrNotImplemented, "Pod端口转发功能尚未实现")
}

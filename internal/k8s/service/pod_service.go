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

// import (
// 	"context"

// 	"io"

// 	"time"

// 	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
// 	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

// 	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
// 	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
// 	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
// 	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
// 	"github.com/GoSimplicity/AI-CloudOps/internal/model"
// 	"go.uber.org/zap"
// 	corev1 "k8s.io/api/core/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// )

// type PodService interface {
// 	GetPodsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sPod, error)
// 	GetPodList(ctx context.Context, req *model.K8sGetResourceListReq) ([]*model.K8sPodResponse, error)
// 	GetPodsByNodeName(ctx context.Context, clusterID int, nodeName string) ([]*model.K8sPod, error)
// 	GetPod(ctx context.Context, req *model.K8sGetResourceReq) (*model.K8sPodResponse, error)
// 	GetPodYaml(ctx context.Context, clusterID int, namespace, podName string) (*corev1.Pod, error)
// 	GetContainersByPod(ctx context.Context, clusterID int, namespace string, podName string) ([]*model.K8sPodContainer, error)
// 	GetContainerLogs(ctx context.Context, clusterID int, namespace, podName, containerName string) (string, error)
// 	GetPodLogs(ctx context.Context, req *model.PodLogReq) (string, error)
// 	DeletePod(ctx context.Context, clusterId int, namespace, podName string) error
// 	DeletePodWithOptions(ctx context.Context, req *model.K8sDeleteResourceReq) error
// 	ExecInPod(ctx context.Context, req *model.PodExecReq) error
// 	PortForward(ctx context.Context, req *model.PodPortForwardReq) error
// }

// type podService struct {
// 	dao        dao.ClusterDAO
// 	client     client.K8sClient   // 保持向后兼容
// 	podManager manager.PodManager // 新的依赖注入
// 	logger     *zap.Logger
// }

// func NewPodService(dao dao.ClusterDAO, client client.K8sClient, podManager manager.PodManager, logger *zap.Logger) PodService {
// 	return &podService{
// 		dao:        dao,
// 		client:     client,     // 保持向后兼容，某些方法可能仍需要
// 		podManager: podManager, // 使用新的 manager
// 		logger:     logger,
// 	}
// }

// // GetPodsByNamespace 获取命名空间中的pod列表
// func (p *podService) GetPodsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sPod, error) {
// 	// 使用新的 PodManager
// 	podList, err := p.podManager.GetPodList(ctx, clusterID, namespace, metav1.ListOptions{})
// 	if err != nil {
// 		p.logger.Error("获取 Pod 列表失败",
// 			zap.Int("clusterID", clusterID),
// 			zap.String("namespace", namespace),
// 			zap.Error(err))
// 		return nil, err
// 	}

// 	return k8sutils.BuildK8sPods(podList.Items), nil
// }

// // GetContainersByPod 获取指定 Pod 中的容器列表
// func (p *podService) GetContainersByPod(ctx context.Context, clusterID int, namespace, podName string) ([]*model.K8sPodContainer, error) {
// 	// 使用新的 PodManager
// 	pod, err := p.podManager.GetPod(ctx, clusterID, namespace, podName)
// 	if err != nil {
// 		p.logger.Error("获取 Pod 失败",
// 			zap.Int("clusterID", clusterID),
// 			zap.String("namespace", namespace),
// 			zap.String("podName", podName),
// 			zap.Error(err))
// 		return nil, err
// 	}

// 	return k8sutils.BuildK8sContainersWithPointer(k8sutils.BuildK8sContainers(pod.Spec.Containers)), nil
// }

// // GetContainerLogs 获取指定容器的日志
// func (p *podService) GetContainerLogs(ctx context.Context, clusterID int, namespace, podName, containerName string) (string, error) {
// 	// 使用新的 PodManager
// 	logOptions := &corev1.PodLogOptions{Container: containerName}
// 	logs, err := p.podManager.GetPodLogs(ctx, clusterID, namespace, podName, logOptions)
// 	if err != nil {
// 		p.logger.Error("获取容器日志失败",
// 			zap.Int("clusterID", clusterID),
// 			zap.String("namespace", namespace),
// 			zap.String("podName", podName),
// 			zap.String("containerName", containerName),
// 			zap.Error(err))
// 		return "", err
// 	}

// 	return logs, nil
// }

// // GetPodYaml 获取pod的YAML
// func (p *podService) GetPodYaml(ctx context.Context, clusterID int, namespace, podName string) (*corev1.Pod, error) {
// 	// 使用新的 PodManager
// 	pod, err := p.podManager.GetPod(ctx, clusterID, namespace, podName)
// 	if err != nil {
// 		p.logger.Error("获取 Pod YAML 失败",
// 			zap.Int("clusterID", clusterID),
// 			zap.String("namespace", namespace),
// 			zap.String("podName", podName),
// 			zap.Error(err))
// 		return nil, err
// 	}

// 	return pod, nil
// }

// // GetPodsByNodeName 获取节点的pod列表
// func (p *podService) GetPodsByNodeName(ctx context.Context, clusterID int, nodeName string) ([]*model.K8sPod, error) {
// 	// 使用新的 PodManager
// 	pods, err := p.podManager.GetPodsByNodeName(ctx, clusterID, nodeName)
// 	if err != nil {
// 		p.logger.Error("获取节点 Pod 列表失败",
// 			zap.Int("clusterID", clusterID),
// 			zap.String("nodeName", nodeName),
// 			zap.Error(err))
// 		return nil, err
// 	}

// 	return k8sutils.BuildK8sPods(pods.Items), nil
// }

// // DeletePod 删除pod
// func (p *podService) DeletePod(ctx context.Context, clusterID int, namespace, podName string) error {
// 	// 使用新的 PodManager
// 	err := p.podManager.DeletePod(ctx, clusterID, namespace, podName, metav1.DeleteOptions{})
// 	if err != nil {
// 		p.logger.Error("删除 Pod 失败",
// 			zap.Int("clusterID", clusterID),
// 			zap.String("namespace", namespace),
// 			zap.String("podName", podName),
// 			zap.Error(err))
// 		return err
// 	}

// 	return nil
// }

// // ==================== 新增的标准化Service方法 ====================

// // GetPodList 获取pod列表
// func (p *podService) GetPodList(ctx context.Context, req *model.K8sGetResourceListReq) ([]*model.K8sPodResponse, error) {
// 	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
// 	if err != nil {
// 		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
// 	}

// 	listOptions := k8sutils.ConvertToMetaV1ListOptions(req)
// 	podList, err := kubeClient.CoreV1().Pods(req.Namespace).List(ctx, listOptions)
// 	if err != nil {
// 		p.logger.Error("获取Pod列表失败",
// 			zap.String("Namespace", req.Namespace),
// 			zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sResourceList, "获取Pod列表失败")
// 	}

// 	pods := make([]*model.K8sPodResponse, 0, len(podList.Items))
// 	for _, pod := range podList.Items {
// 		podResponse := p.convertPodToResponse(&pod)
// 		pods = append(pods, podResponse)
// 	}

// 	return pods, nil
// }

// // GetPod 获取pod详情
// func (p *podService) GetPod(ctx context.Context, req *model.K8sGetResourceReq) (*model.K8sPodResponse, error) {
// 	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
// 	if err != nil {
// 		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
// 	}

// 	pod, err := kubeClient.CoreV1().Pods(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
// 	if err != nil {
// 		p.logger.Error("获取Pod详情失败",
// 			zap.String("Namespace", req.Namespace),
// 			zap.String("PodName", req.ResourceName),
// 			zap.Error(err))
// 		return nil, pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取Pod详情失败")
// 	}

// 	return p.convertPodToResponse(pod), nil
// }

// // GetPodLogs 获取pod日志
// func (p *podService) GetPodLogs(ctx context.Context, req *model.PodLogReq) (string, error) {
// 	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
// 	if err != nil {
// 		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
// 		return "", pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
// 	}

// 	logOptions := &corev1.PodLogOptions{
// 		Container:    req.Container,
// 		Follow:       req.Follow,
// 		Previous:     req.Previous,
// 		SinceSeconds: req.SinceSeconds,
// 		Timestamps:   req.Timestamps,
// 		TailLines:    req.TailLines,
// 		LimitBytes:   req.LimitBytes,
// 	}

// 	if req.SinceTime != "" {
// 		sinceTime, err := time.Parse(time.RFC3339, req.SinceTime)
// 		if err != nil {
// 			p.logger.Error("解析时间参数失败", zap.String("SinceTime", req.SinceTime), zap.Error(err))
// 			return "", pkg.NewBusinessError(constants.ErrInvalidParam, "时间参数格式错误")
// 		}
// 		metaTime := metav1.NewTime(sinceTime)
// 		logOptions.SinceTime = &metaTime
// 	}

// 	podLogRequest := kubeClient.CoreV1().Pods(req.Namespace).GetLogs(req.ResourceName, logOptions)
// 	podLogs, err := podLogRequest.Stream(ctx)
// 	if err != nil {
// 		p.logger.Error("获取Pod日志失败",
// 			zap.String("Namespace", req.Namespace),
// 			zap.String("PodName", req.ResourceName),
// 			zap.String("Container", req.Container),
// 			zap.Error(err))
// 		return "", pkg.NewBusinessError(constants.ErrK8sResourceOperation, "获取Pod日志失败")
// 	}
// 	defer podLogs.Close()

// 	logData, err := io.ReadAll(podLogs)
// 	if err != nil {
// 		p.logger.Error("读取日志数据失败", zap.Error(err))
// 		return "", pkg.NewBusinessError(constants.ErrK8sResourceOperation, "读取日志数据失败")
// 	}

// 	return string(logData), nil
// }

// // DeletePodWithOptions 删除pod
// func (p *podService) DeletePodWithOptions(ctx context.Context, req *model.K8sDeleteResourceReq) error {
// 	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
// 	if err != nil {
// 		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
// 		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
// 	}

// 	deleteOptions := metav1.DeleteOptions{}
// 	if req.GracePeriodSeconds != nil {
// 		deleteOptions.GracePeriodSeconds = req.GracePeriodSeconds
// 	}

// 	if req.Force {
// 		// 强制删除需要设置GracePeriodSeconds为0
// 		zero := int64(0)
// 		deleteOptions.GracePeriodSeconds = &zero
// 	}

// 	err = kubeClient.CoreV1().Pods(req.Namespace).Delete(ctx, req.ResourceName, deleteOptions)
// 	if err != nil {
// 		p.logger.Error("删除Pod失败",
// 			zap.String("Namespace", req.Namespace),
// 			zap.String("PodName", req.ResourceName),
// 			zap.Error(err))
// 		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, "删除Pod失败")
// 	}

// 	p.logger.Info("成功删除Pod",
// 		zap.String("Namespace", req.Namespace),
// 		zap.String("PodName", req.ResourceName))
// 	return nil
// }

// // convertPodToResponse 将Kubernetes Pod对象转换为响应模型
// func (p *podService) convertPodToResponse(pod *corev1.Pod) *model.K8sPodResponse {
// 	// 计算重启次数
// 	var totalRestartCount int32
// 	containers := make([]model.ContainerInfo, 0, len(pod.Spec.Containers))

// 	for _, container := range pod.Spec.Containers {
// 		containerInfo := model.ContainerInfo{
// 			Name:  container.Name,
// 			Image: container.Image,
// 			Resources: model.ContainerResources{
// 				CpuRequest:    container.Resources.Requests.Cpu().String(),
// 				CpuLimit:      container.Resources.Limits.Cpu().String(),
// 				MemoryRequest: container.Resources.Requests.Memory().String(),
// 				MemoryLimit:   container.Resources.Limits.Memory().String(),
// 			},
// 			Ports:        container.Ports,
// 			Env:          container.Env,
// 			VolumeMounts: container.VolumeMounts,
// 		}

// 		// 从容器状态获取重启次数和状态
// 		for _, containerStatus := range pod.Status.ContainerStatuses {
// 			if containerStatus.Name == container.Name {
// 				containerInfo.RestartCount = containerStatus.RestartCount
// 				containerInfo.Ready = containerStatus.Ready
// 				totalRestartCount += containerStatus.RestartCount

// 				if containerStatus.State.Running != nil {
// 					containerInfo.Status = "Running"
// 				} else if containerStatus.State.Waiting != nil {
// 					containerInfo.Status = "Waiting"
// 				} else if containerStatus.State.Terminated != nil {
// 					containerInfo.Status = "Terminated"
// 				}
// 				break
// 			}
// 		}

// 		containers = append(containers, containerInfo)
// 	}

// 	return &model.K8sPodResponse{
// 		Name:              pod.Name,
// 		UID:               string(pod.UID),
// 		Namespace:         pod.Namespace,
// 		Status:            string(pod.Status.Phase),
// 		Phase:             string(pod.Status.Phase),
// 		NodeName:          pod.Spec.NodeName,
// 		PodIP:             pod.Status.PodIP,
// 		HostIP:            pod.Status.HostIP,
// 		RestartCount:      totalRestartCount,
// 		Age:               pkg.GetAge(pod.CreationTimestamp.Time),
// 		Labels:            pod.Labels,
// 		Annotations:       pod.Annotations,
// 		OwnerReferences:   pod.OwnerReferences,
// 		CreationTimestamp: pod.CreationTimestamp.Time,
// 		Containers:        containers,
// 	}
// }

// // ExecInPod pod命令执行
// func (p *podService) ExecInPod(ctx context.Context, req *model.PodExecReq) error {
// 	// TODO: 实现Pod命令执行功能
// 	return pkg.NewBusinessError(constants.ErrNotImplemented, "Pod命令执行功能尚未实现")
// }

// // PortForward pod端口转发
// func (p *podService) PortForward(ctx context.Context, req *model.PodPortForwardReq) error {
// 	// TODO: 实现Pod端口转发功能
// 	return pkg.NewBusinessError(constants.ErrNotImplemented, "Pod端口转发功能尚未实现")
// }

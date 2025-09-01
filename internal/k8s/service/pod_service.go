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
	"archive/tar"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"

	"io"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/kubectl/pkg/scheme"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkgutils "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/terminal"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodService interface {
	// 获取Pod列表
	GetPodsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sPod, error)
	GetPodList(ctx context.Context, req *model.K8sGetResourceListReq) ([]*model.K8sPodResponse, error)
	GetPodsByNodeName(ctx context.Context, clusterID int, nodeName string) ([]*model.K8sPod, error)

	// 获取Pod详情
	GetPod(ctx context.Context, req *model.K8sGetResourceReq) (*model.K8sPodResponse, error)
	GetPodYaml(ctx context.Context, clusterID int, namespace, podName string) (*corev1.Pod, error)

	// 获取容器相关信息
	GetContainersByPod(ctx context.Context, clusterID int, namespace string, podName string) ([]*model.K8sPodContainer, error)
	GetContainerLogs(ctx context.Context, clusterID int, namespace, podName, containerName string) (string, error)
	GetPodLogs(ctx context.Context, req *model.PodLogReq) (string, error)

	// Pod操作
	DeletePod(ctx context.Context, clusterId int, namespace, podName string) error
	DeletePodWithOptions(ctx context.Context, req *model.K8sDeleteResourceReq) error

	// 批量操作
	BatchDeletePods(ctx context.Context, req *model.K8sBatchDeleteReq) error

	// 高级功能
	ExecInPod(ctx *gin.Context, req *model.PodExecReq) error
	PortForward(ctx context.Context, req *model.PodPortForwardReq) error
	DownloadPodFile(ctx *gin.Context, req *model.PodFileReq) error
	UploadFileToFile(ctx *gin.Context, req *model.PodFileReq) error
}

type fileWithHeader struct {
	file   multipart.File
	header *multipart.FileHeader
}

type podService struct {
	dao        dao.ClusterDAO
	client     client.K8sClient   // 保持向后兼容
	podManager manager.PodManager // 新的依赖注入

	//term   terminal.Interface
	logger *zap.Logger
}

func NewPodService(dao dao.ClusterDAO, client client.K8sClient, podManager manager.PodManager, logger *zap.Logger) PodService {
	return &podService{
		dao:        dao,
		client:     client,     // 保持向后兼容，某些方法可能仍需要
		podManager: podManager, // 使用新的 manager
		logger:     logger,
		//term:       term,
	}
}

// GetPodsByNamespace 获取指定命名空间中的 Pod 列表
func (p *podService) GetPodsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sPod, error) {
	// 使用新的 PodManager
	podList, err := p.podManager.GetPodList(ctx, clusterID, namespace, metav1.ListOptions{})
	if err != nil {
		p.logger.Error("获取 Pod 列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, err
	}

	return k8sutils.BuildK8sPods(podList), nil
}

// GetContainersByPod 获取指定 Pod 中的容器列表
func (p *podService) GetContainersByPod(ctx context.Context, clusterID int, namespace, podName string) ([]*model.K8sPodContainer, error) {
	// 使用新的 PodManager
	pod, err := p.podManager.GetPod(ctx, clusterID, namespace, podName)
	if err != nil {
		p.logger.Error("获取 Pod 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("podName", podName),
			zap.Error(err))
		return nil, err
	}

	return k8sutils.BuildK8sContainersWithPointer(k8sutils.BuildK8sContainers(pod.Spec.Containers)), nil
}

// GetContainerLogs 获取指定容器的日志
func (p *podService) GetContainerLogs(ctx context.Context, clusterID int, namespace, podName, containerName string) (string, error) {
	// 使用新的 PodManager
	logOptions := &corev1.PodLogOptions{Container: containerName}
	logs, err := p.podManager.GetPodLogs(ctx, clusterID, namespace, podName, logOptions)
	if err != nil {
		p.logger.Error("获取容器日志失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("podName", podName),
			zap.String("containerName", containerName),
			zap.Error(err))
		return "", err
	}

	return logs, nil
}

// GetPodYaml 获取指定 Pod 的 YAML 配置
func (p *podService) GetPodYaml(ctx context.Context, clusterID int, namespace, podName string) (*corev1.Pod, error) {
	// 使用新的 PodManager
	pod, err := p.podManager.GetPod(ctx, clusterID, namespace, podName)
	if err != nil {
		p.logger.Error("获取 Pod YAML 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("podName", podName),
			zap.Error(err))
		return nil, err
	}

	return pod, nil
}

// GetPodsByNodeName 获取指定节点的 Pod 列表
func (p *podService) GetPodsByNodeName(ctx context.Context, clusterID int, nodeName string) ([]*model.K8sPod, error) {
	// 使用新的 PodManager
	pods, err := p.podManager.GetPodsByNodeName(ctx, clusterID, nodeName)
	if err != nil {
		p.logger.Error("获取节点 Pod 列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("nodeName", nodeName),
			zap.Error(err))
		return nil, err
	}

	return k8sutils.BuildK8sPods(pods), nil
}

// DeletePod 删除 Pod
func (p *podService) DeletePod(ctx context.Context, clusterID int, namespace, podName string) error {
	// 使用新的 PodManager
	err := p.podManager.DeletePod(ctx, clusterID, namespace, podName, metav1.DeleteOptions{})
	if err != nil {
		p.logger.Error("删除 Pod 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("podName", podName),
			zap.Error(err))
		return err
	}

	return nil
}

// ==================== 新增的标准化Service方法 ====================

// GetPodList 获取Pod列表（使用新的请求结构体）
func (p *podService) GetPodList(ctx context.Context, req *model.K8sGetResourceListReq) ([]*model.K8sPodResponse, error) {
	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	listOptions := k8sutils.ConvertToMetaV1ListOptions(req)
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
func (p *podService) GetPod(ctx context.Context, req *model.K8sGetResourceReq) (*model.K8sPodResponse, error) {
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
func (p *podService) GetPodLogs(ctx context.Context, req *model.PodLogReq) (string, error) {
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
func (p *podService) DeletePodWithOptions(ctx context.Context, req *model.K8sDeleteResourceReq) error {
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
func (p *podService) BatchDeletePods(ctx context.Context, req *model.K8sBatchDeleteReq) error {
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

// ExecInPod Pod命令执行
func (p *podService) ExecInPod(ctx *gin.Context, req *model.PodExecReq) error {

	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	restConfig, err := p.getRestConfig(ctx, req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubeconfig配置失败",
			zap.Error(err))
		return err
	}

	conn, err := pkgutils.UpGrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		p.logger.Error("升级ws失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrorWsUpgradeFailed, "初始化ws失败")
	}

	if len(req.Shell) > 0 {
		req.Shell = "sh"
	}

	terminal.NewTerminalerHandler(kubeClient, restConfig).
		HandleSession(ctx.Request.Context(), req.Shell, req.Namespace, req.PodName, req.Container, conn)

	return nil
}

func (p *podService) UploadFileToFile(ctx *gin.Context, req *model.PodFileReq) error {

	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败",
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	restConfig, err := p.getRestConfig(ctx, req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubeconfig配置失败",
			zap.Error(err))
		return err
	}

	targetDir := req.FilePath
	if targetDir == "" {
		targetDir = "/"
	}
	// 解析上传的文件
	files, err := parseMultipartFiles(ctx)
	if err != nil {
		return err
	}

	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		if tarErr := writeFilesToTar(files, writer); tarErr != nil {
			p.logger.Error("打包文件成 tar 失败", zap.Error(tarErr))
			_ = writer.CloseWithError(tarErr)
		}
	}()

	execReq := kubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(req.PodName).
		Namespace(req.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: req.ContainerName,
			Command:   []string{"tar", "-xmf", "-", "-C", targetDir},
			Stdin:     true,
			Stdout:    false,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(restConfig, http.MethodPost, execReq.URL())
	if err != nil {
		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, "创建 Executor 失败")
	}

	if err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             reader,
		Stdout:            nil,
		Stderr:            ctx.Writer,
		TerminalSizeQueue: nil,
	}); err != nil {
		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, "执行上传失败")
	}
	return nil
}

// PortForward Pod端口转发
func (p *podService) PortForward(ctx context.Context, req *model.PodPortForwardReq) error {

	// 获取 restconfig
	restConfig, err := p.getRestConfig(ctx, req.ClusterID)
	if err != nil {
		return err
	}

	// 构造 SPDY dialer
	dialer, err := p.buildDialer(restConfig, req.Namespace, req.ResourceName)
	if err != nil {
		return err
	}

	// 构造端口映射
	portsSpec := buildPortsSpec(req.Ports)

	// 创建 PortForwarder
	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})
	forwarder, err := portforward.New(dialer, portsSpec, stopChan, readyChan, io.Discard, io.Discard)
	if err != nil {
		return pkg.NewBusinessError(constants.ErrK8sPortForward, err.Error())
	}

	// 自动关闭转发
	go func() {
		<-ctx.Done()
		close(stopChan)
	}()

	// 异步开启转发
	go func() {
		if err := forwarder.ForwardPorts(); err != nil {
			p.logger.Error("创建端口转发失败",
				zap.Error(err),
				zap.Strings("ports", portsSpec),
			)
		}

	}()
	// 等待就绪
	<-readyChan
	return nil
}

// DownloadPodFile 实现Pod内文件下载
func (p *podService) DownloadPodFile(ctx *gin.Context, req *model.PodFileReq) error {

	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败",
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	restConfig, err := p.getRestConfig(ctx, req.ClusterID)
	if err != nil {
		return err
	}

	fileName := filepath.Base(req.FilePath)
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename=%s.tar`, fileName))
	ctx.Header("Content-Type", "application/octet-stream")

	reader, err := k8sutils.NewPodFileStreamPipe(
		ctx, restConfig, kubeClient, req.Namespace, req.PodName, req.ContainerName, req.FilePath)

	if err != nil {
		p.logger.Error("创建Pod文件流失败",
			zap.Error(err),
			zap.String("Namespace", req.Namespace),
			zap.String("PodName", req.PodName),
			zap.String("ContainerName", req.ContainerName),
			zap.String("FilePath", req.FilePath),
		)
		return pkg.NewBusinessError(constants.ErrPodFileStream, "无法创建 Pod 文件流")
	}
	defer reader.Close()
	// 把流复制到响应
	if _, err := io.Copy(ctx.Writer, reader); err != nil {
		p.logger.Error("文件下载过程中出错", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrPodFileStream, "下载文件过程中发生错误")
	}
	return nil
}

func (p *podService) getRestConfig(ctx context.Context, clusterID int) (*rest.Config, error) {
	cluster, err := p.dao.GetClusterByID(ctx, clusterID)
	if err != nil {
		p.logger.Error("获取集群信息失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClusterInfo, "无法获取集群信息")
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
	if err != nil {
		p.logger.Error("解析 kubeconfig 失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sParseKubeConfig, "无法解析 kubeconfig")
	}
	restConfig.QPS = 100
	restConfig.Burst = 200
	return restConfig, nil
}

func (p *podService) buildDialer(restConfig *rest.Config, namespace, podName string) (httpstream.Dialer, error) {
	roundTripper, upgrader, err := spdy.RoundTripperFor(restConfig)
	if err != nil {
		return nil, pkg.NewBusinessError(constants.ErrK8sPortForward, err.Error())
	}
	// 基础 API URL
	baseURL, err := url.Parse(restConfig.Host)
	if err != nil {
		return nil, fmt.Errorf("非法 k8s API host: %w", err)
	}

	podPath := path.Join("/api/v1/namespaces", namespace, "pods", podName, "portforward")
	finalURL := baseURL.ResolveReference(&url.URL{Path: podPath})
	return spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, finalURL), nil
}

func buildPortsSpec(ports []model.PortForwardPort) []string {
	specs := make([]string, len(ports))
	for i, port := range ports {
		specs[i] = fmt.Sprintf("%d:%d", port.LocalPort, port.RemotePort)
	}
	return specs
}

func parseMultipartFiles(ctx *gin.Context) ([]fileWithHeader, error) {
	if err := ctx.Request.ParseMultipartForm(10 << 20); err != nil {
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceDelete, "读取上传文件失败")
	}
	files := make([]fileWithHeader, 0)
	for name := range ctx.Request.MultipartForm.File {
		file, header, err := ctx.Request.FormFile(name)
		if err != nil {
			return nil, pkg.NewBusinessError(constants.ErrK8sResourceDelete, "解析上传文件失败")
		}
		files = append(files, fileWithHeader{file: file, header: header})
	}
	return files, nil
}
func writeFilesToTar(files []fileWithHeader, w io.Writer) error {
	tarWriter := tar.NewWriter(w)
	defer tarWriter.Close()
	for _, f := range files {
		func(f fileWithHeader) {
			defer f.file.Close()
			hdr := &tar.Header{
				Name: f.header.Filename,
				Mode: 0600,
				Size: f.header.Size,
			}
			if err := tarWriter.WriteHeader(hdr); err != nil {
				_ = w.(io.WriteCloser).Close()
				return
			}
			if _, err := io.Copy(tarWriter, f.file); err != nil {
				_ = w.(io.WriteCloser).Close()
				return
			}
		}(f)
	}
	return nil
}

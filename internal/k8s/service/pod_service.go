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
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils/query"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	utilsStream "github.com/GoSimplicity/AI-CloudOps/pkg/utils/stream"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport/spdy"
)

type PodService interface {
	// 核心 CRUD 操作
	CreatePod(ctx context.Context, req *model.CreatePodReq) error
	GetPodList(ctx context.Context, req *model.GetPodListReq) (*model.ListResp[*model.K8sPod], error)
	GetPodDetails(ctx context.Context, req *model.GetPodDetailsReq) (*model.K8sPod, error)
	GetPodYaml(ctx context.Context, req *model.GetPodYamlReq) (*model.K8sYaml, error)
	UpdatePod(ctx context.Context, req *model.UpdatePodReq) error
	DeletePod(ctx context.Context, req *model.DeletePodReq) error
	BatchDeletePods(ctx context.Context, req *model.BatchDeletePodsReq) error

	// YAML 操作
	CreatePodByYaml(ctx context.Context, req *model.CreatePodByYamlReq) error
	UpdatePodByYaml(ctx context.Context, req *model.UpdatePodByYamlReq) error

	// 扩展功能
	GetPodsByNodeName(ctx context.Context, req *model.GetPodsByNodeReq) ([]*model.K8sPod, error)
	GetPodContainers(ctx context.Context, req *model.GetPodContainersReq) ([]*model.PodContainer, error)
	GetPodLogs(ctx *gin.Context, req *model.GetPodLogsReq) error
	PodExec(ctx *gin.Context, req *model.PodExecReq) error
	PodPortForward(ctx context.Context, req *model.PodPortForwardReq) error
	PodFileDownload(ctx *gin.Context, req *model.PodFileDownloadReq) error
	PodFileUpload(ctx *gin.Context, req *model.PodFileUploadReq) error
}

type podService struct {
	podManager manager.PodManager
	dao        dao.ClusterDAO
	logger     *zap.Logger
}

func NewPodService(podManager manager.PodManager, dao dao.ClusterDAO, logger *zap.Logger) PodService {
	return &podService{
		podManager: podManager,
		dao:        dao,
		logger:     logger,
	}
}

// CreatePod 创建Pod
func (p *podService) CreatePod(ctx context.Context, req *model.CreatePodReq) error {
	if req == nil {
		return fmt.Errorf("创建Pod请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Pod名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	// 从请求构建Pod对象
	pod, err := utils.BuildPodFromRequest(req)
	if err != nil {
		p.logger.Error("CreatePod: 构建Pod对象失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("构建Pod对象失败: %w", err)
	}

	// 验证Pod配置
	if err := utils.ValidatePod(pod); err != nil {
		p.logger.Error("CreatePod: Pod配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("pod配置验证失败: %w", err)
	}

	// 创建Pod
	_, err = p.podManager.CreatePod(ctx, req.ClusterID, pod)
	if err != nil {
		p.logger.Error("CreatePod: 创建Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("创建Pod失败: %w", err)
	}

	return nil
}

// GetPodList 获取Pod列表
func (p *podService) GetPodList(ctx context.Context, req *model.GetPodListReq) (*model.ListResp[*model.K8sPod], error) {
	if req == nil {
		return nil, fmt.Errorf("获取Pod列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	// 构建查询参数
	queryParams := &query.Query{
		Filters:    make(map[query.Field]query.Value),
		Pagination: &query.Pagination{Limit: req.Size, Offset: (req.Page - 1) * req.Size},
		SortBy:     query.FieldCreationTimeStamp,
		Ascending:  false,
	}

	// 添加标签选择器 (注意：GetPodListReq可能没有Labels字段，这里暂时跳过)
	// queryParams.AppendLabelSelector(req.Labels)

	// 设置命名空间
	namespace := req.Namespace
	if namespace == "" {
		namespace = corev1.NamespaceAll
	}

	pods, err := p.podManager.GetPodList(ctx, req.ClusterID, namespace, queryParams)
	if err != nil {
		p.logger.Error("GetPodList: 获取Pod列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取Pod列表失败: %w", err)
	}

	return pods, nil
}

// GetPodDetails 获取Pod详情
func (p *podService) GetPodDetails(ctx context.Context, req *model.GetPodDetailsReq) (*model.K8sPod, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Pod详情请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("Pod名称不能为空")
	}

	pod, err := p.podManager.GetPod(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		p.logger.Error("GetPodDetails: 获取Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Pod失败: %w", err)
	}

	return utils.ConvertToK8sPod(pod), nil
}

// GetPodYaml 获取Pod YAML
func (p *podService) GetPodYaml(ctx context.Context, req *model.GetPodYamlReq) (*model.K8sYaml, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Pod YAML请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("Pod名称不能为空")
	}

	pod, err := p.podManager.GetPod(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		p.logger.Error("GetPodYaml: 获取Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Pod失败: %w", err)
	}

	// 转换为YAML
	yamlContent, err := utils.PodToYAML(pod)
	if err != nil {
		p.logger.Error("GetPodYaml: 转换为YAML失败",
			zap.Error(err),
			zap.String("podName", pod.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

// UpdatePod 更新Pod
func (p *podService) UpdatePod(ctx context.Context, req *model.UpdatePodReq) error {
	if req == nil {
		return fmt.Errorf("更新Pod请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Pod名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	// 先获取当前Pod
	currentPod, err := p.podManager.GetPod(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		p.logger.Error("UpdatePod: 获取当前Pod失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("获取当前Pod失败: %w", err)
	}

	// 更新标签和注解
	if req.Labels != nil {
		currentPod.Labels = req.Labels
	}
	if req.Annotations != nil {
		currentPod.Annotations = req.Annotations
	}

	// 验证Pod配置
	if err := utils.ValidatePod(currentPod); err != nil {
		p.logger.Error("UpdatePod: Pod配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("pod配置验证失败: %w", err)
	}

	// 更新Pod
	_, err = p.podManager.UpdatePod(ctx, req.ClusterID, currentPod)
	if err != nil {
		p.logger.Error("UpdatePod: 更新Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("更新Pod失败: %w", err)
	}

	return nil
}

// DeletePod 删除Pod
func (p *podService) DeletePod(ctx context.Context, req *model.DeletePodReq) error {
	if req == nil {
		return fmt.Errorf("删除Pod请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Pod名称不能为空")
	}

	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: req.GracePeriodSeconds,
	}

	if req.Force {
		policy := metav1.DeletePropagationBackground
		deleteOptions.PropagationPolicy = &policy
	}

	err := p.podManager.DeletePod(ctx, req.ClusterID, req.Namespace, req.Name, deleteOptions)
	if err != nil {
		p.logger.Error("DeletePod: 删除Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除Pod失败: %w", err)
	}

	return nil
}

// BatchDeletePods 批量删除Pod
func (p *podService) BatchDeletePods(ctx context.Context, req *model.BatchDeletePodsReq) error {
	if req == nil {
		return fmt.Errorf("批量删除Pod请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if len(req.Names) == 0 {
		return fmt.Errorf("Pod名称列表不能为空")
	}

	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: req.GracePeriodSeconds,
	}

	if req.Force {
		policy := metav1.DeletePropagationBackground
		deleteOptions.PropagationPolicy = &policy
	}

	err := p.podManager.BatchDeletePods(ctx, req.ClusterID, req.Namespace, req.Names, deleteOptions)
	if err != nil {
		p.logger.Error("BatchDeletePods: 批量删除Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.Int("count", len(req.Names)))
		return fmt.Errorf("批量删除Pod失败: %w", err)
	}

	return nil
}

// GetPodsByNodeName 获取指定节点上的Pod列表
func (p *podService) GetPodsByNodeName(ctx context.Context, req *model.GetPodsByNodeReq) ([]*model.K8sPod, error) {
	if req == nil {
		return nil, fmt.Errorf("获取节点Pod列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.NodeName == "" {
		return nil, fmt.Errorf("节点名称不能为空")
	}

	pods, err := p.podManager.GetPodsByNodeName(ctx, req.ClusterID, req.NodeName)
	if err != nil {
		p.logger.Error("GetPodsByNodeName: 获取节点Pod列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName))
		return nil, fmt.Errorf("获取节点Pod列表失败: %w", err)
	}

	return utils.ConvertToK8sPods(pods.Items), nil
}

// GetPodContainers 获取Pod的容器列表
func (p *podService) GetPodContainers(ctx context.Context, req *model.GetPodContainersReq) ([]*model.PodContainer, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Pod容器列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.PodName == "" {
		return nil, fmt.Errorf("Pod名称不能为空")
	}

	pod, err := p.podManager.GetPod(ctx, req.ClusterID, req.Namespace, req.PodName)
	if err != nil {
		p.logger.Error("GetPodContainers: 获取Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("podName", req.PodName))
		return nil, fmt.Errorf("获取Pod失败: %w", err)
	}

	// 转换容器信息
	containers := utils.ConvertPodContainers(pod)
	// 转换为指针slice
	result := make([]*model.PodContainer, len(containers))
	for i := range containers {
		result[i] = &containers[i]
	}
	return result, nil
}

// GetPodLogs 获取Pod日志
func (p *podService) GetPodLogs(ctx *gin.Context, req *model.GetPodLogsReq) error {
	if req == nil {
		return fmt.Errorf("获取Pod日志请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.PodName == "" {
		return fmt.Errorf("Pod名称不能为空")
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
			p.logger.Error("解析时间参数失败", zap.String("sinceTime", req.SinceTime), zap.Error(err))
			return fmt.Errorf("时间参数格式错误: %w", err)
		}
		metaTime := metav1.NewTime(sinceTime)
		logOptions.SinceTime = &metaTime
	}

	out, err := p.podManager.GetPodLogs(ctx, req.ClusterID, req.Namespace, req.PodName, logOptions)
	if err != nil {
		return err
	}

	utilsStream.SseStream(ctx, func(ctx context.Context, msgChan chan interface{}) {
		defer func() {
			if err := out.Close(); err != nil {
				p.logger.Error("关闭 Pod 日志流失败", zap.Error(err))
			}
		}()

		reader := bufio.NewReader(out)
		wait.UntilWithContext(ctx, func(ctx context.Context) {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					line = strings.TrimSpace(line)
					if len(line) > 0 {
						msgChan <- line
					}
					return
				}
				p.logger.Error("读取 Pod 日志失败", zap.Error(err))
				return
			}

			// 处理 \r 覆盖输出
			if strings.ContainsRune(line, '\r') {
				segments := strings.Split(line, "\r")
				for _, seg := range segments {
					seg = strings.TrimSpace(seg)
					if seg != "" {
						msgChan <- seg
					}
				}
			} else {
				msgChan <- strings.TrimSpace(line)
			}
		}, 0)
	}, p.logger)

	return nil
}

// PodExec Pod命令执行
func (p *podService) PodExec(ctx *gin.Context, req *model.PodExecReq) error {
	if req == nil {
		return fmt.Errorf("Pod命令执行请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.PodName == "" {
		return fmt.Errorf("Pod名称不能为空")
	}

	restConfig, err := p.getRestConfig(ctx, req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubeconfig配置失败", zap.Error(err))
		return err
	}

	conn, err := pkg.UpGrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		p.logger.Error("升级ws失败", zap.Error(err))
		return fmt.Errorf("初始化ws失败: %w", err)
	}

	return p.podManager.PodTerminalSession(ctx, req.ClusterID, req.Namespace, req.PodName, req.Container, req.Shell, conn, restConfig)
}

// PodPortForward Pod端口转发
func (p *podService) PodPortForward(ctx context.Context, req *model.PodPortForwardReq) error {
	if req == nil {
		return fmt.Errorf("Pod端口转发请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.PodName == "" {
		return fmt.Errorf("Pod名称不能为空")
	}

	restConfig, err := p.getRestConfig(ctx, req.ClusterID)
	if err != nil {
		return err
	}

	// 构造端口映射
	portsSpec := p.buildPortsSpec(req.Ports)

	// 构造 SPDY dialer
	dialer, err := p.buildDialer(restConfig, req.Namespace, req.PodName)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return p.podManager.PortForward(ctx, portsSpec, dialer)
}

// PodFileUpload 上传文件到Pod
func (p *podService) PodFileUpload(ctx *gin.Context, req *model.PodFileUploadReq) error {
	if req == nil {
		return fmt.Errorf("上传文件到Pod请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.PodName == "" {
		return fmt.Errorf("Pod名称不能为空")
	}

	restConfig, err := p.getRestConfig(ctx, req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubeconfig配置失败", zap.Error(err))
		return err
	}

	return p.podManager.UploadFileToPod(ctx, req.ClusterID, req.Namespace, req.PodName, req.ContainerName, req.FilePath, restConfig)
}

// PodFileDownload 从Pod下载文件
func (p *podService) PodFileDownload(ctx *gin.Context, req *model.PodFileDownloadReq) error {
	if req == nil {
		return fmt.Errorf("从Pod下载文件请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.PodName == "" {
		return fmt.Errorf("Pod名称不能为空")
	}

	restConfig, err := p.getRestConfig(ctx, req.ClusterID)
	if err != nil {
		return err
	}

	fileName := filepath.Base(req.FilePath)
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename=%s.tar`, fileName))
	ctx.Header("Content-Type", "application/octet-stream")

	reader, err := p.podManager.DownloadPodFile(ctx.Request.Context(), req.ClusterID, req.Namespace, req.PodName, req.ContainerName, req.FilePath, restConfig)
	if err != nil {
		p.logger.Error("创建Pod文件流失败",
			zap.Error(err),
			zap.String("Namespace", req.Namespace),
			zap.String("PodName", req.PodName),
			zap.String("ContainerName", req.ContainerName),
			zap.String("FilePath", req.FilePath))
		return fmt.Errorf("无法创建 Pod 文件流: %w", err)
	}
	defer reader.Close()

	// 把流复制到响应
	if _, err := io.Copy(ctx.Writer, reader); err != nil {
		p.logger.Error("文件下载过程中出错", zap.Error(err))
		return fmt.Errorf("下载文件过程中发生错误: %w", err)
	}

	return nil
}

// getRestConfig 获取REST配置
func (p *podService) getRestConfig(ctx context.Context, clusterID int) (*rest.Config, error) {
	cluster, err := p.dao.GetClusterByID(ctx, clusterID)
	if err != nil {
		p.logger.Error("获取集群信息失败", zap.Error(err))
		return nil, fmt.Errorf("无法获取集群信息: %w", err)
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
	if err != nil {
		p.logger.Error("解析 kubeconfig 失败", zap.Error(err))
		return nil, fmt.Errorf("无法解析 kubeconfig: %w", err)
	}

	// 设置合理的QPS和Burst参数
	restConfig.QPS = 50
	restConfig.Burst = 100
	return restConfig, nil
}

// buildDialer 构建SPDY dialer
func (p *podService) buildDialer(restConfig *rest.Config, namespace, podName string) (httpstream.Dialer, error) {
	roundTripper, upgrader, err := spdy.RoundTripperFor(restConfig)
	if err != nil {
		return nil, fmt.Errorf("创建RoundTripper失败: %w", err)
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

// buildPortsSpec 构建端口映射规范
func (p *podService) buildPortsSpec(ports []model.PodPortForwardPort) []string {
	specs := make([]string, len(ports))
	for i, port := range ports {
		specs[i] = fmt.Sprintf("%d:%d", port.LocalPort, port.RemotePort)
	}
	return specs
}

// CreatePodByYaml 通过YAML创建Pod
func (p *podService) CreatePodByYaml(ctx context.Context, req *model.CreatePodByYamlReq) error {
	if req == nil {
		return fmt.Errorf("通过YAML创建Pod请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.YAML == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	// 解析YAML为Pod对象
	pod, err := utils.YAMLToPod(req.YAML)
	if err != nil {
		p.logger.Error("CreatePodByYaml: 解析YAML失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 验证Pod配置
	if err := utils.ValidatePod(pod); err != nil {
		p.logger.Error("CreatePodByYaml: Pod配置验证失败",
			zap.Error(err),
			zap.String("name", pod.Name))
		return fmt.Errorf("Pod配置验证失败: %w", err)
	}

	// 创建Pod
	_, err = p.podManager.CreatePod(ctx, req.ClusterID, pod)
	if err != nil {
		p.logger.Error("CreatePodByYaml: 创建Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", pod.Name))
		return fmt.Errorf("创建Pod失败: %w", err)
	}

	return nil
}

// UpdatePodByYaml 通过YAML更新Pod
func (p *podService) UpdatePodByYaml(ctx context.Context, req *model.UpdatePodByYamlReq) error {
	if req == nil {
		return fmt.Errorf("通过YAML更新Pod请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.YAML == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	// 解析YAML为Pod对象
	pod, err := utils.YAMLToPod(req.YAML)
	if err != nil {
		p.logger.Error("UpdatePodByYaml: 解析YAML失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 验证Pod配置
	if err := utils.ValidatePod(pod); err != nil {
		p.logger.Error("UpdatePodByYaml: Pod配置验证失败",
			zap.Error(err),
			zap.String("name", pod.Name))
		return fmt.Errorf("Pod配置验证失败: %w", err)
	}

	// 更新Pod
	_, err = p.podManager.UpdatePod(ctx, req.ClusterID, pod)
	if err != nil {
		p.logger.Error("UpdatePodByYaml: 更新Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", pod.Name))
		return fmt.Errorf("更新Pod失败: %w", err)
	}

	return nil
}

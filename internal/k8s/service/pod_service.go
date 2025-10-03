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
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/sse"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodService interface {
	CreatePod(ctx context.Context, req *model.CreatePodReq) error
	GetPodList(ctx context.Context, req *model.GetPodListReq) (model.ListResp[*model.K8sPod], error)
	GetPodDetails(ctx context.Context, req *model.GetPodDetailsReq) (*model.K8sPod, error)
	GetPodYaml(ctx context.Context, req *model.GetPodYamlReq) (*model.K8sYaml, error)
	UpdatePod(ctx context.Context, req *model.UpdatePodReq) error
	DeletePod(ctx context.Context, req *model.DeletePodReq) error
	CreatePodByYaml(ctx context.Context, req *model.CreatePodByYamlReq) error
	UpdatePodByYaml(ctx context.Context, req *model.UpdatePodByYamlReq) error
	GetPodsByNodeName(ctx context.Context, req *model.GetPodsByNodeReq) (model.ListResp[*model.K8sPod], error)
	GetPodContainers(ctx context.Context, req *model.GetPodContainersReq) (model.ListResp[*model.PodContainer], error)
	GetPodLogs(ctx *gin.Context, req *model.GetPodLogsReq) error
	PodExec(ctx *gin.Context, req *model.PodExecReq) error
	PodPortForward(ctx context.Context, req *model.PodPortForwardReq) error
	PodFileDownload(ctx *gin.Context, req *model.PodFileDownloadReq) error
	PodFileUpload(ctx *gin.Context, req *model.PodFileUploadReq) error
}

type podService struct {
	podManager manager.PodManager
	sseHandler sse.Handler
	logger     *zap.Logger
}

func NewPodService(podManager manager.PodManager, sseHandler sse.Handler, logger *zap.Logger) PodService {
	return &podService{
		podManager: podManager,
		sseHandler: sseHandler,
		logger:     logger,
	}
}

// CreatePod 创建Pod
func (s *podService) CreatePod(ctx context.Context, req *model.CreatePodReq) error {
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
		s.logger.Error("CreatePod: 构建Pod对象失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("构建Pod对象失败: %w", err)
	}

	// 验证Pod配置
	if err := utils.ValidatePod(pod); err != nil {
		s.logger.Error("CreatePod: Pod配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("pod配置验证失败: %w", err)
	}

	// 创建Pod
	_, err = s.podManager.CreatePod(ctx, req.ClusterID, req.Namespace, pod)
	if err != nil {
		s.logger.Error("CreatePod: 创建Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建Pod失败: %w", err)
	}

	return nil
}

// GetPodList 获取Pod列表
func (s *podService) GetPodList(ctx context.Context, req *model.GetPodListReq) (model.ListResp[*model.K8sPod], error) {
	if req == nil {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("获取Pod列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("集群ID不能为空")
	}

	// 设置命名空间
	namespace := req.Namespace
	if namespace == "" {
		namespace = corev1.NamespaceAll
	}

	// 构建查询选项
	listOptions := metav1.ListOptions{}
	if req.Search != "" {
		// 简单名称匹配，复杂搜索可在此扩展
		// 注意：K8s标签选择器语法有限制，先获取所有再过滤
	}

	k8sPods, err := s.podManager.GetPodList(ctx, req.ClusterID, namespace, listOptions)
	if err != nil {
		s.logger.Error("GetPodList: 获取Pod列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", namespace))
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("获取Pod列表失败: %w", err)
	}

	// 过滤Pod列表
	var filteredPods []*model.K8sPod
	for _, pod := range k8sPods {
		// 状态过滤
		if req.Status != "" && pod.Status != req.Status {
			continue
		}
		// 名称搜索
		if req.Search != "" && !strings.Contains(pod.Name, req.Search) {
			continue
		}
		filteredPods = append(filteredPods, pod)
	}

	// 分页处理
	page := req.Page
	size := req.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10 // 默认每页10条
	}

	total := int64(len(filteredPods))
	start := (page - 1) * size
	end := start + size

	if start >= len(filteredPods) {
		filteredPods = []*model.K8sPod{}
	} else if end > len(filteredPods) {
		filteredPods = filteredPods[start:]
	} else {
		filteredPods = filteredPods[start:end]
	}

	return model.ListResp[*model.K8sPod]{
		Items: filteredPods,
		Total: total,
	}, nil
}

// GetPodDetails 获取Pod详情
func (s *podService) GetPodDetails(ctx context.Context, req *model.GetPodDetailsReq) (*model.K8sPod, error) {
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

	pod, err := s.podManager.GetPod(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("GetPodDetails: 获取Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Pod失败: %w", err)
	}

	return utils.ConvertToK8sPod(pod), nil
}

// GetPodYaml 获取Pod YAML
func (s *podService) GetPodYaml(ctx context.Context, req *model.GetPodYamlReq) (*model.K8sYaml, error) {
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

	pod, err := s.podManager.GetPod(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("GetPodYaml: 获取Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Pod失败: %w", err)
	}

	// 转换为YAML
	yamlContent, err := utils.PodToYAML(pod)
	if err != nil {
		s.logger.Error("GetPodYaml: 转换为YAML失败",
			zap.Error(err),
			zap.String("podName", pod.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

// UpdatePod 更新Pod
func (s *podService) UpdatePod(ctx context.Context, req *model.UpdatePodReq) error {
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
	currentPod, err := s.podManager.GetPod(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("UpdatePod: 获取当前Pod失败",
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
		s.logger.Error("UpdatePod: Pod配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("pod配置验证失败: %w", err)
	}

	// 更新Pod
	_, err = s.podManager.UpdatePod(ctx, req.ClusterID, req.Namespace, currentPod)
	if err != nil {
		s.logger.Error("UpdatePod: 更新Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新Pod失败: %w", err)
	}

	return nil
}

// DeletePod 删除Pod
func (s *podService) DeletePod(ctx context.Context, req *model.DeletePodReq) error {
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

	err := s.podManager.DeletePod(ctx, req.ClusterID, req.Namespace, req.Name, deleteOptions)
	if err != nil {
		s.logger.Error("DeletePod: 删除Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除Pod失败: %w", err)
	}

	return nil
}

// GetPodsByNodeName 获取指定节点上的Pod列表
func (s *podService) GetPodsByNodeName(ctx context.Context, req *model.GetPodsByNodeReq) (model.ListResp[*model.K8sPod], error) {
	if req == nil {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("获取节点Pod列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("集群ID不能为空")
	}

	if req.NodeName == "" {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("节点名称不能为空")
	}

	pods, err := s.podManager.GetPodsByNodeName(ctx, req.ClusterID, req.NodeName)
	if err != nil {
		s.logger.Error("GetPodsByNodeName: 获取节点Pod列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName))
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("获取节点Pod列表失败: %w", err)
	}

	return model.ListResp[*model.K8sPod]{
		Items: pods,
		Total: int64(len(pods)),
	}, nil
}

// GetPodContainers 获取Pod的容器列表
func (s *podService) GetPodContainers(ctx context.Context, req *model.GetPodContainersReq) (model.ListResp[*model.PodContainer], error) {
	if req == nil {
		return model.ListResp[*model.PodContainer]{}, fmt.Errorf("获取Pod容器列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.PodContainer]{}, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return model.ListResp[*model.PodContainer]{}, fmt.Errorf("命名空间不能为空")
	}

	if req.PodName == "" {
		return model.ListResp[*model.PodContainer]{}, fmt.Errorf("Pod名称不能为空")
	}

	pod, err := s.podManager.GetPod(ctx, req.ClusterID, req.Namespace, req.PodName)
	if err != nil {
		s.logger.Error("GetPodContainers: 获取Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("podName", req.PodName))
		return model.ListResp[*model.PodContainer]{}, fmt.Errorf("获取Pod失败: %w", err)
	}

	// 转换容器信息
	containers := utils.ConvertPodContainers(pod)
	// 转换为指针切片
	result := make([]*model.PodContainer, len(containers))
	for i := range containers {
		result[i] = &containers[i]
	}
	return model.ListResp[*model.PodContainer]{
		Items: result,
		Total: int64(len(result)),
	}, nil
}

// GetPodLogs 获取Pod日志
func (s *podService) GetPodLogs(ctx *gin.Context, req *model.GetPodLogsReq) error {
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
			s.logger.Error("解析时间参数失败", zap.String("sinceTime", req.SinceTime), zap.Error(err))
			return fmt.Errorf("时间参数格式错误: %w", err)
		}
		metaTime := metav1.NewTime(sinceTime)
		logOptions.SinceTime = &metaTime
	}

	out, err := s.podManager.GetPodLogs(ctx, req.ClusterID, req.Namespace, req.PodName, logOptions)
	if err != nil {
		// 检查是否请求previous日志但容器未重启
		if strings.Contains(err.Error(), "previous terminated container") &&
			strings.Contains(err.Error(), "not found") {
			return s.sseHandler.Stream(ctx, func(ctx context.Context, msgChan chan<- interface{}) {
				// 发送友好提示
				msgChan <- "该容器没有重启过，无法获取之前的日志。请取消 'Previous' 选项查看当前日志。"
				s.logger.Info("容器没有previous日志",
					zap.Int("clusterID", req.ClusterID),
					zap.String("namespace", req.Namespace),
					zap.String("podName", req.PodName))
			})
		}
		return err
	}

	return s.sseHandler.Stream(ctx, func(ctx context.Context, msgChan chan<- interface{}) {
		defer func() {
			if err := out.Close(); err != nil {
				// 检查是否是客户端断开连接
				if errors.Is(err, context.Canceled) ||
					strings.Contains(err.Error(), "request canceled") ||
					strings.Contains(err.Error(), "context cancellation") {
					s.logger.Debug("Pod日志流已正常关闭（客户端断开）", zap.Error(err))
				} else {
					s.logger.Error("关闭Pod日志流失败", zap.Error(err))
				}
			}
		}()

		reader := bufio.NewReader(out)
		retryCount := 0
		maxRetries := 5

		for {
			select {
			case <-ctx.Done():
				s.logger.Info("上下文已取消，停止读取Pod日志")
				return
			default:
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						line = strings.TrimSpace(line)
						if len(line) > 0 {
							msgChan <- line
						}

						// Follow模式不立即关闭连接
						// EOF可能只是暂时无新日志，继续等待
						if logOptions.Follow {
							s.logger.Debug("Pod日志暂时无新内容，继续等待...")
							// 短暂等待避免CPU密集循环
							select {
							case <-ctx.Done():
								return
							case <-time.After(time.Millisecond * 100):
								continue
							}
						} else {
							s.logger.Info("Pod日志流已结束")
							return
						}
					}

					// 检查是否是上下文取消或网络问题
					if errors.Is(err, context.Canceled) ||
						errors.Is(err, context.DeadlineExceeded) ||
						strings.Contains(err.Error(), "Client.Timeout") ||
						strings.Contains(err.Error(), "context cancellation") ||
						strings.Contains(err.Error(), "request canceled") {
						s.logger.Info("客户端断开连接或请求超时，停止读取Pod日志")
						return
					}

					// 检查网络超时
					var netErr net.Error
					if errors.As(err, &netErr) && netErr.Timeout() {
						s.logger.Info("网络超时，停止读取Pod日志")
						return
					}

					retryCount++
					if retryCount > maxRetries {
						s.logger.Error("读取Pod日志失败，已达到最大重试次数",
							zap.Error(err),
							zap.Int("retryCount", retryCount))
						return
					}

					s.logger.Warn("读取Pod日志失败，将重试",
						zap.Error(err),
						zap.Int("retryCount", retryCount),
						zap.Int("maxRetries", maxRetries))

					// 短暂等待后重试
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Millisecond * 100):
						continue
					}
				}

				// 重置重试计数器
				retryCount = 0

				// 处理回车符覆盖输出
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
			}
		}
	})
}

// PodExec Pod终端执行
func (s *podService) PodExec(ctx *gin.Context, req *model.PodExecReq) error {
	if req == nil {
		return fmt.Errorf("Pod终端执行请求不能为空")
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

	conn, err := pkg.UpGrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		s.logger.Error("升级ws失败", zap.Error(err))
		return fmt.Errorf("初始化WebSocket失败: %w", err)
	}

	return s.podManager.PodTerminalSession(ctx, req.ClusterID, req.Namespace, req.PodName, req.Container, req.Shell, conn)
}

// PodPortForward Pod端口转发
func (s *podService) PodPortForward(ctx context.Context, req *model.PodPortForwardReq) error {
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

	if len(req.Ports) == 0 {
		return fmt.Errorf("端口转发配置不能为空")
	}

	// 验证端口范围
	for _, port := range req.Ports {
		if port.LocalPort <= 0 || port.LocalPort > 65535 {
			return fmt.Errorf("本地端口范围无效: %d", port.LocalPort)
		}
		if port.RemotePort <= 0 || port.RemotePort > 65535 {
			return fmt.Errorf("远程端口范围无效: %d", port.RemotePort)
		}
	}

	// 调用Manager层进行端口转发
	return s.podManager.PodPortForward(ctx, req.ClusterID, req.Namespace, req.PodName, req.Ports)
}

// PodFileUpload 上传文件到Pod
func (s *podService) PodFileUpload(ctx *gin.Context, req *model.PodFileUploadReq) error {
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

	if req.ContainerName == "" {
		return fmt.Errorf("容器名称不能为空")
	}

	if req.FilePath == "" {
		return fmt.Errorf("文件路径不能为空")
	}

	s.logger.Info("开始上传文件到Pod",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("podName", req.PodName),
		zap.String("containerName", req.ContainerName),
		zap.String("filePath", req.FilePath))

	err := s.podManager.UploadFileToPod(ctx, req.ClusterID, req.Namespace, req.PodName, req.ContainerName, req.FilePath)
	if err != nil {
		s.logger.Error("上传文件到Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("podName", req.PodName),
			zap.String("containerName", req.ContainerName),
			zap.String("filePath", req.FilePath))
		return fmt.Errorf("上传文件到Pod失败: %w", err)
	}

	s.logger.Info("成功上传文件到Pod",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("podName", req.PodName),
		zap.String("containerName", req.ContainerName),
		zap.String("filePath", req.FilePath))

	return nil
}

// PodFileDownload 从Pod下载文件
func (s *podService) PodFileDownload(ctx *gin.Context, req *model.PodFileDownloadReq) error {
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

	if req.ContainerName == "" {
		return fmt.Errorf("容器名称不能为空")
	}

	if req.FilePath == "" {
		return fmt.Errorf("文件路径不能为空")
	}

	s.logger.Info("开始从Pod下载文件",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("podName", req.PodName),
		zap.String("containerName", req.ContainerName),
		zap.String("filePath", req.FilePath))

	// 生成文件名
	fileName := filepath.Base(req.FilePath)
	if fileName == "." || fileName == "/" || fileName == "" {
		fileName = "download"
	}

	// 设置下载响应头
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.tar"`, fileName))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	ctx.Header("Pragma", "no-cache")
	ctx.Header("Expires", "0")

	// 创建文件流
	reader, err := s.podManager.DownloadPodFile(ctx.Request.Context(), req.ClusterID, req.Namespace, req.PodName, req.ContainerName, req.FilePath)
	if err != nil {
		s.logger.Error("创建Pod文件流失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("podName", req.PodName),
			zap.String("containerName", req.ContainerName),
			zap.String("filePath", req.FilePath))
		return fmt.Errorf("无法创建Pod文件流: %w", err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			s.logger.Error("关闭文件流失败", zap.Error(closeErr))
		}
	}()

	// 直接使用io.Copy进行高效复制
	bytesWritten, err := io.Copy(ctx.Writer, reader)
	if err != nil {
		s.logger.Error("文件传输失败",
			zap.Error(err),
			zap.Int64("bytesWritten", bytesWritten))
		return fmt.Errorf("文件传输失败: %w", err)
	}

	s.logger.Info("成功下载文件从Pod",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("podName", req.PodName),
		zap.String("containerName", req.ContainerName),
		zap.String("filePath", req.FilePath),
		zap.Int64("bytesDownloaded", bytesWritten))

	return nil
}

// CreatePodByYaml 通过YAML创建Pod
func (s *podService) CreatePodByYaml(ctx context.Context, req *model.CreatePodByYamlReq) error {
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
		s.logger.Error("CreatePodByYaml: 解析YAML失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 验证Pod配置
	if err := utils.ValidatePod(pod); err != nil {
		s.logger.Error("CreatePodByYaml: Pod配置验证失败",
			zap.Error(err),
			zap.String("name", pod.Name))
		return fmt.Errorf("Pod配置验证失败: %w", err)
	}

	// 创建Pod
	_, err = s.podManager.CreatePod(ctx, req.ClusterID, pod.Namespace, pod)
	if err != nil {
		s.logger.Error("CreatePodByYaml: 创建Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", pod.Namespace),
			zap.String("name", pod.Name))
		return fmt.Errorf("创建Pod失败: %w", err)
	}

	return nil
}

// UpdatePodByYaml 通过YAML更新Pod
func (s *podService) UpdatePodByYaml(ctx context.Context, req *model.UpdatePodByYamlReq) error {
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
		s.logger.Error("UpdatePodByYaml: 解析YAML失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 验证Pod配置
	if err := utils.ValidatePod(pod); err != nil {
		s.logger.Error("UpdatePodByYaml: Pod配置验证失败",
			zap.Error(err),
			zap.String("name", pod.Name))
		return fmt.Errorf("Pod配置验证失败: %w", err)
	}

	// 更新Pod
	_, err = s.podManager.UpdatePod(ctx, req.ClusterID, req.Namespace, pod)
	if err != nil {
		s.logger.Error("UpdatePodByYaml: 更新Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", pod.Name))
		return fmt.Errorf("更新Pod失败: %w", err)
	}

	return nil
}

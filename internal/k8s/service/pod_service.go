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

	pod, err := utils.BuildPodFromRequest(req)
	if err != nil {
		s.logger.Error("构建Pod对象失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("构建Pod对象失败: %w", err)
	}

	if err := utils.ValidatePod(pod); err != nil {
		s.logger.Error("Pod配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("pod配置验证失败: %w", err)
	}

	_, err = s.podManager.CreatePod(ctx, req.ClusterID, req.Namespace, pod)
	if err != nil {
		s.logger.Error("创建Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建Pod失败: %w", err)
	}

	s.logger.Info("创建Pod成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

func (s *podService) GetPodList(ctx context.Context, req *model.GetPodListReq) (model.ListResp[*model.K8sPod], error) {
	if req == nil {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("获取Pod列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("集群ID不能为空")
	}

	namespace := req.Namespace
	if namespace == "" {
		namespace = corev1.NamespaceAll
	}

	listOptions := metav1.ListOptions{}
	// K8s的label selector不支持模糊匹配
	// 对于复杂查询条件，需要先获取全量数据再在内存中过滤

	k8sPods, err := s.podManager.GetPodList(ctx, req.ClusterID, namespace, listOptions)
	if err != nil {
		s.logger.Error("获取Pod列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", namespace))
		return model.ListResp[*model.K8sPod]{}, fmt.Errorf("获取Pod列表失败: %w", err)
	}

	// 应用过滤条件
	var filteredPods []*model.K8sPod
	for _, pod := range k8sPods {
		// 状态过滤
		if req.Status != "" && pod.Status != req.Status {
			continue
		}
		// 名称过滤（使用通用的Search字段，支持不区分大小写）
		if !utils.FilterByName(pod.Name, req.Search) {
			continue
		}
		filteredPods = append(filteredPods, pod)
	}

	// 按创建时间排序（最新的在前）
	utils.SortByCreationTime(filteredPods, func(pod *model.K8sPod) time.Time {
		return pod.CreatedAt
	})

	// 分页处理
	pagedPods, total := utils.Paginate(filteredPods, req.Page, req.Size)

	return model.ListResp[*model.K8sPod]{
		Items: pagedPods,
		Total: total,
	}, nil
}

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
		s.logger.Error("获取Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Pod失败: %w", err)
	}

	return utils.ConvertToK8sPod(pod), nil
}

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
		s.logger.Error("获取Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Pod失败: %w", err)
	}

	yamlContent, err := utils.PodToYAML(pod)
	if err != nil {
		s.logger.Error("转换为YAML失败",
			zap.Error(err),
			zap.String("podName", pod.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

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

	currentPod, err := s.podManager.GetPod(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("获取当前Pod失败",
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

	if err := utils.ValidatePod(currentPod); err != nil {
		s.logger.Error("Pod配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("pod配置验证失败: %w", err)
	}

	_, err = s.podManager.UpdatePod(ctx, req.ClusterID, req.Namespace, currentPod)
	if err != nil {
		s.logger.Error("更新Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新Pod失败: %w", err)
	}

	s.logger.Info("更新Pod成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

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
		s.logger.Error("删除Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除Pod失败: %w", err)
	}

	s.logger.Info("删除Pod成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

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
		s.logger.Error("获取节点Pod列表失败",
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
		s.logger.Error("获取Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("podName", req.PodName))
		return model.ListResp[*model.PodContainer]{}, fmt.Errorf("获取Pod失败: %w", err)
	}

	containers := utils.ConvertPodContainers(pod)

	result := make([]*model.PodContainer, len(containers))
	for i := range containers {
		result[i] = &containers[i]
	}
	return model.ListResp[*model.PodContainer]{
		Items: result,
		Total: int64(len(result)),
	}, nil
}

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

	// 使用SSE流式传输日志，避免内存占用过大
	// Follow模式下保持连接直到客户端断开或Pod终止
	return s.sseHandler.Stream(ctx, func(ctx context.Context, msgChan chan<- interface{}) {
		defer func() {
			if err := out.Close(); err != nil {
				// 区分正常断开和异常错误，避免误报
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

						// Follow模式：EOF只是暂时无新日志，需要继续等待
						// 非Follow模式：EOF表示日志结束，可以关闭连接
						if logOptions.Follow {
							s.logger.Debug("Pod日志暂时无新内容，继续等待...")
							// 避免CPU密集循环，100ms的等待既不会错过新日志，也不会过度消耗CPU
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

					// 网络异常或客户端断开：直接退出，不进行重试
					// 这些错误通常不可恢复，重试只会浪费资源
					if errors.Is(err, context.Canceled) ||
						errors.Is(err, context.DeadlineExceeded) ||
						strings.Contains(err.Error(), "Client.Timeout") ||
						strings.Contains(err.Error(), "context cancellation") ||
						strings.Contains(err.Error(), "request canceled") {
						s.logger.Info("客户端断开连接或请求超时，停止读取Pod日志")
						return
					}

					var netErr net.Error
					if errors.As(err, &netErr) && netErr.Timeout() {
						s.logger.Info("网络超时，停止读取Pod日志")
						return
					}

					// 其他错误进行有限次数重试，防止临时网络抖动导致日志中断
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

					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Millisecond * 100):
						continue
					}
				}

				retryCount = 0

				// 处理进度条等使用\r覆盖输出的场景
				// 将每个\r分隔的片段单独发送，保证前端能正确渲染
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

	for _, port := range req.Ports {
		if port.LocalPort <= 0 || port.LocalPort > 65535 {
			return fmt.Errorf("本地端口范围无效: %d", port.LocalPort)
		}
		if port.RemotePort <= 0 || port.RemotePort > 65535 {
			return fmt.Errorf("远程端口范围无效: %d", port.RemotePort)
		}
	}

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

	s.logger.Info("开始通过YAML创建Pod",
		zap.Int("clusterID", req.ClusterID))

	// 解析YAML为Pod对象
	pod, err := utils.YAMLToPod(req.YAML)
	if err != nil {
		s.logger.Error("解析YAML失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID))
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 如果YAML中没有指定namespace，使用default
	if pod.Namespace == "" {
		pod.Namespace = "default"
	}

	// YAML中必须包含name信息
	if pod.Name == "" {
		s.logger.Error("YAML中必须指定name",
			zap.Int("clusterID", req.ClusterID))
		return fmt.Errorf("YAML中必须指定name")
	}

	if err := utils.ValidatePod(pod); err != nil {
		s.logger.Error("Pod配置验证失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", pod.Name))
		return fmt.Errorf("Pod配置验证失败: %w", err)
	}

	_, err = s.podManager.CreatePod(ctx, req.ClusterID, pod.Namespace, pod)
	if err != nil {
		s.logger.Error("创建Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", pod.Namespace),
			zap.String("name", pod.Name))
		return fmt.Errorf("创建Pod失败: %w", err)
	}

	s.logger.Info("创建Pod成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", pod.Namespace),
		zap.String("name", pod.Name))

	return nil
}

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

	s.logger.Info("开始通过YAML更新Pod",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	// 解析YAML为Pod对象
	pod, err := utils.YAMLToPod(req.YAML)
	if err != nil {
		s.logger.Error("解析YAML失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 确保YAML中的namespace和name与请求参数一致
	if pod.Namespace != "" && pod.Namespace != req.Namespace {
		s.logger.Error("YAML中的namespace与请求参数不一致",
			zap.Int("clusterID", req.ClusterID),
			zap.String("yamlNamespace", pod.Namespace),
			zap.String("reqNamespace", req.Namespace))
		return fmt.Errorf("YAML中的namespace (%s) 与请求参数不一致 (%s)", pod.Namespace, req.Namespace)
	}

	if pod.Name != "" && pod.Name != req.Name {
		s.logger.Error("YAML中的name与请求参数不一致",
			zap.Int("clusterID", req.ClusterID),
			zap.String("yamlName", pod.Name),
			zap.String("reqName", req.Name))
		return fmt.Errorf("YAML中的name (%s) 与请求参数不一致 (%s)", pod.Name, req.Name)
	}

	// 如果YAML中没有指定，使用请求参数
	if pod.Namespace == "" {
		pod.Namespace = req.Namespace
	}

	if pod.Name == "" {
		pod.Name = req.Name
	}

	if err := utils.ValidatePod(pod); err != nil {
		s.logger.Error("Pod配置验证失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", pod.Name))
		return fmt.Errorf("Pod配置验证失败: %w", err)
	}

	_, err = s.podManager.UpdatePod(ctx, req.ClusterID, req.Namespace, pod)
	if err != nil {
		s.logger.Error("更新Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", pod.Name))
		return fmt.Errorf("更新Pod失败: %w", err)
	}

	s.logger.Info("更新Pod成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", pod.Name))

	return nil
}

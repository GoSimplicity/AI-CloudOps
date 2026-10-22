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

package api

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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/sse"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type K8sPodHandler struct {
	podService service.PodService
	sseHandler sse.Handler
	logger     *zap.Logger
	wsUpgrader *websocket.Upgrader
}

func NewK8sPodHandler(podService service.PodService, sseHandler sse.Handler, logger *zap.Logger, wsUpgrader *websocket.Upgrader) *K8sPodHandler {
	return &K8sPodHandler{
		podService: podService,
		sseHandler: sseHandler,
		logger:     logger,
		wsUpgrader: wsUpgrader,
	}
}

func (h *K8sPodHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/pod/:cluster_id/list", h.GetPodList)
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/detail", h.GetPodDetails)
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/detail/yaml", h.GetPodYaml)
		k8sGroup.POST("/pod/:cluster_id/create", h.CreatePod)
		k8sGroup.POST("/pod/:cluster_id/create/yaml", h.CreatePodByYaml)
		k8sGroup.PUT("/pod/:cluster_id/:namespace/:name/update", h.UpdatePod)
		k8sGroup.PUT("/pod/:cluster_id/:namespace/:name/update/yaml", h.UpdatePodByYaml)
		k8sGroup.DELETE("/pod/:cluster_id/:namespace/:name/delete", h.DeletePod)
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/containers", h.GetPodContainers)
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/containers/:container/logs", h.GetPodLogs)
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/containers/:container/exec", h.PodExec)
		k8sGroup.POST("/pod/:cluster_id/:namespace/:name/port-forward", h.PodPortForward)
		k8sGroup.POST("/pod/:cluster_id/:namespace/:name/containers/:container/files/upload", h.PodFileUpload)
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/containers/:container/files/download", h.PodFileDownload)
	}
}

func (h *K8sPodHandler) GetPodDetails(ctx *gin.Context) {
	var req model.GetPodDetailsReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.podService.GetPodDetails(ctx.Request.Context(), &req)
	})
}

func (h *K8sPodHandler) GetPodList(ctx *gin.Context) {
	var req model.GetPodListReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.podService.GetPodList(ctx.Request.Context(), &req)
	})
}

func (h *K8sPodHandler) GetPodContainers(ctx *gin.Context) {
	var req model.GetPodContainersReq

	if err := ctx.ShouldBindUri(&req); err != nil {
		base.BadRequestError(ctx, "参数绑定失败: "+err.Error())
		return
	}

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.podService.GetPodContainers(ctx.Request.Context(), &req)
	})
}

func (h *K8sPodHandler) GetPodLogs(ctx *gin.Context) {
	var req model.GetPodLogsReq

	if err := ctx.ShouldBindUri(&req); err != nil {
		base.BadRequestError(ctx, "参数绑定失败: "+err.Error())
		return
	}

	if err := base.BindRequest(ctx, &req); err != nil {
		base.BadRequestError(ctx, "查询参数绑定失败: "+err.Error())
		return
	}

	out, err := h.podService.GetPodLogs(ctx.Request.Context(), &req)
	if err != nil {
		var previousErr *service.PodLogsPreviousNotFoundError
		if errors.As(err, &previousErr) {
			if streamErr := h.sseHandler.Stream(ctx, func(streamCtx context.Context, msgChan chan<- interface{}) {
				msgChan <- previousErr.Error()
			}); streamErr != nil {
				base.BadRequestError(ctx, streamErr.Error())
			}
			return
		}

		base.BadRequestError(ctx, err.Error())
		return
	}

	if streamErr := h.sseHandler.Stream(ctx, func(streamCtx context.Context, msgChan chan<- interface{}) {
		h.streamPodLogLines(streamCtx, out, req.Follow, msgChan)
	}); streamErr != nil {
		h.logger.Error("推送Pod日志失败", zap.Error(streamErr))
		return
	}
}

func (h *K8sPodHandler) GetPodYaml(ctx *gin.Context) {
	var req model.GetPodYamlReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.podService.GetPodYaml(ctx.Request.Context(), &req)
	})
}

func (h *K8sPodHandler) CreatePod(ctx *gin.Context) {
	var req model.CreatePodReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.podService.CreatePod(ctx.Request.Context(), &req)
	})
}

func (h *K8sPodHandler) CreatePodByYaml(ctx *gin.Context) {
	var req model.CreatePodByYamlReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.podService.CreatePodByYaml(ctx.Request.Context(), &req)
	})
}

func (h *K8sPodHandler) UpdatePod(ctx *gin.Context) {
	var req model.UpdatePodReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.podService.UpdatePod(ctx.Request.Context(), &req)
	})
}

func (h *K8sPodHandler) UpdatePodByYaml(ctx *gin.Context) {
	var req model.UpdatePodByYamlReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.podService.UpdatePodByYaml(ctx.Request.Context(), &req)
	})
}

func (h *K8sPodHandler) DeletePod(ctx *gin.Context) {
	var req model.DeletePodReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.podService.DeletePod(ctx.Request.Context(), &req)
	})
}

func (h *K8sPodHandler) PodExec(ctx *gin.Context) {
	var req model.PodExecReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, "集群ID参数错误: "+err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, "命名空间参数错误: "+err.Error())
		return
	}

	podName, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, "Pod名称参数错误: "+err.Error())
		return
	}

	container, err := base.GetParamCustomName(ctx, "container")
	if err != nil {
		base.BadRequestError(ctx, "容器名称参数错误: "+err.Error())
		return
	}

	shell := ctx.DefaultQuery("shell", "sh")

	if clusterID <= 0 {
		base.BadRequestError(ctx, "集群ID必须大于0")
		return
	}

	if namespace == "" {
		base.BadRequestError(ctx, "命名空间不能为空")
		return
	}

	if podName == "" {
		base.BadRequestError(ctx, "Pod名称不能为空")
		return
	}

	if container == "" {
		base.BadRequestError(ctx, "容器名称不能为空")
		return
	}

	// 安全性：只允许常见的安全shell，防止命令注入
	validShells := []string{"sh", "bash", "zsh", "fish", "ash"}
	isValidShell := false
	for _, validShell := range validShells {
		if shell == validShell {
			isValidShell = true
			break
		}
	}
	if !isValidShell {
		base.BadRequestError(ctx, "不支持的shell类型: "+shell)
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.PodName = podName
	req.Container = container
	req.Shell = shell

	if h.wsUpgrader == nil {
		base.BadRequestError(ctx, "WebSocket升级器未初始化")
		return
	}

	conn, err := h.wsUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		base.BadRequestError(ctx, "初始化WebSocket失败: "+err.Error())
		return
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			h.logger.Error("关闭WebSocket连接失败", zap.Error(closeErr))
		}
	}()

	if err := h.podService.PodExec(ctx.Request.Context(), &req, conn); err != nil {
		h.logger.Error("建立终端连接失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("podName", req.PodName),
			zap.String("container", req.Container))
		return
	}
}

func (h *K8sPodHandler) PodPortForward(ctx *gin.Context) {
	var req model.PodPortForwardReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	podName, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.PodName = podName

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.podService.PodPortForward(ctx.Request.Context(), &req)
	})
}

func (h *K8sPodHandler) PodFileUpload(ctx *gin.Context) {
	var req model.PodFileUploadReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, "集群ID参数错误: "+err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, "命名空间参数错误: "+err.Error())
		return
	}

	podName, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, "Pod名称参数错误: "+err.Error())
		return
	}

	container, err := base.GetParamCustomName(ctx, "container")
	if err != nil {
		base.BadRequestError(ctx, "容器名称参数错误: "+err.Error())
		return
	}

	filePath := ctx.Query("file_path")
	if filePath == "" {
		filePath = ctx.PostForm("file_path")
	}
	if filePath == "" {
		filePath = "/tmp"
	}

	if clusterID <= 0 {
		base.BadRequestError(ctx, "集群ID必须大于0")
		return
	}

	if namespace == "" {
		base.BadRequestError(ctx, "命名空间不能为空")
		return
	}

	if podName == "" {
		base.BadRequestError(ctx, "Pod名称不能为空")
		return
	}

	if container == "" {
		base.BadRequestError(ctx, "容器名称不能为空")
		return
	}

	if !isValidPath(filePath) {
		base.BadRequestError(ctx, "无效的文件路径格式")
		return
	}

	if ctx.Request.MultipartForm == nil {
		if err := ctx.Request.ParseMultipartForm(32 << 20); err != nil {
			base.BadRequestError(ctx, "解析上传文件失败: "+err.Error())
			return
		}
	}

	if ctx.Request.MultipartForm == nil || len(ctx.Request.MultipartForm.File) == 0 {
		base.BadRequestError(ctx, "未找到上传的文件")
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.PodName = podName
	req.ContainerName = container
	req.FilePath = filePath

	if ctx.Request.MultipartForm == nil {
		base.BadRequestError(ctx, "未找到上传的文件")
		return
	}

	if err := h.podService.PodFileUpload(ctx.Request.Context(), &req, ctx.Request.MultipartForm); err != nil {
		base.BadRequestError(ctx, "文件上传失败: "+err.Error())
		return
	}

	base.Success(ctx)
}

func (h *K8sPodHandler) PodFileDownload(ctx *gin.Context) {
	var req model.PodFileDownloadReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, "集群ID参数错误: "+err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, "命名空间参数错误: "+err.Error())
		return
	}

	podName, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, "Pod名称参数错误: "+err.Error())
		return
	}

	container, err := base.GetParamCustomName(ctx, "container")
	if err != nil {
		base.BadRequestError(ctx, "容器名称参数错误: "+err.Error())
		return
	}

	filePath := ctx.Query("file_path")

	if clusterID <= 0 {
		base.BadRequestError(ctx, "集群ID必须大于0")
		return
	}

	if namespace == "" {
		base.BadRequestError(ctx, "命名空间不能为空")
		return
	}

	if podName == "" {
		base.BadRequestError(ctx, "Pod名称不能为空")
		return
	}

	if container == "" {
		base.BadRequestError(ctx, "容器名称不能为空")
		return
	}

	if filePath == "" {
		base.BadRequestError(ctx, "文件路径不能为空")
		return
	}

	if !isValidPath(filePath) {
		base.BadRequestError(ctx, "无效的文件路径格式")
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.PodName = podName
	req.ContainerName = container
	req.FilePath = filePath

	reader, fileName, err := h.podService.PodFileDownload(ctx.Request.Context(), &req)
	if err != nil {
		base.BadRequestError(ctx, "文件下载失败: "+err.Error())
		return
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			h.logger.Error("关闭文件流失败", zap.Error(closeErr))
		}
	}()

	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.tar"`, fileName))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	ctx.Header("Pragma", "no-cache")
	ctx.Header("Expires", "0")

	bytesWritten, err := io.Copy(ctx.Writer, reader)
	if err != nil {
		h.logger.Error("文件传输失败",
			zap.Error(err),
			zap.Int64("bytesWritten", bytesWritten),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("podName", req.PodName),
			zap.String("container", req.ContainerName),
			zap.String("filePath", req.FilePath))
		return
	}

	h.logger.Info("文件下载完成",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("podName", req.PodName),
		zap.String("container", req.ContainerName),
		zap.String("filePath", req.FilePath),
		zap.Int64("bytesWritten", bytesWritten))
}

func (h *K8sPodHandler) streamPodLogLines(ctx context.Context, out io.ReadCloser, follow bool, msgChan chan<- interface{}) {
	defer func() {
		if err := out.Close(); err != nil {
			if errors.Is(err, context.Canceled) ||
				errors.Is(err, context.DeadlineExceeded) ||
				strings.Contains(err.Error(), "request canceled") ||
				strings.Contains(err.Error(), "context cancellation") {
				h.logger.Debug("Pod日志流已正常关闭", zap.Error(err))
				return
			}
			h.logger.Error("关闭Pod日志流失败", zap.Error(err))
		}
	}()

	reader := bufio.NewReader(out)
	retryCount := 0
	maxRetries := 5

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("上下文已取消，停止读取Pod日志")
			return
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if errors.Is(err, context.Canceled) ||
					errors.Is(err, context.DeadlineExceeded) ||
					strings.Contains(err.Error(), "Client.Timeout") ||
					strings.Contains(err.Error(), "context cancellation") ||
					strings.Contains(err.Error(), "request canceled") {
					h.logger.Info("客户端断开连接或请求超时，停止读取Pod日志")
					return
				}

				var netErr net.Error
				if errors.As(err, &netErr) && netErr.Timeout() {
					h.logger.Info("网络超时，停止读取Pod日志")
					return
				}

				if errors.Is(err, io.EOF) {
					line = strings.TrimSpace(line)
					if line != "" {
						msgChan <- line
					}

					if follow {
						select {
						case <-ctx.Done():
							return
						case <-time.After(time.Millisecond * 100):
							continue
						}
					}
					return
				}

				retryCount++
				if retryCount > maxRetries {
					h.logger.Error("读取Pod日志失败，已达到最大重试次数",
						zap.Error(err),
						zap.Int("retryCount", retryCount),
						zap.Int("maxRetries", maxRetries))
					msgChan <- fmt.Sprintf("日志读取失败: %v", err)
					return
				}

				h.logger.Warn("读取Pod日志失败，将重试",
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

			if strings.ContainsRune(line, '\r') {
				segments := strings.Split(line, "\r")
				for _, seg := range segments {
					seg = strings.TrimSpace(seg)
					if seg != "" {
						msgChan <- seg
					}
				}
				continue
			}

			line = strings.TrimSpace(line)
			if line != "" {
				msgChan <- line
			}
		}
	}
}

// isValidPath 验证文件路径的安全性，防止路径遍历等安全攻击
// 1. 路径遍历攻击 (..)
// 2. 注入攻击 (\n, \r)
// 3. 空字节注入 (\x00)
// 4. 缓冲区溢出 (长度限制)
// 5. 用户主目录访问 (~)
func isValidPath(path string) bool {
	if path == "" {
		return false
	}

	cleanPath := filepath.Clean(path)

	dangerousPatterns := []string{"..", "\n", "\r", "\x00"}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(cleanPath, pattern) {
			return false
		}
	}

	if len(cleanPath) > 4096 {
		return false
	}

	if strings.HasPrefix(cleanPath, "~") {
		return false
	}

	return true
}

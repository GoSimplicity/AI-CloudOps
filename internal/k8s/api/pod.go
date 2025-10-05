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
	"path/filepath"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type K8sPodHandler struct {
	podService service.PodService
}

func NewK8sPodHandler(podService service.PodService) *K8sPodHandler {
	return &K8sPodHandler{
		podService: podService,
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

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.podService.GetPodDetails(ctx, &req)
	})
}

func (h *K8sPodHandler) GetPodList(ctx *gin.Context) {
	var req model.GetPodListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.podService.GetPodList(ctx, &req)
	})
}

func (h *K8sPodHandler) GetPodContainers(ctx *gin.Context) {
	var req model.GetPodContainersReq

	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, "参数绑定失败: "+err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.podService.GetPodContainers(ctx, &req)
	})
}

func (h *K8sPodHandler) GetPodLogs(ctx *gin.Context) {
	var req model.GetPodLogsReq

	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, "参数绑定失败: "+err.Error())
		return
	}

	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, "查询参数绑定失败: "+err.Error())
		return
	}

	// SSE (Server-Sent Events) 响应头配置
	// 用于实现日志的实时流式推送，避免轮询带来的性能问题
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("Access-Control-Allow-Headers", "Cache-Control")

	if err := h.podService.GetPodLogs(ctx, &req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
}

func (h *K8sPodHandler) GetPodYaml(ctx *gin.Context) {
	var req model.GetPodYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.podService.GetPodYaml(ctx, &req)
	})
}

func (h *K8sPodHandler) CreatePod(ctx *gin.Context) {
	var req model.CreatePodReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.podService.CreatePod(ctx, &req)
	})
}

func (h *K8sPodHandler) CreatePodByYaml(ctx *gin.Context) {
	var req model.CreatePodByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.podService.CreatePodByYaml(ctx, &req)
	})
}

func (h *K8sPodHandler) UpdatePod(ctx *gin.Context) {
	var req model.UpdatePodReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.podService.UpdatePod(ctx, &req)
	})
}

func (h *K8sPodHandler) UpdatePodByYaml(ctx *gin.Context) {
	var req model.UpdatePodByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.podService.UpdatePodByYaml(ctx, &req)
	})
}

func (h *K8sPodHandler) DeletePod(ctx *gin.Context) {
	var req model.DeletePodReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.podService.DeletePod(ctx, &req)
	})
}

func (h *K8sPodHandler) PodExec(ctx *gin.Context) {
	var req model.PodExecReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, "集群ID参数错误: "+err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, "命名空间参数错误: "+err.Error())
		return
	}

	podName, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, "Pod名称参数错误: "+err.Error())
		return
	}

	container, err := utils.GetParamCustomName(ctx, "container")
	if err != nil {
		utils.BadRequestError(ctx, "容器名称参数错误: "+err.Error())
		return
	}

	shell := ctx.DefaultQuery("shell", "sh")

	if clusterID <= 0 {
		utils.BadRequestError(ctx, "集群ID必须大于0")
		return
	}

	if namespace == "" {
		utils.BadRequestError(ctx, "命名空间不能为空")
		return
	}

	if podName == "" {
		utils.BadRequestError(ctx, "Pod名称不能为空")
		return
	}

	if container == "" {
		utils.BadRequestError(ctx, "容器名称不能为空")
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
		utils.BadRequestError(ctx, "不支持的shell类型: "+shell)
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.PodName = podName
	req.Container = container
	req.Shell = shell

	if err := h.podService.PodExec(ctx, &req); err != nil {
		utils.BadRequestError(ctx, "建立终端连接失败: "+err.Error())
		return
	}
}

func (h *K8sPodHandler) PodPortForward(ctx *gin.Context) {
	var req model.PodPortForwardReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	podName, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.PodName = podName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.podService.PodPortForward(ctx, &req)
	})
}

func (h *K8sPodHandler) PodFileUpload(ctx *gin.Context) {
	var req model.PodFileUploadReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, "集群ID参数错误: "+err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, "命名空间参数错误: "+err.Error())
		return
	}

	podName, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, "Pod名称参数错误: "+err.Error())
		return
	}

	container, err := utils.GetParamCustomName(ctx, "container")
	if err != nil {
		utils.BadRequestError(ctx, "容器名称参数错误: "+err.Error())
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
		utils.BadRequestError(ctx, "集群ID必须大于0")
		return
	}

	if namespace == "" {
		utils.BadRequestError(ctx, "命名空间不能为空")
		return
	}

	if podName == "" {
		utils.BadRequestError(ctx, "Pod名称不能为空")
		return
	}

	if container == "" {
		utils.BadRequestError(ctx, "容器名称不能为空")
		return
	}

	if !isValidPath(filePath) {
		utils.BadRequestError(ctx, "无效的文件路径格式")
		return
	}

	if ctx.Request.MultipartForm == nil {
		if err := ctx.Request.ParseMultipartForm(32 << 20); err != nil {
			utils.BadRequestError(ctx, "解析上传文件失败: "+err.Error())
			return
		}
	}

	if ctx.Request.MultipartForm == nil || len(ctx.Request.MultipartForm.File) == 0 {
		utils.BadRequestError(ctx, "未找到上传的文件")
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.PodName = podName
	req.ContainerName = container
	req.FilePath = filePath

	if err := h.podService.PodFileUpload(ctx, &req); err != nil {
		utils.BadRequestError(ctx, "文件上传失败: "+err.Error())
		return
	}

	utils.Success(ctx)
}

func (h *K8sPodHandler) PodFileDownload(ctx *gin.Context) {
	var req model.PodFileDownloadReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, "集群ID参数错误: "+err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, "命名空间参数错误: "+err.Error())
		return
	}

	podName, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, "Pod名称参数错误: "+err.Error())
		return
	}

	container, err := utils.GetParamCustomName(ctx, "container")
	if err != nil {
		utils.BadRequestError(ctx, "容器名称参数错误: "+err.Error())
		return
	}

	filePath := ctx.Query("file_path")

	if clusterID <= 0 {
		utils.BadRequestError(ctx, "集群ID必须大于0")
		return
	}

	if namespace == "" {
		utils.BadRequestError(ctx, "命名空间不能为空")
		return
	}

	if podName == "" {
		utils.BadRequestError(ctx, "Pod名称不能为空")
		return
	}

	if container == "" {
		utils.BadRequestError(ctx, "容器名称不能为空")
		return
	}

	if filePath == "" {
		utils.BadRequestError(ctx, "文件路径不能为空")
		return
	}

	if !isValidPath(filePath) {
		utils.BadRequestError(ctx, "无效的文件路径格式")
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.PodName = podName
	req.ContainerName = container
	req.FilePath = filePath

	if err := h.podService.PodFileDownload(ctx, &req); err != nil {
		utils.BadRequestError(ctx, "文件下载失败: "+err.Error())
		return
	}
}

// isValidPath 验证文件路径的安全性，防止路径遍历等安全攻击
// 关键安全检查：
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

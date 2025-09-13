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

func (k *K8sPodHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		// Pod相关接口
		k8sGroup.GET("/pod/:cluster_id/list", k.GetPodList)
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/detail", k.GetPodDetails)
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/detail/yaml", k.GetPodYaml)
		k8sGroup.POST("/pod/:cluster_id/create", k.CreatePod)
		k8sGroup.POST("/pod/:cluster_id/create/yaml", k.CreatePodByYaml)
		k8sGroup.PUT("/pod/:cluster_id/:namespace/:name/update", k.UpdatePod)
		k8sGroup.PUT("/pod/:cluster_id/:namespace/:name/update/yaml", k.UpdatePodByYaml)
		k8sGroup.DELETE("/pod/:cluster_id/:namespace/:name/delete", k.DeletePod)
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/containers", k.GetPodContainers)
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/containers/:container/logs", k.GetPodLogs)
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/containers/:container/exec", k.PodExec)
		k8sGroup.POST("/pod/:cluster_id/:namespace/:name/port-forward", k.PodPortForward)
		k8sGroup.POST("/pod/:cluster_id/:namespace/:name/containers/:container/files/upload", k.PodFileUpload)
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/containers/:container/files/download", k.PodFileDownload)
	}
}

// GetPodDetails 获取Pod详情
func (k *K8sPodHandler) GetPodDetails(ctx *gin.Context) {
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
		return k.podService.GetPodDetails(ctx, &req)
	})
}

// GetPodList 获取Pod列表
func (k *K8sPodHandler) GetPodList(ctx *gin.Context) {
	var req model.GetPodListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.podService.GetPodList(ctx, &req)
	})
}

// GetPodContainers 获取Pod容器列表
func (k *K8sPodHandler) GetPodContainers(ctx *gin.Context) {
	var req model.GetPodContainersReq

	// 绑定路径参数
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, "参数绑定失败: "+err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.podService.GetPodContainers(ctx, &req)
	})
}

// GetPodLogs 获取容器日志
func (k *K8sPodHandler) GetPodLogs(ctx *gin.Context) {
	var req model.GetPodLogsReq

	// 绑定路径参数
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, "参数绑定失败: "+err.Error())
		return
	}

	// 绑定查询参数
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, "查询参数绑定失败: "+err.Error())
		return
	}

	// 设置SSE响应头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("Access-Control-Allow-Headers", "Cache-Control")

	// 调用service层进行流式推送
	if err := k.podService.GetPodLogs(ctx, &req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
}

// GetPodYaml 获取Pod的YAML配置
func (k *K8sPodHandler) GetPodYaml(ctx *gin.Context) {
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
		return k.podService.GetPodYaml(ctx, &req)
	})
}

// CreatePod 创建Pod
func (k *K8sPodHandler) CreatePod(ctx *gin.Context) {
	var req model.CreatePodReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.podService.CreatePod(ctx, &req)
	})
}

// CreatePodByYaml 通过YAML创建Pod
func (k *K8sPodHandler) CreatePodByYaml(ctx *gin.Context) {
	var req model.CreatePodByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.podService.CreatePodByYaml(ctx, &req)
	})
}

// UpdatePod 更新Pod
func (k *K8sPodHandler) UpdatePod(ctx *gin.Context) {
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
		return nil, k.podService.UpdatePod(ctx, &req)
	})
}

// UpdatePodByYaml 通过YAML更新Pod
func (k *K8sPodHandler) UpdatePodByYaml(ctx *gin.Context) {
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
		return nil, k.podService.UpdatePodByYaml(ctx, &req)
	})
}

// DeletePod 删除Pod
func (k *K8sPodHandler) DeletePod(ctx *gin.Context) {
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
		return nil, k.podService.DeletePod(ctx, &req)
	})
}

// PodExec Pod终端连接
func (k *K8sPodHandler) PodExec(ctx *gin.Context) {
	var req model.PodExecReq

	// 获取路径参数
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

	// 获取shell类型
	shell := ctx.DefaultQuery("shell", "sh")

	// 参数验证
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

	// 验证shell类型
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

	// 建立WebSocket连接
	if err := k.podService.PodExec(ctx, &req); err != nil {
		utils.BadRequestError(ctx, "建立终端连接失败: "+err.Error())
		return
	}
}

// PodPortForward Pod端口转发
func (k *K8sPodHandler) PodPortForward(ctx *gin.Context) {
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
		return nil, k.podService.PodPortForward(ctx, &req)
	})
}

// PodFileUpload 上传文件到Pod
func (k *K8sPodHandler) PodFileUpload(ctx *gin.Context) {
	var req model.PodFileUploadReq

	// 获取路径参数
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

	// 获取目标路径
	filePath := ctx.DefaultPostForm("file_path", "/tmp")

	// 参数验证
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
		utils.BadRequestError(ctx, "上传路径不能为空")
		return
	}

	// 验证路径格式
	if !isValidPath(filePath) {
		utils.BadRequestError(ctx, "无效的文件路径格式")
		return
	}

	// 检查是否有文件上传
	if ctx.Request.MultipartForm == nil {
		if err := ctx.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max
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

	// 调用文件上传服务
	if err := k.podService.PodFileUpload(ctx, &req); err != nil {
		utils.BadRequestError(ctx, "文件上传失败: "+err.Error())
		return
	}

	utils.Success(ctx)
}

// PodFileDownload 从Pod下载文件
func (k *K8sPodHandler) PodFileDownload(ctx *gin.Context) {
	var req model.PodFileDownloadReq

	// 获取路径参数
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

	// 获取文件路径
	filePath := ctx.Query("file_path")

	// 参数验证
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

	// 验证路径格式
	if !isValidPath(filePath) {
		utils.BadRequestError(ctx, "无效的文件路径格式")
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.PodName = podName
	req.ContainerName = container
	req.FilePath = filePath

	// 调用文件下载服务
	if err := k.podService.PodFileDownload(ctx, &req); err != nil {
		utils.BadRequestError(ctx, "文件下载失败: "+err.Error())
		return
	}
}

// isValidPath 验证文件路径安全性
func isValidPath(path string) bool {
	if path == "" {
		return false
	}

	// 清理路径
	cleanPath := filepath.Clean(path)

	// 只检查最关键的安全问题
	dangerousPatterns := []string{
		"..",   // 路径遍历
		"\n",   // 换行符
		"\r",   // 回车符
		"\x00", // 空字节
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(cleanPath, pattern) {
			return false
		}
	}

	// 长度限制
	if len(cleanPath) > 4096 {
		return false
	}

	// 拒绝以某些特殊字符开头的路径
	if strings.HasPrefix(cleanPath, "~") {
		return false
	}

	return true
}

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
		// Pod基础管理
		k8sGroup.GET("/pod/:cluster_id/list", k.GetPodList)                                                       // 获取Pod列表
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/detail", k.GetPodDetails)                                 // 获取Pod详情
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/detail/yaml", k.GetPodYaml)                               // 获取Pod YAML
		k8sGroup.POST("/pod/:cluster_id/create", k.CreatePod)                                                     // 创建Pod
		k8sGroup.POST("/pod/:cluster_id/create/yaml", k.CreatePodByYaml)                                          // 通过YAML创建Pod
		k8sGroup.PUT("/pod/:cluster_id/:namespace/:name/update", k.UpdatePod)                                     // 更新Pod
		k8sGroup.PUT("/pod/:cluster_id/:namespace/:name/update/yaml", k.UpdatePodByYaml)                          // 通过YAML更新Pod
		k8sGroup.DELETE("/pod/:cluster_id/:namespace/:name/delete", k.DeletePod)                                  // 删除Pod
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/containers", k.GetPodContainers)                          // 获取Pod容器列表
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/containers/:container/logs", k.GetPodLogs)                // 获取容器日志
		k8sGroup.POST("/pod/:cluster_id/:namespace/:name/containers/:container/exec", k.PodExec)                  // Pod执行命令
		k8sGroup.POST("/pod/:cluster_id/:namespace/:name/port-forward", k.PodPortForward)                         // Pod端口转发
		k8sGroup.POST("/pod/:cluster_id/:namespace/:name/containers/:container/files/upload", k.PodFileUpload)    // Pod文件上传
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/containers/:container/files/download", k.PodFileDownload) // Pod文件下载
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

// GetPodContainers 获取Pod的容器列表
func (k *K8sPodHandler) GetPodContainers(ctx *gin.Context) {
	var req model.GetPodContainersReq

	// 绑定URL路径参数
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, "URL参数绑定失败: "+err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.podService.GetPodContainers(ctx, &req)
	})
}

// GetPodLogs 获取容器日志 - SSE流式推送
func (k *K8sPodHandler) GetPodLogs(ctx *gin.Context) {
	var req model.GetPodLogsReq

	// 绑定URL路径参数
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, "URL参数绑定失败: "+err.Error())
		return
	}

	// 绑定查询参数（日志选项）
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

	// 直接调用service层进行SSE流式推送
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

// PodExec Pod执行命令 - WebSocket连接
func (k *K8sPodHandler) PodExec(ctx *gin.Context) {
	var req model.PodExecReq

	// 从URL路径参数获取基本信息
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

	// 从查询参数获取shell类型（可选）
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

	// 对于WebSocket连接，直接调用service，不使用HandleRequest
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

// PodFileUpload Pod文件上传
func (k *K8sPodHandler) PodFileUpload(ctx *gin.Context) {
	var req model.PodFileUploadReq

	// 从URL路径参数获取基本信息
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

	// 从表单参数获取目标路径
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

	// 对于文件上传，直接调用service，不使用HandleRequest包装
	if err := k.podService.PodFileUpload(ctx, &req); err != nil {
		utils.BadRequestError(ctx, "文件上传失败: "+err.Error())
		return
	}

	utils.Success(ctx)
}

// PodFileDownload Pod文件下载
func (k *K8sPodHandler) PodFileDownload(ctx *gin.Context) {
	var req model.PodFileDownloadReq

	// 从URL路径参数获取基本信息
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

	// 从查询参数获取文件路径
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

	// 对于文件下载，直接调用service，不使用HandleRequest包装
	if err := k.podService.PodFileDownload(ctx, &req); err != nil {
		utils.BadRequestError(ctx, "文件下载失败: "+err.Error())
		return
	}
}

// isValidPath 验证文件路径是否安全和有效
func isValidPath(path string) bool {
	if path == "" {
		return false
	}

	// 清理路径
	cleanPath := filepath.Clean(path)

	// 检查危险的路径模式
	dangerousPatterns := []string{
		"..",   // 路径遍历
		"~",    // 用户主目录
		"$",    // 环境变量
		"`",    // 命令替换
		";",    // 命令分隔符
		"|",    // 管道
		"&",    // 后台运行
		">",    // 重定向
		"<",    // 重定向
		"*",    // 通配符
		"?",    // 通配符
		"[",    // 通配符
		"]",    // 通配符
		"{",    // 花括号展开
		"}",    // 花括号展开
		"\n",   // 换行符
		"\r",   // 回车符
		"\t",   // 制表符
		"\x00", // 空字节
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(cleanPath, pattern) {
			return false
		}
	}

	// 检查是否是绝对路径或相对路径
	if !filepath.IsAbs(cleanPath) && !strings.HasPrefix(cleanPath, "./") && !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}

	// 路径长度限制
	if len(cleanPath) > 4096 {
		return false
	}

	return true
}

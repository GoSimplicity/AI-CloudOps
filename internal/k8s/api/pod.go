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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sPodHandler struct {
	logger     *zap.Logger
	podService service.PodService
}

func NewK8sPodHandler(logger *zap.Logger, podService service.PodService) *K8sPodHandler {
	return &K8sPodHandler{
		logger:     logger,
		podService: podService,
	}
}

func (k *K8sPodHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	pods := k8sGroup.Group("/pods")
	{
		pods.GET("/:id", k.GetPodListByNamespace)                     // 根据命名空间获取 Pods 列表
		pods.GET("/:id/:podName/containers", k.GetPodContainers)      // 获取指定 Pod 的容器列表
		pods.GET("/:id/:podName/:container/logs", k.GetContainerLogs) // 获取指定容器的日志
		pods.GET("/:id/:podName/yaml", k.GetPodYaml)                  // 获取指定 Pod 的 YAML 配置
		pods.DELETE("/delete/:id", k.DeletePod)                       // 删除指定名称的 Pod
	}
}

// GetPodListByNamespace 根据命名空间获取Pod列表
// @Summary 根据命名空间获取Pod列表
// @Description 根据指定的命名空间获取K8s集群中的Pod列表
// @Tags Pod管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=[]object} "成功获取Pod列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/pods/{id} [get]
func (k *K8sPodHandler) GetPodListByNamespace(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.podService.GetPodsByNamespace(ctx, id, namespace)
	})
}

// GetPodContainers 获取Pod的容器列表
// @Summary 获取Pod的容器列表
// @Description 获取指定Pod的所有容器信息
// @Tags Pod管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param podName path string true "Pod名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=[]object} "成功获取容器列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/pods/{id}/{podName}/containers [get]
func (k *K8sPodHandler) GetPodContainers(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	podName := ctx.Param("podName")
	if podName == "" {
		utils.BadRequestError(ctx, "缺少 'podName' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.podService.GetContainersByPod(ctx, id, namespace, podName)
	})
}

// GetPodsListByNodeName 根据节点名获取Pod列表
// @Summary 根据节点名获取Pod列表
// @Description 获取指定节点上运行的所有Pod列表
// @Tags Pod管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param name query string true "节点名称"
// @Success 200 {object} utils.ApiResponse{data=[]object} "成功获取Pod列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/pods/{id}/node [get]
func (k *K8sPodHandler) GetPodsListByNodeName(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name := ctx.Query("name")
	if name == "" {
		utils.BadRequestError(ctx, "缺少 'name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.podService.GetPodsByNodeName(ctx, id, name)
	})
}

// GetContainerLogs 获取容器日志
// @Summary 获取容器日志
// @Description 获取指定Pod中容器的运行日志
// @Tags Pod管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param podName path string true "Pod名称"
// @Param container path string true "容器名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=string} "成功获取容器日志"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/pods/{id}/{podName}/{container}/logs [get]
func (k *K8sPodHandler) GetContainerLogs(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	podName := ctx.Param("podName")
	if podName == "" {
		utils.BadRequestError(ctx, "缺少 'podName' 参数")
		return
	}

	containerName := ctx.Param("container")
	if containerName == "" {
		utils.BadRequestError(ctx, "缺少 'container' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.podService.GetContainerLogs(ctx, id, namespace, podName, containerName)
	})
}

// GetPodYaml 获取Pod的YAML配置
// @Summary 获取Pod的YAML配置
// @Description 获取指定Pod的完整YAML配置文件
// @Tags Pod管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param podName path string true "Pod名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=string} "成功获取YAML配置"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/pods/{id}/{podName}/yaml [get]
func (k *K8sPodHandler) GetPodYaml(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	podName := ctx.Param("podName")
	if podName == "" {
		utils.BadRequestError(ctx, "缺少 'podName' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.podService.GetPodYaml(ctx, id, namespace, podName)
	})
}

// DeletePod 删除Pod
// @Summary 删除Pod
// @Description 删除指定命名空间中的Pod
// @Tags Pod管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param namespace query string true "命名空间"
// @Param podName query string true "Pod名称"
// @Success 200 {object} utils.ApiResponse "成功删除Pod"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/pods/delete/{id} [delete]
func (k *K8sPodHandler) DeletePod(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace := ctx.Query("namespace")
	podName := ctx.Query("podName")

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.podService.DeletePod(ctx, id, namespace, podName)
	})
}

package api

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

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sPodHandler struct {
	logger     *zap.Logger
	podService admin.PodService
}

func NewK8sPodHandler(logger *zap.Logger, podService admin.PodService) *K8sPodHandler {
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

// GetPodListByNamespace 根据命名空间获取 Pods 列表
func (k *K8sPodHandler) GetPodListByNamespace(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.podService.GetPodsByNamespace(ctx, id, namespace)
	})
}

// GetPodContainers 获取 Pod 的容器列表
func (k *K8sPodHandler) GetPodContainers(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	podName := ctx.Param("podName")
	if podName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'podName' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.podService.GetContainersByPod(ctx, id, namespace, podName)
	})
}

// GetPodsListByNodeName 获取指定节点上的 Pods 列表
func (k *K8sPodHandler) GetPodsListByNodeName(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	name := ctx.Query("name")
	if name == "" {
		apiresponse.BadRequestError(ctx, "缺少 'name' 参数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.podService.GetPodsByNodeName(ctx, id, name)
	})
}

// GetContainerLogs 获取容器日志
func (k *K8sPodHandler) GetContainerLogs(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	podName := ctx.Param("podName")
	if podName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'podName' 参数")
		return
	}

	containerName := ctx.Param("container")
	if containerName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'container' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.podService.GetContainerLogs(ctx, id, namespace, podName, containerName)
	})
}

// GetPodYaml 获取 Pod 的 YAML 配置
func (k *K8sPodHandler) GetPodYaml(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	podName := ctx.Param("podName")
	if podName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'podName' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.podService.GetPodYaml(ctx, id, namespace, podName)
	})
}

// DeletePod 删除指定名称的 Pod
func (k *K8sPodHandler) DeletePod(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	namespace := ctx.Query("namespace")
	podName := ctx.Query("podName")

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.podService.DeletePod(ctx, id, namespace, podName)
	})
}

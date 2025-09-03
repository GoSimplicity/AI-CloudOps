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

// import (
// 	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
// 	"github.com/GoSimplicity/AI-CloudOps/internal/model"
// 	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
// 	"github.com/gin-gonic/gin"
// )

// type K8sPodHandler struct {
// 	podService service.PodService
// }

// func NewK8sPodHandler(podService service.PodService) *K8sPodHandler {
// 	return &K8sPodHandler{
// 		podService: podService,
// 	}
// }

// func (k *K8sPodHandler) RegisterRouters(server *gin.Engine) {
// 	k8sGroup := server.Group("/api/k8s")
// 	{
// 		k8sGroup.GET("/pods/:id", k.GetPodListByNamespace)
// 		k8sGroup.GET("/pods/:id/node", k.GetPodsListByNodeName)
// 		k8sGroup.GET("/pods/:id/:podName/containers", k.GetPodContainers)
// 		k8sGroup.GET("/pods/:id/:podName/:container/logs", k.GetContainerLogs)
// 		k8sGroup.GET("/pods/:id/:podName/yaml", k.GetPodYaml)
// 		k8sGroup.DELETE("/pods/delete/:id", k.DeletePod)
// 	}
// }

// // GetPodListByNamespace 获取pod列表
// func (k *K8sPodHandler) GetPodListByNamespace(ctx *gin.Context) {
// 	var req model.K8sGetResourceListReq
// 	if err := ctx.ShouldBindUri(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}
// 	if err := ctx.ShouldBindQuery(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}

// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return k.podService.GetPodsByNamespace(ctx, req.ClusterID, req.Namespace)
// 	})
// }

// // GetPodContainers 获取pod容器列表
// func (k *K8sPodHandler) GetPodContainers(ctx *gin.Context) {
// 	var req model.PodContainersReq
// 	if err := ctx.ShouldBindUri(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}
// 	if err := ctx.ShouldBindQuery(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}

// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return k.podService.GetContainersByPod(ctx, req.ClusterID, req.Namespace, req.PodName)
// 	})
// }

// // GetPodsListByNodeName 获取节点pod列表
// func (k *K8sPodHandler) GetPodsListByNodeName(ctx *gin.Context) {
// 	var req model.PodsByNodeReq
// 	if err := ctx.ShouldBindUri(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}
// 	if err := ctx.ShouldBindQuery(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}

// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return k.podService.GetPodsByNodeName(ctx, req.ClusterID, req.NodeName)
// 	})
// }

// // GetContainerLogs 获取容器日志
// func (k *K8sPodHandler) GetContainerLogs(ctx *gin.Context) {
// 	var req model.PodLogReq
// 	if err := ctx.ShouldBindUri(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}
// 	if err := ctx.ShouldBindQuery(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}

// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return k.podService.GetContainerLogs(ctx, req.ClusterID, req.Namespace, req.ResourceName, req.Container)
// 	})
// }

// // GetPodYaml 获取pod的YAML
// func (k *K8sPodHandler) GetPodYaml(ctx *gin.Context) {
// 	var req model.K8sGetResourceYamlReq
// 	if err := ctx.ShouldBindUri(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}
// 	if err := ctx.ShouldBindQuery(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}

// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return k.podService.GetPodYaml(ctx, req.ClusterID, req.Namespace, req.ResourceName)
// 	})
// }

// // DeletePod 删除pod
// func (k *K8sPodHandler) DeletePod(ctx *gin.Context) {
// 	var req model.K8sDeleteResourceReq
// 	if err := ctx.ShouldBindUri(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}
// 	if err := ctx.ShouldBindQuery(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}

// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return nil, k.podService.DeletePod(ctx, req.ClusterID, req.Namespace, req.ResourceName)
// 	})
// }

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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils/query"
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
		k8sGroup.GET("/pods/:cluster_id", k.GetPodList)
		k8sGroup.GET("/pods/:cluster_id/:node_name", k.GetPodsListByNodeName)
		k8sGroup.GET("/pods/:cluster_id/:namespace/:pod_name", k.GetPodDetail) // 获取单个Ingress详情
		k8sGroup.GET("/pods/:cluster_id/:namespace/:pod_name/containers", k.GetPodContainers)
		k8sGroup.GET("/pods/:cluster_id/:namespace/:pod_name/:container/logs", k.GetContainerLogs)
		k8sGroup.GET("/pods/:cluster_id/:namespace/:pod_name/yaml", k.GetPodYaml)
		k8sGroup.DELETE("/pods/delete/:cluster_id/:namespace/:pod_name", k.DeletePod)
		k8sGroup.GET("/pods/:cluster_id/:namespace/:pod_name/:container/download_file", k.DownloadPodFile)
		k8sGroup.POST("/pods/:cluster_id/:namespace/:pod_name/:container/upload_file", k.UploadFileToPod)
		k8sGroup.POST("/pods/:cluster_id/:namespace/:pod_name/port_forward", k.PortForward)
		k8sGroup.POST("/pods/:cluster_id/:namespace/:pod_name/:container/exec", k.HandlePodTerminalSession)
	}
}

// GetPodDetail 获取Pod列表
func (k *K8sPodHandler) GetPodDetail(ctx *gin.Context) {
	var req = new(model.K8sGetPodReq)

	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		return k.podService.GetPod(ctx, req)
	})
}

// GetPodList 获取Pod列表
func (k *K8sPodHandler) GetPodList(ctx *gin.Context) {
	var req = new(model.GetPodListReq)

	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		// 分页相关参数
		queryParams := query.ParseQueryWithParameters(ctx)

		return k.podService.GetPodList(ctx, queryParams, req)
	})
}

// GetPodContainers 获取Pod的容器列表
func (k *K8sPodHandler) GetPodContainers(ctx *gin.Context) {
	var req = new(model.PodContainersReq)

	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		return k.podService.GetContainersByPod(ctx, req)
	})
}

// GetPodsListByNodeName 根据节点名获取Pod列表
func (k *K8sPodHandler) GetPodsListByNodeName(ctx *gin.Context) {
	var req = new(model.PodsByNodeReq)

	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}
	//if err := ctx.ShouldBindUri(&req); err != nil {
	//	utils.BadRequestError(ctx, err.Error())
	//	return
	//}
	//if err := ctx.ShouldBindQuery(&req); err != nil {
	//	utils.BadRequestError(ctx, err.Error())
	//	return
	//}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		return k.podService.GetPodsByNodeName(ctx, req)
	})
}

// GetContainerLogs 获取容器日志
func (k *K8sPodHandler) GetContainerLogs(ctx *gin.Context) {
	var req = new(model.PodLogReq)
	//if err := ctx.ShouldBindUri(&req); err != nil {
	//	utils.BadRequestError(ctx, err.Error())
	//	return
	//}
	//if err := ctx.ShouldBindQuery(&req); err != nil {
	//	utils.BadRequestError(ctx, err.Error())
	//	return
	//}

	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {

		return nil, k.podService.GetPodLogs(ctx, req)
	})
}

// GetPodYaml 获取Pod的YAML配置
func (k *K8sPodHandler) GetPodYaml(ctx *gin.Context) {
	var req = new(model.K8sGetPodReq)

	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	//if err := ctx.ShouldBindQuery(&req); err != nil {
	//	utils.BadRequestError(ctx, err.Error())
	//	return
	//}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		return k.podService.GetPodYaml(ctx, req)
	})
}

// DeletePod 删除Pod
func (k *K8sPodHandler) DeletePod(ctx *gin.Context) {
	var req = new(model.K8sDeletePodReq)

	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	//if err := ctx.ShouldBindQuery(&req); err != nil {
	//	utils.BadRequestError(ctx, err.Error())
	//	return
	//}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		return nil, k.podService.DeletePodWithOptions(ctx, req)
	})
}

// DownloadPodFile 下载pod内文件
func (k *K8sPodHandler) DownloadPodFile(ctx *gin.Context) {
	var req = new(model.PodFileReq)

	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	//if err := ctx.ShouldBindQuery(&req); err != nil {
	//	utils.BadRequestError(ctx, err.Error())
	//	return
	//}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		return nil, k.podService.DownloadPodFile(ctx, req)
	})
}

// UploadFileToPod 上传文件至pod
func (k *K8sPodHandler) UploadFileToPod(ctx *gin.Context) {
	var req = new(model.PodFileReq)

	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	//if err := ctx.ShouldBindQuery(&req); err != nil {
	//	utils.BadRequestError(ctx, err.Error())
	//	return
	//}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		return nil, k.podService.UploadFileToPod(ctx, req)
	})
}

// PortForward Pod端口转发
func (k *K8sPodHandler) PortForward(ctx *gin.Context) {
	var req = new(model.PodPortForwardReq)

	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	//if err := ctx.ShouldBindBodyWithJSON(&req); err != nil {
	//	utils.BadRequestError(ctx, err.Error())
	//	return
	//}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		return nil, k.podService.PortForward(ctx, req)
	})
}

// HandlePodTerminalSession Pod shell终端
func (k *K8sPodHandler) HandlePodTerminalSession(ctx *gin.Context) {
	var req = new(model.PodExecReq)

	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	//
	//if err := ctx.ShouldBindQuery(&req); err != nil {
	//	utils.BadRequestError(ctx, err.Error())
	//	return
	//}

	utils.HandleRequest(ctx, req, func() (interface{}, error) {
		return nil, k.podService.ExecInPod(ctx, req)
	})
}

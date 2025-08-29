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
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sDaemonSetHandler struct {
	logger           *zap.Logger
	daemonSetService service.DaemonSetService
}

func NewK8sDaemonSetHandler(logger *zap.Logger, daemonSetService service.DaemonSetService) *K8sDaemonSetHandler {
	return &K8sDaemonSetHandler{
		logger:           logger,
		daemonSetService: daemonSetService,
	}
}

func (k *K8sDaemonSetHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	daemonSets := k8sGroup.Group("/daemonsets")
	{
		// 基础操作
		daemonSets.GET("/list", k.GetDaemonSetList)                   // 获取DaemonSet列表
		daemonSets.GET("/:cluster_id", k.GetDaemonSetsByNamespace)    // 根据命名空间获取DaemonSet列表
		daemonSets.GET("/:cluster_id/:name", k.GetDaemonSet)          // 获取单个DaemonSet详情
		daemonSets.GET("/:cluster_id/:name/yaml", k.GetDaemonSetYaml) // 获取DaemonSet YAML配置
		daemonSets.POST("/create", k.CreateDaemonSet)                 // 创建DaemonSet
		daemonSets.PUT("/update", k.UpdateDaemonSet)                  // 更新DaemonSet
		daemonSets.DELETE("/delete", k.DeleteDaemonSet)               // 删除DaemonSet
		daemonSets.POST("/restart", k.RestartDaemonSet)               // 重启DaemonSet

		// 批量操作

		// 高级功能
		daemonSets.GET("/:cluster_id/:name/history", k.GetDaemonSetHistory)    // 获取DaemonSet历史版本
		daemonSets.GET("/:cluster_id/:name/events", k.GetDaemonSetEvents)      // 获取DaemonSet事件
		daemonSets.GET("/:cluster_id/:name/node-pods", k.GetDaemonSetNodePods) // 获取指定节点的DaemonSet Pod
	}
}

// GetDaemonSetList 获取DaemonSet列表
// @Summary 获取DaemonSet列表
// @Description 根据查询条件获取K8s集群中的DaemonSet列表
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param request query model.K8sDaemonSetListReq true "DaemonSet列表查询请求"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sDaemonSetEntity} "成功获取DaemonSet列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/daemonsets/list [get]
func (k *K8sDaemonSetHandler) GetDaemonSetList(ctx *gin.Context) {
	var req model.K8sDaemonSetListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSetList(ctx, &req)
	})
}

// GetDaemonSetsByNamespace 根据命名空间获取DaemonSet列表
// @Summary 根据命名空间获取DaemonSet列表
// @Description 根据指定的命名空间获取K8s集群中的DaemonSet列表
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace query string false "命名空间，为空则获取所有命名空间"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sDaemonSetEntity} "成功获取DaemonSet列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/daemonsets/{cluster_id} [get]
func (k *K8sDaemonSetHandler) GetDaemonSetsByNamespace(ctx *gin.Context) {
	var req model.K8sGetResourceListReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSetsByNamespace(ctx, req.ClusterID, req.Namespace)
	})
}

// GetDaemonSet 获取DaemonSet详情
// @Summary 获取DaemonSet详情
// @Description 获取指定DaemonSet的详细信息
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "DaemonSet名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=model.K8sDaemonSetEntity} "成功获取DaemonSet详情"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/daemonsets/{cluster_id}/{name} [get]
func (k *K8sDaemonSetHandler) GetDaemonSet(ctx *gin.Context) {
	var req model.K8sGetResourceReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSet(ctx, req.ClusterID, req.Namespace, req.ResourceName)
	})
}

// GetDaemonSetYaml 获取DaemonSet的YAML配置
// @Summary 获取DaemonSet的YAML配置
// @Description 获取指定DaemonSet的完整YAML配置文件
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "DaemonSet名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=string} "成功获取YAML配置"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/daemonsets/{cluster_id}/{name}/yaml [get]
func (k *K8sDaemonSetHandler) GetDaemonSetYaml(ctx *gin.Context) {
	var req model.K8sGetResourceYamlReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSetYaml(ctx, req.ClusterID, req.Namespace, req.ResourceName)
	})
}

// CreateDaemonSet 创建DaemonSet
// @Summary 创建DaemonSet
// @Description 创建新的DaemonSet资源
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param request body model.K8sDaemonSetCreateReq true "DaemonSet创建请求"
// @Success 200 {object} utils.ApiResponse "成功创建DaemonSet"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/daemonsets/create [post]
func (k *K8sDaemonSetHandler) CreateDaemonSet(ctx *gin.Context) {
	var req model.K8sDaemonSetCreateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.daemonSetService.CreateDaemonSet(ctx, &req)
	})
}

// UpdateDaemonSet 更新DaemonSet
// @Summary 更新DaemonSet
// @Description 更新指定的DaemonSet资源配置
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param request body model.K8sDaemonSetUpdateReq true "DaemonSet更新请求"
// @Success 200 {object} utils.ApiResponse "成功更新DaemonSet"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/daemonsets/update [put]
func (k *K8sDaemonSetHandler) UpdateDaemonSet(ctx *gin.Context) {
	var req model.K8sDaemonSetUpdateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.daemonSetService.UpdateDaemonSet(ctx, &req)
	})
}

// DeleteDaemonSet 删除DaemonSet
// @Summary 删除DaemonSet
// @Description 删除指定的DaemonSet资源
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param request body model.K8sDaemonSetDeleteReq true "DaemonSet删除请求"
// @Success 200 {object} utils.ApiResponse "成功删除DaemonSet"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/daemonsets/delete [delete]
func (k *K8sDaemonSetHandler) DeleteDaemonSet(ctx *gin.Context) {
	var req model.K8sDaemonSetDeleteReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.daemonSetService.DeleteDaemonSet(ctx, &req)
	})
}

// RestartDaemonSet 重启DaemonSet
// @Summary 重启DaemonSet
// @Description 重启指定的DaemonSet资源
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param request body model.K8sDaemonSetRestartReq true "DaemonSet重启请求"
// @Success 200 {object} utils.ApiResponse "成功重启DaemonSet"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/daemonsets/restart [post]
func (k *K8sDaemonSetHandler) RestartDaemonSet(ctx *gin.Context) {
	var req model.K8sDaemonSetRestartReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.daemonSetService.RestartDaemonSet(ctx, &req)
	})
}

// GetDaemonSetHistory 获取DaemonSet历史版本
// @Summary 获取DaemonSet历史版本
// @Description 获取指定DaemonSet的历史版本信息
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "DaemonSet名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sResourceHistory} "成功获取历史版本"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/daemonsets/{cluster_id}/{name}/history [get]
func (k *K8sDaemonSetHandler) GetDaemonSetHistory(ctx *gin.Context) {
	var req model.K8sDaemonSetHistoryReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSetHistory(ctx, &req)
	})
}

// GetDaemonSetEvents 获取DaemonSet事件
// @Summary 获取DaemonSet事件
// @Description 获取指定DaemonSet相关的事件信息
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "DaemonSet名称"
// @Param namespace query string true "命名空间"
// @Param limit_days query int false "限制天数内的事件"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sEvent} "成功获取事件"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/daemonsets/{cluster_id}/{name}/events [get]
func (k *K8sDaemonSetHandler) GetDaemonSetEvents(ctx *gin.Context) {
	var req model.K8sDaemonSetEventReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSetEvents(ctx, &req)
	})
}

// GetDaemonSetNodePods 获取DaemonSet在指定节点的Pod
// @Summary 获取DaemonSet在指定节点的Pod
// @Description 获取指定DaemonSet在指定节点上运行的Pod列表
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "DaemonSet名称"
// @Param namespace query string true "命名空间"
// @Param node_name query string true "节点名称"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sPod} "成功获取Pod列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/daemonsets/{cluster_id}/{name}/node-pods [get]
func (k *K8sDaemonSetHandler) GetDaemonSetNodePods(ctx *gin.Context) {
	var req model.K8sDaemonSetNodePodsReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSetNodePods(ctx, &req)
	})
}

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

type K8sEventHandler struct {
	logger       *zap.Logger
	eventService service.EventService
}

func NewK8sEventHandler(logger *zap.Logger, eventService service.EventService) *K8sEventHandler {
	return &K8sEventHandler{
		logger:       logger,
		eventService: eventService,
	}
}

func (k *K8sEventHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	events := k8sGroup.Group("/events")
	{
		// 基础操作
		events.GET("/list", k.GetEventList)                // 获取Event列表
		events.GET("/:cluster_id", k.GetEventsByNamespace) // 根据命名空间获取Event列表
		events.GET("/:cluster_id/:name", k.GetEvent)       // 获取单个Event详情

		// 根据对象获取事件
		events.GET("/by-object", k.GetEventsByObject)         // 根据对象获取相关事件
		events.GET("/by-pod", k.GetEventsByPod)               // 获取Pod相关事件
		events.GET("/by-deployment", k.GetEventsByDeployment) // 获取Deployment相关事件
		events.GET("/by-service", k.GetEventsByService)       // 获取Service相关事件
		events.GET("/by-node", k.GetEventsByNode)             // 获取Node相关事件

		// 事件分析
		events.GET("/statistics", k.GetEventStatistics) // 获取事件统计
		events.GET("/timeline", k.GetEventTimeline)     // 获取事件时间线

		// 事件管理
		events.POST("/cleanup", k.CleanupOldEvents) // 清理旧事件
	}
}

// GetEventList 获取Event列表
// @Summary 获取Event列表
// @Description 根据查询条件获取K8s集群中的Event列表
// @Tags Event管理
// @Accept json
// @Produce json
// @Param request query model.K8sEventListReq true "Event列表查询请求"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sEventEntity} "成功获取Event列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/events/list [get]
func (k *K8sEventHandler) GetEventList(ctx *gin.Context) {
	var req model.K8sEventListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.eventService.GetEventList(ctx, &req)
	})
}

// GetEventsByNamespace 根据命名空间获取Event列表
// @Summary 根据命名空间获取Event列表
// @Description 根据指定的命名空间获取K8s集群中的Event列表
// @Tags Event管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace query string false "命名空间，为空则获取所有命名空间"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sEventEntity} "成功获取Event列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/events/{cluster_id} [get]
func (k *K8sEventHandler) GetEventsByNamespace(ctx *gin.Context) {
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
		return k.eventService.GetEventsByNamespace(ctx, req.ClusterID, req.Namespace)
	})
}

// GetEvent 获取Event详情
// @Summary 获取Event详情
// @Description 获取指定Event的详细信息
// @Tags Event管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "Event名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=model.K8sEventEntity} "成功获取Event详情"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/events/{cluster_id}/{name} [get]
func (k *K8sEventHandler) GetEvent(ctx *gin.Context) {
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
		return k.eventService.GetEvent(ctx, req.ClusterID, req.Namespace, req.ResourceName)
	})
}

// GetEventsByObject 根据对象获取相关事件
// @Summary 根据对象获取相关事件
// @Description 根据指定的Kubernetes对象获取相关的事件列表
// @Tags Event管理
// @Accept json
// @Produce json
// @Param request query model.K8sEventByObjectReq true "对象事件查询请求"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sEventEntity} "成功获取事件列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/events/by-object [get]
func (k *K8sEventHandler) GetEventsByObject(ctx *gin.Context) {
	var req model.K8sEventByObjectReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.eventService.GetEventsByObject(ctx, &req)
	})
}

// GetEventsByPod 获取Pod相关事件
// @Summary 获取Pod相关事件
// @Description 获取指定Pod的相关事件列表
// @Tags Event管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string true "命名空间"
// @Param pod_name query string true "Pod名称"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sEventEntity} "成功获取事件列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/events/by-pod [get]
func (k *K8sEventHandler) GetEventsByPod(ctx *gin.Context) {
	var req struct {
		ClusterID int    `form:"cluster_id" binding:"required"`
		Namespace string `form:"namespace" binding:"required"`
		PodName   string `form:"pod_name" binding:"required"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.eventService.GetEventsByPod(ctx, req.ClusterID, req.Namespace, req.PodName)
	})
}

// GetEventsByDeployment 获取Deployment相关事件
// @Summary 获取Deployment相关事件
// @Description 获取指定Deployment的相关事件列表
// @Tags Event管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string true "命名空间"
// @Param deployment_name query string true "Deployment名称"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sEventEntity} "成功获取事件列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/events/by-deployment [get]
func (k *K8sEventHandler) GetEventsByDeployment(ctx *gin.Context) {
	var req struct {
		ClusterID      int    `form:"cluster_id" binding:"required"`
		Namespace      string `form:"namespace" binding:"required"`
		DeploymentName string `form:"deployment_name" binding:"required"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.eventService.GetEventsByDeployment(ctx, req.ClusterID, req.Namespace, req.DeploymentName)
	})
}

// GetEventsByService 获取Service相关事件
// @Summary 获取Service相关事件
// @Description 获取指定Service的相关事件列表
// @Tags Event管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string true "命名空间"
// @Param service_name query string true "Service名称"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sEventEntity} "成功获取事件列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/events/by-service [get]
func (k *K8sEventHandler) GetEventsByService(ctx *gin.Context) {
	var req struct {
		ClusterID   int    `form:"cluster_id" binding:"required"`
		Namespace   string `form:"namespace" binding:"required"`
		ServiceName string `form:"service_name" binding:"required"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.eventService.GetEventsByService(ctx, req.ClusterID, req.Namespace, req.ServiceName)
	})
}

// GetEventsByNode 获取Node相关事件
// @Summary 获取Node相关事件
// @Description 获取指定Node的相关事件列表
// @Tags Event管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param node_name query string true "Node名称"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sEventEntity} "成功获取事件列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/events/by-node [get]
func (k *K8sEventHandler) GetEventsByNode(ctx *gin.Context) {
	var req struct {
		ClusterID int    `form:"cluster_id" binding:"required"`
		NodeName  string `form:"node_name" binding:"required"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.eventService.GetEventsByNode(ctx, req.ClusterID, req.NodeName)
	})
}

// GetEventStatistics 获取事件统计
// @Summary 获取事件统计
// @Description 获取指定条件下的事件统计信息
// @Tags Event管理
// @Accept json
// @Produce json
// @Param request query model.K8sEventStatisticsReq true "事件统计请求"
// @Success 200 {object} utils.ApiResponse{data=model.K8sEventStatistics} "成功获取事件统计"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/events/statistics [get]
func (k *K8sEventHandler) GetEventStatistics(ctx *gin.Context) {
	var req model.K8sEventStatisticsReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.eventService.GetEventStatistics(ctx, &req)
	})
}

// GetEventTimeline 获取事件时间线
// @Summary 获取事件时间线
// @Description 获取指定条件下的事件时间线
// @Tags Event管理
// @Accept json
// @Produce json
// @Param request query model.K8sEventTimelineReq true "事件时间线请求"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sEventTimelineItem} "成功获取事件时间线"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/events/timeline [get]
func (k *K8sEventHandler) GetEventTimeline(ctx *gin.Context) {
	var req model.K8sEventTimelineReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.eventService.GetEventTimeline(ctx, &req)
	})
}

// CleanupOldEvents 清理旧事件
// @Summary 清理旧事件
// @Description 清理指定条件下的旧事件
// @Tags Event管理
// @Accept json
// @Produce json
// @Param request body model.K8sEventCleanupReq true "事件清理请求"
// @Success 200 {object} utils.ApiResponse{data=model.K8sEventCleanupResult} "成功清理旧事件"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/events/cleanup [post]
func (k *K8sEventHandler) CleanupOldEvents(ctx *gin.Context) {
	var req model.K8sEventCleanupReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.eventService.CleanupOldEvents(ctx, &req)
	})
}

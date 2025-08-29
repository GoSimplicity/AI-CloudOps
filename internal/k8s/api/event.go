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
func (k *K8sEventHandler) CleanupOldEvents(ctx *gin.Context) {
	var req model.K8sEventCleanupReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.eventService.CleanupOldEvents(ctx, &req)
	})
}

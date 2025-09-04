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
)

type K8sEventHandler struct {
	eventService service.EventService
}

func NewK8sEventHandler(eventService service.EventService) *K8sEventHandler {
	return &K8sEventHandler{

		eventService: eventService,
	}
}

func (k *K8sEventHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/events/list", k.GetEventList)                                                           // 获取事件列表
		k8sGroup.GET("/events/:cluster_id/:namespace/detail/:name", k.GetEventDetail)                          // 获取单个事件详情
		k8sGroup.GET("/events/:cluster_id/:namespace/by-pod/:pod_name", k.GetEventsByPod)                      // 获取Pod相关事件
		k8sGroup.GET("/events/:cluster_id/:namespace/by-deployment/:deployment_name", k.GetEventsByDeployment) // 获取Deployment相关事件
		k8sGroup.GET("/events/:cluster_id/:namespace/by-service/:service_name", k.GetEventsByService)          // 获取Service相关事件
		k8sGroup.GET("/events/:cluster_id/:namespace/by-ingress/:ingress_name", k.GetEventsByIngress)          // 获取Ingress相关事件
		k8sGroup.GET("/events/:cluster_id/by-node/:node_name", k.GetEventsByNode)                              // 获取Node相关事件
		k8sGroup.GET("/events/statistics", k.GetEventStatistics)                                               // 获取事件统计信息
		k8sGroup.GET("/events/summary", k.GetEventSummary)                                                     // 获取事件汇总
		k8sGroup.GET("/events/timeline", k.GetEventTimeline)                                                   // 获取事件时间线
		k8sGroup.GET("/events/trends", k.GetEventTrends)                                                       // 获取事件趋势
		k8sGroup.GET("/events/group", k.GetEventGroupData)                                                     // 获取事件分组数据
		k8sGroup.DELETE("/events/:cluster_id/:namespace/delete/:name", k.DeleteEvent)                          // 删除单个事件
		k8sGroup.POST("/events/cleanup", k.CleanupOldEvents)                                                   // 清理旧事件
	}
}

// GetEventList 获取Event列表
func (k *K8sEventHandler) GetEventList(ctx *gin.Context) {
	var req model.GetEventListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.eventService.GetEventList(ctx, &req)
	})
}

// GetEventDetail 获取Event详情
func (k *K8sEventHandler) GetEventDetail(ctx *gin.Context) {
	var req model.GetEventDetailReq

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
		return k.eventService.GetEvent(ctx, &req)
	})
}

// GetEventsByPod 获取Pod相关事件
func (k *K8sEventHandler) GetEventsByPod(ctx *gin.Context) {
	var req model.GetEventsByPodReq

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

	podName, err := utils.GetParamCustomName(ctx, "pod_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.PodName = podName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.eventService.GetEventsByPod(ctx, &req)
	})
}

// GetEventsByDeployment 获取Deployment相关事件
func (k *K8sEventHandler) GetEventsByDeployment(ctx *gin.Context) {
	var req model.GetEventsByDeploymentReq

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

	deploymentName, err := utils.GetParamCustomName(ctx, "deployment_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.DeploymentName = deploymentName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.eventService.GetEventsByDeployment(ctx, &req)
	})
}

// GetEventsByService 获取Service相关事件
func (k *K8sEventHandler) GetEventsByService(ctx *gin.Context) {
	var req model.GetEventsByServiceReq

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

	serviceName, err := utils.GetParamCustomName(ctx, "service_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.ServiceName = serviceName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.eventService.GetEventsByService(ctx, &req)
	})
}

// GetEventsByNode 获取Node相关事件
func (k *K8sEventHandler) GetEventsByNode(ctx *gin.Context) {
	var req model.GetEventsByNodeReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := utils.GetParamCustomName(ctx, "node_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.eventService.GetEventsByNode(ctx, &req)
	})
}

// GetEventStatistics 获取事件统计
func (k *K8sEventHandler) GetEventStatistics(ctx *gin.Context) {
	var req model.GetEventStatisticsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.eventService.GetEventStatistics(ctx, &req)
	})
}

// GetEventSummary 获取事件汇总
func (k *K8sEventHandler) GetEventSummary(ctx *gin.Context) {
	var req model.GetEventSummaryReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.eventService.GetEventSummary(ctx, &req)
	})
}

// GetEventTimeline 获取事件时间线
func (k *K8sEventHandler) GetEventTimeline(ctx *gin.Context) {
	var req model.GetEventTimelineReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.eventService.GetEventTimeline(ctx, &req)
	})
}

// GetEventTrends 获取事件趋势
func (k *K8sEventHandler) GetEventTrends(ctx *gin.Context) {
	var req model.GetEventTrendsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.eventService.GetEventTrends(ctx, &req)
	})
}

// GetEventGroupData 获取事件分组数据
func (k *K8sEventHandler) GetEventGroupData(ctx *gin.Context) {
	var req model.GetEventGroupDataReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.eventService.GetEventGroupData(ctx, &req)
	})
}

// DeleteEvent 删除单个事件
func (k *K8sEventHandler) DeleteEvent(ctx *gin.Context) {
	var req model.DeleteEventReq

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
		return nil, k.eventService.DeleteEvent(ctx, &req)
	})
}

// CleanupOldEvents 清理旧事件
func (k *K8sEventHandler) CleanupOldEvents(ctx *gin.Context) {
	var req model.CleanupOldEventsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.eventService.CleanupOldEvents(ctx, &req)
	})
}

// GetEventsByIngress 获取Ingress相关事件
func (k *K8sEventHandler) GetEventsByIngress(ctx *gin.Context) {
	var req model.K8sIngressEventReq

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

	ingressName, err := utils.GetParamCustomName(ctx, "ingress_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.IngressName = ingressName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.eventService.GetEventsByIngress(ctx, &req)
	})
}

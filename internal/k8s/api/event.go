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

func (h *K8sEventHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/clusters/:cluster_id/events", h.GetEventList)
		k8sGroup.GET("/clusters/:cluster_id/events/:namespace/:name", h.GetEventDetail)
		k8sGroup.DELETE("/clusters/:cluster_id/events/:namespace/:name", h.DeleteEvent)
		k8sGroup.GET("/clusters/:cluster_id/events/:namespace/pods/:pod_name", h.GetEventsByPod)
		k8sGroup.GET("/clusters/:cluster_id/events/:namespace/deployments/:deployment_name", h.GetEventsByDeployment)
		k8sGroup.GET("/clusters/:cluster_id/events/:namespace/services/:service_name", h.GetEventsByService)
		k8sGroup.GET("/clusters/:cluster_id/events/nodes/:node_name", h.GetEventsByNode)
		k8sGroup.GET("/clusters/:cluster_id/events/statistics", h.GetEventStatistics)
		k8sGroup.GET("/clusters/:cluster_id/events/summary", h.GetEventSummary)
		k8sGroup.GET("/clusters/:cluster_id/events/timeline", h.GetEventTimeline)
		k8sGroup.GET("/clusters/:cluster_id/events/trends", h.GetEventTrends)
		k8sGroup.GET("/clusters/:cluster_id/events/groups", h.GetEventGroupData)
		k8sGroup.POST("/clusters/:cluster_id/events/cleanup", h.CleanupOldEvents)
	}
}

func (h *K8sEventHandler) GetEventList(ctx *gin.Context) {
	var req model.GetEventListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.eventService.GetEventList(ctx, &req)
	})
}

func (h *K8sEventHandler) GetEventDetail(ctx *gin.Context) {
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
		return h.eventService.GetEvent(ctx, &req)
	})
}

func (h *K8sEventHandler) GetEventsByPod(ctx *gin.Context) {
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
		return h.eventService.GetEventsByPod(ctx, &req)
	})
}

func (h *K8sEventHandler) GetEventsByDeployment(ctx *gin.Context) {
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
		return h.eventService.GetEventsByDeployment(ctx, &req)
	})
}

func (h *K8sEventHandler) GetEventsByService(ctx *gin.Context) {
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
		return h.eventService.GetEventsByService(ctx, &req)
	})
}

func (h *K8sEventHandler) GetEventsByNode(ctx *gin.Context) {
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
		return h.eventService.GetEventsByNode(ctx, &req)
	})
}

func (h *K8sEventHandler) GetEventStatistics(ctx *gin.Context) {
	var req model.GetEventStatisticsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.eventService.GetEventStatistics(ctx, &req)
	})
}

func (h *K8sEventHandler) GetEventSummary(ctx *gin.Context) {
	var req model.GetEventSummaryReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.eventService.GetEventSummary(ctx, &req)
	})
}

func (h *K8sEventHandler) GetEventTimeline(ctx *gin.Context) {
	var req model.GetEventTimelineReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.eventService.GetEventTimeline(ctx, &req)
	})
}

func (h *K8sEventHandler) GetEventTrends(ctx *gin.Context) {
	var req model.GetEventTrendsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.eventService.GetEventTrends(ctx, &req)
	})
}

func (h *K8sEventHandler) GetEventGroupData(ctx *gin.Context) {
	var req model.GetEventGroupDataReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.eventService.GetEventGroupData(ctx, &req)
	})
}

func (h *K8sEventHandler) DeleteEvent(ctx *gin.Context) {
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
		return nil, h.eventService.DeleteEvent(ctx, &req)
	})
}

// CleanupOldEvents 清理旧事件
func (h *K8sEventHandler) CleanupOldEvents(ctx *gin.Context) {
	var req model.CleanupOldEventsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.eventService.CleanupOldEvents(ctx, &req)
	})
}

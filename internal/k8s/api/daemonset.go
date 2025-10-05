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

type K8sDaemonSetHandler struct {
	daemonSetService service.DaemonSetService
}

func NewK8sDaemonSetHandler(daemonSetService service.DaemonSetService) *K8sDaemonSetHandler {
	return &K8sDaemonSetHandler{
		daemonSetService: daemonSetService,
	}
}

func (h *K8sDaemonSetHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/daemonset/:cluster_id/list", h.GetDaemonSetList)
		k8sGroup.GET("/daemonset/:cluster_id/:namespace/:name/detail", h.GetDaemonSetDetails)
		k8sGroup.GET("/daemonset/:cluster_id/:namespace/:name/detail/yaml", h.GetDaemonSetYaml)
		k8sGroup.POST("/daemonset/:cluster_id/create", h.CreateDaemonSet)
		k8sGroup.POST("/daemonset/:cluster_id/create/yaml", h.CreateDaemonSetByYaml)
		k8sGroup.PUT("/daemonset/:cluster_id/:namespace/:name/update", h.UpdateDaemonSet)
		k8sGroup.PUT("/daemonset/:cluster_id/:namespace/:name/update/yaml", h.UpdateDaemonSetByYaml)
		k8sGroup.DELETE("/daemonset/:cluster_id/:namespace/:name/delete", h.DeleteDaemonSet)
		k8sGroup.POST("/daemonset/:cluster_id/:namespace/:name/restart", h.RestartDaemonSet)
		k8sGroup.POST("/daemonset/:cluster_id/:namespace/:name/rollback", h.RollbackDaemonSet)
		k8sGroup.GET("/daemonset/:cluster_id/:namespace/:name/pods", h.GetDaemonSetPods)
		k8sGroup.GET("/daemonset/:cluster_id/:namespace/:name/history", h.GetDaemonSetHistory)
	}
}

func (h *K8sDaemonSetHandler) GetDaemonSetList(ctx *gin.Context) {
	var req model.GetDaemonSetListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.daemonSetService.GetDaemonSetList(ctx, &req)
	})
}

func (h *K8sDaemonSetHandler) GetDaemonSetDetails(ctx *gin.Context) {
	var req model.GetDaemonSetDetailsReq

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
		return h.daemonSetService.GetDaemonSetDetails(ctx, &req)
	})
}

func (h *K8sDaemonSetHandler) GetDaemonSetYaml(ctx *gin.Context) {
	var req model.GetDaemonSetYamlReq

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
		return h.daemonSetService.GetDaemonSetYaml(ctx, &req)
	})
}

func (h *K8sDaemonSetHandler) CreateDaemonSet(ctx *gin.Context) {
	var req model.CreateDaemonSetReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.daemonSetService.CreateDaemonSet(ctx, &req)
	})
}

func (h *K8sDaemonSetHandler) CreateDaemonSetByYaml(ctx *gin.Context) {
	var req model.CreateDaemonSetByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.daemonSetService.CreateDaemonSetByYaml(ctx, &req)
	})
}

func (h *K8sDaemonSetHandler) UpdateDaemonSet(ctx *gin.Context) {
	var req model.UpdateDaemonSetReq

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
		return nil, h.daemonSetService.UpdateDaemonSet(ctx, &req)
	})
}

func (h *K8sDaemonSetHandler) UpdateDaemonSetByYaml(ctx *gin.Context) {
	var req model.UpdateDaemonSetByYamlReq

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
		return nil, h.daemonSetService.UpdateDaemonSetByYaml(ctx, &req)
	})
}

func (h *K8sDaemonSetHandler) DeleteDaemonSet(ctx *gin.Context) {
	var req model.DeleteDaemonSetReq

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
		return nil, h.daemonSetService.DeleteDaemonSet(ctx, &req)
	})
}

func (h *K8sDaemonSetHandler) RestartDaemonSet(ctx *gin.Context) {
	var req model.RestartDaemonSetReq

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
		return nil, h.daemonSetService.RestartDaemonSet(ctx, &req)
	})
}

func (h *K8sDaemonSetHandler) GetDaemonSetPods(ctx *gin.Context) {
	var req model.GetDaemonSetPodsReq

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
		return h.daemonSetService.GetDaemonSetPods(ctx, &req)
	})
}

func (h *K8sDaemonSetHandler) GetDaemonSetHistory(ctx *gin.Context) {
	var req model.GetDaemonSetHistoryReq

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
		return h.daemonSetService.GetDaemonSetHistory(ctx, &req)
	})
}

func (h *K8sDaemonSetHandler) RollbackDaemonSet(ctx *gin.Context) {
	var req model.RollbackDaemonSetReq

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
		return nil, h.daemonSetService.RollbackDaemonSet(ctx, &req)
	})
}

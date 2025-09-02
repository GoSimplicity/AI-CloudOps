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

type K8sClusterRoleHandler struct {
	clusterRoleService service.ClusterRoleService
}

func NewK8sClusterRoleHandler(clusterRoleService service.ClusterRoleService) *K8sClusterRoleHandler {
	return &K8sClusterRoleHandler{
		clusterRoleService: clusterRoleService,
	}
}

func (k *K8sClusterRoleHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/clusterroles", k.GetClusterRoleList)
		k8sGroup.GET("/clusterroles/:cluster_id/:name", k.GetClusterRoleDetails)
		k8sGroup.GET("/clusterroles/:cluster_id/:name/yaml", k.GetClusterRoleYaml)
		k8sGroup.POST("/clusterroles", k.CreateClusterRole)
		k8sGroup.PUT("/clusterroles/:cluster_id/:name", k.UpdateClusterRole)
		k8sGroup.DELETE("/clusterroles/:cluster_id/:name", k.DeleteClusterRole)
		k8sGroup.GET("/clusterroles/:cluster_id/:name/events", k.GetClusterRoleEvents)
		k8sGroup.GET("/clusterroles/:cluster_id/:name/usage", k.GetClusterRoleUsage)
		k8sGroup.GET("/clusterroles/:cluster_id/:name/metrics", k.GetClusterRoleMetrics)

		k8sGroup.PUT("/clusterroles/:cluster_id/:name/yaml", k.UpdateClusterRoleYaml)
	}
}

func (k *K8sClusterRoleHandler) GetClusterRoleList(ctx *gin.Context) {
	var req model.GetClusterRoleListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.clusterRoleService.GetClusterRoleList(ctx, &req)
	})
}

func (k *K8sClusterRoleHandler) GetClusterRoleDetails(ctx *gin.Context) {
	var req model.GetClusterRoleDetailsReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
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
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.clusterRoleService.GetClusterRoleDetails(ctx, &req)
	})
}

func (k *K8sClusterRoleHandler) CreateClusterRole(ctx *gin.Context) {
	var req model.CreateClusterRoleReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterRoleService.CreateClusterRole(ctx, &req)
	})
}

func (k *K8sClusterRoleHandler) UpdateClusterRole(ctx *gin.Context) {
	var req model.UpdateClusterRoleReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
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
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterRoleService.UpdateClusterRole(ctx, &req)
	})
}

func (k *K8sClusterRoleHandler) DeleteClusterRole(ctx *gin.Context) {
	var req model.DeleteClusterRoleReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
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
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterRoleService.DeleteClusterRole(ctx, &req)
	})
}

func (k *K8sClusterRoleHandler) GetClusterRoleYaml(ctx *gin.Context) {
	var req model.GetClusterRoleYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
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
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.clusterRoleService.GetClusterRoleYaml(ctx, &req)
	})
}

func (k *K8sClusterRoleHandler) UpdateClusterRoleYaml(ctx *gin.Context) {
	var req model.UpdateClusterRoleYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
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
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterRoleService.UpdateClusterRoleYaml(ctx, &req)
	})
}

func (k *K8sClusterRoleHandler) GetClusterRoleEvents(ctx *gin.Context) {
	var req model.GetClusterRoleEventsReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
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
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.clusterRoleService.GetClusterRoleEvents(ctx, &req)
	})
}

func (k *K8sClusterRoleHandler) GetClusterRoleUsage(ctx *gin.Context) {
	var req model.GetClusterRoleUsageReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
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
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.clusterRoleService.GetClusterRoleUsage(ctx, &req)
	})
}

func (k *K8sClusterRoleHandler) GetClusterRoleMetrics(ctx *gin.Context) {
	var req model.GetClusterRoleMetricsReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
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
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.clusterRoleService.GetClusterRoleMetrics(ctx, &req)
	})
}

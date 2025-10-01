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

func (h *K8sClusterRoleHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/clusterrole/:cluster_id/list", h.GetClusterRoleList)
		k8sGroup.GET("/clusterrole/:cluster_id/:name/detail", h.GetClusterRoleDetails)
		k8sGroup.GET("/clusterrole/:cluster_id/:name/detail/yaml", h.GetClusterRoleYaml)
		k8sGroup.POST("/clusterrole/:cluster_id/create", h.CreateClusterRole)
		k8sGroup.POST("/clusterrole/:cluster_id/create/yaml", h.CreateClusterRoleByYaml)
		k8sGroup.PUT("/clusterrole/:cluster_id/:name/update", h.UpdateClusterRole)
		k8sGroup.PUT("/clusterrole/:cluster_id/:name/update/yaml", h.UpdateClusterRoleByYaml)
		k8sGroup.DELETE("/clusterrole/:cluster_id/:name/delete", h.DeleteClusterRole)
	}
}

// GetClusterRoleList 获取ClusterRole列表
func (h *K8sClusterRoleHandler) GetClusterRoleList(ctx *gin.Context) {
	var req model.GetClusterRoleListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.clusterRoleService.GetClusterRoleList(ctx, &req)
	})
}

// GetClusterRoleDetails 获取ClusterRole详情
func (h *K8sClusterRoleHandler) GetClusterRoleDetails(ctx *gin.Context) {
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
		return h.clusterRoleService.GetClusterRoleDetails(ctx, &req)
	})
}

// GetClusterRoleYaml 获取ClusterRole的YAML配置
func (h *K8sClusterRoleHandler) GetClusterRoleYaml(ctx *gin.Context) {
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
		return h.clusterRoleService.GetClusterRoleYaml(ctx, &req)
	})
}

// CreateClusterRole 创建ClusterRole
func (h *K8sClusterRoleHandler) CreateClusterRole(ctx *gin.Context) {
	var req model.CreateClusterRoleReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.clusterRoleService.CreateClusterRole(ctx, &req)
	})
}

// CreateClusterRoleByYaml 通过YAML创建ClusterRole
func (h *K8sClusterRoleHandler) CreateClusterRoleByYaml(ctx *gin.Context) {
	var req model.CreateClusterRoleByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.clusterRoleService.CreateClusterRoleByYaml(ctx, &req)
	})
}

// UpdateClusterRole 更新ClusterRole
func (h *K8sClusterRoleHandler) UpdateClusterRole(ctx *gin.Context) {
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
		return nil, h.clusterRoleService.UpdateClusterRole(ctx, &req)
	})
}

// UpdateClusterRoleYaml 通过YAML更新ClusterRole
func (h *K8sClusterRoleHandler) UpdateClusterRoleByYaml(ctx *gin.Context) {
	var req model.UpdateClusterRoleByYamlReq

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
		return nil, h.clusterRoleService.UpdateClusterRoleYaml(ctx, &req)
	})
}

// DeleteClusterRole 删除ClusterRole
func (h *K8sClusterRoleHandler) DeleteClusterRole(ctx *gin.Context) {
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
		return nil, h.clusterRoleService.DeleteClusterRole(ctx, &req)
	})
}

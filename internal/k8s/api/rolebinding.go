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

type K8sRoleBindingHandler struct {
	roleBindingService service.RoleBindingService
}

func NewK8sRoleBindingHandler(roleBindingService service.RoleBindingService) *K8sRoleBindingHandler {
	return &K8sRoleBindingHandler{
		roleBindingService: roleBindingService,
	}
}

func (h *K8sRoleBindingHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/rolebinding/:cluster_id/list", h.GetRoleBindingList)
		k8sGroup.GET("/rolebinding/:cluster_id/:namespace/:name/detail", h.GetRoleBindingDetails)
		k8sGroup.GET("/rolebinding/:cluster_id/:namespace/:name/detail/yaml", h.GetRoleBindingYaml)
		k8sGroup.POST("/rolebinding/:cluster_id/create", h.CreateRoleBinding)
		k8sGroup.POST("/rolebinding/:cluster_id/create/yaml", h.CreateRoleBindingByYaml)
		k8sGroup.PUT("/rolebinding/:cluster_id/:namespace/:name/update", h.UpdateRoleBinding)
		k8sGroup.PUT("/rolebinding/:cluster_id/:namespace/:name/update/yaml", h.UpdateRoleBindingYaml)
		k8sGroup.DELETE("/rolebinding/:cluster_id/:namespace/:name/delete", h.DeleteRoleBinding)
	}
}

// GetRoleBindingList 获取 RoleBinding 列表
func (h *K8sRoleBindingHandler) GetRoleBindingList(ctx *gin.Context) {
	var req model.GetRoleBindingListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.roleBindingService.GetRoleBindingList(ctx, &req)
	})
}

// GetRoleBindingDetails 获取 RoleBinding 详情
func (h *K8sRoleBindingHandler) GetRoleBindingDetails(ctx *gin.Context) {
	var req model.GetRoleBindingDetailsReq

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
		return h.roleBindingService.GetRoleBindingDetails(ctx, &req)
	})
}

// CreateRoleBinding 创建 RoleBinding
func (h *K8sRoleBindingHandler) CreateRoleBinding(ctx *gin.Context) {
	var req model.CreateRoleBindingReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.roleBindingService.CreateRoleBinding(ctx, &req)
	})
}

// CreateRoleBindingByYaml 通过YAML创建 RoleBinding
func (h *K8sRoleBindingHandler) CreateRoleBindingByYaml(ctx *gin.Context) {
	var req model.CreateRoleBindingByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.roleBindingService.CreateRoleBindingByYaml(ctx, &req)
	})
}

// UpdateRoleBinding 更新 RoleBinding
func (h *K8sRoleBindingHandler) UpdateRoleBinding(ctx *gin.Context) {
	var req model.UpdateRoleBindingReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.roleBindingService.UpdateRoleBinding(ctx, &req)
	})
}

// DeleteRoleBinding 删除 RoleBinding
func (h *K8sRoleBindingHandler) DeleteRoleBinding(ctx *gin.Context) {
	var req model.DeleteRoleBindingReq

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
		return nil, h.roleBindingService.DeleteRoleBinding(ctx, &req)
	})
}

// GetRoleBindingYaml 获取 RoleBinding YAML
func (h *K8sRoleBindingHandler) GetRoleBindingYaml(ctx *gin.Context) {
	var req model.GetRoleBindingYamlReq

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
		return h.roleBindingService.GetRoleBindingYaml(ctx, &req)
	})
}

// UpdateRoleBindingYaml 更新 RoleBinding YAML
func (h *K8sRoleBindingHandler) UpdateRoleBindingYaml(ctx *gin.Context) {
	var req model.UpdateRoleBindingByYamlReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.roleBindingService.UpdateRoleBindingYaml(ctx, &req)
	})
}

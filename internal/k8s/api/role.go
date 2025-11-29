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
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/gin-gonic/gin"
)

type K8sRoleHandler struct {
	roleService service.RoleService
}

func NewK8sRoleHandler(roleService service.RoleService) *K8sRoleHandler {
	return &K8sRoleHandler{
		roleService: roleService,
	}
}

func (h *K8sRoleHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/role/:cluster_id/list", h.GetRoleList)
		k8sGroup.GET("/role/:cluster_id/:namespace/:name/detail", h.GetRoleDetails)
		k8sGroup.GET("/role/:cluster_id/:namespace/:name/detail/yaml", h.GetRoleYaml)
		k8sGroup.POST("/role/:cluster_id/create", h.CreateRole)
		k8sGroup.POST("/role/:cluster_id/create/yaml", h.CreateRoleByYaml)
		k8sGroup.PUT("/role/:cluster_id/:namespace/:name/update", h.UpdateRole)
		k8sGroup.PUT("/role/:cluster_id/:namespace/:name/update/yaml", h.UpdateRoleByYaml)
		k8sGroup.DELETE("/role/:cluster_id/:namespace/:name/delete", h.DeleteRole)
	}
}

func (h *K8sRoleHandler) GetRoleList(ctx *gin.Context) {
	var req model.GetRoleListReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.roleService.GetRoleList(ctx, &req)
	})
}

func (h *K8sRoleHandler) GetRoleDetails(ctx *gin.Context) {
	var req model.GetRoleDetailsReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.roleService.GetRoleDetails(ctx, &req)
	})
}

func (h *K8sRoleHandler) GetRoleYaml(ctx *gin.Context) {
	var req model.GetRoleYamlReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.roleService.GetRoleYaml(ctx, &req)
	})
}

func (h *K8sRoleHandler) CreateRole(ctx *gin.Context) {
	var req model.CreateRoleReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.roleService.CreateRole(ctx, &req)
	})
}

func (h *K8sRoleHandler) CreateRoleByYaml(ctx *gin.Context) {
	var req model.CreateRoleByYamlReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.roleService.CreateRoleByYaml(ctx, &req)
	})
}

func (h *K8sRoleHandler) UpdateRole(ctx *gin.Context) {
	var req model.UpdateRoleReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.roleService.UpdateRole(ctx, &req)
	})
}

func (h *K8sRoleHandler) UpdateRoleByYaml(ctx *gin.Context) {
	var req model.UpdateRoleByYamlReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.roleService.UpdateRoleYaml(ctx, &req)
	})
}

func (h *K8sRoleHandler) DeleteRole(ctx *gin.Context) {
	var req model.DeleteRoleReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.roleService.DeleteRole(ctx, &req)
	})
}

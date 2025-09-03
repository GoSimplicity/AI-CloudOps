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

type K8sRoleHandler struct {
	roleService service.RoleService
}

func NewK8sRoleHandler(roleService service.RoleService) *K8sRoleHandler {
	return &K8sRoleHandler{
		roleService: roleService,
	}
}

func (k *K8sRoleHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/roles", k.GetRoleList)
		k8sGroup.GET("/roles/:cluster_id/:namespace/:name", k.GetRoleDetails)
		k8sGroup.POST("/roles", k.CreateRole)
		k8sGroup.PUT("/roles/:cluster_id/:namespace/:name", k.UpdateRole)
		k8sGroup.DELETE("/roles/:cluster_id/:namespace/:name", k.DeleteRole)
		k8sGroup.GET("/roles/:cluster_id/:namespace/:name/yaml", k.GetRoleYaml)
		k8sGroup.PUT("/roles/:cluster_id/:namespace/:name/yaml", k.UpdateRoleYaml)
	}
}

func (k *K8sRoleHandler) GetRoleList(ctx *gin.Context) {
	var req model.GetRoleListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.roleService.GetRoleList(ctx, &req)
	})
}

func (k *K8sRoleHandler) GetRoleDetails(ctx *gin.Context) {
	var req model.GetRoleDetailsReq

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
		return k.roleService.GetRoleDetails(ctx, &req)
	})
}

func (k *K8sRoleHandler) CreateRole(ctx *gin.Context) {
	var req model.CreateRoleReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.roleService.CreateRole(ctx, &req)
	})
}

func (k *K8sRoleHandler) UpdateRole(ctx *gin.Context) {
	var req model.UpdateRoleReq

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
		return nil, k.roleService.UpdateRole(ctx, &req)
	})
}

func (k *K8sRoleHandler) DeleteRole(ctx *gin.Context) {
	var req model.DeleteRoleReq

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
		return nil, k.roleService.DeleteRole(ctx, &req)
	})
}

func (k *K8sRoleHandler) GetRoleYaml(ctx *gin.Context) {
	var req model.GetRoleYamlReq

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
		return k.roleService.GetRoleYaml(ctx, &req)
	})
}

func (k *K8sRoleHandler) UpdateRoleYaml(ctx *gin.Context) {
	var req model.UpdateRoleYamlReq

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
		return nil, k.roleService.UpdateRoleYaml(ctx, &req)
	})
}

func (k *K8sRoleHandler) GetRoleEvents(ctx *gin.Context) {
	var req model.GetRoleEventsReq

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
		return k.roleService.GetRoleEvents(ctx, &req)
	})
}

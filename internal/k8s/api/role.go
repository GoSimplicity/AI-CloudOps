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
		// Role 基本操作
		k8sGroup.GET("/roles", k.GetRoleList)                                 // 获取列表
		k8sGroup.GET("/roles/:cluster_id/:namespace/:name", k.GetRoleDetails) // 获取详情
		k8sGroup.POST("/roles", k.CreateRole)                                 // 创建
		k8sGroup.PUT("/roles/:cluster_id/:namespace/:name", k.UpdateRole)     // 更新
		k8sGroup.DELETE("/roles/:cluster_id/:namespace/:name", k.DeleteRole)  // 删除

		// Role YAML 操作
		k8sGroup.GET("/roles/:cluster_id/:namespace/:name/yaml", k.GetRoleYaml)    // 获取YAML
		k8sGroup.PUT("/roles/:cluster_id/:namespace/:name/yaml", k.UpdateRoleYaml) // 更新YAML

		// Role 扩展功能
		k8sGroup.GET("/roles/:cluster_id/:namespace/:name/events", k.GetRoleEvents)   // 获取事件
		k8sGroup.GET("/roles/:cluster_id/:namespace/:name/usage", k.GetRoleUsage)     // 获取使用情况
		k8sGroup.GET("/roles/:cluster_id/:namespace/:name/metrics", k.GetRoleMetrics) // 获取指标
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

func (k *K8sRoleHandler) GetRoleUsage(ctx *gin.Context) {
	var req model.GetRoleUsageReq

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
		return k.roleService.GetRoleUsage(ctx, &req)
	})
}

func (k *K8sRoleHandler) GetRoleMetrics(ctx *gin.Context) {
	var req model.GetRoleMetricsReq

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
		return k.roleService.GetRoleMetrics(ctx, &req)
	})
}

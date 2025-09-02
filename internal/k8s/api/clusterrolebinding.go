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

type K8sClusterRoleBindingHandler struct {
	clusterRoleBindingService service.ClusterRoleBindingService
}

func NewK8sClusterRoleBindingHandler(clusterRoleBindingService service.ClusterRoleBindingService) *K8sClusterRoleBindingHandler {
	return &K8sClusterRoleBindingHandler{
		clusterRoleBindingService: clusterRoleBindingService,
	}
}

func (k *K8sClusterRoleBindingHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		// ClusterRoleBinding 基本操作
		k8sGroup.GET("/cluster-role-bindings", k.GetClusterRoleBindingList)                      // 获取列表
		k8sGroup.GET("/cluster-role-bindings/:cluster_id/:name", k.GetClusterRoleBindingDetails) // 获取详情
		k8sGroup.POST("/cluster-role-bindings", k.CreateClusterRoleBinding)                      // 创建
		k8sGroup.PUT("/cluster-role-bindings/:cluster_id/:name", k.UpdateClusterRoleBinding)     // 更新
		k8sGroup.DELETE("/cluster-role-bindings/:cluster_id/:name", k.DeleteClusterRoleBinding)  // 删除

		// ClusterRoleBinding YAML 操作
		k8sGroup.GET("/cluster-role-bindings/:cluster_id/:name/yaml", k.GetClusterRoleBindingYaml)    // 获取YAML
		k8sGroup.PUT("/cluster-role-bindings/:cluster_id/:name/yaml", k.UpdateClusterRoleBindingYaml) // 更新YAML

		// ClusterRoleBinding 扩展功能
		k8sGroup.GET("/cluster-role-bindings/:cluster_id/:name/events", k.GetClusterRoleBindingEvents)   // 获取事件
		k8sGroup.GET("/cluster-role-bindings/:cluster_id/:name/usage", k.GetClusterRoleBindingUsage)     // 获取使用情况
		k8sGroup.GET("/cluster-role-bindings/:cluster_id/:name/metrics", k.GetClusterRoleBindingMetrics) // 获取指标
	}
}

func (k *K8sClusterRoleBindingHandler) GetClusterRoleBindingList(ctx *gin.Context) {
	var req model.GetClusterRoleBindingListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.clusterRoleBindingService.GetClusterRoleBindingList(ctx, &req)
	})
}

func (k *K8sClusterRoleBindingHandler) GetClusterRoleBindingDetails(ctx *gin.Context) {
	var req model.GetClusterRoleBindingDetailsReq

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
		return k.clusterRoleBindingService.GetClusterRoleBindingDetails(ctx, &req)
	})
}

func (k *K8sClusterRoleBindingHandler) CreateClusterRoleBinding(ctx *gin.Context) {
	var req model.CreateClusterRoleBindingReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterRoleBindingService.CreateClusterRoleBinding(ctx, &req)
	})
}

func (k *K8sClusterRoleBindingHandler) UpdateClusterRoleBinding(ctx *gin.Context) {
	var req model.UpdateClusterRoleBindingReq

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
		return nil, k.clusterRoleBindingService.UpdateClusterRoleBinding(ctx, &req)
	})
}

func (k *K8sClusterRoleBindingHandler) DeleteClusterRoleBinding(ctx *gin.Context) {
	var req model.DeleteClusterRoleBindingReq

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
		return nil, k.clusterRoleBindingService.DeleteClusterRoleBinding(ctx, &req)
	})
}

func (k *K8sClusterRoleBindingHandler) GetClusterRoleBindingYaml(ctx *gin.Context) {
	var req model.GetClusterRoleBindingYamlReq

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
		return k.clusterRoleBindingService.GetClusterRoleBindingYaml(ctx, &req)
	})
}

func (k *K8sClusterRoleBindingHandler) UpdateClusterRoleBindingYaml(ctx *gin.Context) {
	var req model.UpdateClusterRoleBindingYamlReq

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
		return nil, k.clusterRoleBindingService.UpdateClusterRoleBindingYaml(ctx, &req)
	})
}

func (k *K8sClusterRoleBindingHandler) GetClusterRoleBindingEvents(ctx *gin.Context) {
	var req model.GetClusterRoleBindingEventsReq

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
		return k.clusterRoleBindingService.GetClusterRoleBindingEvents(ctx, &req)
	})
}

func (k *K8sClusterRoleBindingHandler) GetClusterRoleBindingUsage(ctx *gin.Context) {
	var req model.GetClusterRoleBindingUsageReq

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
		return k.clusterRoleBindingService.GetClusterRoleBindingUsage(ctx, &req)
	})
}

func (k *K8sClusterRoleBindingHandler) GetClusterRoleBindingMetrics(ctx *gin.Context) {
	var req model.GetClusterRoleBindingMetricsReq

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
		return k.clusterRoleBindingService.GetClusterRoleBindingMetrics(ctx, &req)
	})
}

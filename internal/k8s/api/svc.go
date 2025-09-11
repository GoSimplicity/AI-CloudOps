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

type K8sSvcHandler struct {
	svcService service.SvcService
}

func NewK8sSvcHandler(svcService service.SvcService) *K8sSvcHandler {
	return &K8sSvcHandler{
		svcService: svcService,
	}
}

func (k *K8sSvcHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		// Service基础管理
		k8sGroup.GET("/clusters/:cluster_id/services/list", k.GetServiceList)                              // 获取Service列表
		k8sGroup.GET("/clusters/:cluster_id/services/:namespace/:name/detail", k.GetServiceDetails)        // 获取Service详情
		k8sGroup.GET("/clusters/:cluster_id/services/:namespace/:name/detail/yaml", k.GetServiceYaml)      // 获取Service YAML
		k8sGroup.POST("/clusters/:cluster_id/services/create", k.CreateService)                            // 创建Service
		k8sGroup.POST("/clusters/:cluster_id/services/create/yaml", k.CreateServiceByYaml)                 // 通过YAML创建Service
		k8sGroup.PUT("/clusters/:cluster_id/services/:namespace/:name/update", k.UpdateService)            // 更新Service
		k8sGroup.PUT("/clusters/:cluster_id/services/:namespace/:name/update/yaml", k.UpdateServiceByYaml) // 通过YAML更新Service
		k8sGroup.DELETE("/clusters/:cluster_id/services/:namespace/:name/delete", k.DeleteService)         // 删除Service
		k8sGroup.GET("/clusters/:cluster_id/services/:namespace/:name/endpoints", k.GetServiceEndpoints)   // 获取Service端点
	}
}

// GetServiceList 获取Service列表
func (k *K8sSvcHandler) GetServiceList(ctx *gin.Context) {
	var req model.GetServiceListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.svcService.GetServiceList(ctx, &req)
	})
}

// GetServiceDetails 获取Service详情
func (k *K8sSvcHandler) GetServiceDetails(ctx *gin.Context) {
	var req model.GetServiceDetailsReq

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
		return k.svcService.GetServiceDetails(ctx, &req)
	})
}

// GetServiceYaml 获取Service YAML
func (k *K8sSvcHandler) GetServiceYaml(ctx *gin.Context) {
	var req model.GetServiceYamlReq

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
		return k.svcService.GetServiceYaml(ctx, &req)
	})
}

// CreateService 创建Service
func (k *K8sSvcHandler) CreateService(ctx *gin.Context) {
	var req model.CreateServiceReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.svcService.CreateService(ctx, &req)
	})
}

// UpdateService 更新Service
func (k *K8sSvcHandler) UpdateService(ctx *gin.Context) {
	var req model.UpdateServiceReq

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
		return nil, k.svcService.UpdateService(ctx, &req)
	})
}

// DeleteService 删除Service
func (k *K8sSvcHandler) DeleteService(ctx *gin.Context) {
	var req model.DeleteServiceReq

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
		return nil, k.svcService.DeleteService(ctx, &req)
	})
}

// GetServiceEndpoints 获取Service端点
func (k *K8sSvcHandler) GetServiceEndpoints(ctx *gin.Context) {
	var req model.GetServiceEndpointsReq

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
		return k.svcService.GetServiceEndpoints(ctx, &req)
	})
}

// YAML操作方法

// CreateServiceByYaml 通过YAML创建Service
func (k *K8sSvcHandler) CreateServiceByYaml(ctx *gin.Context) {
	var req model.CreateResourceByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.svcService.CreateServiceByYaml(ctx, &req)
	})
}

// UpdateServiceByYaml 通过YAML更新Service
func (k *K8sSvcHandler) UpdateServiceByYaml(ctx *gin.Context) {
	var req model.UpdateResourceByYamlReq

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
		return nil, k.svcService.UpdateServiceByYaml(ctx, &req)
	})
}

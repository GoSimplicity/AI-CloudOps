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
		k8sGroup.GET("/services", k.GetServiceList)                                             // 获取Service列表
		k8sGroup.GET("/services/:cluster_id/:namespace/:name", k.GetServiceDetails)             // 获取Service详情
		k8sGroup.GET("/services/:cluster_id/:namespace/:name/yaml", k.GetServiceYaml)           // 获取Service YAML
		k8sGroup.POST("/services", k.CreateService)                                             // 创建Service
		k8sGroup.PUT("/services/:cluster_id/:namespace/:name", k.UpdateService)                 // 更新Service
		k8sGroup.DELETE("/services/:cluster_id/:namespace/:name", k.DeleteService)              // 删除Service
		k8sGroup.GET("/services/:cluster_id/:namespace/:name/endpoints", k.GetServiceEndpoints) // 获取Service端点
		k8sGroup.GET("/services/:cluster_id/:namespace/:name/metrics", k.GetServiceMetrics)     // 获取Service指标
		k8sGroup.GET("/services/:cluster_id/:namespace/:name/events", k.GetServiceEvents)       // 获取Service事件
	}
}

// GetServiceList 获取Service列表
func (k *K8sSvcHandler) GetServiceList(ctx *gin.Context) {
	var req model.GetServiceListReq

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

// GetServiceMetrics 获取Service指标
func (k *K8sSvcHandler) GetServiceMetrics(ctx *gin.Context) {
	var req model.GetServiceMetricsReq

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
		return k.svcService.GetServiceMetrics(ctx, &req)
	})
}

// GetServiceEvents 获取Service事件
func (k *K8sSvcHandler) GetServiceEvents(ctx *gin.Context) {
	var req model.GetServiceEventsReq

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
		return k.svcService.GetServiceEvents(ctx, &req)
	})
}

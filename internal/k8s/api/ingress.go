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

type K8sIngressHandler struct {
	ingressService service.IngressService
}

func NewK8sIngressHandler(ingressService service.IngressService) *K8sIngressHandler {
	return &K8sIngressHandler{
		ingressService: ingressService,
	}
}

func (k *K8sIngressHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		// Ingress基础管理
		k8sGroup.GET("/ingress/:cluster_id/list", k.GetIngressList)                              // 获取Ingress列表
		k8sGroup.GET("/ingress/:cluster_id/:namespace/:name/detail", k.GetIngressDetails)        // 获取Ingress详情
		k8sGroup.GET("/ingress/:cluster_id/:namespace/:name/detail/yaml", k.GetIngressYaml)      // 获取Ingress YAML
		k8sGroup.POST("/ingress/:cluster_id/create", k.CreateIngress)                            // 创建Ingress
		k8sGroup.POST("/ingress/:cluster_id/create/yaml", k.CreateIngressByYaml)                 // 通过YAML创建Ingress
		k8sGroup.PUT("/ingress/:cluster_id/:namespace/:name/update", k.UpdateIngress)            // 更新Ingress
		k8sGroup.PUT("/ingress/:cluster_id/:namespace/:name/update/yaml", k.UpdateIngressByYaml) // 通过YAML更新Ingress
		k8sGroup.DELETE("/ingress/:cluster_id/:namespace/:name/delete", k.DeleteIngress)         // 删除Ingress
	}
}

// GetIngressList 获取Ingress列表
func (k *K8sIngressHandler) GetIngressList(ctx *gin.Context) {
	var req model.GetIngressListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.ingressService.GetIngressList(ctx, &req)
	})
}

// GetIngressDetails 获取Ingress详情
func (k *K8sIngressHandler) GetIngressDetails(ctx *gin.Context) {
	var req model.GetIngressDetailsReq

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
		return k.ingressService.GetIngressDetails(ctx, &req)
	})
}

// GetIngressYaml 获取Ingress的YAML配置
func (k *K8sIngressHandler) GetIngressYaml(ctx *gin.Context) {
	var req model.GetIngressYamlReq

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
		return k.ingressService.GetIngressYaml(ctx, &req)
	})
}

// CreateIngress 创建Ingress
func (k *K8sIngressHandler) CreateIngress(ctx *gin.Context) {
	var req model.CreateIngressReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.ingressService.CreateIngress(ctx, &req)
	})
}

// CreateIngressByYaml 通过YAML创建Ingress
func (k *K8sIngressHandler) CreateIngressByYaml(ctx *gin.Context) {
	var req model.CreateIngressByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.ingressService.CreateIngressByYaml(ctx, &req)
	})
}

// UpdateIngress 更新Ingress
func (k *K8sIngressHandler) UpdateIngress(ctx *gin.Context) {
	var req model.UpdateIngressReq

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
		return nil, k.ingressService.UpdateIngress(ctx, &req)
	})
}

// UpdateIngressByYaml 通过YAML更新Ingress
func (k *K8sIngressHandler) UpdateIngressByYaml(ctx *gin.Context) {
	var req model.UpdateIngressByYamlReq

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
		return nil, k.ingressService.UpdateIngressByYaml(ctx, &req)
	})
}

// DeleteIngress 删除Ingress
func (k *K8sIngressHandler) DeleteIngress(ctx *gin.Context) {
	var req model.DeleteIngressReq

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
		return nil, k.ingressService.DeleteIngress(ctx, &req)
	})
}

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
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type K8sNodeAffinityHandler struct {
	nodeAffinityService admin.NodeAffinityService
}

func NewK8sNodeAffinityHandler(nodeAffinityService admin.NodeAffinityService) *K8sNodeAffinityHandler {
	return &K8sNodeAffinityHandler{
		nodeAffinityService: nodeAffinityService,
	}
}

// SetNodeAffinity 设置节点亲和性
func (k *K8sNodeAffinityHandler) SetNodeAffinity(ctx *gin.Context) {
	var req model.K8sNodeAffinityRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.nodeAffinityService.SetNodeAffinity(ctx, &req)
	})
}

// GetNodeAffinity 获取节点亲和性
func (k *K8sNodeAffinityHandler) GetNodeAffinity(ctx *gin.Context) {
	clusterID, err := strconv.Atoi(ctx.Query("cluster_id"))
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的集群ID")
		return
	}

	namespace := ctx.Query("namespace")
	resourceType := ctx.Query("resource_type")
	resourceName := ctx.Query("resource_name")

	if namespace == "" || resourceType == "" || resourceName == "" {
		utils.ErrorWithMessage(ctx, "命名空间、资源类型和资源名称不能为空")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.nodeAffinityService.GetNodeAffinity(ctx, clusterID, namespace, resourceType, resourceName)
	})
}

// UpdateNodeAffinity 更新节点亲和性
func (k *K8sNodeAffinityHandler) UpdateNodeAffinity(ctx *gin.Context) {
	var req model.K8sNodeAffinityRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.nodeAffinityService.UpdateNodeAffinity(ctx, &req)
	})
}

// DeleteNodeAffinity 删除节点亲和性
func (k *K8sNodeAffinityHandler) DeleteNodeAffinity(ctx *gin.Context) {
	var req model.K8sNodeAffinityRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.nodeAffinityService.DeleteNodeAffinity(ctx, &req)
	})
}

// ValidateNodeAffinity 验证节点亲和性
func (k *K8sNodeAffinityHandler) ValidateNodeAffinity(ctx *gin.Context) {
	var req model.K8sNodeAffinityValidationRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.nodeAffinityService.ValidateNodeAffinity(ctx, &req)
	})
}

// GetNodeAffinityRecommendations 获取节点亲和性建议
func (k *K8sNodeAffinityHandler) GetNodeAffinityRecommendations(ctx *gin.Context) {
	clusterID, err := strconv.Atoi(ctx.Query("cluster_id"))
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的集群ID")
		return
	}

	namespace := ctx.Query("namespace")
	resourceType := ctx.Query("resource_type")

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.nodeAffinityService.GetNodeAffinityRecommendations(ctx, clusterID, namespace, resourceType)
	})
}

// RegisterRouters 注册路由
func (k *K8sNodeAffinityHandler) RegisterRouters(g *gin.Engine) {
	k8sGroup := g.Group("/api/k8s")
	{
		// 节点亲和性管理
		k8sGroup.POST("/node-affinity", k.SetNodeAffinity)
		k8sGroup.GET("/node-affinity", k.GetNodeAffinity)
		k8sGroup.PUT("/node-affinity", k.UpdateNodeAffinity)
		k8sGroup.DELETE("/node-affinity", k.DeleteNodeAffinity)

		// 节点亲和性验证
		k8sGroup.POST("/node-affinity/validate", k.ValidateNodeAffinity)

		// 节点亲和性建议
		k8sGroup.GET("/node-affinity/recommendations", k.GetNodeAffinityRecommendations)
	}
}

type K8sPodAffinityHandler struct {
	podAffinityService admin.PodAffinityService
}

func NewK8sPodAffinityHandler(podAffinityService admin.PodAffinityService) *K8sPodAffinityHandler {
	return &K8sPodAffinityHandler{
		podAffinityService: podAffinityService,
	}
}

// SetPodAffinity 设置Pod亲和性
func (k *K8sPodAffinityHandler) SetPodAffinity(ctx *gin.Context) {
	var req model.K8sPodAffinityRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.podAffinityService.SetPodAffinity(ctx, &req)
	})
}

// GetPodAffinity 获取Pod亲和性
func (k *K8sPodAffinityHandler) GetPodAffinity(ctx *gin.Context) {
	clusterID, err := strconv.Atoi(ctx.Query("cluster_id"))
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的集群ID")
		return
	}

	namespace := ctx.Query("namespace")
	resourceType := ctx.Query("resource_type")
	resourceName := ctx.Query("resource_name")

	if namespace == "" || resourceType == "" || resourceName == "" {
		utils.ErrorWithMessage(ctx, "命名空间、资源类型和资源名称不能为空")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.podAffinityService.GetPodAffinity(ctx, clusterID, namespace, resourceType, resourceName)
	})
}

// UpdatePodAffinity 更新Pod亲和性
func (k *K8sPodAffinityHandler) UpdatePodAffinity(ctx *gin.Context) {
	var req model.K8sPodAffinityRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.podAffinityService.UpdatePodAffinity(ctx, &req)
	})
}

// DeletePodAffinity 删除Pod亲和性
func (k *K8sPodAffinityHandler) DeletePodAffinity(ctx *gin.Context) {
	var req model.K8sPodAffinityRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.podAffinityService.DeletePodAffinity(ctx, &req)
	})
}

// ValidatePodAffinity 验证Pod亲和性
func (k *K8sPodAffinityHandler) ValidatePodAffinity(ctx *gin.Context) {
	var req model.K8sPodAffinityValidationRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.podAffinityService.ValidatePodAffinity(ctx, &req)
	})
}

// GetTopologyDomains 获取拓扑域信息
func (k *K8sPodAffinityHandler) GetTopologyDomains(ctx *gin.Context) {
	clusterID, err := strconv.Atoi(ctx.Query("cluster_id"))
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的集群ID")
		return
	}

	namespace := ctx.Query("namespace")

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.podAffinityService.GetTopologyDomains(ctx, clusterID, namespace)
	})
}

// RegisterRouters 注册路由
func (k *K8sPodAffinityHandler) RegisterRouters(g *gin.Engine) {
	k8sGroup := g.Group("/api/k8s")
	{
		// Pod亲和性管理
		k8sGroup.POST("/pod-affinity", k.SetPodAffinity)
		k8sGroup.GET("/pod-affinity", k.GetPodAffinity)
		k8sGroup.PUT("/pod-affinity", k.UpdatePodAffinity)
		k8sGroup.DELETE("/pod-affinity", k.DeletePodAffinity)

		// Pod亲和性验证
		k8sGroup.POST("/pod-affinity/validate", k.ValidatePodAffinity)

		// 拓扑域信息
		k8sGroup.GET("/pod-affinity/topology-domains", k.GetTopologyDomains)
	}
}

type K8sAffinityVisualizationHandler struct {
	visualizationService admin.AffinityVisualizationService
}

func NewK8sAffinityVisualizationHandler(visualizationService admin.AffinityVisualizationService) *K8sAffinityVisualizationHandler {
	return &K8sAffinityVisualizationHandler{
		visualizationService: visualizationService,
	}
}

// GetAffinityVisualization 获取亲和性可视化
func (k *K8sAffinityVisualizationHandler) GetAffinityVisualization(ctx *gin.Context) {
	var req model.K8sAffinityVisualizationRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.visualizationService.GetAffinityVisualization(ctx, &req)
	})
}

// RegisterRouters 注册路由
func (k *K8sAffinityVisualizationHandler) RegisterRouters(g *gin.Engine) {
	k8sGroup := g.Group("/api/k8s")
	{
		// 亲和性可视化
		k8sGroup.POST("/affinity/visualization", k.GetAffinityVisualization)
	}
}

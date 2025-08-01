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
// @Summary 设置节点亲和性
// @Description 为指定的K8s资源（Deployment、StatefulSet等）设置节点亲和性规则，控制Pod调度到特定节点
// @Tags 亲和性管理
// @Accept json
// @Produce json
// @Param request body model.K8sNodeAffinityRequest true "节点亲和性设置请求"
// @Success 200 {object} utils.ApiResponse "设置成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/node-affinity [post]
// @Security BearerAuth
func (k *K8sNodeAffinityHandler) SetNodeAffinity(ctx *gin.Context) {
	var req model.K8sNodeAffinityRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.nodeAffinityService.SetNodeAffinity(ctx, &req)
	})
}

// GetNodeAffinity 获取节点亲和性
// @Summary 获取节点亲和性配置
// @Description 获取指定K8s资源的节点亲和性配置信息，包括必需和偏好的亲和性规则
// @Tags 亲和性管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string true "命名空间"
// @Param resource_type query string true "资源类型（deployment、statefulset等）"
// @Param resource_name query string true "资源名称"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/node-affinity [get]
// @Security BearerAuth
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
// @Summary 更新节点亲和性配置
// @Description 更新指定K8s资源的节点亲和性规则，支持修改必需和偏好的亲和性配置
// @Tags 亲和性管理
// @Accept json
// @Produce json
// @Param request body model.K8sNodeAffinityRequest true "节点亲和性更新请求"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/node-affinity [put]
// @Security BearerAuth
func (k *K8sNodeAffinityHandler) UpdateNodeAffinity(ctx *gin.Context) {
	var req model.K8sNodeAffinityRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.nodeAffinityService.UpdateNodeAffinity(ctx, &req)
	})
}

// DeleteNodeAffinity 删除节点亲和性
// @Summary 删除节点亲和性配置
// @Description 删除指定K8s资源的节点亲和性配置，恢复默认调度策略
// @Tags 亲和性管理
// @Accept json
// @Produce json
// @Param request body model.K8sNodeAffinityRequest true "节点亲和性删除请求"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/node-affinity [delete]
// @Security BearerAuth
func (k *K8sNodeAffinityHandler) DeleteNodeAffinity(ctx *gin.Context) {
	var req model.K8sNodeAffinityRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.nodeAffinityService.DeleteNodeAffinity(ctx, &req)
	})
}

// ValidateNodeAffinity 验证节点亲和性
// @Summary 验证节点亲和性配置
// @Description 验证节点亲和性配置的正确性和语法，检查标签选择器和表达式是否有效
// @Tags 亲和性管理
// @Accept json
// @Produce json
// @Param request body model.K8sNodeAffinityValidationRequest true "节点亲和性验证请求"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "验证成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/node-affinity/validate [post]
// @Security BearerAuth
func (k *K8sNodeAffinityHandler) ValidateNodeAffinity(ctx *gin.Context) {
	var req model.K8sNodeAffinityValidationRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.nodeAffinityService.ValidateNodeAffinity(ctx, &req)
	})
}

// GetNodeAffinityRecommendations 获取节点亲和性建议
// @Summary 获取节点亲和性配置建议
// @Description 基于集群节点标签和资源需求，智能推荐合适的节点亲和性配置
// @Tags 亲和性管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string true "命名空间"
// @Param resource_type query string true "资源类型（deployment、statefulset等）"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/node-affinity/recommendations [get]
// @Security BearerAuth
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
// @Summary 设置Pod亲和性规则
// @Description 为指定的K8s资源设置Pod亲和性或反亲和性规则，控制Pod间的调度关系
// @Tags 亲和性管理
// @Accept json
// @Produce json
// @Param request body model.K8sPodAffinityRequest true "Pod亲和性设置请求"
// @Success 200 {object} utils.ApiResponse "设置成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/pod-affinity [post]
// @Security BearerAuth
func (k *K8sPodAffinityHandler) SetPodAffinity(ctx *gin.Context) {
	var req model.K8sPodAffinityRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.podAffinityService.SetPodAffinity(ctx, &req)
	})
}

// GetPodAffinity 获取Pod亲和性
// @Summary 获取Pod亲和性配置
// @Description 获取指定K8s资源的Pod亲和性和反亲和性配置信息
// @Tags 亲和性管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string true "命名空间"
// @Param resource_type query string true "资源类型（deployment、statefulset等）"
// @Param resource_name query string true "资源名称"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/pod-affinity [get]
// @Security BearerAuth
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
// @Summary 更新Pod亲和性配置
// @Description 更新指定K8s资源的Pod亲和性和反亲和性规则配置
// @Tags 亲和性管理
// @Accept json
// @Produce json
// @Param request body model.K8sPodAffinityRequest true "Pod亲和性更新请求"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/pod-affinity [put]
// @Security BearerAuth
func (k *K8sPodAffinityHandler) UpdatePodAffinity(ctx *gin.Context) {
	var req model.K8sPodAffinityRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.podAffinityService.UpdatePodAffinity(ctx, &req)
	})
}

// DeletePodAffinity 删除Pod亲和性
// @Summary 删除Pod亲和性配置
// @Description 删除指定K8s资源的Pod亲和性和反亲和性配置，恢复默认调度策略
// @Tags 亲和性管理
// @Accept json
// @Produce json
// @Param request body model.K8sPodAffinityRequest true "Pod亲和性删除请求"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/pod-affinity [delete]
// @Security BearerAuth
func (k *K8sPodAffinityHandler) DeletePodAffinity(ctx *gin.Context) {
	var req model.K8sPodAffinityRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.podAffinityService.DeletePodAffinity(ctx, &req)
	})
}

// ValidatePodAffinity 验证Pod亲和性
// @Summary 验证Pod亲和性配置
// @Description 验证Pod亲和性和反亲和性配置的正确性，检查标签选择器和拓扑域设置
// @Tags 亲和性管理
// @Accept json
// @Produce json
// @Param request body model.K8sPodAffinityValidationRequest true "Pod亲和性验证请求"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "验证成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/pod-affinity/validate [post]
// @Security BearerAuth
func (k *K8sPodAffinityHandler) ValidatePodAffinity(ctx *gin.Context) {
	var req model.K8sPodAffinityValidationRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.podAffinityService.ValidatePodAffinity(ctx, &req)
	})
}

// GetTopologyDomains 获取拓扑域信息
// @Summary 获取拓扑域信息
// @Description 获取集群中可用的拓扑域信息，用于Pod亲和性配置参考
// @Tags 亲和性管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string false "命名空间（可选）"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/pod-affinity/topology-domains [get]
// @Security BearerAuth
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
// @Summary 获取亲和性可视化图表
// @Description 生成集群中Pod和节点亲和性配置的可视化图表，便于理解调度关系
// @Tags 亲和性管理
// @Accept json
// @Produce json
// @Param request body model.K8sAffinityVisualizationRequest true "亲和性可视化请求"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/affinity/visualization [post]
// @Security BearerAuth
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

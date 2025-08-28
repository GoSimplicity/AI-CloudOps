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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sStatefulSetHandler struct {
	logger             *zap.Logger
	statefulSetService service.StatefulSetService
}

func NewK8sStatefulSetHandler(logger *zap.Logger, statefulSetService service.StatefulSetService) *K8sStatefulSetHandler {
	return &K8sStatefulSetHandler{
		logger:             logger,
		statefulSetService: statefulSetService,
	}
}

func (h *K8sStatefulSetHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	statefulSets := k8sGroup.Group("/statefulsets")
	{
		statefulSets.GET("/list", h.GetStatefulSetList)                              // 获取StatefulSet列表
		statefulSets.GET("/:cluster_id/:namespace/:name", h.GetStatefulSet)          // 获取单个StatefulSet详情
		statefulSets.POST("/create", h.CreateStatefulSet)                            // 创建StatefulSet
		statefulSets.PUT("/update", h.UpdateStatefulSet)                             // 更新StatefulSet
		statefulSets.POST("/scale", h.ScaleStatefulSet)                              // 扩缩容StatefulSet
		statefulSets.DELETE("/:cluster_id/:namespace/:name", h.DeleteStatefulSet)    // 删除StatefulSet
		statefulSets.DELETE("/batch", h.BatchDeleteStatefulSets)                     // 批量删除StatefulSet
		statefulSets.GET("/:cluster_id/:namespace/:name/yaml", h.GetStatefulSetYAML) // 获取StatefulSet的YAML配置
	}
}

// GetStatefulSetList 获取StatefulSet列表
// @Summary 获取StatefulSet列表
// @Description 根据集群和命名空间获取StatefulSet列表
// @Tags 工作负载管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string false "命名空间"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sStatefulSet} "获取成功"
// @Router /api/k8s/statefulsets/list [get]
// @Security BearerAuth
func (h *K8sStatefulSetHandler) GetStatefulSetList(ctx *gin.Context) {
	var req model.K8sListReq

	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, "参数绑定错误: "+err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.statefulSetService.GetStatefulSetList(ctx, &req)
	})
}

// GetStatefulSet 获取单个StatefulSet详情
// @Summary 获取StatefulSet详情
// @Description 根据集群ID、命名空间和名称获取指定StatefulSet的详细信息
// @Tags 工作负载管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间"
// @Param name path string true "StatefulSet名称"
// @Success 200 {object} utils.ApiResponse{data=model.K8sStatefulSet} "获取成功"
// @Router /api/k8s/statefulsets/{cluster_id}/{namespace}/{name} [get]
// @Security BearerAuth
func (h *K8sStatefulSetHandler) GetStatefulSet(ctx *gin.Context) {
	var req model.K8sResourceIdentifierReq

	clusterIDStr := ctx.Param("cluster_id")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		utils.BadRequestError(ctx, "无效的集群ID: "+err.Error())
		return
	}
	req.ClusterID = clusterID
	req.Namespace = ctx.Param("namespace")
	req.ResourceName = ctx.Param("name")

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.statefulSetService.GetStatefulSet(ctx, &req)
	})
}

// CreateStatefulSet 创建StatefulSet
// @Summary 创建StatefulSet
// @Description 在指定集群和命名空间中创建新的StatefulSet
// @Tags 工作负载管理
// @Accept json
// @Produce json
// @Param request body model.StatefulSetCreateReq true "StatefulSet创建请求"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Router /api/k8s/statefulsets/create [post]
// @Security BearerAuth
func (h *K8sStatefulSetHandler) CreateStatefulSet(ctx *gin.Context) {
	var req model.StatefulSetCreateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.statefulSetService.CreateStatefulSet(ctx, &req)
	})
}

// UpdateStatefulSet 更新StatefulSet
// @Summary 更新StatefulSet
// @Description 更新指定的StatefulSet配置
// @Tags 工作负载管理
// @Accept json
// @Produce json
// @Param request body model.StatefulSetUpdateReq true "StatefulSet更新请求"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Router /api/k8s/statefulsets/update [put]
// @Security BearerAuth
func (h *K8sStatefulSetHandler) UpdateStatefulSet(ctx *gin.Context) {
	var req model.StatefulSetUpdateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.statefulSetService.UpdateStatefulSet(ctx, &req)
	})
}

// ScaleStatefulSet 扩缩容StatefulSet
// @Summary 扩缩容StatefulSet
// @Description 调整StatefulSet的副本数量
// @Tags 工作负载管理
// @Accept json
// @Produce json
// @Param request body model.StatefulSetScaleReq true "StatefulSet扩缩容请求"
// @Success 200 {object} utils.ApiResponse "扩缩容成功"
// @Router /api/k8s/statefulsets/scale [post]
// @Security BearerAuth
func (h *K8sStatefulSetHandler) ScaleStatefulSet(ctx *gin.Context) {
	var req model.StatefulSetScaleReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.statefulSetService.ScaleStatefulSet(ctx, &req)
	})
}

// DeleteStatefulSet 删除StatefulSet
// @Summary 删除StatefulSet
// @Description 删除指定的StatefulSet资源
// @Tags 工作负载管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间"
// @Param name path string true "StatefulSet名称"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Router /api/k8s/statefulsets/{cluster_id}/{namespace}/{name} [delete]
// @Security BearerAuth
func (h *K8sStatefulSetHandler) DeleteStatefulSet(ctx *gin.Context) {
	var req model.K8sResourceIdentifierReq

	clusterIDStr := ctx.Param("cluster_id")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		utils.BadRequestError(ctx, "无效的集群ID: "+err.Error())
		return
	}
	req.ClusterID = clusterID
	req.Namespace = ctx.Param("namespace")
	req.ResourceName = ctx.Param("name")

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.statefulSetService.DeleteStatefulSet(ctx, &req)
	})
}

// BatchDeleteStatefulSets 批量删除StatefulSet
// @Summary 批量删除StatefulSet
// @Description 批量删除指定命名空间中的多个StatefulSet
// @Tags 工作负载管理
// @Accept json
// @Produce json
// @Param request body model.K8sBatchDeleteReq true "批量删除请求"
// @Success 200 {object} utils.ApiResponse "批量删除成功"
// @Router /api/k8s/statefulsets/batch [delete]
// @Security BearerAuth
func (h *K8sStatefulSetHandler) BatchDeleteStatefulSets(ctx *gin.Context) {
	var req model.K8sBatchDeleteReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.statefulSetService.BatchDeleteStatefulSets(ctx, &req)
	})
}

// GetStatefulSetYAML 获取StatefulSet的YAML配置
// @Summary 获取StatefulSet的YAML配置
// @Description 获取指定StatefulSet的完整YAML配置文件
// @Tags 工作负载管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间"
// @Param name path string true "StatefulSet名称"
// @Success 200 {object} utils.ApiResponse{data=string} "获取成功"
// @Router /api/k8s/statefulsets/{cluster_id}/{namespace}/{name}/yaml [get]
// @Security BearerAuth
func (h *K8sStatefulSetHandler) GetStatefulSetYAML(ctx *gin.Context) {
	var req model.K8sResourceIdentifierReq

	clusterIDStr := ctx.Param("cluster_id")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		utils.BadRequestError(ctx, "无效的集群ID: "+err.Error())
		return
	}
	req.ClusterID = clusterID
	req.Namespace = ctx.Param("namespace")
	req.ResourceName = ctx.Param("name")

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.statefulSetService.GetStatefulSetYAML(ctx, &req)
	})
}

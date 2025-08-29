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

type K8sConfigMapHandler struct {
	logger           *zap.Logger
	configMapService service.ConfigMapService
}

func NewK8sConfigMapHandler(logger *zap.Logger, configMapService service.ConfigMapService) *K8sConfigMapHandler {
	return &K8sConfigMapHandler{
		logger:           logger,
		configMapService: configMapService,
	}
}

func (h *K8sConfigMapHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	configMaps := k8sGroup.Group("/configmaps")
	{
		configMaps.GET("/list", h.GetConfigMapList)                           // 获取ConfigMap列表
		configMaps.GET("/:cluster_id/:namespace/:name", h.GetConfigMap)       // 获取单个ConfigMap详情
		configMaps.POST("/create", h.CreateConfigMap)                         // 创建ConfigMap
		configMaps.PUT("/update", h.UpdateConfigMap)                          // 更新ConfigMap
		configMaps.DELETE("/:cluster_id/:namespace/:name", h.DeleteConfigMap) // 删除ConfigMap

		configMaps.GET("/:cluster_id/:namespace/:name/yaml", h.GetConfigMapYAML) // 获取ConfigMap的YAML配置
	}
}

// GetConfigMapList 获取ConfigMap列表
// @Summary 获取ConfigMap列表
// @Description 根据集群和命名空间获取ConfigMap列表，支持标签和字段选择器过滤
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string false "命名空间，为空时获取所有命名空间"
// @Param label_selector query string false "标签选择器"
// @Param field_selector query string false "字段选择器"
// @Param limit query int false "限制结果数量"
// @Param continue query string false "分页续订令牌"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sConfigMap} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/list [get]
// @Security BearerAuth
func (h *K8sConfigMapHandler) GetConfigMapList(ctx *gin.Context) {
	var req model.K8sListReq

	// 从查询参数中获取请求参数
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, "参数绑定错误: "+err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.configMapService.GetConfigMapList(ctx, &req)
	})
}

// GetConfigMap 获取单个ConfigMap详情
// @Summary 获取ConfigMap详情
// @Description 根据集群ID、命名空间和名称获取指定ConfigMap的详细信息
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间"
// @Param name path string true "ConfigMap名称"
// @Success 200 {object} utils.ApiResponse{data=model.K8sConfigMap} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 404 {object} utils.ApiResponse "ConfigMap不存在"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/{cluster_id}/{namespace}/{name} [get]
// @Security BearerAuth
func (h *K8sConfigMapHandler) GetConfigMap(ctx *gin.Context) {
	var req model.K8sResourceIdentifierReq

	// 从路径参数中获取请求参数
	clusterIDStr := ctx.Param("cluster_id")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		utils.BadRequestError(ctx, "无效的集群ID: "+err.Error())
		return
	}
	req.ClusterID = clusterID

	req.Namespace = ctx.Param("namespace")
	req.ResourceName = ctx.Param("name")

	// 验证必要参数
	if req.Namespace == "" || req.ResourceName == "" {
		utils.BadRequestError(ctx, "命名空间和ConfigMap名称不能为空")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.configMapService.GetConfigMap(ctx, &req)
	})
}

// CreateConfigMap 创建ConfigMap
// @Summary 创建ConfigMap
// @Description 在指定集群和命名空间中创建新的ConfigMap
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param request body model.ConfigMapCreateReq true "ConfigMap创建请求"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 409 {object} utils.ApiResponse "ConfigMap已存在"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/create [post]
// @Security BearerAuth
func (h *K8sConfigMapHandler) CreateConfigMap(ctx *gin.Context) {
	var req model.ConfigMapCreateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.configMapService.CreateConfigMap(ctx, &req)
	})
}

// UpdateConfigMap 更新ConfigMap
// @Summary 更新ConfigMap
// @Description 更新指定的ConfigMap配置数据
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param request body model.ConfigMapUpdateReq true "ConfigMap更新请求"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 404 {object} utils.ApiResponse "ConfigMap不存在"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/update [put]
// @Security BearerAuth
func (h *K8sConfigMapHandler) UpdateConfigMap(ctx *gin.Context) {
	var req model.ConfigMapUpdateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.configMapService.UpdateConfigMap(ctx, &req)
	})
}

// DeleteConfigMap 删除ConfigMap
// @Summary 删除ConfigMap
// @Description 删除指定的ConfigMap资源
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间"
// @Param name path string true "ConfigMap名称"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 404 {object} utils.ApiResponse "ConfigMap不存在"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/{cluster_id}/{namespace}/{name} [delete]
// @Security BearerAuth
func (h *K8sConfigMapHandler) DeleteConfigMap(ctx *gin.Context) {
	var req model.K8sResourceIdentifierReq

	// 从路径参数中获取请求参数
	clusterIDStr := ctx.Param("cluster_id")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		utils.BadRequestError(ctx, "无效的集群ID: "+err.Error())
		return
	}
	req.ClusterID = clusterID

	req.Namespace = ctx.Param("namespace")
	req.ResourceName = ctx.Param("name")

	// 验证必要参数
	if req.Namespace == "" || req.ResourceName == "" {
		utils.BadRequestError(ctx, "命名空间和ConfigMap名称不能为空")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.configMapService.DeleteConfigMap(ctx, &req)
	})
}

// GetConfigMapYAML 获取ConfigMap的YAML配置
// @Summary 获取ConfigMap的YAML配置
// @Description 获取指定ConfigMap的完整YAML配置文件
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间"
// @Param name path string true "ConfigMap名称"
// @Success 200 {object} utils.ApiResponse{data=string} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 404 {object} utils.ApiResponse "ConfigMap不存在"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/{cluster_id}/{namespace}/{name}/yaml [get]
// @Security BearerAuth
func (h *K8sConfigMapHandler) GetConfigMapYAML(ctx *gin.Context) {
	var req model.K8sResourceIdentifierReq

	// 从路径参数中获取请求参数
	clusterIDStr := ctx.Param("cluster_id")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		utils.BadRequestError(ctx, "无效的集群ID: "+err.Error())
		return
	}
	req.ClusterID = clusterID

	req.Namespace = ctx.Param("namespace")
	req.ResourceName = ctx.Param("name")

	// 验证必要参数
	if req.Namespace == "" || req.ResourceName == "" {
		utils.BadRequestError(ctx, "命名空间和ConfigMap名称不能为空")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.configMapService.GetConfigMapYAML(ctx, &req)
	})
}

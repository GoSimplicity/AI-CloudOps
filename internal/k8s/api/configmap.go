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
	{
		k8sGroup.GET("/configmaps/list", h.GetConfigMapList)                              // 获取ConfigMap列表
		k8sGroup.GET("/configmaps/:cluster_id/:namespace/:name", h.GetConfigMap)          // 获取单个ConfigMap详情
		k8sGroup.POST("/configmaps/create", h.CreateConfigMap)                            // 创建ConfigMap
		k8sGroup.PUT("/configmaps/update", h.UpdateConfigMap)                             // 更新ConfigMap
		k8sGroup.DELETE("/configmaps/:cluster_id/:namespace/:name", h.DeleteConfigMap)    // 删除ConfigMap
		k8sGroup.GET("/configmaps/:cluster_id/:namespace/:name/yaml", h.GetConfigMapYAML) // 获取ConfigMap的YAML配置
	}
}

// GetConfigMapList 获取ConfigMap列表
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
func (h *K8sConfigMapHandler) CreateConfigMap(ctx *gin.Context) {
	var req model.ConfigMapCreateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.configMapService.CreateConfigMap(ctx, &req)
	})
}

// UpdateConfigMap 更新ConfigMap
func (h *K8sConfigMapHandler) UpdateConfigMap(ctx *gin.Context) {
	var req model.ConfigMapUpdateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.configMapService.UpdateConfigMap(ctx, &req)
	})
}

// DeleteConfigMap 删除ConfigMap
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

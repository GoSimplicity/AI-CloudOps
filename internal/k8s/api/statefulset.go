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
	{
		k8sGroup.GET("/statefulsets/list", h.GetStatefulSetList)                              // 获取StatefulSet列表
		k8sGroup.GET("/statefulsets/:cluster_id/:namespace/:name", h.GetStatefulSet)          // 获取单个StatefulSet详情
		k8sGroup.POST("/statefulsets/create", h.CreateStatefulSet)                            // 创建StatefulSet
		k8sGroup.PUT("/statefulsets/update", h.UpdateStatefulSet)                             // 更新StatefulSet
		k8sGroup.POST("/statefulsets/scale", h.ScaleStatefulSet)                              // 扩缩容StatefulSet
		k8sGroup.DELETE("/statefulsets/:cluster_id/:namespace/:name", h.DeleteStatefulSet)    // 删除StatefulSet
		k8sGroup.GET("/statefulsets/:cluster_id/:namespace/:name/yaml", h.GetStatefulSetYAML) // 获取StatefulSet的YAML配置
	}
}

// GetStatefulSetList 获取StatefulSet列表
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
func (h *K8sStatefulSetHandler) CreateStatefulSet(ctx *gin.Context) {
	var req model.StatefulSetCreateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.statefulSetService.CreateStatefulSet(ctx, &req)
	})
}

// UpdateStatefulSet 更新StatefulSet
func (h *K8sStatefulSetHandler) UpdateStatefulSet(ctx *gin.Context) {
	var req model.StatefulSetUpdateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.statefulSetService.UpdateStatefulSet(ctx, &req)
	})
}

// ScaleStatefulSet 扩缩容StatefulSet
func (h *K8sStatefulSetHandler) ScaleStatefulSet(ctx *gin.Context) {
	var req model.StatefulSetScaleReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.statefulSetService.ScaleStatefulSet(ctx, &req)
	})
}

// DeleteStatefulSet 删除StatefulSet
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

// GetStatefulSetYAML 获取StatefulSet的YAML配置
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

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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sLimitRangeHandler struct {
	l                  *zap.Logger
	limitRangeService  admin.LimitRangeService
}

func NewK8sLimitRangeHandler(l *zap.Logger, limitRangeService admin.LimitRangeService) *K8sLimitRangeHandler {
	return &K8sLimitRangeHandler{
		l:                  l,
		limitRangeService:  limitRangeService,
	}
}

func (k *K8sLimitRangeHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	limitRanges := k8sGroup.Group("/limitranges")
	{
		limitRanges.POST("/create", k.CreateLimitRange)                // 创建 LimitRange
		limitRanges.GET("/list/:id", k.ListLimitRanges)               // 获取 LimitRange 列表
		limitRanges.GET("/:id", k.GetLimitRange)                      // 获取 LimitRange 详情
		limitRanges.PUT("/:id", k.UpdateLimitRange)                   // 更新 LimitRange
		limitRanges.DELETE("/:id", k.DeleteLimitRange)                // 删除 LimitRange
		limitRanges.GET("/:id/yaml", k.GetLimitRangeYaml)             // 获取 LimitRange YAML
		limitRanges.POST("/batch_delete", k.BatchDeleteLimitRanges)   // 批量删除 LimitRange
	}
}

// CreateLimitRange 创建 LimitRange
func (k *K8sLimitRangeHandler) CreateLimitRange(ctx *gin.Context) {
	var req model.K8sLimitRangeRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.limitRangeService.CreateLimitRange(ctx, &req)
	})
}

// ListLimitRanges 获取 LimitRange 列表
func (k *K8sLimitRangeHandler) ListLimitRanges(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.limitRangeService.ListLimitRanges(ctx, id, namespace)
	})
}

// GetLimitRange 获取 LimitRange 详情
func (k *K8sLimitRangeHandler) GetLimitRange(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	limitRangeName := ctx.Query("name")
	if limitRangeName == "" {
		utils.BadRequestError(ctx, "缺少 'name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.limitRangeService.GetLimitRange(ctx, id, namespace, limitRangeName)
	})
}

// UpdateLimitRange 更新 LimitRange
func (k *K8sLimitRangeHandler) UpdateLimitRange(ctx *gin.Context) {
	var req model.K8sLimitRangeRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.limitRangeService.UpdateLimitRange(ctx, &req)
	})
}

// DeleteLimitRange 删除 LimitRange
func (k *K8sLimitRangeHandler) DeleteLimitRange(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	limitRangeName := ctx.Query("name")
	if limitRangeName == "" {
		utils.BadRequestError(ctx, "缺少 'name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.limitRangeService.DeleteLimitRange(ctx, id, namespace, limitRangeName)
	})
}

// GetLimitRangeYaml 获取 LimitRange YAML
func (k *K8sLimitRangeHandler) GetLimitRangeYaml(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	limitRangeName := ctx.Query("name")
	if limitRangeName == "" {
		utils.BadRequestError(ctx, "缺少 'name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.limitRangeService.GetLimitRangeYaml(ctx, id, namespace, limitRangeName)
	})
}

// BatchDeleteLimitRanges 批量删除 LimitRange
func (k *K8sLimitRangeHandler) BatchDeleteLimitRanges(ctx *gin.Context) {
	var req model.K8sLimitRangeRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.limitRangeService.BatchDeleteLimitRanges(ctx, req.ClusterID, req.Namespace, req.LimitRangeNames)
	})
}
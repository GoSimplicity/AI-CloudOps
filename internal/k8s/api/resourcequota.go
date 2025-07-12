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

type K8sResourceQuotaHandler struct {
	l                     *zap.Logger
	resourceQuotaService  admin.ResourceQuotaService
}

func NewK8sResourceQuotaHandler(l *zap.Logger, resourceQuotaService admin.ResourceQuotaService) *K8sResourceQuotaHandler {
	return &K8sResourceQuotaHandler{
		l:                     l,
		resourceQuotaService:  resourceQuotaService,
	}
}

func (k *K8sResourceQuotaHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	resourceQuotas := k8sGroup.Group("/resourcequotas")
	{
		resourceQuotas.POST("/create", k.CreateResourceQuota)                // 创建 ResourceQuota
		resourceQuotas.GET("/list/:id", k.ListResourceQuotas)               // 获取 ResourceQuota 列表
		resourceQuotas.GET("/:id", k.GetResourceQuota)                      // 获取 ResourceQuota 详情
		resourceQuotas.PUT("/:id", k.UpdateResourceQuota)                   // 更新 ResourceQuota
		resourceQuotas.DELETE("/:id", k.DeleteResourceQuota)                // 删除 ResourceQuota
		resourceQuotas.GET("/:id/usage", k.GetResourceQuotaUsage)           // 获取配额使用情况
		resourceQuotas.GET("/:id/yaml", k.GetResourceQuotaYaml)             // 获取 ResourceQuota YAML
		resourceQuotas.POST("/batch_delete", k.BatchDeleteResourceQuotas)   // 批量删除 ResourceQuota
	}
}

// CreateResourceQuota 创建 ResourceQuota
func (k *K8sResourceQuotaHandler) CreateResourceQuota(ctx *gin.Context) {
	var req model.K8sResourceQuotaRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.resourceQuotaService.CreateResourceQuota(ctx, &req)
	})
}

// ListResourceQuotas 获取 ResourceQuota 列表
func (k *K8sResourceQuotaHandler) ListResourceQuotas(ctx *gin.Context) {
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
		return k.resourceQuotaService.ListResourceQuotas(ctx, id, namespace)
	})
}

// GetResourceQuota 获取 ResourceQuota 详情
func (k *K8sResourceQuotaHandler) GetResourceQuota(ctx *gin.Context) {
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

	resourceQuotaName := ctx.Query("name")
	if resourceQuotaName == "" {
		utils.BadRequestError(ctx, "缺少 'name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.resourceQuotaService.GetResourceQuota(ctx, id, namespace, resourceQuotaName)
	})
}

// UpdateResourceQuota 更新 ResourceQuota
func (k *K8sResourceQuotaHandler) UpdateResourceQuota(ctx *gin.Context) {
	var req model.K8sResourceQuotaRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.resourceQuotaService.UpdateResourceQuota(ctx, &req)
	})
}

// DeleteResourceQuota 删除 ResourceQuota
func (k *K8sResourceQuotaHandler) DeleteResourceQuota(ctx *gin.Context) {
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

	resourceQuotaName := ctx.Query("name")
	if resourceQuotaName == "" {
		utils.BadRequestError(ctx, "缺少 'name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.resourceQuotaService.DeleteResourceQuota(ctx, id, namespace, resourceQuotaName)
	})
}

// GetResourceQuotaUsage 获取配额使用情况
func (k *K8sResourceQuotaHandler) GetResourceQuotaUsage(ctx *gin.Context) {
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

	resourceQuotaName := ctx.Query("name")
	if resourceQuotaName == "" {
		utils.BadRequestError(ctx, "缺少 'name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.resourceQuotaService.GetResourceQuotaUsage(ctx, id, namespace, resourceQuotaName)
	})
}

// GetResourceQuotaYaml 获取 ResourceQuota YAML
func (k *K8sResourceQuotaHandler) GetResourceQuotaYaml(ctx *gin.Context) {
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

	resourceQuotaName := ctx.Query("name")
	if resourceQuotaName == "" {
		utils.BadRequestError(ctx, "缺少 'name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.resourceQuotaService.GetResourceQuotaYaml(ctx, id, namespace, resourceQuotaName)
	})
}

// BatchDeleteResourceQuotas 批量删除 ResourceQuota
func (k *K8sResourceQuotaHandler) BatchDeleteResourceQuotas(ctx *gin.Context) {
	var req model.K8sResourceQuotaRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.resourceQuotaService.BatchDeleteResourceQuotas(ctx, req.ClusterID, req.Namespace, req.ResourceQuotaNames)
	})
}
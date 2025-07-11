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

type K8sEndpointHandler struct {
	l               *zap.Logger
	endpointService admin.EndpointService
}

func NewK8sEndpointHandler(l *zap.Logger, endpointService admin.EndpointService) *K8sEndpointHandler {
	return &K8sEndpointHandler{
		l:               l,
		endpointService: endpointService,
	}
}

func (k *K8sEndpointHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	endpoints := k8sGroup.Group("/endpoints")
	{
		endpoints.GET("/:id", k.GetEndpointsByNamespace)          // 根据命名空间获取 Endpoint 列表
		endpoints.POST("/create", k.CreateEndpoint)               // 创建 Endpoint
		endpoints.DELETE("/delete/:id", k.DeleteEndpoint)         // 删除指定 Endpoint
		endpoints.DELETE("/batch_delete", k.BatchDeleteEndpoint)  // 批量删除 Endpoint
		endpoints.GET("/:id/yaml", k.GetEndpointYaml)            // 获取 Endpoint YAML 配置
		endpoints.GET("/:id/status", k.GetEndpointStatus)        // 获取 Endpoint 状态
		endpoints.GET("/:id/health", k.CheckEndpointHealth)      // 检查 Endpoint 健康状态
		endpoints.GET("/:id/service", k.GetEndpointService)      // 获取关联的 Service
	}
}

// GetEndpointsByNamespace 根据命名空间获取 Endpoint 列表
func (k *K8sEndpointHandler) GetEndpointsByNamespace(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.endpointService.GetEndpointsByNamespace(ctx, id, namespace)
	})
}

// CreateEndpoint 创建 Endpoint
func (k *K8sEndpointHandler) CreateEndpoint(ctx *gin.Context) {
	var req model.K8sEndpointRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.endpointService.CreateEndpoint(ctx, &req)
	})
}

// BatchDeleteEndpoint 批量删除 Endpoint
func (k *K8sEndpointHandler) BatchDeleteEndpoint(ctx *gin.Context) {
	var req model.K8sEndpointRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.endpointService.BatchDeleteEndpoint(ctx, req.ClusterID, req.Namespace, req.EndpointNames)
	})
}

// GetEndpointYaml 获取 Endpoint 的 YAML 配置
func (k *K8sEndpointHandler) GetEndpointYaml(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	endpointName := ctx.Query("endpoint_name")
	if endpointName == "" {
		k.l.Error("缺少必需的 endpoint_name 参数")
		utils.BadRequestError(ctx, "缺少 'endpoint_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.endpointService.GetEndpointYaml(ctx, id, namespace, endpointName)
	})
}

// DeleteEndpoint 删除指定的 Endpoint
func (k *K8sEndpointHandler) DeleteEndpoint(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	endpointName := ctx.Query("endpoint_name")
	if endpointName == "" {
		k.l.Error("缺少必需的 endpoint_name 参数")
		utils.BadRequestError(ctx, "缺少 'endpoint_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.endpointService.DeleteEndpoint(ctx, id, namespace, endpointName)
	})
}

// GetEndpointStatus 获取 Endpoint 状态
func (k *K8sEndpointHandler) GetEndpointStatus(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	endpointName := ctx.Query("endpoint_name")
	if endpointName == "" {
		k.l.Error("缺少必需的 endpoint_name 参数")
		utils.BadRequestError(ctx, "缺少 'endpoint_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.endpointService.GetEndpointStatus(ctx, id, namespace, endpointName)
	})
}

// CheckEndpointHealth 检查 Endpoint 健康状态
func (k *K8sEndpointHandler) CheckEndpointHealth(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	endpointName := ctx.Query("endpoint_name")
	if endpointName == "" {
		k.l.Error("缺少必需的 endpoint_name 参数")
		utils.BadRequestError(ctx, "缺少 'endpoint_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.endpointService.CheckEndpointHealth(ctx, id, namespace, endpointName)
	})
}

// GetEndpointService 获取 Endpoint 关联的 Service
func (k *K8sEndpointHandler) GetEndpointService(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	endpointName := ctx.Query("endpoint_name")
	if endpointName == "" {
		k.l.Error("缺少必需的 endpoint_name 参数")
		utils.BadRequestError(ctx, "缺少 'endpoint_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.endpointService.GetEndpointService(ctx, id, namespace, endpointName)
	})
}
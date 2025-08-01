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
// @Summary 获取 Endpoint 列表
// @Description 根据集群ID和命名空间获取Endpoint资源列表
// @Tags Kubernetes-Endpoint
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sEndpointStatus} "成功获取Endpoint列表"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/endpoints/{id} [get]
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
// @Summary 创建 Endpoint
// @Description 在指定命名空间中创建新的Endpoint资源
// @Tags Kubernetes-Endpoint
// @Accept json
// @Produce json
// @Param request body model.K8sEndpointRequest true "Endpoint创建请求"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/endpoints/create [post]
func (k *K8sEndpointHandler) CreateEndpoint(ctx *gin.Context) {
	var req model.K8sEndpointRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.endpointService.CreateEndpoint(ctx, &req)
	})
}

// BatchDeleteEndpoint 批量删除 Endpoint
// @Summary 批量删除 Endpoint
// @Description 批量删除指定命名空间中的多个Endpoint资源
// @Tags Kubernetes-Endpoint
// @Accept json
// @Produce json
// @Param request body model.K8sEndpointRequest true "批量删除请求"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/endpoints/batch_delete [delete]
func (k *K8sEndpointHandler) BatchDeleteEndpoint(ctx *gin.Context) {
	var req model.K8sEndpointRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.endpointService.BatchDeleteEndpoint(ctx, req.ClusterID, req.Namespace, req.EndpointNames)
	})
}

// GetEndpointYaml 获取 Endpoint 的 YAML 配置
// @Summary 获取 Endpoint YAML
// @Description 获取指定Endpoint的YAML配置文件
// @Tags Kubernetes-Endpoint
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param endpoint_name query string true "Endpoint名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=string} "成功获取YAML配置"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/endpoints/{id}/yaml [get]
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
// @Summary 删除 Endpoint
// @Description 删除指定命名空间中的Endpoint资源
// @Tags Kubernetes-Endpoint
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param endpoint_name query string true "Endpoint名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/endpoints/delete/{id} [delete]
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
// @Summary 获取 Endpoint 状态
// @Description 获取指定Endpoint的状态信息
// @Tags Kubernetes-Endpoint
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param endpoint_name query string true "Endpoint名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse "成功获取状态"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/endpoints/{id}/status [get]
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
// @Summary 检查 Endpoint 健康状态
// @Description 检查指定Endpoint的健康状态和可用性
// @Tags Kubernetes-Endpoint
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param endpoint_name query string true "Endpoint名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "成功获取健康状态"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/endpoints/{id}/health [get]
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
// @Summary 获取关联的 Service
// @Description 获取指定Endpoint关联的Service信息
// @Tags Kubernetes-Endpoint
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param endpoint_name query string true "Endpoint名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "成功获取关联Service"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/endpoints/{id}/service [get]
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
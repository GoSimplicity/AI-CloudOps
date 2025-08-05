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

type K8sSvcHandler struct {
	l          *zap.Logger
	svcService admin.SvcService
}

func NewK8sSvcHandler(l *zap.Logger, svcService admin.SvcService) *K8sSvcHandler {
	return &K8sSvcHandler{
		l:          l,
		svcService: svcService,
	}
}

func (k *K8sSvcHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	services := k8sGroup.Group("/services")
	{
		services.GET("/:id", k.GetServiceListByNamespace)       // 根据命名空间获取 Service 列表
		services.GET("/:id/:svcName/yaml", k.GetServiceYaml)    // 获取指定 Service 的 YAML 配置
		services.POST("/update", k.UpdateService)               // 更新指定 Name 的 Service
		services.DELETE("/delete/:id", k.DeleteService)         // 删除指定 Service
		services.DELETE("/batch_delete", k.BatchDeleteServices) // 批量删除 Service

	}
}

// GetServiceListByNamespace 根据命名空间获取 Service 列表
// @Summary 获取命名空间下的Service列表
// @Description 根据集群ID和命名空间获取该命名空间下的所有Service资源列表
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param namespace query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse{data=[]interface{}} "获取Service列表成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/services/{id} [get]
// @Security BearerAuth
func (k *K8sSvcHandler) GetServiceListByNamespace(ctx *gin.Context) {
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
		return k.svcService.GetServicesByNamespace(ctx, id, namespace)
	})
}

// GetServiceYaml 获取 Service 的 YAML 配置
// @Summary 获取Service的YAML配置
// @Description 根据集群ID、命名空间和Service名称获取指定Service的YAML配置信息
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param svcName path string true "Service名称"
// @Param namespace query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "获取Service YAML配置成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/services/{id}/{svcName}/yaml [get]
// @Security BearerAuth
func (k *K8sSvcHandler) GetServiceYaml(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	svcName := ctx.Param("svcName")
	if svcName == "" {
		utils.BadRequestError(ctx, "缺少 'serviceName' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.svcService.GetServiceYaml(ctx, id, namespace, svcName)
	})
}

// UpdateService 更新指定 Name 的 Service
// @Summary 更新Service资源
// @Description 根据提供的Service配置信息更新指定的Service资源
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param serviceRequest body model.K8sServiceRequest true "Service更新请求参数"
// @Success 200 {object} utils.ApiResponse "更新Service成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/services/update [post]
// @Security BearerAuth
func (k *K8sSvcHandler) UpdateService(ctx *gin.Context) {
	var req model.K8sServiceRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.svcService.UpdateService(ctx, &req)
	})
}

// DeleteService 删除指定 Service
// @Summary 删除Service资源
// @Description 根据集群ID、命名空间和Service名称删除指定的Service资源
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param namespace query string true "命名空间名称"
// @Param svcName query string true "Service名称"
// @Success 200 {object} utils.ApiResponse "删除Service成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/services/delete/{id} [delete]
// @Security BearerAuth
func (k *K8sSvcHandler) DeleteService(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace := ctx.Query("namespace")
	svcName := ctx.Query("svcName")
	if namespace == "" || svcName == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 或 'svcName' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.svcService.DeleteService(ctx, id, namespace, svcName)
	})

}

// BatchDeleteServices 批量删除 Service
// @Summary 批量删除Service资源
// @Description 根据提供的Service名称列表批量删除指定命名空间下的多个Service资源
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param serviceRequest body model.K8sServiceRequest true "批量删除Service请求参数"
// @Success 200 {object} utils.ApiResponse "批量删除Service成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/services/batch_delete [delete]
// @Security BearerAuth
func (k *K8sSvcHandler) BatchDeleteServices(ctx *gin.Context) {
	var req model.K8sServiceRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.svcService.BatchDeleteService(ctx, req.ClusterId, req.Namespace, req.ServiceNames)
	})
}

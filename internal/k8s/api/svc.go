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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sSvcHandler struct {
	logger     *zap.Logger
	svcService service.SvcService
}

func NewK8sSvcHandler(logger *zap.Logger, svcService service.SvcService) *K8sSvcHandler {
	return &K8sSvcHandler{
		logger:     logger,
		svcService: svcService,
	}
}

func (k *K8sSvcHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/services/:id", k.GetServiceListByNamespace)    // 根据命名空间获取 Service 列表
		k8sGroup.GET("/services/:id/:svcName/yaml", k.GetServiceYaml) // 获取指定 Service 的 YAML 配置
		k8sGroup.POST("/services/update", k.UpdateService)            // 更新指定 Name 的 Service
		k8sGroup.DELETE("/services/delete/:id", k.DeleteService)      // 删除指定 Service
	}
}

// GetServiceListByNamespace 根据命名空间获取 Service 列表
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
func (k *K8sSvcHandler) UpdateService(ctx *gin.Context) {
	var req model.K8sServiceReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.svcService.UpdateService(ctx, &req)
	})
}

// DeleteService 删除指定 Service
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

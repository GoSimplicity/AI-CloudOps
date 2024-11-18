package api

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

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sConfigMapHandler struct {
	configmapService admin.ConfigMapService
	l                *zap.Logger
}

func NewK8sConfigMapHandler(l *zap.Logger, configmapService admin.ConfigMapService) *K8sConfigMapHandler {
	return &K8sConfigMapHandler{
		l:                l,
		configmapService: configmapService,
	}
}

func (k *K8sConfigMapHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	configMaps := k8sGroup.Group("/configmaps")
	{
		configMaps.GET("/:id", k.GetConfigMapListByNamespace)       // 根据命名空间获取 ConfigMap 列表
		configMaps.GET("/:id/yaml", k.GetConfigMapYaml)             // 获取指定 ConfigMap 的 YAML 配置
		configMaps.POST("/update", k.UpdateConfigMap)               // 更新指定 Name 的 ConfigMap
		configMaps.DELETE("/delete/:id", k.DeleteConfigMaps)        // 删除 ConfigMap
		configMaps.DELETE("/batch_delete", k.BatchDeleteConfigMaps) // 批量删除 ConfigMap
	}
}

// GetConfigMapListByNamespace 根据命名空间获取 ConfigMap 列表
func (k *K8sConfigMapHandler) GetConfigMapListByNamespace(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.configmapService.GetConfigMapsByNamespace(ctx, id, namespace)
	})
}

// UpdateConfigMap 更新指定 Name 的 ConfigMap
func (k *K8sConfigMapHandler) UpdateConfigMap(ctx *gin.Context) {
	var req model.K8sConfigMapRequest

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.configmapService.UpdateConfigMap(ctx, &req)
	})
}

// GetConfigMapYaml 获取 ConfigMap 的 YAML 配置
func (k *K8sConfigMapHandler) GetConfigMapYaml(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	configMapName := ctx.Query("configmap_name")
	if configMapName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'configmap_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.configmapService.GetConfigMapYaml(ctx, id, namespace, configMapName)
	})
}

// BatchDeleteConfigMaps 批量删除 ConfigMap
func (k *K8sConfigMapHandler) BatchDeleteConfigMaps(ctx *gin.Context) {
	var req model.K8sConfigMapRequest

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.configmapService.BatchDeleteConfigMap(ctx, req.ClusterId, req.Namespace, req.ConfigMapNames)
	})
}

func (k *K8sConfigMapHandler) DeleteConfigMaps(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	configMapName := ctx.Query("configmap_name")
	if configMapName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'configmap_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.configmapService.DeleteConfigMap(ctx, id, namespace, configMapName)
	})
}

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
		
		// 版本管理
		configMaps.POST("/versions/create", k.CreateConfigMapVersion)         // 创建 ConfigMap 版本
		configMaps.GET("/:id/versions", k.GetConfigMapVersions)               // 获取 ConfigMap 版本列表
		configMaps.GET("/:id/versions/detail", k.GetConfigMapVersion)         // 获取特定版本的 ConfigMap
		configMaps.DELETE("/:id/versions/delete", k.DeleteConfigMapVersion)   // 删除 ConfigMap 版本
		
		// 热更新
		configMaps.POST("/hot_reload", k.HotReloadConfigMap)                  // 热重载 ConfigMap
		
		// 回滚
		configMaps.POST("/rollback", k.RollbackConfigMap)                     // 回滚 ConfigMap
	}
}

// GetConfigMapListByNamespace 根据命名空间获取 ConfigMap 列表
// @Summary 获取ConfigMap列表
// @Description 根据指定的集群ID和命名空间查询所有的ConfigMap资源
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param namespace query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse{data=[]interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/{id} [get]
// @Security BearerAuth
func (k *K8sConfigMapHandler) GetConfigMapListByNamespace(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.configmapService.GetConfigMapsByNamespace(ctx, id, namespace)
	})
}

// UpdateConfigMap 更新指定 Name 的 ConfigMap
// @Summary 更新ConfigMap配置
// @Description 修改指定ConfigMap的数据内容，支持键值对的增删改查
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param request body model.K8sConfigMapRequest true "ConfigMap 更新信息"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/update [post]
// @Security BearerAuth
func (k *K8sConfigMapHandler) UpdateConfigMap(ctx *gin.Context) {
	var req model.K8sConfigMapRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.configmapService.UpdateConfigMap(ctx, &req)
	})
}

// GetConfigMapYaml 获取 ConfigMap 的 YAML 配置
// @Summary 获取ConfigMap的YAML配置
// @Description 以YAML格式返回指定ConfigMap的完整配置信息
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param configmap_name query string true "ConfigMap 名称"
// @Param namespace query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse{data=string} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/{id}/yaml [get]
// @Security BearerAuth
func (k *K8sConfigMapHandler) GetConfigMapYaml(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	configMapName := ctx.Query("configmap_name")
	if configMapName == "" {
		utils.BadRequestError(ctx, "缺少 'configmap_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.configmapService.GetConfigMapYaml(ctx, id, namespace, configMapName)
	})
}

// BatchDeleteConfigMaps 批量删除 ConfigMap
// @Summary 批量删除ConfigMap
// @Description 同时删除指定命名空间下的多个ConfigMap资源
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param request body model.K8sConfigMapRequest true "批量删除请求"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/batch_delete [delete]
// @Security BearerAuth
func (k *K8sConfigMapHandler) BatchDeleteConfigMaps(ctx *gin.Context) {
	var req model.K8sConfigMapRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.configmapService.BatchDeleteConfigMap(ctx, req.ClusterId, req.Namespace, req.ConfigMapNames)
	})
}

// DeleteConfigMaps 删除 ConfigMap
// @Summary 删除单个ConfigMap
// @Description 删除指定命名空间下的单个ConfigMap资源
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param configmap_name query string true "ConfigMap 名称"
// @Param namespace query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/delete/{id} [delete]
// @Security BearerAuth
func (k *K8sConfigMapHandler) DeleteConfigMaps(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	configMapName := ctx.Query("configmap_name")
	if configMapName == "" {
		utils.BadRequestError(ctx, "缺少 'configmap_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.configmapService.DeleteConfigMap(ctx, id, namespace, configMapName)
	})
}

// CreateConfigMapVersion 创建 ConfigMap 版本
// @Summary 创建ConfigMap版本快照
// @Description 为指定的ConfigMap创建一个版本快照，用于配置的历史管理和回滚
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param request body model.K8sConfigMapVersionRequest true "版本创建请求"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/versions/create [post]
// @Security BearerAuth
func (k *K8sConfigMapHandler) CreateConfigMapVersion(ctx *gin.Context) {
	var req model.K8sConfigMapVersionRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.configmapService.CreateConfigMapVersion(ctx, &req)
	})
}

// GetConfigMapVersions 获取 ConfigMap 版本列表
// @Summary 获取ConfigMap版本历史
// @Description 获取指定ConfigMap的所有历史版本记录，包括创建时间和版本说明
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param namespace query string true "命名空间名称"
// @Param configmap_name query string true "ConfigMap 名称"
// @Success 200 {object} utils.ApiResponse{data=[]interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/{id}/versions [get]
// @Security BearerAuth
func (k *K8sConfigMapHandler) GetConfigMapVersions(ctx *gin.Context) {
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

	configMapName := ctx.Query("configmap_name")
	if configMapName == "" {
		k.l.Error("缺少必需的 configmap_name 参数")
		utils.BadRequestError(ctx, "缺少 'configmap_name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.configmapService.GetConfigMapVersions(ctx, id, namespace, configMapName)
	})
}

// GetConfigMapVersion 获取特定版本的 ConfigMap
// @Summary 获取ConfigMap特定版本
// @Description 获取指定版本号的ConfigMap详细配置信息和内容
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param namespace query string true "命名空间名称"
// @Param configmap_name query string true "ConfigMap 名称"
// @Param version query string true "版本号"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/{id}/versions/detail [get]
// @Security BearerAuth
func (k *K8sConfigMapHandler) GetConfigMapVersion(ctx *gin.Context) {
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

	configMapName := ctx.Query("configmap_name")
	if configMapName == "" {
		k.l.Error("缺少必需的 configmap_name 参数")
		utils.BadRequestError(ctx, "缺少 'configmap_name' 参数")
		return
	}

	version := ctx.Query("version")
	if version == "" {
		k.l.Error("缺少必需的 version 参数")
		utils.BadRequestError(ctx, "缺少 'version' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.configmapService.GetConfigMapVersion(ctx, id, namespace, configMapName, version)
	})
}

// DeleteConfigMapVersion 删除 ConfigMap 版本
// @Summary 删除ConfigMap版本
// @Description 删除指定版本的ConfigMap历史记录（不影响当前活跃版本）
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param namespace query string true "命名空间名称"
// @Param configmap_name query string true "ConfigMap 名称"
// @Param version query string true "版本号"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/{id}/versions/delete [delete]
// @Security BearerAuth
func (k *K8sConfigMapHandler) DeleteConfigMapVersion(ctx *gin.Context) {
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

	configMapName := ctx.Query("configmap_name")
	if configMapName == "" {
		k.l.Error("缺少必需的 configmap_name 参数")
		utils.BadRequestError(ctx, "缺少 'configmap_name' 参数")
		return
	}

	version := ctx.Query("version")
	if version == "" {
		k.l.Error("缺少必需的 version 参数")
		utils.BadRequestError(ctx, "缺少 'version' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.configmapService.DeleteConfigMapVersion(ctx, id, namespace, configMapName, version)
	})
}

// HotReloadConfigMap 热重载 ConfigMap
// @Summary 热重载ConfigMap配置
// @Description 对使用指定ConfigMap的Pod进行热重载，使更新后的配置立即生效
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param request body model.K8sConfigMapHotReloadRequest true "热重载请求"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "重载成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/hot_reload [post]
// @Security BearerAuth
func (k *K8sConfigMapHandler) HotReloadConfigMap(ctx *gin.Context) {
	var req model.K8sConfigMapHotReloadRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.configmapService.HotReloadConfigMap(ctx, &req)
	})
}

// RollbackConfigMap 回滚 ConfigMap
// @Summary 回滚 ConfigMap到指定版本
// @Description 将ConfigMap配置回滚到指定的历史版本，用于快速恢复配置
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param request body model.K8sConfigMapRollbackRequest true "回滚请求"
// @Success 200 {object} utils.ApiResponse "回滚成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/configmaps/rollback [post]
// @Security BearerAuth
func (k *K8sConfigMapHandler) RollbackConfigMap(ctx *gin.Context) {
	var req model.K8sConfigMapRollbackRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.configmapService.RollbackConfigMap(ctx, &req)
	})
}

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

type K8sStatefulSetHandler struct {
	l                 *zap.Logger
	statefulSetService admin.StatefulSetService
}

func NewK8sStatefulSetHandler(l *zap.Logger, statefulSetService admin.StatefulSetService) *K8sStatefulSetHandler {
	return &K8sStatefulSetHandler{
		l:                 l,
		statefulSetService: statefulSetService,
	}
}

func (k *K8sStatefulSetHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	statefulsets := k8sGroup.Group("/statefulsets")
	{
		statefulsets.GET("/:id", k.GetStatefulSetsByNamespace)          // 根据命名空间获取 StatefulSet 列表
		statefulsets.GET("/:id/yaml", k.GetStatefulSetYaml)            // 获取指定 StatefulSet 的 YAML 配置
		statefulsets.POST("/update", k.UpdateStatefulSet)              // 更新指定 StatefulSet
		statefulsets.POST("/create", k.CreateStatefulSet)              // 创建 StatefulSet
		statefulsets.DELETE("/batch_delete", k.BatchDeleteStatefulSet) // 批量删除 StatefulSet
		statefulsets.DELETE("/delete/:id", k.DeleteStatefulSet)        // 删除指定 StatefulSet
		statefulsets.POST("/restart/:id", k.RestartStatefulSet)        // 重启 StatefulSet
		statefulsets.POST("/scale", k.ScaleStatefulSet)                // 扩缩容 StatefulSet
		statefulsets.GET("/:id/status", k.GetStatefulSetStatus)        // 获取 StatefulSet 状态
	}
}

// GetStatefulSetsByNamespace 根据命名空间获取 StatefulSet 列表
// @Summary 获取StatefulSet列表
// @Description 根据指定的集群ID和命名空间查询所有的有状态应用(StatefulSet)资源
// @Tags 有状态应用管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param namespace query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse{data=[]interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/statefulsets/{id} [get]
// @Security BearerAuth
func (k *K8sStatefulSetHandler) GetStatefulSetsByNamespace(ctx *gin.Context) {
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
		return k.statefulSetService.GetStatefulSetsByNamespace(ctx, id, namespace)
	})
}

// CreateStatefulSet 创建 StatefulSet
// @Summary 创建StatefulSet
// @Description 在指定集群的命名空间中创建新的有状态应用(StatefulSet)资源
// @Tags 有状态应用管理
// @Accept json
// @Produce json
// @Param request body model.K8sStatefulSetRequest true "StatefulSet创建信息"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/statefulsets/create [post]
// @Security BearerAuth
func (k *K8sStatefulSetHandler) CreateStatefulSet(ctx *gin.Context) {
	var req model.K8sStatefulSetRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.statefulSetService.CreateStatefulSet(ctx, &req)
	})
}

// UpdateStatefulSet 更新指定的 StatefulSet
// @Summary 更新StatefulSet
// @Description 修改指定StatefulSet的配置和参数信息
// @Tags 有状态应用管理
// @Accept json
// @Produce json
// @Param request body model.K8sStatefulSetRequest true "StatefulSet更新信息"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/statefulsets/update [post]
// @Security BearerAuth
func (k *K8sStatefulSetHandler) UpdateStatefulSet(ctx *gin.Context) {
	var req model.K8sStatefulSetRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.statefulSetService.UpdateStatefulSet(ctx, &req)
	})
}

// BatchDeleteStatefulSet 批量删除 StatefulSet
// @Summary 批量删除StatefulSet
// @Description 同时删除指定命名空间下的多个StatefulSet资源
// @Tags 有状态应用管理
// @Accept json
// @Produce json
// @Param request body model.K8sStatefulSetRequest true "批量删除请求"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/statefulsets/batch_delete [delete]
// @Security BearerAuth
func (k *K8sStatefulSetHandler) BatchDeleteStatefulSet(ctx *gin.Context) {
	var req model.K8sStatefulSetRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.statefulSetService.BatchDeleteStatefulSet(ctx, req.ClusterID, req.Namespace, req.StatefulSetNames)
	})
}

// GetStatefulSetYaml 获取 StatefulSet 的 YAML 配置
// @Summary 获取StatefulSet的YAML配置
// @Description 以YAML格式返回指定StatefulSet的完整配置信息
// @Tags 有状态应用管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param statefulset_name query string true "StatefulSet名称"
// @Param namespace query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse{data=string} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/statefulsets/{id}/yaml [get]
// @Security BearerAuth
func (k *K8sStatefulSetHandler) GetStatefulSetYaml(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	statefulSetName := ctx.Query("statefulset_name")
	if statefulSetName == "" {
		utils.BadRequestError(ctx, "缺少 'statefulset_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.statefulSetService.GetStatefulSetYaml(ctx, id, namespace, statefulSetName)
	})
}

// DeleteStatefulSet 删除指定的 StatefulSet
// @Summary 删除单个StatefulSet
// @Description 删除指定命名空间下的单个StatefulSet资源
// @Tags 有状态应用管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param statefulset_name query string true "StatefulSet名称"
// @Param namespace query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/statefulsets/delete/{id} [delete]
// @Security BearerAuth
func (k *K8sStatefulSetHandler) DeleteStatefulSet(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	statefulSetName := ctx.Query("statefulset_name")
	if statefulSetName == "" {
		utils.BadRequestError(ctx, "缺少 'statefulset_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.statefulSetService.DeleteStatefulSet(ctx, id, namespace, statefulSetName)
	})
}

// RestartStatefulSet 重启 StatefulSet
// @Summary 重启StatefulSet
// @Description 对指定的StatefulSet执行重启操作，重新创建Pod实例
// @Tags 有状态应用管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param statefulset_name query string true "StatefulSet名称"
// @Param namespace query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse "重启成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/statefulsets/restart/{id} [post]
// @Security BearerAuth
func (k *K8sStatefulSetHandler) RestartStatefulSet(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	statefulSetName := ctx.Query("statefulset_name")
	if statefulSetName == "" {
		utils.BadRequestError(ctx, "缺少 'statefulset_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.statefulSetService.RestartStatefulSet(ctx, id, namespace, statefulSetName)
	})
}

// ScaleStatefulSet 扩缩容 StatefulSet
// @Summary 扩缩容StatefulSet
// @Description 调整StatefulSet的副本数量，实现应用的扩容或缩容
// @Tags 有状态应用管理
// @Accept json
// @Produce json
// @Param request body model.K8sStatefulSetScaleRequest true "扩缩容请求"
// @Success 200 {object} utils.ApiResponse "扩缩容成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/statefulsets/scale [post]
// @Security BearerAuth
func (k *K8sStatefulSetHandler) ScaleStatefulSet(ctx *gin.Context) {
	var req model.K8sStatefulSetScaleRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.statefulSetService.ScaleStatefulSet(ctx, &req)
	})
}

// GetStatefulSetStatus 获取 StatefulSet 状态
// @Summary 获取StatefulSet状态
// @Description 获取指定StatefulSet的详细状态信息，包括副本数、就绪状态等
// @Tags 有状态应用管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param statefulset_name query string true "StatefulSet名称"
// @Param namespace query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/statefulsets/{id}/status [get]
// @Security BearerAuth
func (k *K8sStatefulSetHandler) GetStatefulSetStatus(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	statefulSetName := ctx.Query("statefulset_name")
	if statefulSetName == "" {
		utils.BadRequestError(ctx, "缺少 'statefulset_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.statefulSetService.GetStatefulSetStatus(ctx, id, namespace, statefulSetName)
	})
}
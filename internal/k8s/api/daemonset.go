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

type K8sDaemonSetHandler struct {
	l                *zap.Logger
	daemonSetService admin.DaemonSetService
}

func NewK8sDaemonSetHandler(l *zap.Logger, daemonSetService admin.DaemonSetService) *K8sDaemonSetHandler {
	return &K8sDaemonSetHandler{
		l:                l,
		daemonSetService: daemonSetService,
	}
}

func (k *K8sDaemonSetHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	daemonsets := k8sGroup.Group("/daemonsets")
	{
		daemonsets.GET("/:id", k.GetDaemonSetsByNamespace)          // 根据命名空间获取 DaemonSet 列表
		daemonsets.GET("/:id/yaml", k.GetDaemonSetYaml)            // 获取指定 DaemonSet 的 YAML 配置
		daemonsets.POST("/update", k.UpdateDaemonSet)              // 更新指定 DaemonSet
		daemonsets.POST("/create", k.CreateDaemonSet)              // 创建 DaemonSet
		daemonsets.DELETE("/batch_delete", k.BatchDeleteDaemonSet) // 批量删除 DaemonSet
		daemonsets.DELETE("/delete/:id", k.DeleteDaemonSet)        // 删除指定 DaemonSet
		daemonsets.POST("/restart/:id", k.RestartDaemonSet)        // 重启 DaemonSet
		daemonsets.GET("/:id/status", k.GetDaemonSetStatus)        // 获取 DaemonSet 状态
	}
}

// GetDaemonSetsByNamespace 根据命名空间获取 DaemonSet 列表
// @Summary 根据命名空间获取 DaemonSet 列表
// @Description 根据指定的集群ID和命名空间获取 DaemonSet 列表
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=[]object} "成功获取DaemonSet列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/daemonsets/{id} [get]
// @Security BearerAuth
func (k *K8sDaemonSetHandler) GetDaemonSetsByNamespace(ctx *gin.Context) {
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
		return k.daemonSetService.GetDaemonSetsByNamespace(ctx, id, namespace)
	})
}

// CreateDaemonSet 创建 DaemonSet
// @Summary 创建 DaemonSet
// @Description 创建新的 Kubernetes DaemonSet 资源
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param request body model.K8sDaemonSetRequest true "DaemonSet 创建信息"
// @Success 200 {object} utils.ApiResponse "成功创建DaemonSet"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/daemonsets/create [post]
// @Security BearerAuth
func (k *K8sDaemonSetHandler) CreateDaemonSet(ctx *gin.Context) {
	var req model.K8sDaemonSetRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.daemonSetService.CreateDaemonSet(ctx, &req)
	})
}

// UpdateDaemonSet 更新指定的 DaemonSet
// @Summary 更新 DaemonSet
// @Description 更新指定的 Kubernetes DaemonSet 资源配置
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param request body model.K8sDaemonSetRequest true "DaemonSet 更新信息"
// @Success 200 {object} utils.ApiResponse "成功更新DaemonSet"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/daemonsets/update [post]
// @Security BearerAuth
func (k *K8sDaemonSetHandler) UpdateDaemonSet(ctx *gin.Context) {
	var req model.K8sDaemonSetRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.daemonSetService.UpdateDaemonSet(ctx, &req)
	})
}

// BatchDeleteDaemonSet 批量删除 DaemonSet
// @Summary 批量删除 DaemonSet
// @Description 批量删除指定命名空间下的多个 DaemonSet
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param request body model.K8sDaemonSetRequest true "批量删除请求"
// @Success 200 {object} utils.ApiResponse "成功批量删除DaemonSet"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/daemonsets/batch_delete [delete]
// @Security BearerAuth
func (k *K8sDaemonSetHandler) BatchDeleteDaemonSet(ctx *gin.Context) {
	var req model.K8sDaemonSetRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.daemonSetService.BatchDeleteDaemonSet(ctx, req.ClusterID, req.Namespace, req.DaemonSetNames)
	})
}

// GetDaemonSetYaml 获取 DaemonSet 的 YAML 配置
// @Summary 获取 DaemonSet YAML 配置
// @Description 获取指定 DaemonSet 的 YAML 格式配置文件
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param daemonset_name query string true "DaemonSet 名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=string} "成功获取DaemonSet YAML配置"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/daemonsets/{id}/yaml [get]
// @Security BearerAuth
func (k *K8sDaemonSetHandler) GetDaemonSetYaml(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	daemonSetName := ctx.Query("daemonset_name")
	if daemonSetName == "" {
		utils.BadRequestError(ctx, "缺少 'daemonset_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSetYaml(ctx, id, namespace, daemonSetName)
	})
}

// DeleteDaemonSet 删除指定的 DaemonSet
// @Summary 删除 DaemonSet
// @Description 删除指定名称的 DaemonSet 资源
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param daemonset_name query string true "DaemonSet 名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse "成功删除DaemonSet"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/daemonsets/delete/{id} [delete]
// @Security BearerAuth
func (k *K8sDaemonSetHandler) DeleteDaemonSet(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	daemonSetName := ctx.Query("daemonset_name")
	if daemonSetName == "" {
		utils.BadRequestError(ctx, "缺少 'daemonset_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.daemonSetService.DeleteDaemonSet(ctx, id, namespace, daemonSetName)
	})
}

// RestartDaemonSet 重启 DaemonSet
// @Summary 重启 DaemonSet
// @Description 重启指定的 DaemonSet，触发 Pod 重新创建
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param daemonset_name query string true "DaemonSet 名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse "成功重启DaemonSet"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/daemonsets/restart/{id} [post]
// @Security BearerAuth
func (k *K8sDaemonSetHandler) RestartDaemonSet(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	daemonSetName := ctx.Query("daemonset_name")
	if daemonSetName == "" {
		utils.BadRequestError(ctx, "缺少 'daemonset_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.daemonSetService.RestartDaemonSet(ctx, id, namespace, daemonSetName)
	})
}

// GetDaemonSetStatus 获取 DaemonSet 状态
// @Summary 获取 DaemonSet 状态
// @Description 获取指定 DaemonSet 的运行状态和节点分布情况
// @Tags DaemonSet管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param daemonset_name query string true "DaemonSet 名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=object} "成功获取DaemonSet状态"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/daemonsets/{id}/status [get]
// @Security BearerAuth
func (k *K8sDaemonSetHandler) GetDaemonSetStatus(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	daemonSetName := ctx.Query("daemonset_name")
	if daemonSetName == "" {
		utils.BadRequestError(ctx, "缺少 'daemonset_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSetStatus(ctx, id, namespace, daemonSetName)
	})
}
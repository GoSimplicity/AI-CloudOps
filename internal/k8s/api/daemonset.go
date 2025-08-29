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

type K8sDaemonSetHandler struct {
	logger           *zap.Logger
	daemonSetService service.DaemonSetService
}

func NewK8sDaemonSetHandler(logger *zap.Logger, daemonSetService service.DaemonSetService) *K8sDaemonSetHandler {
	return &K8sDaemonSetHandler{
		logger:           logger,
		daemonSetService: daemonSetService,
	}
}

func (k *K8sDaemonSetHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	daemonSets := k8sGroup.Group("/daemonsets")
	{
		// 基础操作
		daemonSets.GET("/list", k.GetDaemonSetList)                   // 获取DaemonSet列表
		daemonSets.GET("/:cluster_id", k.GetDaemonSetsByNamespace)    // 根据命名空间获取DaemonSet列表
		daemonSets.GET("/:cluster_id/:name", k.GetDaemonSet)          // 获取单个DaemonSet详情
		daemonSets.GET("/:cluster_id/:name/yaml", k.GetDaemonSetYaml) // 获取DaemonSet YAML配置
		daemonSets.POST("/create", k.CreateDaemonSet)                 // 创建DaemonSet
		daemonSets.PUT("/update", k.UpdateDaemonSet)                  // 更新DaemonSet
		daemonSets.DELETE("/delete", k.DeleteDaemonSet)               // 删除DaemonSet
		daemonSets.POST("/restart", k.RestartDaemonSet)               // 重启DaemonSet

		// 批量操作

		// 高级功能
		daemonSets.GET("/:cluster_id/:name/history", k.GetDaemonSetHistory)    // 获取DaemonSet历史版本
		daemonSets.GET("/:cluster_id/:name/events", k.GetDaemonSetEvents)      // 获取DaemonSet事件
		daemonSets.GET("/:cluster_id/:name/node-pods", k.GetDaemonSetNodePods) // 获取指定节点的DaemonSet Pod
	}
}

// GetDaemonSetList 获取DaemonSet列表
func (k *K8sDaemonSetHandler) GetDaemonSetList(ctx *gin.Context) {
	var req model.K8sDaemonSetListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSetList(ctx, &req)
	})
}

// GetDaemonSetsByNamespace 根据命名空间获取DaemonSet列表
func (k *K8sDaemonSetHandler) GetDaemonSetsByNamespace(ctx *gin.Context) {
	var req model.K8sGetResourceListReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSetsByNamespace(ctx, req.ClusterID, req.Namespace)
	})
}

// GetDaemonSet 获取DaemonSet详情
func (k *K8sDaemonSetHandler) GetDaemonSet(ctx *gin.Context) {
	var req model.K8sGetResourceReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSet(ctx, req.ClusterID, req.Namespace, req.ResourceName)
	})
}

// GetDaemonSetYaml 获取DaemonSet的YAML配置
func (k *K8sDaemonSetHandler) GetDaemonSetYaml(ctx *gin.Context) {
	var req model.K8sGetResourceYamlReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSetYaml(ctx, req.ClusterID, req.Namespace, req.ResourceName)
	})
}

// CreateDaemonSet 创建DaemonSet
func (k *K8sDaemonSetHandler) CreateDaemonSet(ctx *gin.Context) {
	var req model.K8sDaemonSetCreateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.daemonSetService.CreateDaemonSet(ctx, &req)
	})
}

// UpdateDaemonSet 更新DaemonSet
func (k *K8sDaemonSetHandler) UpdateDaemonSet(ctx *gin.Context) {
	var req model.K8sDaemonSetUpdateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.daemonSetService.UpdateDaemonSet(ctx, &req)
	})
}

// DeleteDaemonSet 删除DaemonSet
func (k *K8sDaemonSetHandler) DeleteDaemonSet(ctx *gin.Context) {
	var req model.K8sDaemonSetDeleteReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.daemonSetService.DeleteDaemonSet(ctx, &req)
	})
}

// RestartDaemonSet 重启DaemonSet
func (k *K8sDaemonSetHandler) RestartDaemonSet(ctx *gin.Context) {
	var req model.K8sDaemonSetRestartReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.daemonSetService.RestartDaemonSet(ctx, &req)
	})
}

// GetDaemonSetHistory 获取DaemonSet历史版本
func (k *K8sDaemonSetHandler) GetDaemonSetHistory(ctx *gin.Context) {
	var req model.K8sDaemonSetHistoryReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSetHistory(ctx, &req)
	})
}

// GetDaemonSetEvents 获取DaemonSet事件
func (k *K8sDaemonSetHandler) GetDaemonSetEvents(ctx *gin.Context) {
	var req model.K8sDaemonSetEventReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSetEvents(ctx, &req)
	})
}

// GetDaemonSetNodePods 获取DaemonSet在指定节点的Pod
func (k *K8sDaemonSetHandler) GetDaemonSetNodePods(ctx *gin.Context) {
	var req model.K8sDaemonSetNodePodsReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSetNodePods(ctx, &req)
	})
}

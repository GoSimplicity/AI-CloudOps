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

type K8sPVHandler struct {
	logger    *zap.Logger
	pvService service.PVService
}

func NewK8sPVHandler(logger *zap.Logger, pvService service.PVService) *K8sPVHandler {
	return &K8sPVHandler{
		logger:    logger,
		pvService: pvService,
	}
}

func (k *K8sPVHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	pvs := k8sGroup.Group("/pvs")
	{
		// 基础操作
		pvs.GET("/list", k.GetPVList)                   // 获取PV列表
		pvs.GET("/:cluster_id", k.GetPVsByCluster)      // 根据集群获取PV列表
		pvs.GET("/:cluster_id/:name", k.GetPV)          // 获取单个PV详情
		pvs.GET("/:cluster_id/:name/yaml", k.GetPVYaml) // 获取PV YAML配置
		pvs.POST("/create", k.CreatePV)                 // 创建PV
		pvs.PUT("/update", k.UpdatePV)                  // 更新PV
		pvs.DELETE("/delete", k.DeletePV)               // 删除PV

		// 批量操作

		// 高级功能
		pvs.GET("/:cluster_id/:name/events", k.GetPVEvents) // 获取PV事件
		pvs.GET("/:cluster_id/:name/usage", k.GetPVUsage)   // 获取PV使用情况
		pvs.POST("/:cluster_id/:name/reclaim", k.ReclaimPV) // 回收PV
	}
}

// GetPVList 获取PV列表
func (k *K8sPVHandler) GetPVList(ctx *gin.Context) {
	var req model.K8sPVListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvService.GetPVList(ctx, &req)
	})
}

// GetPVsByCluster 根据集群获取PV列表
func (k *K8sPVHandler) GetPVsByCluster(ctx *gin.Context) {
	var req struct {
		ClusterID int `uri:"cluster_id" binding:"required"`
	}
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvService.GetPVsByCluster(ctx, req.ClusterID)
	})
}

// GetPV 获取PV详情
func (k *K8sPVHandler) GetPV(ctx *gin.Context) {
	var req struct {
		ClusterID int    `uri:"cluster_id" binding:"required"`
		Name      string `uri:"name" binding:"required"`
	}
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvService.GetPV(ctx, req.ClusterID, req.Name)
	})
}

// GetPVYaml 获取PV的YAML配置
func (k *K8sPVHandler) GetPVYaml(ctx *gin.Context) {
	var req struct {
		ClusterID int    `uri:"cluster_id" binding:"required"`
		Name      string `uri:"name" binding:"required"`
	}
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvService.GetPVYaml(ctx, req.ClusterID, req.Name)
	})
}

// CreatePV 创建PV
func (k *K8sPVHandler) CreatePV(ctx *gin.Context) {
	var req model.K8sPVCreateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvService.CreatePV(ctx, &req)
	})
}

// UpdatePV 更新PV
func (k *K8sPVHandler) UpdatePV(ctx *gin.Context) {
	var req model.K8sPVUpdateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvService.UpdatePV(ctx, &req)
	})
}

// DeletePV 删除PV
func (k *K8sPVHandler) DeletePV(ctx *gin.Context) {
	var req model.K8sPVDeleteReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvService.DeletePV(ctx, &req)
	})
}

// GetPVEvents 获取PV事件
func (k *K8sPVHandler) GetPVEvents(ctx *gin.Context) {
	var req model.K8sPVEventReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvService.GetPVEvents(ctx, &req)
	})
}

// GetPVUsage 获取PV使用情况
func (k *K8sPVHandler) GetPVUsage(ctx *gin.Context) {
	var req model.K8sPVUsageReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvService.GetPVUsage(ctx, &req)
	})
}

// ReclaimPV 回收PV
func (k *K8sPVHandler) ReclaimPV(ctx *gin.Context) {
	var req model.K8sPVReclaimReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvService.ReclaimPV(ctx, req.ClusterID, req.Name)
	})
}

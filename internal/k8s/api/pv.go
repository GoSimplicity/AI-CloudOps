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

type K8sPVHandler struct {
	l         *zap.Logger
	pvService admin.PVService
}

func NewK8sPVHandler(l *zap.Logger, pvService admin.PVService) *K8sPVHandler {
	return &K8sPVHandler{
		l:         l,
		pvService: pvService,
	}
}

func (k *K8sPVHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	pvs := k8sGroup.Group("/pvs")
	{
		pvs.GET("/:id", k.GetPVs)                   // 获取 PV 列表
		pvs.POST("/create", k.CreatePV)             // 创建 PV
		pvs.DELETE("/delete/:id", k.DeletePV)       // 删除指定 PV
		pvs.DELETE("/batch_delete", k.BatchDeletePV) // 批量删除 PV
		pvs.GET("/:id/yaml", k.GetPVYaml)          // 获取 PV YAML 配置
		pvs.GET("/:id/status", k.GetPVStatus)      // 获取 PV 状态
		pvs.GET("/:id/capacity", k.GetPVCapacity)  // 获取 PV 容量信息
	}
}

// GetPVs 获取 PV 列表
// @Summary 获取PV列表
// @Description 获取指定集群中所有的持久卷(PV)资源列表
// @Tags 存储卷管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} utils.ApiResponse{data=[]interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/pvs/{id} [get]
// @Security BearerAuth
func (k *K8sPVHandler) GetPVs(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvService.GetPVs(ctx, id)
	})
}

// CreatePV 创建 PV
// @Summary 创建PV
// @Description 在指定集群中创建新的持久卷(PV)资源
// @Tags 存储卷管理
// @Accept json
// @Produce json
// @Param request body model.K8sPVRequest true "PV创建信息"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/pvs/create [post]
// @Security BearerAuth
func (k *K8sPVHandler) CreatePV(ctx *gin.Context) {
	var req model.K8sPVRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvService.CreatePV(ctx, &req)
	})
}

// BatchDeletePV 批量删除 PV
// @Summary 批量删除PV
// @Description 同时删除多个持久卷(PV)资源
// @Tags 存储卷管理
// @Accept json
// @Produce json
// @Param request body model.K8sPVRequest true "批量删除请求"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/pvs/batch_delete [delete]
// @Security BearerAuth
func (k *K8sPVHandler) BatchDeletePV(ctx *gin.Context) {
	var req model.K8sPVRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvService.BatchDeletePV(ctx, req.ClusterID, req.PVNames)
	})
}

// GetPVYaml 获取 PV 的 YAML 配置
// @Summary 获取PV的YAML配置
// @Description 以YAML格式返回指定PV的完整配置信息
// @Tags 存储卷管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param pv_name query string true "PV名称"
// @Success 200 {object} utils.ApiResponse{data=string} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/pvs/{id}/yaml [get]
// @Security BearerAuth
func (k *K8sPVHandler) GetPVYaml(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	pvName := ctx.Query("pv_name")
	if pvName == "" {
		k.l.Error("缺少必需的 pv_name 参数")
		utils.BadRequestError(ctx, "缺少 'pv_name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvService.GetPVYaml(ctx, id, pvName)
	})
}

// DeletePV 删除指定的 PV
// @Summary 删除单个PV
// @Description 删除指定的持久卷(PV)资源
// @Tags 存储卷管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param pv_name query string true "PV名称"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/pvs/delete/{id} [delete]
// @Security BearerAuth
func (k *K8sPVHandler) DeletePV(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	pvName := ctx.Query("pv_name")
	if pvName == "" {
		k.l.Error("缺少必需的 pv_name 参数")
		utils.BadRequestError(ctx, "缺少 'pv_name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.pvService.DeletePV(ctx, id, pvName)
	})
}

// GetPVStatus 获取 PV 状态
// @Summary 获取PV状态
// @Description 获取指定PV的详细状态信息，包括绑定状态、容量等
// @Tags 存储卷管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param pv_name query string true "PV名称"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/pvs/{id}/status [get]
// @Security BearerAuth
func (k *K8sPVHandler) GetPVStatus(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	pvName := ctx.Query("pv_name")
	if pvName == "" {
		k.l.Error("缺少必需的 pv_name 参数")
		utils.BadRequestError(ctx, "缺少 'pv_name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvService.GetPVStatus(ctx, id, pvName)
	})
}

// GetPVCapacity 获取 PV 容量信息
// @Summary 获取PV容量信息
// @Description 获取指定PV的容量详细信息和使用情况
// @Tags 存储卷管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param pv_name query string true "PV名称"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/pvs/{id}/capacity [get]
// @Security BearerAuth
func (k *K8sPVHandler) GetPVCapacity(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	pvName := ctx.Query("pv_name")
	if pvName == "" {
		k.l.Error("缺少必需的 pv_name 参数")
		utils.BadRequestError(ctx, "缺少 'pv_name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvService.GetPVCapacity(ctx, id, pvName)
	})
}
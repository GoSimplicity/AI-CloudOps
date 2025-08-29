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

type K8sPVCHandler struct {
	logger     *zap.Logger
	pvcService service.PVCService
}

func NewK8sPVCHandler(logger *zap.Logger, pvcService service.PVCService) *K8sPVCHandler {
	return &K8sPVCHandler{
		logger:     logger,
		pvcService: pvcService,
	}
}

func (k *K8sPVCHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	pvcs := k8sGroup.Group("/pvcs")
	{
		// 基础操作
		pvcs.GET("/list", k.GetPVCList)                   // 获取PVC列表
		pvcs.GET("/:cluster_id", k.GetPVCsByNamespace)    // 根据命名空间获取PVC列表
		pvcs.GET("/:cluster_id/:name", k.GetPVC)          // 获取单个PVC详情
		pvcs.GET("/:cluster_id/:name/yaml", k.GetPVCYaml) // 获取PVC YAML配置
		pvcs.POST("/create", k.CreatePVC)                 // 创建PVC
		pvcs.PUT("/update", k.UpdatePVC)                  // 更新PVC
		pvcs.DELETE("/delete", k.DeletePVC)               // 删除PVC

		// 批量操作

		// 高级功能
		pvcs.GET("/:cluster_id/:name/events", k.GetPVCEvents) // 获取PVC事件
		pvcs.GET("/:cluster_id/:name/usage", k.GetPVCUsage)   // 获取PVC使用情况
		pvcs.POST("/:cluster_id/:name/expand", k.ExpandPVC)   // 扩容PVC
	}
}

// GetPVCList 获取PVC列表
// @Summary 获取PVC列表
// @Description 根据查询条件获取K8s集群中的PVC列表
// @Tags PVC管理
// @Accept json
// @Produce json
// @Param request query model.K8sPVCListReq true "PVC列表查询请求"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sPVCEntity} "成功获取PVC列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/pvcs/list [get]
func (k *K8sPVCHandler) GetPVCList(ctx *gin.Context) {
	var req model.K8sPVCListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvcService.GetPVCList(ctx, &req)
	})
}

// GetPVCsByNamespace 根据命名空间获取PVC列表
// @Summary 根据命名空间获取PVC列表
// @Description 根据指定的命名空间获取K8s集群中的PVC列表
// @Tags PVC管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace query string false "命名空间，为空则获取所有命名空间"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sPVCEntity} "成功获取PVC列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/pvcs/{cluster_id} [get]
func (k *K8sPVCHandler) GetPVCsByNamespace(ctx *gin.Context) {
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
		return k.pvcService.GetPVCsByNamespace(ctx, req.ClusterID, req.Namespace)
	})
}

// GetPVC 获取PVC详情
// @Summary 获取PVC详情
// @Description 获取指定PVC的详细信息
// @Tags PVC管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "PVC名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=model.K8sPVCEntity} "成功获取PVC详情"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/pvcs/{cluster_id}/{name} [get]
func (k *K8sPVCHandler) GetPVC(ctx *gin.Context) {
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
		return k.pvcService.GetPVC(ctx, req.ClusterID, req.Namespace, req.ResourceName)
	})
}

// GetPVCYaml 获取PVC的YAML配置
// @Summary 获取PVC的YAML配置
// @Description 获取指定PVC的完整YAML配置文件
// @Tags PVC管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "PVC名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=string} "成功获取YAML配置"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/pvcs/{cluster_id}/{name}/yaml [get]
func (k *K8sPVCHandler) GetPVCYaml(ctx *gin.Context) {
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
		return k.pvcService.GetPVCYaml(ctx, req.ClusterID, req.Namespace, req.ResourceName)
	})
}

// CreatePVC 创建PVC
// @Summary 创建PVC
// @Description 创建新的PVC资源
// @Tags PVC管理
// @Accept json
// @Produce json
// @Param request body model.K8sPVCCreateReq true "PVC创建请求"
// @Success 200 {object} utils.ApiResponse "成功创建PVC"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/pvcs/create [post]
func (k *K8sPVCHandler) CreatePVC(ctx *gin.Context) {
	var req model.K8sPVCCreateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvcService.CreatePVC(ctx, &req)
	})
}

// UpdatePVC 更新PVC
// @Summary 更新PVC
// @Description 更新指定的PVC资源配置
// @Tags PVC管理
// @Accept json
// @Produce json
// @Param request body model.K8sPVCUpdateReq true "PVC更新请求"
// @Success 200 {object} utils.ApiResponse "成功更新PVC"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/pvcs/update [put]
func (k *K8sPVCHandler) UpdatePVC(ctx *gin.Context) {
	var req model.K8sPVCUpdateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvcService.UpdatePVC(ctx, &req)
	})
}

// DeletePVC 删除PVC
// @Summary 删除PVC
// @Description 删除指定的PVC资源
// @Tags PVC管理
// @Accept json
// @Produce json
// @Param request body model.K8sPVCDeleteReq true "PVC删除请求"
// @Success 200 {object} utils.ApiResponse "成功删除PVC"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/pvcs/delete [delete]
func (k *K8sPVCHandler) DeletePVC(ctx *gin.Context) {
	var req model.K8sPVCDeleteReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvcService.DeletePVC(ctx, &req)
	})
}

// GetPVCEvents 获取PVC事件
// @Summary 获取PVC事件
// @Description 获取指定PVC相关的事件信息
// @Tags PVC管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "PVC名称"
// @Param namespace query string true "命名空间"
// @Param limit_days query int false "限制天数内的事件"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sEvent} "成功获取事件"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/pvcs/{cluster_id}/{name}/events [get]
func (k *K8sPVCHandler) GetPVCEvents(ctx *gin.Context) {
	var req model.K8sPVCEventReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvcService.GetPVCEvents(ctx, &req)
	})
}

// GetPVCUsage 获取PVC使用情况
// @Summary 获取PVC使用情况
// @Description 获取指定PVC的使用情况信息
// @Tags PVC管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "PVC名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=model.K8sPVCUsageInfo} "成功获取使用情况"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/pvcs/{cluster_id}/{name}/usage [get]
func (k *K8sPVCHandler) GetPVCUsage(ctx *gin.Context) {
	var req model.K8sPVCUsageReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvcService.GetPVCUsage(ctx, &req)
	})
}

// ExpandPVC 扩容PVC
// @Summary 扩容PVC
// @Description 扩容指定的PVC资源
// @Tags PVC管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "PVC名称"
// @Param request body model.K8sPVCExpandReq true "PVC扩容请求"
// @Success 200 {object} utils.ApiResponse "成功扩容PVC"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/pvcs/{cluster_id}/{name}/expand [post]
func (k *K8sPVCHandler) ExpandPVC(ctx *gin.Context) {
	var req model.K8sPVCExpandReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvcService.ExpandPVC(ctx, &req)
	})
}

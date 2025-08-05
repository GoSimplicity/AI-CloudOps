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
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sClusterHandler struct {
	clusterService admin.ClusterService
	l              *zap.Logger
}

func NewK8sClusterHandler(l *zap.Logger, clusterService admin.ClusterService) *K8sClusterHandler {
	return &K8sClusterHandler{
		l:              l,
		clusterService: clusterService,
	}
}

func (k *K8sClusterHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	clusters := k8sGroup.Group("/clusters")
	{
		clusters.GET("/list", k.GetAllClusters)                 // 获取集群列表
		clusters.GET("/:id", k.GetCluster)                      // 获取指定集群
		clusters.POST("/create", k.CreateCluster)               // 创建新的集群
		clusters.POST("/update", k.UpdateCluster)               // 更新指定 ID 的集群
		clusters.DELETE("/delete/:id", k.DeleteCluster)         // 删除指定 ID 的集群
		clusters.DELETE("/batch_delete", k.BatchDeleteClusters) // 批量删除集群
		clusters.POST("/refresh/:id", k.RefreshCluster)         // 刷新集群状态
	}
}

// GetAllClusters 获取集群列表
// @Summary 获取所有K8s集群列表
// @Description 查询所有可用的Kubernetes集群，包括集群状态、版本信息和连接情况
// @Tags 集群管理
// @Accept json
// @Produce json
// @Success 200 {object} utils.ApiResponse{data=[]interface{}} "获取成功"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/clusters/list [get]
// @Security BearerAuth
func (k *K8sClusterHandler) GetAllClusters(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.clusterService.ListAllClusters(ctx)
	})
}

// GetCluster 获取指定集群
// @Summary 获取K8s集群详情
// @Description 根据集群ID获取指定Kubernetes集群的详细信息，包括节点数量、资源统计等
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/clusters/{id} [get]
// @Security BearerAuth
func (k *K8sClusterHandler) GetCluster(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.clusterService.GetClusterByID(ctx, id)
	})
}

// CreateCluster 创建集群
// @Summary 创建K8s集群配置
// @Description 添加新的Kubernetes集群到系统，需要提供kubeconfig文件或者连接信息
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param request body model.K8sCluster true "集群创建信息"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/clusters/create [post]
// @Security BearerAuth
func (k *K8sClusterHandler) CreateCluster(ctx *gin.Context) {
	var req model.K8sCluster

	uc := ctx.MustGet("user").(ijwt.UserClaims) // 获取用户信息

	req.UserID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.CreateCluster(ctx, &req)
	})
}

// UpdateCluster 更新集群
// @Summary 更新K8s集群配置
// @Description 修改指定Kubernetes集群的配置信息，包括kubeconfig、描述等
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param request body model.K8sCluster true "集群更新信息"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/clusters/update [post]
// @Security BearerAuth
func (k *K8sClusterHandler) UpdateCluster(ctx *gin.Context) {
	var req model.K8sCluster

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.UpdateCluster(ctx, &req)
	})
}

// DeleteCluster 删除集群
// @Summary 删除K8s集群
// @Description 从系统中移除指定的Kubernetes集群配置（不会影响实际集群）
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/clusters/delete/{id} [delete]
// @Security BearerAuth
func (k *K8sClusterHandler) DeleteCluster(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.clusterService.DeleteCluster(ctx, id)
	})
}

// BatchDeleteClusters 批量删除集群
// @Summary 批量删除K8s集群
// @Description 同时从系统中移除多个Kubernetes集群配置
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param request body model.BatchDeleteReq true "批量删除请求"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/clusters/batch_delete [delete]
// @Security BearerAuth
func (k *K8sClusterHandler) BatchDeleteClusters(ctx *gin.Context) {
	var req model.BatchDeleteReq

	if len(req.IDs) == 0 {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.BatchDeleteClusters(ctx, req.IDs)
	})
}

// RefreshCluster 刷新集群状态
// @Summary 刷新K8s集群状态
// @Description 重新检测指定Kubernetes集群的连接状态和基本信息
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} utils.ApiResponse "刷新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/clusters/refresh/{id} [post]
// @Security BearerAuth
func (k *K8sClusterHandler) RefreshCluster(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.clusterService.RefreshClusterStatus(ctx, id)
	})
}

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

type K8sClusterHandler struct {
	logger         *zap.Logger
	clusterService service.ClusterService
}

func NewK8sClusterHandler(logger *zap.Logger, clusterService service.ClusterService) *K8sClusterHandler {
	return &K8sClusterHandler{
		logger:         logger,
		clusterService: clusterService,
	}
}

func (k *K8sClusterHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	clusters := k8sGroup.Group("/clusters")
	{
		clusters.GET("/list", k.GetAllClusters)
		clusters.GET("/:id", k.GetCluster)
		clusters.POST("/create", k.CreateCluster)
		clusters.POST("/update", k.UpdateCluster)
		clusters.DELETE("/delete/:id", k.DeleteCluster)
		clusters.DELETE("/batch_delete", k.BatchDeleteClusters)
		clusters.POST("/refresh/:id", k.RefreshCluster)
	}
}

// GetAllClusters 获取集群列表
// @Summary 获取集群列表
// @Description 获取所有Kubernetes集群
// @Tags 集群管理
// @Accept json
// @Produce json
// @Success 200 {object} utils.ApiResponse{data=[]interface{}}
// @Failure 500 {object} utils.ApiResponse
// @Router /api/k8s/clusters/list [get]
// @Security BearerAuth
func (k *K8sClusterHandler) GetAllClusters(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.clusterService.ListAllClusters(ctx)
	})
}

// GetCluster 获取集群详情
// @Summary 获取集群详情
// @Description 根据ID获取集群信息
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} utils.ApiResponse{data=interface{}}
// @Failure 400 {object} utils.ApiResponse
// @Failure 500 {object} utils.ApiResponse
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
// @Summary 创建集群
// @Description 添加新的Kubernetes集群
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param request body model.K8sCluster true "集群信息"
// @Success 200 {object} utils.ApiResponse
// @Failure 400 {object} utils.ApiResponse
// @Failure 500 {object} utils.ApiResponse
// @Router /api/k8s/clusters/create [post]
// @Security BearerAuth
func (k *K8sClusterHandler) CreateCluster(ctx *gin.Context) {
	var req model.ClusterCreateReq

	uc := ctx.MustGet("user").(utils.UserClaims)

	req.UserID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.CreateCluster(ctx, &req.K8sCluster)
	})
}

// UpdateCluster 更新集群
// @Summary 更新集群
// @Description 修改集群配置
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param request body model.K8sCluster true "集群信息"
// @Success 200 {object} utils.ApiResponse
// @Failure 400 {object} utils.ApiResponse
// @Failure 500 {object} utils.ApiResponse
// @Router /api/k8s/clusters/update [post]
// @Security BearerAuth
func (k *K8sClusterHandler) UpdateCluster(ctx *gin.Context) {
	var req model.ClusterUpdateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.UpdateCluster(ctx, &req.K8sCluster)
	})
}

// DeleteCluster 删除集群
// @Summary 删除集群
// @Description 删除集群配置
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} utils.ApiResponse
// @Failure 400 {object} utils.ApiResponse
// @Failure 500 {object} utils.ApiResponse
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
// @Summary 批量删除集群
// @Description 批量删除多个集群
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param request body model.BatchDeleteReq true "删除请求"
// @Success 200 {object} utils.ApiResponse
// @Failure 400 {object} utils.ApiResponse
// @Failure 500 {object} utils.ApiResponse
// @Router /api/k8s/clusters/batch_delete [delete]
// @Security BearerAuth
func (k *K8sClusterHandler) BatchDeleteClusters(ctx *gin.Context) {
	var req model.BatchDeleteReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.BatchDeleteClusters(ctx, req.IDs)
	})
}

// RefreshCluster 刷新集群状态
// @Summary 刷新集群状态
// @Description 重新检测集群连接状态
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} utils.ApiResponse
// @Failure 400 {object} utils.ApiResponse
// @Failure 500 {object} utils.ApiResponse
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

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
func (k *K8sClusterHandler) GetAllClusters(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.clusterService.ListAllClusters(ctx)
	})
}

// GetCluster 获取指定 ID 的集群详情
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

// CreateCluster 创建新的集群
func (k *K8sClusterHandler) CreateCluster(ctx *gin.Context) {
	var req model.K8sCluster

	uc := ctx.MustGet("user").(ijwt.UserClaims) // 获取用户信息

	req.UserID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.CreateCluster(ctx, &req)
	})
}

// UpdateCluster 更新指定 ID 的集群
func (k *K8sClusterHandler) UpdateCluster(ctx *gin.Context) {
	var req model.K8sCluster

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.UpdateCluster(ctx, &req)
	})
}

// DeleteCluster 删除指定 ID 的集群
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

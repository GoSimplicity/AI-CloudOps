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
)

type K8sClusterHandler struct {
	clusterService service.ClusterService
}

func NewK8sClusterHandler(clusterService service.ClusterService) *K8sClusterHandler {
	return &K8sClusterHandler{
		clusterService: clusterService,
	}
}

func (h *K8sClusterHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		// 集群基础管理
		k8sGroup.GET("/cluster/list", h.GetClusterList)                  // 获取集群列表
		k8sGroup.GET("/cluster/:cluster_id/detail", h.GetCluster)        // 获取集群详情
		k8sGroup.POST("/cluster/create", h.CreateCluster)                // 创建集群
		k8sGroup.PUT("/cluster/:cluster_id/update", h.UpdateCluster)     // 更新集群
		k8sGroup.DELETE("/cluster/:cluster_id/delete", h.DeleteCluster)  // 删除集群
		k8sGroup.POST("/clusters/:cluster_id/refresh", h.RefreshCluster) // 刷新集群状态
	}
}

// GetClusterList 获取集群列表
func (h *K8sClusterHandler) GetClusterList(ctx *gin.Context) {
	var req model.ListClustersReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.clusterService.ListClusters(ctx, &req)
	})
}

// GetCluster 获取集群详情
func (h *K8sClusterHandler) GetCluster(ctx *gin.Context) {
	var req model.GetClusterReq

	id, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.clusterService.GetClusterByID(ctx, &req)
	})
}

// CreateCluster 创建集群
func (h *K8sClusterHandler) CreateCluster(ctx *gin.Context) {
	var req model.CreateClusterReq

	uc := ctx.MustGet("user").(utils.UserClaims)

	req.CreateUserID = uc.Uid
	req.CreateUserName = uc.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.clusterService.CreateCluster(ctx, &req)
	})
}

// UpdateCluster 更新集群
func (h *K8sClusterHandler) UpdateCluster(ctx *gin.Context) {
	var req model.UpdateClusterReq

	id, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.clusterService.UpdateCluster(ctx, &req)
	})
}

// DeleteCluster 删除集群
func (h *K8sClusterHandler) DeleteCluster(ctx *gin.Context) {
	var req model.DeleteClusterReq

	id, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.clusterService.DeleteCluster(ctx, &req)
	})
}

// RefreshCluster 刷新集群状态
func (h *K8sClusterHandler) RefreshCluster(ctx *gin.Context) {
	var req model.RefreshClusterReq

	id, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.clusterService.RefreshClusterStatus(ctx, &req)
	})
}

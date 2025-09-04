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

func (k *K8sClusterHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/clusters/list", k.GetClusterList)
		k8sGroup.GET("/clusters/:id/detail", k.GetCluster)
		k8sGroup.POST("/clusters/create", k.CreateCluster)
		k8sGroup.PUT("/clusters/:id/update", k.UpdateCluster)
		k8sGroup.DELETE("/clusters/:id/delete", k.DeleteCluster)
		k8sGroup.POST("/clusters/:id/refresh", k.RefreshCluster)
	}
}

func (k *K8sClusterHandler) GetClusterList(ctx *gin.Context) {
	var req model.ListClustersReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.clusterService.ListClusters(ctx, &req)
	})
}

func (k *K8sClusterHandler) GetCluster(ctx *gin.Context) {
	var req model.GetClusterReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.clusterService.GetClusterByID(ctx, &req)
	})
}

func (k *K8sClusterHandler) CreateCluster(ctx *gin.Context) {
	var req model.CreateClusterReq

	uc := ctx.MustGet("user").(utils.UserClaims)

	req.CreateUserID = uc.Uid
	req.CreateUserName = uc.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.CreateCluster(ctx, &req)
	})
}

func (k *K8sClusterHandler) UpdateCluster(ctx *gin.Context) {
	var req model.UpdateClusterReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.UpdateCluster(ctx, &req)
	})
}

func (k *K8sClusterHandler) DeleteCluster(ctx *gin.Context) {
	var req model.DeleteClusterReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.DeleteCluster(ctx, &req)
	})
}

func (k *K8sClusterHandler) RefreshCluster(ctx *gin.Context) {
	var req model.RefreshClusterReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.RefreshClusterStatus(ctx, &req)
	})
}

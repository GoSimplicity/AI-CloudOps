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
	{
		k8sGroup.GET("/clusters/list", k.GetAllClusters)
		k8sGroup.GET("/clusters/detail/:id", k.GetCluster)
		k8sGroup.POST("/clusters/create", k.CreateCluster)
		k8sGroup.PUT("/clusters/update/:id", k.UpdateCluster)
		k8sGroup.DELETE("/clusters/delete/:id", k.DeleteCluster)
		k8sGroup.POST("/clusters/refresh/:id", k.RefreshCluster)
		k8sGroup.GET("/clusters/health/:id", k.CheckClusterHealth)
		k8sGroup.GET("/clusters/stats/:id", k.GetClusterStats) // 获取集群统计信息
	}
}

// GetAllClusters 获取集群列表
func (k *K8sClusterHandler) GetAllClusters(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.clusterService.ListAllClusters(ctx)
	})
}

// GetCluster 获取集群详情
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
func (k *K8sClusterHandler) CreateCluster(ctx *gin.Context) {
	var req model.ClusterCreateReq

	uc := ctx.MustGet("user").(utils.UserClaims)

	req.UserID = uc.Uid

	// 将请求转换为K8sCluster
	cluster := &model.K8sCluster{
		Name:                 req.Name,
		NameZh:               req.NameZh,
		UserID:               req.UserID,
		CpuRequest:           req.CpuRequest,
		CpuLimit:             req.CpuLimit,
		MemoryRequest:        req.MemoryRequest,
		MemoryLimit:          req.MemoryLimit,
		RestrictedNameSpace:  req.RestrictedNameSpace,
		Status:               req.Status,
		Env:                  req.Env,
		Version:              req.Version,
		ApiServerAddr:        req.ApiServerAddr,
		KubeConfigContent:    req.KubeConfigContent,
		ActionTimeoutSeconds: req.ActionTimeoutSeconds,
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.CreateCluster(ctx, cluster)
	})
}

// UpdateCluster 更新集群
func (k *K8sClusterHandler) UpdateCluster(ctx *gin.Context) {
	var req model.ClusterUpdateReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ID = id

	// 将请求转换为K8sCluster
	cluster := &model.K8sCluster{
		Model:                model.Model{ID: req.ID},
		Name:                 req.Name,
		NameZh:               req.NameZh,
		UserID:               req.UserID,
		CpuRequest:           req.CpuRequest,
		CpuLimit:             req.CpuLimit,
		MemoryRequest:        req.MemoryRequest,
		MemoryLimit:          req.MemoryLimit,
		RestrictedNameSpace:  req.RestrictedNameSpace,
		Status:               req.Status,
		Env:                  req.Env,
		Version:              req.Version,
		ApiServerAddr:        req.ApiServerAddr,
		KubeConfigContent:    req.KubeConfigContent,
		ActionTimeoutSeconds: req.ActionTimeoutSeconds,
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.UpdateCluster(ctx, cluster)
	})
}

// DeleteCluster 删除集群
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

// RefreshCluster 刷新集群状态
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

// CheckClusterHealth 检查集群健康状态
func (k *K8sClusterHandler) CheckClusterHealth(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.clusterService.CheckClusterHealth(ctx, id)
	})
}

// GetClusterStats 获取集群统计信息
func (k *K8sClusterHandler) GetClusterStats(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.clusterService.GetClusterStats(ctx, id)
	})
}

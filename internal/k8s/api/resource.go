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

// import (
// 	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
// 	"github.com/GoSimplicity/AI-CloudOps/internal/model"
// 	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
// 	"github.com/gin-gonic/gin"
// )

// type K8sResourceHandler struct {
// 	resourceService service.ResourceService
// }

// func NewK8sResourceHandler(resourceService service.ResourceService) *K8sResourceHandler {
// 	return &K8sResourceHandler{
// 		resourceService: resourceService,
// 	}
// }

// func (k *K8sResourceHandler) RegisterRouters(server *gin.Engine) {
// 	k8sGroup := server.Group("/api/k8s")
// 	{
// 		// 资源概览相关路由
// 		k8sGroup.GET("/resources/overview/:cluster_id", h.GetResourceOverview)
// 		k8sGroup.GET("/resources/statistics/:cluster_id", h.GetResourceStatistics)
// 		k8sGroup.GET("/resources/distribution/:cluster_id", h.GetResourceDistribution)

// 		// 资源分析和趋势路由
// 		k8sGroup.GET("/resources/trend/:cluster_id", h.GetResourceTrend)
// 		k8sGroup.GET("/resources/utilization/:cluster_id", h.GetResourceUtilization)
// 		k8sGroup.GET("/resources/health/:cluster_id", h.GetResourceHealth)

// 		// 工作负载分布路由
// 		k8sGroup.GET("/resources/workloads/:cluster_id", h.GetWorkloadDistribution)
// 		k8sGroup.GET("/resources/namespaces/:cluster_id", h.GetNamespaceResources)

// 		// 存储和网络资源路由
// 		k8sGroup.GET("/resources/storage/:cluster_id", h.GetStorageOverview)
// 		k8sGroup.GET("/resources/network/:cluster_id", h.GetNetworkOverview)

// 		// 多集群资源对比
// 		k8sGroup.POST("/resources/clusters/compare", h.CompareClusterResources)
// 		k8sGroup.GET("/resources/clusters/summary", h.GetAllClustersSummary)
// 	}
// }

// // GetResourceOverview 获取集群资源总览
// func (k *K8sResourceHandler) GetResourceOverview(ctx *gin.Context) {
// 	var req model.ResourceOverviewReq
// 	if err := ctx.ShouldBindUri(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}

// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return h.resourceService.GetResourceOverview(ctx, req.ClusterID)
// 	})
// }

// // GetResourceStatistics 获取资源统计信息
// func (k *K8sResourceHandler) GetResourceStatistics(ctx *gin.Context) {
// 	var req model.ResourceStatisticsReq
// 	if err := ctx.ShouldBindUri(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}

// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return h.resourceService.GetResourceStatistics(ctx, req.ClusterID)
// 	})
// }

// // GetResourceDistribution 获取资源分布信息
// func (k *K8sResourceHandler) GetResourceDistribution(ctx *gin.Context) {
// 	var req model.ResourceDistributionReq
// 	if err := ctx.ShouldBindUri(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}

// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return h.resourceService.GetResourceDistribution(ctx, req.ClusterID)
// 	})
// }

// // GetResourceTrend 获取资源趋势信息
// func (k *K8sResourceHandler) GetResourceTrend(ctx *gin.Context) {
// 	var req model.ResourceTrendReq
// 	if err := ctx.ShouldBindUri(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}
// 	if err := ctx.ShouldBindQuery(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}

// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return h.resourceService.GetResourceTrend(ctx, &req)
// 	})
// }

// // GetResourceUtilization 获取资源利用率信息
// func (k *K8sResourceHandler) GetResourceUtilization(ctx *gin.Context) {
// 	var req model.ResourceUtilizationReq
// 	if err := ctx.ShouldBindUri(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}

// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return h.resourceService.GetResourceUtilization(ctx, req.ClusterID)
// 	})
// }

// // GetResourceHealth 获取资源健康状态
// func (k *K8sResourceHandler) GetResourceHealth(ctx *gin.Context) {
// 	var req model.ResourceHealthReq
// 	if err := ctx.ShouldBindUri(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}

// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return h.resourceService.GetResourceHealth(ctx, req.ClusterID)
// 	})
// }

// // GetWorkloadDistribution 获取工作负载分布
// func (k *K8sResourceHandler) GetWorkloadDistribution(ctx *gin.Context) {
// 	var req model.WorkloadDistributionReq
// 	if err := ctx.ShouldBindUri(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}

// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return h.resourceService.GetWorkloadDistribution(ctx, req.ClusterID)
// 	})
// }

// // GetNamespaceResources 获取命名空间资源信息
// func (k *K8sResourceHandler) GetNamespaceResources(ctx *gin.Context) {
// 	var req model.NamespaceResourcesReq
// 	if err := ctx.ShouldBindUri(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}

// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return h.resourceService.GetNamespaceResources(ctx, req.ClusterID)
// 	})
// }

// // GetStorageOverview 获取存储概览
// func (k *K8sResourceHandler) GetStorageOverview(ctx *gin.Context) {
// 	var req model.StorageOverviewReq
// 	if err := ctx.ShouldBindUri(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}

// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return h.resourceService.GetStorageOverview(ctx, req.ClusterID)
// 	})
// }

// // GetNetworkOverview 获取网络概览
// func (k *K8sResourceHandler) GetNetworkOverview(ctx *gin.Context) {
// 	var req model.NetworkOverviewReq
// 	if err := ctx.ShouldBindUri(&req); err != nil {
// 		utils.BadRequestError(ctx, err.Error())
// 		return
// 	}

// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return h.resourceService.GetNetworkOverview(ctx, req.ClusterID)
// 	})
// }

// // CompareClusterResources 对比多个集群的资源使用情况
// func (k *K8sResourceHandler) CompareClusterResources(ctx *gin.Context) {
// 	var req model.CompareClusterResourcesReq

// 	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
// 		return h.resourceService.CompareClusterResources(ctx, req.ClusterIDs)
// 	})
// }

// // GetAllClustersSummary 获取所有集群资源汇总
// func (k *K8sResourceHandler) GetAllClustersSummary(ctx *gin.Context) {
// 	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
// 		return h.resourceService.GetAllClustersSummary(ctx)
// 	})
// }

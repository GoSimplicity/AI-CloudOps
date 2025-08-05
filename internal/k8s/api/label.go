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
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type K8sLabelHandler struct {
	labelService admin.LabelService
}

func NewK8sLabelHandler(labelService admin.LabelService) *K8sLabelHandler {
	return &K8sLabelHandler{
		labelService: labelService,
	}
}

// RegisterRouters 注册路由
func (k *K8sLabelHandler) RegisterRouters(g *gin.Engine) {
	k8sGroup := g.Group("/api/k8s")
	{
		// 基础标签管理
		k8sGroup.POST("/labels/add", k.AddResourceLabels)
		k8sGroup.PUT("/labels/update", k.UpdateResourceLabels)
		k8sGroup.DELETE("/labels/delete", k.DeleteResourceLabels)
		k8sGroup.GET("/labels", k.GetResourceLabels)

		// 标签选择器查询
		k8sGroup.POST("/labels/select", k.ListResourcesByLabels)

		// 批量标签操作
		k8sGroup.POST("/labels/batch", k.BatchUpdateLabels)

		// 标签策略管理
		k8sGroup.POST("/labels/policies", k.CreateLabelPolicy)
		k8sGroup.PUT("/labels/policies", k.UpdateLabelPolicy)
		k8sGroup.DELETE("/labels/policies/:cluster_id/:policy_name", k.DeleteLabelPolicy)
		k8sGroup.GET("/labels/policies/:cluster_id/:policy_name", k.GetLabelPolicy)
		k8sGroup.GET("/labels/policies", k.ListLabelPolicies)

		// 标签合规性检查
		k8sGroup.POST("/labels/compliance/check", k.CheckLabelCompliance)

		// 标签历史记录
		k8sGroup.POST("/labels/history", k.GetLabelHistory)
	}
}

// AddResourceLabels 添加资源标签
// @Summary 添加资源标签
// @Description 为指定Kubernetes资源添加标签
// @Tags Kubernetes-Label
// @Accept json
// @Produce json
// @Param request body model.K8sLabelRequest true "标签添加请求"
// @Success 200 {object} utils.ApiResponse "成功添加标签"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/labels/add [post]
func (k *K8sLabelHandler) AddResourceLabels(ctx *gin.Context) {
	var req model.K8sLabelRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.labelService.AddResourceLabels(ctx, &req)
	})
}

// UpdateResourceLabels 更新资源标签
// @Summary 更新资源标签
// @Description 更新指定Kubernetes资源的标签
// @Tags Kubernetes-Label
// @Accept json
// @Produce json
// @Param request body model.K8sLabelRequest true "标签更新请求"
// @Success 200 {object} utils.ApiResponse "成功更新标签"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/labels/update [put]
func (k *K8sLabelHandler) UpdateResourceLabels(ctx *gin.Context) {
	var req model.K8sLabelRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.labelService.UpdateResourceLabels(ctx, &req)
	})
}

// DeleteResourceLabels 删除资源标签
// @Summary 删除资源标签
// @Description 删除指定Kubernetes资源的标签
// @Tags Kubernetes-Label
// @Accept json
// @Produce json
// @Param request body model.K8sLabelRequest true "标签删除请求"
// @Success 200 {object} utils.ApiResponse "成功删除标签"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/labels/delete [delete]
func (k *K8sLabelHandler) DeleteResourceLabels(ctx *gin.Context) {
	var req model.K8sLabelRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.labelService.DeleteResourceLabels(ctx, &req)
	})
}

// GetResourceLabels 获取资源标签
// @Summary 获取资源标签
// @Description 获取指定Kubernetes资源的所有标签
// @Tags Kubernetes-Label
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string false "命名空间"
// @Param resource_type query string true "资源类型"
// @Param resource_name query string false "资源名称"
// @Success 200 {object} utils.ApiResponse{data=map[string]string} "成功获取标签"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/labels [get]
func (k *K8sLabelHandler) GetResourceLabels(ctx *gin.Context) {
	clusterID, err := strconv.Atoi(ctx.Query("cluster_id"))
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的集群ID")
		return
	}

	namespace := ctx.Query("namespace")
	resourceType := ctx.Query("resource_type")
	resourceName := ctx.Query("resource_name")

	if resourceType == "" {
		utils.ErrorWithMessage(ctx, "资源类型不能为空")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.labelService.GetResourceLabels(ctx, clusterID, namespace, resourceType, resourceName)
	})
}

// ListResourcesByLabels 根据标签选择器查询资源
// @Summary 根据标签查询资源
// @Description 使用标签选择器查询匹配的Kubernetes资源
// @Tags Kubernetes-Label
// @Accept json
// @Produce json
// @Param request body model.K8sLabelSelectorRequest true "标签选择器查询请求"
// @Success 200 {object} utils.ApiResponse{data=[]interface{}} "成功查询资源"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/labels/select [post]
func (k *K8sLabelHandler) ListResourcesByLabels(ctx *gin.Context) {
	var req model.K8sLabelSelectorRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.labelService.ListResourcesByLabels(ctx, &req)
	})
}

// BatchUpdateLabels 批量更新标签
// @Summary 批量更新标签
// @Description 批量更新多个资源的标签
// @Tags Kubernetes-Label
// @Accept json
// @Produce json
// @Param request body model.K8sLabelBatchRequest true "批量标签更新请求"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sLabelResponse} "成功批量更新标签"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/labels/batch [post]
func (k *K8sLabelHandler) BatchUpdateLabels(ctx *gin.Context) {
	var req model.K8sLabelBatchRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.labelService.BatchUpdateLabels(ctx, &req)
	})
}

// CreateLabelPolicy 创建标签策略
// @Summary 创建标签策略
// @Description 创建新的标签管理策略
// @Tags Kubernetes-Label
// @Accept json
// @Produce json
// @Param request body model.K8sLabelPolicyRequest true "标签策略创建请求"
// @Success 200 {object} utils.ApiResponse "成功创建策略"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/labels/policies [post]
func (k *K8sLabelHandler) CreateLabelPolicy(ctx *gin.Context) {
	var req model.K8sLabelPolicyRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.labelService.CreateLabelPolicy(ctx, &req)
	})
}

// UpdateLabelPolicy 更新标签策略
// @Summary 更新标签策略
// @Description 更新已存在的标签管理策略
// @Tags Kubernetes-Label
// @Accept json
// @Produce json
// @Param request body model.K8sLabelPolicyRequest true "标签策略更新请求"
// @Success 200 {object} utils.ApiResponse "成功更新策略"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/labels/policies [put]
func (k *K8sLabelHandler) UpdateLabelPolicy(ctx *gin.Context) {
	var req model.K8sLabelPolicyRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.labelService.UpdateLabelPolicy(ctx, &req)
	})
}

// DeleteLabelPolicy 删除标签策略
// @Summary 删除标签策略
// @Description 删除指定的标签管理策略
// @Tags Kubernetes-Label
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param policy_name path string true "策略名称"
// @Success 200 {object} utils.ApiResponse "成功删除策略"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/labels/policies/{cluster_id}/{policy_name} [delete]
func (k *K8sLabelHandler) DeleteLabelPolicy(ctx *gin.Context) {
	clusterID, err := strconv.Atoi(ctx.Param("cluster_id"))
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的集群ID")
		return
	}

	policyName := ctx.Param("policy_name")
	if policyName == "" {
		utils.ErrorWithMessage(ctx, "策略名称不能为空")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.labelService.DeleteLabelPolicy(ctx, clusterID, policyName)
	})
}

// GetLabelPolicy 获取标签策略
// @Summary 获取标签策略
// @Description 获取指定的标签管理策略详情
// @Tags Kubernetes-Label
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param policy_name path string true "策略名称"
// @Success 200 {object} utils.ApiResponse "成功获取策略"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/labels/policies/{cluster_id}/{policy_name} [get]
func (k *K8sLabelHandler) GetLabelPolicy(ctx *gin.Context) {
	clusterID, err := strconv.Atoi(ctx.Param("cluster_id"))
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的集群ID")
		return
	}

	policyName := ctx.Param("policy_name")
	if policyName == "" {
		utils.ErrorWithMessage(ctx, "策略名称不能为空")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.labelService.GetLabelPolicy(ctx, clusterID, policyName)
	})
}

// ListLabelPolicies 获取标签策略列表
// @Summary 获取标签策略列表
// @Description 获取指定集群的所有标签管理策略
// @Tags Kubernetes-Label
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string false "命名空间"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sLabelPolicyRequest} "成功获取策略列表"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/labels/policies [get]
func (k *K8sLabelHandler) ListLabelPolicies(ctx *gin.Context) {
	clusterID, err := strconv.Atoi(ctx.Query("cluster_id"))
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的集群ID")
		return
	}

	namespace := ctx.Query("namespace")

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.labelService.ListLabelPolicies(ctx, clusterID, namespace)
	})
}

// CheckLabelCompliance 检查标签合规性
// @Summary 检查标签合规性
// @Description 检查资源标签是否符合定义的策略规则
// @Tags Kubernetes-Label
// @Accept json
// @Produce json
// @Param request body model.K8sLabelComplianceRequest true "合规性检查请求"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sLabelComplianceResponse} "成功检查合规性"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/labels/compliance/check [post]
func (k *K8sLabelHandler) CheckLabelCompliance(ctx *gin.Context) {
	var req model.K8sLabelComplianceRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.labelService.CheckLabelCompliance(ctx, &req)
	})
}

// GetLabelHistory 获取标签历史记录
// @Summary 获取标签历史记录
// @Description 获取资源标签的历史变更记录
// @Tags Kubernetes-Label
// @Accept json
// @Produce json
// @Param request body model.K8sLabelHistoryRequest true "标签历史查询请求"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sLabelHistoryResponse} "成功获取历史记录"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/labels/history [post]
func (k *K8sLabelHandler) GetLabelHistory(ctx *gin.Context) {
	var req model.K8sLabelHistoryRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.labelService.GetLabelHistory(ctx, &req)
	})
}

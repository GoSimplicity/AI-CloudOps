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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sServiceAccountHandler struct {
	logger                *zap.Logger
	serviceAccountService service.ServiceAccountService
}

func NewK8sServiceAccountHandler(logger *zap.Logger, serviceAccountService service.ServiceAccountService) *K8sServiceAccountHandler {
	return &K8sServiceAccountHandler{
		logger:                logger,
		serviceAccountService: serviceAccountService,
	}
}

func (h *K8sServiceAccountHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/v1/k8s")

	serviceAccountGroup := k8sGroup.Group("/serviceaccount")
	{
		serviceAccountGroup.GET("/list", h.GetServiceAccountList)       // 获取ServiceAccount列表
		serviceAccountGroup.GET("/details", h.GetServiceAccountDetails) // 获取ServiceAccount详情
		serviceAccountGroup.POST("/create", h.CreateServiceAccount)     // 创建ServiceAccount
		serviceAccountGroup.PUT("/update", h.UpdateServiceAccount)      // 更新ServiceAccount
		serviceAccountGroup.DELETE("/delete", h.DeleteServiceAccount)   // 删除ServiceAccount

		serviceAccountGroup.GET("/statistics", h.GetServiceAccountStatistics) // 获取ServiceAccount统计信息
		serviceAccountGroup.POST("/token", h.GetServiceAccountToken)          // 获取ServiceAccount令牌
		serviceAccountGroup.GET("/yaml", h.GetServiceAccountYaml)             // 获取ServiceAccount YAML
		serviceAccountGroup.PUT("/yaml", h.UpdateServiceAccountYaml)          // 更新ServiceAccount YAML
	}
}

// GetServiceAccountList 获取ServiceAccount列表
// @Summary 获取ServiceAccount列表
// @Description 根据指定条件获取K8s集群中的ServiceAccount列表
// @Tags ServiceAccount管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string false "命名空间"
// @Param label_selector query string false "标签选择器"
// @Param field_selector query string false "字段选择器"
// @Param page query int false "页码"
// @Param page_size query int false "每页大小"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sServiceAccountResponse} "成功获取ServiceAccount列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/v1/k8s/serviceaccount/list [get]
func (h *K8sServiceAccountHandler) GetServiceAccountList(ctx *gin.Context) {
	var req model.ServiceAccountListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.serviceAccountService.GetServiceAccountList(ctx, &req)
	})
}

// GetServiceAccountDetails 获取ServiceAccount详情
// @Summary 获取ServiceAccount详情
// @Description 获取指定ServiceAccount的详细信息
// @Tags ServiceAccount管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string true "命名空间"
// @Param name query string true "ServiceAccount名称"
// @Success 200 {object} utils.ApiResponse{data=model.K8sServiceAccountResponse} "成功获取ServiceAccount详情"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/v1/k8s/serviceaccount/details [get]
func (h *K8sServiceAccountHandler) GetServiceAccountDetails(ctx *gin.Context) {
	clusterIDStr := ctx.Query("cluster_id")
	namespace := ctx.Query("namespace")
	name := ctx.Query("name")

	if clusterIDStr == "" || namespace == "" || name == "" {
		utils.BadRequestError(ctx, "缺少必要参数: cluster_id, namespace, name")
		return
	}

	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		utils.BadRequestError(ctx, "cluster_id必须是有效的整数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.serviceAccountService.GetServiceAccountDetails(ctx, clusterID, namespace, name)
	})
}

// CreateServiceAccount 创建ServiceAccount
// @Summary 创建ServiceAccount
// @Description 在指定命名空间中创建新的ServiceAccount
// @Tags ServiceAccount管理
// @Accept json
// @Produce json
// @Param request body model.ServiceAccountCreateReq true "创建ServiceAccount的请求参数"
// @Success 200 {object} utils.ApiResponse "成功创建ServiceAccount"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/v1/k8s/serviceaccount/create [post]
func (h *K8sServiceAccountHandler) CreateServiceAccount(ctx *gin.Context) {
	var req model.ServiceAccountCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.serviceAccountService.CreateServiceAccount(ctx, &req)
	})
}

// UpdateServiceAccount 更新ServiceAccount
// @Summary 更新ServiceAccount
// @Description 更新指定的ServiceAccount配置
// @Tags ServiceAccount管理
// @Accept json
// @Produce json
// @Param request body model.ServiceAccountUpdateReq true "更新ServiceAccount的请求参数"
// @Success 200 {object} utils.ApiResponse "成功更新ServiceAccount"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/v1/k8s/serviceaccount/update [put]
func (h *K8sServiceAccountHandler) UpdateServiceAccount(ctx *gin.Context) {
	var req model.ServiceAccountUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.serviceAccountService.UpdateServiceAccount(ctx, &req)
	})
}

// DeleteServiceAccount 删除ServiceAccount
// @Summary 删除ServiceAccount
// @Description 删除指定的ServiceAccount
// @Tags ServiceAccount管理
// @Accept json
// @Produce json
// @Param request body model.ServiceAccountDeleteReq true "删除ServiceAccount的请求参数"
// @Success 200 {object} utils.ApiResponse "成功删除ServiceAccount"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/v1/k8s/serviceaccount/delete [delete]
func (h *K8sServiceAccountHandler) DeleteServiceAccount(ctx *gin.Context) {
	var req model.ServiceAccountDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.serviceAccountService.DeleteServiceAccount(ctx, &req)
	})
}

// GetServiceAccountStatistics 获取ServiceAccount统计信息
// @Summary 获取ServiceAccount统计信息
// @Description 获取ServiceAccount的统计信息，包括总数、活跃数等
// @Tags ServiceAccount管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string false "命名空间，不指定则统计所有命名空间"
// @Success 200 {object} utils.ApiResponse{data=model.ServiceAccountStatisticsResp} "成功获取ServiceAccount统计信息"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/v1/k8s/serviceaccount/statistics [get]
func (h *K8sServiceAccountHandler) GetServiceAccountStatistics(ctx *gin.Context) {
	var req model.ServiceAccountStatisticsReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.serviceAccountService.GetServiceAccountStatistics(ctx, &req)
	})
}

// GetServiceAccountToken 获取ServiceAccount令牌
// @Summary 获取ServiceAccount令牌
// @Description 为指定的ServiceAccount生成访问令牌
// @Tags ServiceAccount管理
// @Accept json
// @Produce json
// @Param request body model.ServiceAccountTokenReq true "获取ServiceAccount令牌的请求参数"
// @Success 200 {object} utils.ApiResponse{data=model.ServiceAccountTokenResp} "成功获取ServiceAccount令牌"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/v1/k8s/serviceaccount/token [post]
func (h *K8sServiceAccountHandler) GetServiceAccountToken(ctx *gin.Context) {
	var req model.ServiceAccountTokenReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.serviceAccountService.GetServiceAccountToken(ctx, &req)
	})
}

// GetServiceAccountYaml 获取ServiceAccount的YAML配置
// @Summary 获取ServiceAccount的YAML配置
// @Description 获取指定ServiceAccount的完整YAML配置文件
// @Tags ServiceAccount管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string true "命名空间"
// @Param name query string true "ServiceAccount名称"
// @Success 200 {object} utils.ApiResponse{data=model.ServiceAccountYamlResp} "成功获取YAML配置"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/v1/k8s/serviceaccount/yaml [get]
func (h *K8sServiceAccountHandler) GetServiceAccountYaml(ctx *gin.Context) {
	var req model.ServiceAccountYamlReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.serviceAccountService.GetServiceAccountYaml(ctx, &req)
	})
}

// UpdateServiceAccountYaml 更新ServiceAccount的YAML配置
// @Summary 更新ServiceAccount的YAML配置
// @Description 通过YAML更新指定ServiceAccount的配置
// @Tags ServiceAccount管理
// @Accept json
// @Produce json
// @Param request body model.ServiceAccountUpdateYamlReq true "更新ServiceAccount YAML的请求参数"
// @Success 200 {object} utils.ApiResponse "成功更新YAML配置"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/v1/k8s/serviceaccount/yaml [put]
func (h *K8sServiceAccountHandler) UpdateServiceAccountYaml(ctx *gin.Context) {
	var req model.ServiceAccountUpdateYamlReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.serviceAccountService.UpdateServiceAccountYaml(ctx, &req)
	})
}

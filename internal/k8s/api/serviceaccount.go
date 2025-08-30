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
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/serviceaccount/list", h.GetServiceAccountList)             // 获取ServiceAccount列表
		k8sGroup.GET("/serviceaccount/details", h.GetServiceAccountDetails)       // 获取ServiceAccount详情
		k8sGroup.POST("/serviceaccount/create", h.CreateServiceAccount)           // 创建ServiceAccount
		k8sGroup.PUT("/serviceaccount/update", h.UpdateServiceAccount)            // 更新ServiceAccount
		k8sGroup.DELETE("/serviceaccount/delete", h.DeleteServiceAccount)         // 删除ServiceAccount
		k8sGroup.GET("/serviceaccount/statistics", h.GetServiceAccountStatistics) // 获取ServiceAccount统计信息
		k8sGroup.POST("/serviceaccount/token", h.GetServiceAccountToken)          // 获取ServiceAccount令牌
		k8sGroup.GET("/serviceaccount/yaml", h.GetServiceAccountYaml)             // 获取ServiceAccount YAML
		k8sGroup.PUT("/serviceaccount/yaml", h.UpdateServiceAccountYaml)          // 更新ServiceAccount YAML
	}
}

// GetServiceAccountList 获取ServiceAccount列表
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

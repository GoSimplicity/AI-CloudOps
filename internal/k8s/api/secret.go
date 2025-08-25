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

type K8sSecretHandler struct {
	logger        *zap.Logger
	secretService service.SecretService
}

func NewK8sSecretHandler(logger *zap.Logger, secretService service.SecretService) *K8sSecretHandler {
	return &K8sSecretHandler{
		logger:        logger,
		secretService: secretService,
	}
}

func (h *K8sSecretHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	secrets := k8sGroup.Group("/secrets")
	{
		secrets.GET("/list", h.GetSecretList)                           // 获取Secret列表
		secrets.GET("/:cluster_id/:namespace/:name", h.GetSecret)       // 获取单个Secret详情
		secrets.POST("/create", h.CreateSecret)                         // 创建Secret
		secrets.PUT("/update", h.UpdateSecret)                          // 更新Secret
		secrets.DELETE("/:cluster_id/:namespace/:name", h.DeleteSecret) // 删除Secret
		secrets.DELETE("/batch", h.BatchDeleteSecrets)                  // 批量删除Secret
		secrets.GET("/:cluster_id/:namespace/:name/yaml", h.GetSecretYAML) // 获取Secret的YAML配置
	}
}

// GetSecretList 获取Secret列表
// @Summary 获取Secret列表
// @Description 根据集群和命名空间获取Secret列表，支持标签和字段选择器过滤
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string false "命名空间，为空时获取所有命名空间"
// @Param label_selector query string false "标签选择器"
// @Param field_selector query string false "字段选择器"
// @Param limit query int false "限制结果数量"
// @Param continue query string false "分页续订令牌"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sSecret} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/list [get]
// @Security BearerAuth
func (h *K8sSecretHandler) GetSecretList(ctx *gin.Context) {
	var req model.K8sListRequest
	
	// 从查询参数中获取请求参数
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, "参数绑定错误: "+err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.secretService.GetSecretList(ctx, &req)
	})
}

// GetSecret 获取单个Secret详情
// @Summary 获取Secret详情
// @Description 根据集群ID、命名空间和名称获取指定Secret的详细信息
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间"
// @Param name path string true "Secret名称"
// @Success 200 {object} utils.ApiResponse{data=model.K8sSecret} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 404 {object} utils.ApiResponse "Secret不存在"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/{cluster_id}/{namespace}/{name} [get]
// @Security BearerAuth
func (h *K8sSecretHandler) GetSecret(ctx *gin.Context) {
	var req model.K8sResourceIdentifier
	
	// 从路径参数中获取请求参数
	clusterIDStr := ctx.Param("cluster_id")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		utils.BadRequestError(ctx, "无效的集群ID: "+err.Error())
		return
	}
	req.ClusterID = clusterID
	
	req.Namespace = ctx.Param("namespace")
	req.ResourceName = ctx.Param("name")
	
	// 验证必要参数
	if req.Namespace == "" || req.ResourceName == "" {
		utils.BadRequestError(ctx, "命名空间和Secret名称不能为空")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.secretService.GetSecret(ctx, &req)
	})
}

// CreateSecret 创建Secret
// @Summary 创建Secret
// @Description 在指定集群和命名空间中创建新的Secret
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param request body model.SecretCreateRequest true "Secret创建请求"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 409 {object} utils.ApiResponse "Secret已存在"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/create [post]
// @Security BearerAuth
func (h *K8sSecretHandler) CreateSecret(ctx *gin.Context) {
	var req model.SecretCreateRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.secretService.CreateSecret(ctx, &req)
	})
}

// UpdateSecret 更新Secret
// @Summary 更新Secret
// @Description 更新指定的Secret配置数据
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param request body model.SecretUpdateRequest true "Secret更新请求"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 404 {object} utils.ApiResponse "Secret不存在"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/update [put]
// @Security BearerAuth
func (h *K8sSecretHandler) UpdateSecret(ctx *gin.Context) {
	var req model.SecretUpdateRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.secretService.UpdateSecret(ctx, &req)
	})
}

// DeleteSecret 删除Secret
// @Summary 删除Secret
// @Description 删除指定的Secret资源
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间"
// @Param name path string true "Secret名称"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 404 {object} utils.ApiResponse "Secret不存在"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/{cluster_id}/{namespace}/{name} [delete]
// @Security BearerAuth
func (h *K8sSecretHandler) DeleteSecret(ctx *gin.Context) {
	var req model.K8sResourceIdentifier
	
	// 从路径参数中获取请求参数
	clusterIDStr := ctx.Param("cluster_id")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		utils.BadRequestError(ctx, "无效的集群ID: "+err.Error())
		return
	}
	req.ClusterID = clusterID
	
	req.Namespace = ctx.Param("namespace")
	req.ResourceName = ctx.Param("name")
	
	// 验证必要参数
	if req.Namespace == "" || req.ResourceName == "" {
		utils.BadRequestError(ctx, "命名空间和Secret名称不能为空")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.secretService.DeleteSecret(ctx, &req)
	})
}

// BatchDeleteSecrets 批量删除Secret
// @Summary 批量删除Secret
// @Description 批量删除指定命名空间中的多个Secret
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param request body model.K8sBatchDeleteRequest true "批量删除请求"
// @Success 200 {object} utils.ApiResponse "批量删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/batch [delete]
// @Security BearerAuth
func (h *K8sSecretHandler) BatchDeleteSecrets(ctx *gin.Context) {
	var req model.K8sBatchDeleteRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.secretService.BatchDeleteSecrets(ctx, &req)
	})
}

// GetSecretYAML 获取Secret的YAML配置
// @Summary 获取Secret的YAML配置
// @Description 获取指定Secret的完整YAML配置文件
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间"
// @Param name path string true "Secret名称"
// @Success 200 {object} utils.ApiResponse{data=string} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 404 {object} utils.ApiResponse "Secret不存在"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/{cluster_id}/{namespace}/{name}/yaml [get]
// @Security BearerAuth
func (h *K8sSecretHandler) GetSecretYAML(ctx *gin.Context) {
	var req model.K8sResourceIdentifier
	
	// 从路径参数中获取请求参数
	clusterIDStr := ctx.Param("cluster_id")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		utils.BadRequestError(ctx, "无效的集群ID: "+err.Error())
		return
	}
	req.ClusterID = clusterID
	
	req.Namespace = ctx.Param("namespace")
	req.ResourceName = ctx.Param("name")
	
	// 验证必要参数
	if req.Namespace == "" || req.ResourceName == "" {
		utils.BadRequestError(ctx, "命名空间和Secret名称不能为空")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.secretService.GetSecretYAML(ctx, &req)
	})
}
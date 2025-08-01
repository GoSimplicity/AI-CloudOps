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
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sSecretHandler struct {
	l             *zap.Logger
	secretService admin.SecretService
}

func NewK8sSecretHandler(l *zap.Logger, secretService admin.SecretService) *K8sSecretHandler {
	return &K8sSecretHandler{
		l:             l,
		secretService: secretService,
	}
}

func (k *K8sSecretHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	secrets := k8sGroup.Group("/secrets")
	{
		secrets.GET("/:id", k.GetSecretsByNamespace)          // 根据命名空间获取 Secret 列表
		secrets.POST("/create", k.CreateSecret)               // 创建 Secret
		secrets.POST("/create_encrypted", k.CreateEncryptedSecret) // 创建加密 Secret
		secrets.POST("/update", k.UpdateSecret)               // 更新 Secret
		secrets.DELETE("/delete/:id", k.DeleteSecret)         // 删除指定 Secret
		secrets.DELETE("/batch_delete", k.BatchDeleteSecret)  // 批量删除 Secret
		secrets.GET("/:id/yaml", k.GetSecretYaml)            // 获取 Secret YAML 配置
		secrets.GET("/:id/status", k.GetSecretStatus)        // 获取 Secret 状态
		secrets.GET("/:id/types", k.GetSupportedSecretTypes) // 获取支持的 Secret 类型
		secrets.POST("/:id/decrypt", k.DecryptSecret)        // 解密 Secret 数据
	}
}

// GetSecretsByNamespace 根据命名空间获取 Secret 列表
// @Summary 获取Secret列表
// @Description 根据指定的集群ID和命名空间查询所有的Secret资源
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param namespace query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse{data=[]interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/{id} [get]
// @Security BearerAuth
func (k *K8sSecretHandler) GetSecretsByNamespace(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.secretService.GetSecretsByNamespace(ctx, id, namespace)
	})
}

// CreateSecret 创建 Secret
// @Summary 创建Secret
// @Description 在指定集群的命名空间中创建新的Secret资源
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param request body model.K8sSecretRequest true "Secret创建信息"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/create [post]
// @Security BearerAuth
func (k *K8sSecretHandler) CreateSecret(ctx *gin.Context) {
	var req model.K8sSecretRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.secretService.CreateSecret(ctx, &req)
	})
}

// CreateEncryptedSecret 创建加密的 Secret
// @Summary 创建加密Secret
// @Description 创建带有加密数据的Secret资源，提供额外的安全保护
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param request body model.K8sSecretEncryptionRequest true "加密Secret创建信息"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/create_encrypted [post]
// @Security BearerAuth
func (k *K8sSecretHandler) CreateEncryptedSecret(ctx *gin.Context) {
	var req model.K8sSecretEncryptionRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.secretService.CreateEncryptedSecret(ctx, &req)
	})
}

// UpdateSecret 更新 Secret
// @Summary 更新Secret
// @Description 修改指定Secret的数据内容和配置信息
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param request body model.K8sSecretRequest true "Secret更新信息"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/update [post]
// @Security BearerAuth
func (k *K8sSecretHandler) UpdateSecret(ctx *gin.Context) {
	var req model.K8sSecretRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.secretService.UpdateSecret(ctx, &req)
	})
}

// BatchDeleteSecret 批量删除 Secret
// @Summary 批量删除Secret
// @Description 同时删除指定命名空间下的多个Secret资源
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param request body model.K8sSecretRequest true "批量删除请求"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/batch_delete [delete]
// @Security BearerAuth
func (k *K8sSecretHandler) BatchDeleteSecret(ctx *gin.Context) {
	var req model.K8sSecretRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.secretService.BatchDeleteSecret(ctx, req.ClusterID, req.Namespace, req.SecretNames)
	})
}

// GetSecretYaml 获取 Secret 的 YAML 配置
// @Summary 获取Secret的YAML配置
// @Description 以YAML格式返回指定Secret的完整配置信息
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param secret_name query string true "Secret名称"
// @Param namespace query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse{data=string} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/{id}/yaml [get]
// @Security BearerAuth
func (k *K8sSecretHandler) GetSecretYaml(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	secretName := ctx.Query("secret_name")
	if secretName == "" {
		k.l.Error("缺少必需的 secret_name 参数")
		utils.BadRequestError(ctx, "缺少 'secret_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.secretService.GetSecretYaml(ctx, id, namespace, secretName)
	})
}

// DeleteSecret 删除指定的 Secret
// @Summary 删除单个Secret
// @Description 删除指定命名空间下的单个Secret资源
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param secret_name query string true "Secret名称"
// @Param namespace query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/delete/{id} [delete]
// @Security BearerAuth
func (k *K8sSecretHandler) DeleteSecret(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	secretName := ctx.Query("secret_name")
	if secretName == "" {
		k.l.Error("缺少必需的 secret_name 参数")
		utils.BadRequestError(ctx, "缺少 'secret_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.secretService.DeleteSecret(ctx, id, namespace, secretName)
	})
}

// GetSecretStatus 获取 Secret 状态
// @Summary 获取Secret状态
// @Description 获取指定Secret的详细状态信息，包括创建时间、类型等
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param secret_name query string true "Secret名称"
// @Param namespace query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/{id}/status [get]
// @Security BearerAuth
func (k *K8sSecretHandler) GetSecretStatus(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	secretName := ctx.Query("secret_name")
	if secretName == "" {
		k.l.Error("缺少必需的 secret_name 参数")
		utils.BadRequestError(ctx, "缺少 'secret_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.secretService.GetSecretStatus(ctx, id, namespace, secretName)
	})
}

// GetSupportedSecretTypes 获取支持的 Secret 类型
// @Summary 获取支持的Secret类型
// @Description 获取当前集群支持的所有Secret类型列表
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} utils.ApiResponse{data=[]interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/{id}/types [get]
// @Security BearerAuth
func (k *K8sSecretHandler) GetSupportedSecretTypes(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.secretService.GetSupportedSecretTypes(ctx, id)
	})
}

// DecryptSecret 解密 Secret 数据
// @Summary 解密Secret数据
// @Description 解密指定Secret中的加密数据，返回明文内容
// @Tags 密钥管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param secret_name query string true "Secret名称"
// @Param namespace query string true "命名空间名称"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "解密成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/secrets/{id}/decrypt [post]
// @Security BearerAuth
func (k *K8sSecretHandler) DecryptSecret(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	secretName := ctx.Query("secret_name")
	if secretName == "" {
		k.l.Error("缺少必需的 secret_name 参数")
		utils.BadRequestError(ctx, "缺少 'secret_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.secretService.DecryptSecret(ctx, id, namespace, secretName)
	})
}
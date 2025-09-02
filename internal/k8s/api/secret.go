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
)

type K8sSecretHandler struct {
	secretService service.SecretService
}

func NewK8sSecretHandler(secretService service.SecretService) *K8sSecretHandler {
	return &K8sSecretHandler{

		secretService: secretService,
	}
}

func (h *K8sSecretHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/secrets/list", h.GetSecretList)                              // 获取Secret列表
		k8sGroup.GET("/secrets/:cluster_id/:namespace/:name", h.GetSecret)          // 获取单个Secret详情
		k8sGroup.POST("/secrets/create", h.CreateSecret)                            // 创建Secret
		k8sGroup.PUT("/secrets/update", h.UpdateSecret)                             // 更新Secret
		k8sGroup.DELETE("/secrets/:cluster_id/:namespace/:name", h.DeleteSecret)    // 删除Secret
		k8sGroup.GET("/secrets/:cluster_id/:namespace/:name/yaml", h.GetSecretYAML) // 获取Secret的YAML配置

		// YAML操作
		k8sGroup.POST("/secrets/yaml", h.CreateSecretByYaml)                             // 通过YAML创建Secret
		k8sGroup.PUT("/secrets/:cluster_id/:namespace/:name/yaml", h.UpdateSecretByYaml) // 通过YAML更新Secret
	}
}

// GetSecretList 获取Secret列表
func (h *K8sSecretHandler) GetSecretList(ctx *gin.Context) {
	var req model.K8sListReq

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
func (h *K8sSecretHandler) GetSecret(ctx *gin.Context) {
	var req model.K8sResourceIdentifierReq

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
func (h *K8sSecretHandler) CreateSecret(ctx *gin.Context) {
	var req model.SecretCreateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.secretService.CreateSecret(ctx, &req)
	})
}

// UpdateSecret 更新Secret
func (h *K8sSecretHandler) UpdateSecret(ctx *gin.Context) {
	var req model.SecretUpdateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.secretService.UpdateSecret(ctx, &req)
	})
}

// DeleteSecret 删除Secret
func (h *K8sSecretHandler) DeleteSecret(ctx *gin.Context) {
	var req model.K8sResourceIdentifierReq

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

// GetSecretYAML 获取Secret的YAML配置
func (h *K8sSecretHandler) GetSecretYAML(ctx *gin.Context) {
	var req model.K8sResourceIdentifierReq

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

// YAML操作方法

// CreateSecretByYaml 通过YAML创建Secret
func (h *K8sSecretHandler) CreateSecretByYaml(ctx *gin.Context) {
	var req model.CreateResourceByYamlReq
	req.ResourceType = model.ResourceTypeSecret

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.secretService.CreateSecretByYaml(ctx, &req)
	})
}

// UpdateSecretByYaml 通过YAML更新Secret
func (h *K8sSecretHandler) UpdateSecretByYaml(ctx *gin.Context) {
	var req model.UpdateResourceByYamlReq
	req.ResourceType = model.ResourceTypeSecret

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.secretService.UpdateSecretByYaml(ctx, &req)
	})
}

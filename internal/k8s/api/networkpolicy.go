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

type K8sNetworkPolicyHandler struct {
	l                     *zap.Logger
	networkPolicyService  admin.NetworkPolicyService
}

func NewK8sNetworkPolicyHandler(l *zap.Logger, networkPolicyService admin.NetworkPolicyService) *K8sNetworkPolicyHandler {
	return &K8sNetworkPolicyHandler{
		l:                     l,
		networkPolicyService:  networkPolicyService,
	}
}

func (k *K8sNetworkPolicyHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	networkPolicies := k8sGroup.Group("/networkpolicies")
	{
		networkPolicies.GET("/:id", k.GetNetworkPoliciesByNamespace)          // 根据命名空间获取 NetworkPolicy 列表
		networkPolicies.POST("/create", k.CreateNetworkPolicy)               // 创建 NetworkPolicy
		networkPolicies.POST("/update", k.UpdateNetworkPolicy)               // 更新 NetworkPolicy
		networkPolicies.DELETE("/delete/:id", k.DeleteNetworkPolicy)         // 删除指定 NetworkPolicy
		networkPolicies.DELETE("/batch_delete", k.BatchDeleteNetworkPolicy)  // 批量删除 NetworkPolicy
		networkPolicies.GET("/:id/yaml", k.GetNetworkPolicyYaml)            // 获取 NetworkPolicy YAML 配置
		networkPolicies.GET("/:id/status", k.GetNetworkPolicyStatus)        // 获取 NetworkPolicy 状态
		networkPolicies.GET("/:id/rules", k.GetNetworkPolicyRules)          // 获取 NetworkPolicy 规则
		networkPolicies.GET("/:id/pods", k.GetAffectedPods)                 // 获取受影响的 Pod
		networkPolicies.POST("/:id/validate", k.ValidateNetworkPolicy)      // 验证 NetworkPolicy 配置
	}
}

// GetNetworkPoliciesByNamespace 根据命名空间获取 NetworkPolicy 列表
func (k *K8sNetworkPolicyHandler) GetNetworkPoliciesByNamespace(ctx *gin.Context) {
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
		return k.networkPolicyService.GetNetworkPoliciesByNamespace(ctx, id, namespace)
	})
}

// CreateNetworkPolicy 创建 NetworkPolicy
func (k *K8sNetworkPolicyHandler) CreateNetworkPolicy(ctx *gin.Context) {
	var req model.K8sNetworkPolicyRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.networkPolicyService.CreateNetworkPolicy(ctx, &req)
	})
}

// UpdateNetworkPolicy 更新 NetworkPolicy
func (k *K8sNetworkPolicyHandler) UpdateNetworkPolicy(ctx *gin.Context) {
	var req model.K8sNetworkPolicyRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.networkPolicyService.UpdateNetworkPolicy(ctx, &req)
	})
}

// BatchDeleteNetworkPolicy 批量删除 NetworkPolicy
func (k *K8sNetworkPolicyHandler) BatchDeleteNetworkPolicy(ctx *gin.Context) {
	var req model.K8sNetworkPolicyRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.networkPolicyService.BatchDeleteNetworkPolicy(ctx, req.ClusterID, req.Namespace, req.NetworkPolicyNames)
	})
}

// GetNetworkPolicyYaml 获取 NetworkPolicy 的 YAML 配置
func (k *K8sNetworkPolicyHandler) GetNetworkPolicyYaml(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	networkPolicyName := ctx.Query("network_policy_name")
	if networkPolicyName == "" {
		k.l.Error("缺少必需的 network_policy_name 参数")
		utils.BadRequestError(ctx, "缺少 'network_policy_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.networkPolicyService.GetNetworkPolicyYaml(ctx, id, namespace, networkPolicyName)
	})
}

// DeleteNetworkPolicy 删除指定的 NetworkPolicy
func (k *K8sNetworkPolicyHandler) DeleteNetworkPolicy(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	networkPolicyName := ctx.Query("network_policy_name")
	if networkPolicyName == "" {
		k.l.Error("缺少必需的 network_policy_name 参数")
		utils.BadRequestError(ctx, "缺少 'network_policy_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.networkPolicyService.DeleteNetworkPolicy(ctx, id, namespace, networkPolicyName)
	})
}

// GetNetworkPolicyStatus 获取 NetworkPolicy 状态
func (k *K8sNetworkPolicyHandler) GetNetworkPolicyStatus(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	networkPolicyName := ctx.Query("network_policy_name")
	if networkPolicyName == "" {
		k.l.Error("缺少必需的 network_policy_name 参数")
		utils.BadRequestError(ctx, "缺少 'network_policy_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.networkPolicyService.GetNetworkPolicyStatus(ctx, id, namespace, networkPolicyName)
	})
}

// GetNetworkPolicyRules 获取 NetworkPolicy 规则
func (k *K8sNetworkPolicyHandler) GetNetworkPolicyRules(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	networkPolicyName := ctx.Query("network_policy_name")
	if networkPolicyName == "" {
		k.l.Error("缺少必需的 network_policy_name 参数")
		utils.BadRequestError(ctx, "缺少 'network_policy_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.networkPolicyService.GetNetworkPolicyRules(ctx, id, namespace, networkPolicyName)
	})
}

// GetAffectedPods 获取受 NetworkPolicy 影响的 Pod
func (k *K8sNetworkPolicyHandler) GetAffectedPods(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	networkPolicyName := ctx.Query("network_policy_name")
	if networkPolicyName == "" {
		k.l.Error("缺少必需的 network_policy_name 参数")
		utils.BadRequestError(ctx, "缺少 'network_policy_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.networkPolicyService.GetAffectedPods(ctx, id, namespace, networkPolicyName)
	})
}

// ValidateNetworkPolicy 验证 NetworkPolicy 配置
func (k *K8sNetworkPolicyHandler) ValidateNetworkPolicy(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	var req model.K8sNetworkPolicyRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.networkPolicyService.ValidateNetworkPolicy(ctx, id, &req)
	})
}
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

type K8sIngressHandler struct {
	l              *zap.Logger
	ingressService admin.IngressService
}

func NewK8sIngressHandler(l *zap.Logger, ingressService admin.IngressService) *K8sIngressHandler {
	return &K8sIngressHandler{
		l:              l,
		ingressService: ingressService,
	}
}

func (k *K8sIngressHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	ingresses := k8sGroup.Group("/ingresses")
	{
		ingresses.GET("/:id", k.GetIngressesByNamespace)        // 根据命名空间获取 Ingress 列表
		ingresses.POST("/create", k.CreateIngress)              // 创建 Ingress
		ingresses.POST("/update", k.UpdateIngress)              // 更新 Ingress
		ingresses.DELETE("/delete/:id", k.DeleteIngress)        // 删除指定 Ingress
		ingresses.DELETE("/batch_delete", k.BatchDeleteIngress) // 批量删除 Ingress
		ingresses.GET("/:id/yaml", k.GetIngressYaml)            // 获取 Ingress YAML 配置
		ingresses.GET("/:id/status", k.GetIngressStatus)        // 获取 Ingress 状态
		ingresses.GET("/:id/rules", k.GetIngressRules)          // 获取 Ingress 规则
		ingresses.GET("/:id/tls", k.GetIngressTLS)              // 获取 Ingress TLS 配置
		ingresses.GET("/:id/endpoints", k.GetIngressEndpoints)  // 获取 Ingress 后端端点
	}
}

func (k *K8sIngressHandler) GetIngressesByNamespace(ctx *gin.Context) {
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
		return k.ingressService.GetIngressesByNamespace(ctx, id, namespace)
	})
}

func (k *K8sIngressHandler) CreateIngress(ctx *gin.Context) {
	var req model.K8sIngressRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.ingressService.CreateIngress(ctx, &req)
	})
}

func (k *K8sIngressHandler) UpdateIngress(ctx *gin.Context) {
	var req model.K8sIngressRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.ingressService.UpdateIngress(ctx, &req)
	})
}

func (k *K8sIngressHandler) BatchDeleteIngress(ctx *gin.Context) {
	var req model.K8sIngressRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.ingressService.BatchDeleteIngress(ctx, req.ClusterID, req.Namespace, req.IngressNames)
	})
}

func (k *K8sIngressHandler) GetIngressYaml(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	ingressName := ctx.Query("ingress_name")
	if ingressName == "" {
		k.l.Error("缺少必需的 ingress_name 参数")
		utils.BadRequestError(ctx, "缺少 'ingress_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.ingressService.GetIngressYaml(ctx, id, namespace, ingressName)
	})
}

func (k *K8sIngressHandler) DeleteIngress(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	ingressName := ctx.Query("ingress_name")
	if ingressName == "" {
		k.l.Error("缺少必需的 ingress_name 参数")
		utils.BadRequestError(ctx, "缺少 'ingress_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.ingressService.DeleteIngress(ctx, id, namespace, ingressName)
	})
}

func (k *K8sIngressHandler) GetIngressStatus(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	ingressName := ctx.Query("ingress_name")
	if ingressName == "" {
		k.l.Error("缺少必需的 ingress_name 参数")
		utils.BadRequestError(ctx, "缺少 'ingress_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.ingressService.GetIngressStatus(ctx, id, namespace, ingressName)
	})
}

func (k *K8sIngressHandler) GetIngressRules(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	ingressName := ctx.Query("ingress_name")
	if ingressName == "" {
		k.l.Error("缺少必需的 ingress_name 参数")
		utils.BadRequestError(ctx, "缺少 'ingress_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.ingressService.GetIngressRules(ctx, id, namespace, ingressName)
	})
}

func (k *K8sIngressHandler) GetIngressTLS(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	ingressName := ctx.Query("ingress_name")
	if ingressName == "" {
		k.l.Error("缺少必需的 ingress_name 参数")
		utils.BadRequestError(ctx, "缺少 'ingress_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.ingressService.GetIngressTLS(ctx, id, namespace, ingressName)
	})
}

func (k *K8sIngressHandler) GetIngressEndpoints(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	ingressName := ctx.Query("ingress_name")
	if ingressName == "" {
		k.l.Error("缺少必需的 ingress_name 参数")
		utils.BadRequestError(ctx, "缺少 'ingress_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.ingressService.GetIngressEndpoints(ctx, id, namespace, ingressName)
	})
}

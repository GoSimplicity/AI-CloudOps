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
)

type K8sIngressHandler struct {
	ingressService service.IngressService
}

func NewK8sIngressHandler(ingressService service.IngressService) *K8sIngressHandler {
	return &K8sIngressHandler{

		ingressService: ingressService,
	}
}

func (k *K8sIngressHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/ingresses/list", k.GetIngressList)                                        // 获取Ingress列表
		k8sGroup.GET("/ingresses/:cluster_id", k.GetIngressesByNamespace)                        // 根据命名空间获取Ingress列表
		k8sGroup.GET("/ingresses/:cluster_id/:name", k.GetIngress)                               // 获取单个Ingress详情
		k8sGroup.GET("/ingresses/:cluster_id/:name/yaml", k.GetIngressYaml)                      // 获取Ingress YAML配置
		k8sGroup.POST("/ingresses/create", k.CreateIngress)                                      // 创建Ingress
		k8sGroup.PUT("/ingresses/update", k.UpdateIngress)                                       // 更新Ingress
		k8sGroup.DELETE("/ingresses/delete", k.DeleteIngress)                                    // 删除Ingress
		k8sGroup.GET("/ingresses/:cluster_id/:name/events", k.GetIngressEvents)                  // 获取Ingress事件
		k8sGroup.POST("/ingresses/:cluster_id/:name/tls-test", k.TestIngressTLS)                 // 测试Ingress TLS证书
		k8sGroup.GET("/ingresses/:cluster_id/:name/backend-health", k.CheckIngressBackendHealth) // 检查后端健康状态
	}
}

// GetIngressList 获取Ingress列表
func (k *K8sIngressHandler) GetIngressList(ctx *gin.Context) {
	var req model.K8sIngressListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.ingressService.GetIngressList(ctx, &req)
	})
}

// GetIngressesByNamespace 根据命名空间获取Ingress列表
func (k *K8sIngressHandler) GetIngressesByNamespace(ctx *gin.Context) {
	var req model.K8sGetResourceListReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.ingressService.GetIngressesByNamespace(ctx, req.ClusterID, req.Namespace)
	})
}

// GetIngress 获取Ingress详情
func (k *K8sIngressHandler) GetIngress(ctx *gin.Context) {
	var req model.K8sGetResourceReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.ingressService.GetIngress(ctx, req.ClusterID, req.Namespace, req.ResourceName)
	})
}

// GetIngressYaml 获取Ingress的YAML配置
func (k *K8sIngressHandler) GetIngressYaml(ctx *gin.Context) {
	var req model.K8sGetResourceYamlReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.ingressService.GetIngressYaml(ctx, req.ClusterID, req.Namespace, req.ResourceName)
	})
}

// CreateIngress 创建Ingress
func (k *K8sIngressHandler) CreateIngress(ctx *gin.Context) {
	var req model.K8sIngressCreateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.ingressService.CreateIngress(ctx, &req)
	})
}

// UpdateIngress 更新Ingress
func (k *K8sIngressHandler) UpdateIngress(ctx *gin.Context) {
	var req model.K8sIngressUpdateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.ingressService.UpdateIngress(ctx, &req)
	})
}

// DeleteIngress 删除Ingress
func (k *K8sIngressHandler) DeleteIngress(ctx *gin.Context) {
	var req model.K8sIngressDeleteReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.ingressService.DeleteIngress(ctx, &req)
	})
}

// GetIngressEvents 获取Ingress事件
func (k *K8sIngressHandler) GetIngressEvents(ctx *gin.Context) {
	var req model.K8sIngressEventReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.ingressService.GetIngressEvents(ctx, &req)
	})
}

// TestIngressTLS 测试Ingress TLS证书
func (k *K8sIngressHandler) TestIngressTLS(ctx *gin.Context) {
	var req model.K8sIngressTLSTestReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.ingressService.TestIngressTLS(ctx, &req)
	})
}

// CheckIngressBackendHealth 检查Ingress后端健康状态
func (k *K8sIngressHandler) CheckIngressBackendHealth(ctx *gin.Context) {
	var req model.K8sIngressBackendHealthReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.ingressService.CheckIngressBackendHealth(ctx, &req)
	})
}

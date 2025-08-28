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
	"go.uber.org/zap"
)

type K8sIngressHandler struct {
	logger         *zap.Logger
	ingressService service.IngressService
}

func NewK8sIngressHandler(logger *zap.Logger, ingressService service.IngressService) *K8sIngressHandler {
	return &K8sIngressHandler{
		logger:         logger,
		ingressService: ingressService,
	}
}

func (k *K8sIngressHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	ingresses := k8sGroup.Group("/ingresses")
	{
		// 基础操作
		ingresses.GET("/list", k.GetIngressList)                   // 获取Ingress列表
		ingresses.GET("/:cluster_id", k.GetIngressesByNamespace)   // 根据命名空间获取Ingress列表
		ingresses.GET("/:cluster_id/:name", k.GetIngress)          // 获取单个Ingress详情
		ingresses.GET("/:cluster_id/:name/yaml", k.GetIngressYaml) // 获取Ingress YAML配置
		ingresses.POST("/create", k.CreateIngress)                 // 创建Ingress
		ingresses.PUT("/update", k.UpdateIngress)                  // 更新Ingress
		ingresses.DELETE("/delete", k.DeleteIngress)               // 删除Ingress

		// 批量操作
		ingresses.DELETE("/batch_delete", k.BatchDeleteIngresses) // 批量删除Ingress

		// 高级功能
		ingresses.GET("/:cluster_id/:name/events", k.GetIngressEvents)                  // 获取Ingress事件
		ingresses.POST("/:cluster_id/:name/tls-test", k.TestIngressTLS)                 // 测试Ingress TLS证书
		ingresses.GET("/:cluster_id/:name/backend-health", k.CheckIngressBackendHealth) // 检查后端健康状态
	}
}

// GetIngressList 获取Ingress列表
// @Summary 获取Ingress列表
// @Description 根据查询条件获取K8s集群中的Ingress列表
// @Tags Ingress管理
// @Accept json
// @Produce json
// @Param request query model.K8sIngressListReq true "Ingress列表查询请求"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sIngressEntity} "成功获取Ingress列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/ingresses/list [get]
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
// @Summary 根据命名空间获取Ingress列表
// @Description 根据指定的命名空间获取K8s集群中的Ingress列表
// @Tags Ingress管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace query string false "命名空间，为空则获取所有命名空间"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sIngressEntity} "成功获取Ingress列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/ingresses/{cluster_id} [get]
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
// @Summary 获取Ingress详情
// @Description 获取指定Ingress的详细信息
// @Tags Ingress管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "Ingress名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=model.K8sIngressEntity} "成功获取Ingress详情"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/ingresses/{cluster_id}/{name} [get]
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
// @Summary 获取Ingress的YAML配置
// @Description 获取指定Ingress的完整YAML配置文件
// @Tags Ingress管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "Ingress名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=string} "成功获取YAML配置"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/ingresses/{cluster_id}/{name}/yaml [get]
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
// @Summary 创建Ingress
// @Description 创建新的Ingress资源
// @Tags Ingress管理
// @Accept json
// @Produce json
// @Param request body model.K8sIngressCreateReq true "Ingress创建请求"
// @Success 200 {object} utils.ApiResponse "成功创建Ingress"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/ingresses/create [post]
func (k *K8sIngressHandler) CreateIngress(ctx *gin.Context) {
	var req model.K8sIngressCreateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.ingressService.CreateIngress(ctx, &req)
	})
}

// UpdateIngress 更新Ingress
// @Summary 更新Ingress
// @Description 更新指定的Ingress资源配置
// @Tags Ingress管理
// @Accept json
// @Produce json
// @Param request body model.K8sIngressUpdateReq true "Ingress更新请求"
// @Success 200 {object} utils.ApiResponse "成功更新Ingress"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/ingresses/update [put]
func (k *K8sIngressHandler) UpdateIngress(ctx *gin.Context) {
	var req model.K8sIngressUpdateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.ingressService.UpdateIngress(ctx, &req)
	})
}

// DeleteIngress 删除Ingress
// @Summary 删除Ingress
// @Description 删除指定的Ingress资源
// @Tags Ingress管理
// @Accept json
// @Produce json
// @Param request body model.K8sIngressDeleteReq true "Ingress删除请求"
// @Success 200 {object} utils.ApiResponse "成功删除Ingress"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/ingresses/delete [delete]
func (k *K8sIngressHandler) DeleteIngress(ctx *gin.Context) {
	var req model.K8sIngressDeleteReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.ingressService.DeleteIngress(ctx, &req)
	})
}

// BatchDeleteIngresses 批量删除Ingress
// @Summary 批量删除Ingress
// @Description 批量删除指定命名空间中的多个Ingress
// @Tags Ingress管理
// @Accept json
// @Produce json
// @Param request body model.K8sIngressBatchDeleteReq true "批量删除请求"
// @Success 200 {object} utils.ApiResponse "成功批量删除Ingress"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/ingresses/batch_delete [delete]
func (k *K8sIngressHandler) BatchDeleteIngresses(ctx *gin.Context) {
	var req model.K8sIngressBatchDeleteReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.ingressService.BatchDeleteIngresses(ctx, &req)
	})
}

// GetIngressEvents 获取Ingress事件
// @Summary 获取Ingress事件
// @Description 获取指定Ingress相关的事件信息
// @Tags Ingress管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "Ingress名称"
// @Param namespace query string true "命名空间"
// @Param limit_days query int false "限制天数内的事件"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sEvent} "成功获取事件"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/ingresses/{cluster_id}/{name}/events [get]
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
// @Summary 测试Ingress TLS证书
// @Description 测试指定Ingress的TLS证书有效性
// @Tags Ingress管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "Ingress名称"
// @Param request body model.K8sIngressTLSTestReq true "TLS测试请求"
// @Success 200 {object} utils.ApiResponse{data=model.K8sTLSTestResult} "成功获取TLS测试结果"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/ingresses/{cluster_id}/{name}/tls-test [post]
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
// @Summary 检查Ingress后端健康状态
// @Description 检查指定Ingress后端服务的健康状态
// @Tags Ingress管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "Ingress名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sBackendHealth} "成功获取后端健康状态"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/ingresses/{cluster_id}/{name}/backend-health [get]
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

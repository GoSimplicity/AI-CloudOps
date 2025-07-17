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

type K8sPVCHandler struct {
	l          *zap.Logger
	pvcService admin.PVCService
}

func NewK8sPVCHandler(l *zap.Logger, pvcService admin.PVCService) *K8sPVCHandler {
	return &K8sPVCHandler{
		l:          l,
		pvcService: pvcService,
	}
}

func (k *K8sPVCHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	pvcs := k8sGroup.Group("/pvcs")
	{
		pvcs.GET("/:id", k.GetPVCsByNamespace)          // 根据命名空间获取 PVC 列表
		pvcs.POST("/create", k.CreatePVC)               // 创建 PVC
		pvcs.DELETE("/delete/:id", k.DeletePVC)         // 删除指定 PVC
		pvcs.DELETE("/batch_delete", k.BatchDeletePVC)  // 批量删除 PVC
		pvcs.GET("/:id/yaml", k.GetPVCYaml)            // 获取 PVC YAML 配置
		pvcs.GET("/:id/status", k.GetPVCStatus)        // 获取 PVC 状态
		pvcs.GET("/:id/binding", k.GetPVCBinding)      // 获取 PVC 绑定状态
		pvcs.GET("/:id/capacity", k.GetPVCCapacityRequest) // 获取 PVC 容量请求
	}
}

// GetPVCsByNamespace 根据命名空间获取 PVC 列表
func (k *K8sPVCHandler) GetPVCsByNamespace(ctx *gin.Context) {
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
		return k.pvcService.GetPVCsByNamespace(ctx, id, namespace)
	})
}

// CreatePVC 创建 PVC
func (k *K8sPVCHandler) CreatePVC(ctx *gin.Context) {
	var req model.K8sPVCRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvcService.CreatePVC(ctx, &req)
	})
}

// BatchDeletePVC 批量删除 PVC
func (k *K8sPVCHandler) BatchDeletePVC(ctx *gin.Context) {
	var req model.K8sPVCRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvcService.BatchDeletePVC(ctx, req.ClusterID, req.Namespace, req.PVCNames)
	})
}

// GetPVCYaml 获取 PVC 的 YAML 配置
func (k *K8sPVCHandler) GetPVCYaml(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	pvcName := ctx.Query("pvc_name")
	if pvcName == "" {
		k.l.Error("缺少必需的 pvc_name 参数")
		utils.BadRequestError(ctx, "缺少 'pvc_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvcService.GetPVCYaml(ctx, id, namespace, pvcName)
	})
}

// DeletePVC 删除指定的 PVC
func (k *K8sPVCHandler) DeletePVC(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	pvcName := ctx.Query("pvc_name")
	if pvcName == "" {
		k.l.Error("缺少必需的 pvc_name 参数")
		utils.BadRequestError(ctx, "缺少 'pvc_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.pvcService.DeletePVC(ctx, id, namespace, pvcName)
	})
}

// GetPVCStatus 获取 PVC 状态
func (k *K8sPVCHandler) GetPVCStatus(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	pvcName := ctx.Query("pvc_name")
	if pvcName == "" {
		k.l.Error("缺少必需的 pvc_name 参数")
		utils.BadRequestError(ctx, "缺少 'pvc_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvcService.GetPVCStatus(ctx, id, namespace, pvcName)
	})
}

// GetPVCBinding 获取 PVC 绑定状态
func (k *K8sPVCHandler) GetPVCBinding(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	pvcName := ctx.Query("pvc_name")
	if pvcName == "" {
		k.l.Error("缺少必需的 pvc_name 参数")
		utils.BadRequestError(ctx, "缺少 'pvc_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvcService.GetPVCBinding(ctx, id, namespace, pvcName)
	})
}

// GetPVCCapacityRequest 获取 PVC 容量请求
func (k *K8sPVCHandler) GetPVCCapacityRequest(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	pvcName := ctx.Query("pvc_name")
	if pvcName == "" {
		k.l.Error("缺少必需的 pvc_name 参数")
		utils.BadRequestError(ctx, "缺少 'pvc_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		k.l.Error("缺少必需的 namespace 参数")
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvcService.GetPVCCapacityRequest(ctx, id, namespace, pvcName)
	})
}
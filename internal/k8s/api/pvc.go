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

type K8sPVCHandler struct {
	pvcService service.PVCService
}

func NewK8sPVCHandler(pvcService service.PVCService) *K8sPVCHandler {
	return &K8sPVCHandler{
		pvcService: pvcService,
	}
}

func (k *K8sPVCHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/pvcs/list", k.GetPVCList)                   // 获取PVC列表
		k8sGroup.GET("/pvcs/:cluster_id", k.GetPVCsByNamespace)    // 根据命名空间获取PVC列表
		k8sGroup.GET("/pvcs/:cluster_id/:name", k.GetPVC)          // 获取单个PVC详情
		k8sGroup.GET("/pvcs/:cluster_id/:name/yaml", k.GetPVCYaml) // 获取PVC YAML配置
		k8sGroup.POST("/pvcs/create", k.CreatePVC)                 // 创建PVC
		k8sGroup.PUT("/pvcs/update", k.UpdatePVC)                  // 更新PVC
		k8sGroup.DELETE("/pvcs/delete", k.DeletePVC)               // 删除PVC

		// YAML操作
		k8sGroup.POST("/pvcs/yaml", k.CreatePVCByYaml)                             // 通过YAML创建PVC
		k8sGroup.PUT("/pvcs/:cluster_id/:namespace/:name/yaml", k.UpdatePVCByYaml) // 通过YAML更新PVC

		k8sGroup.GET("/pvcs/:cluster_id/:name/events", k.GetPVCEvents) // 获取PVC事件
		k8sGroup.GET("/pvcs/:cluster_id/:name/usage", k.GetPVCUsage)   // 获取PVC使用情况
		k8sGroup.POST("/pvcs/:cluster_id/:name/expand", k.ExpandPVC)   // 扩容PVC
	}
}

// GetPVCList 获取PVC列表
func (k *K8sPVCHandler) GetPVCList(ctx *gin.Context) {
	var req model.K8sPVCListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvcService.GetPVCList(ctx, &req)
	})
}

// GetPVCsByNamespace 根据命名空间获取PVC列表
func (k *K8sPVCHandler) GetPVCsByNamespace(ctx *gin.Context) {
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
		return k.pvcService.GetPVCsByNamespace(ctx, req.ClusterID, req.Namespace)
	})
}

// GetPVC 获取PVC详情
func (k *K8sPVCHandler) GetPVC(ctx *gin.Context) {
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
		return k.pvcService.GetPVC(ctx, req.ClusterID, req.Namespace, req.ResourceName)
	})
}

// GetPVCYaml 获取PVC的YAML配置
func (k *K8sPVCHandler) GetPVCYaml(ctx *gin.Context) {
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
		return k.pvcService.GetPVCYaml(ctx, req.ClusterID, req.Namespace, req.ResourceName)
	})
}

// CreatePVC 创建PVC
func (k *K8sPVCHandler) CreatePVC(ctx *gin.Context) {
	var req model.K8sPVCCreateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvcService.CreatePVC(ctx, &req)
	})
}

// UpdatePVC 更新PVC
func (k *K8sPVCHandler) UpdatePVC(ctx *gin.Context) {
	var req model.K8sPVCUpdateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvcService.UpdatePVC(ctx, &req)
	})
}

// DeletePVC 删除PVC
func (k *K8sPVCHandler) DeletePVC(ctx *gin.Context) {
	var req model.K8sPVCDeleteReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvcService.DeletePVC(ctx, &req)
	})
}

// GetPVCEvents 获取PVC事件
func (k *K8sPVCHandler) GetPVCEvents(ctx *gin.Context) {
	var req model.K8sPVCEventReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvcService.GetPVCEvents(ctx, &req)
	})
}

// GetPVCUsage 获取PVC使用情况
func (k *K8sPVCHandler) GetPVCUsage(ctx *gin.Context) {
	var req model.K8sPVCUsageReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.pvcService.GetPVCUsage(ctx, &req)
	})
}

// ExpandPVC 扩容PVC
func (k *K8sPVCHandler) ExpandPVC(ctx *gin.Context) {
	var req model.K8sPVCExpandReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvcService.ExpandPVC(ctx, &req)
	})
}

// YAML操作方法

// CreatePVCByYaml 通过YAML创建PVC
func (k *K8sPVCHandler) CreatePVCByYaml(ctx *gin.Context) {
	var req model.CreateResourceByYamlReq
	req.ResourceType = model.ResourceTypePVC

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvcService.CreatePVCByYaml(ctx, &req)
	})
}

// UpdatePVCByYaml 通过YAML更新PVC
func (k *K8sPVCHandler) UpdatePVCByYaml(ctx *gin.Context) {
	var req model.UpdateResourceByYamlReq
	req.ResourceType = model.ResourceTypePVC

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
		return nil, k.pvcService.UpdatePVCByYaml(ctx, &req)
	})
}

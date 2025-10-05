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
	return &K8sPVCHandler{pvcService: pvcService}
}

func (h *K8sPVCHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/pvc/:cluster_id/list", h.GetPVCList)
		k8sGroup.GET("/pvc/:cluster_id/:namespace/:name/detail", h.GetPVCDetails)
		k8sGroup.GET("/pvc/:cluster_id/:namespace/:name/detail/yaml", h.GetPVCYaml)
		k8sGroup.POST("/pvc/:cluster_id/create", h.CreatePVC)
		k8sGroup.POST("/pvc/:cluster_id/create/yaml", h.CreatePVCByYaml)
		k8sGroup.PUT("/pvc/:cluster_id/:namespace/:name/update", h.UpdatePVC)
		k8sGroup.PUT("/pvc/:cluster_id/:namespace/:name/update/yaml", h.UpdatePVCByYaml)
		k8sGroup.DELETE("/pvc/:cluster_id/:namespace/:name/delete", h.DeletePVC)
		k8sGroup.POST("/pvc/:cluster_id/:namespace/:name/expand", h.ExpandPVC)
		k8sGroup.GET("/pvc/:cluster_id/:namespace/:name/pods", h.GetPVCPods)
	}
}

func (h *K8sPVCHandler) GetPVCList(ctx *gin.Context) {
	var req model.GetPVCListReq
	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	req.ClusterID = clusterID
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.pvcService.GetPVCList(ctx, &req)
	})
}

func (h *K8sPVCHandler) GetPVCDetails(ctx *gin.Context) {
	var req model.GetPVCDetailsReq
	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	ns, err := utils.GetParamCustomName(ctx, "namespace")
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
	req.Namespace = ns
	req.Name = name
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.pvcService.GetPVC(ctx, &req)
	})
}

func (h *K8sPVCHandler) GetPVCYaml(ctx *gin.Context) {
	var req model.GetPVCYamlReq
	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	ns, err := utils.GetParamCustomName(ctx, "namespace")
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
	req.Namespace = ns
	req.Name = name
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.pvcService.GetPVCYaml(ctx, &req)
	})
}

func (h *K8sPVCHandler) CreatePVC(ctx *gin.Context) {
	var req model.CreatePVCReq
	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	req.ClusterID = clusterID
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.pvcService.CreatePVC(ctx, &req)
	})
}

func (h *K8sPVCHandler) UpdatePVC(ctx *gin.Context) {
	var req model.UpdatePVCReq
	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	ns, err := utils.GetParamCustomName(ctx, "namespace")
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
	req.Namespace = ns
	req.Name = name
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.pvcService.UpdatePVC(ctx, &req)
	})
}

func (h *K8sPVCHandler) DeletePVC(ctx *gin.Context) {
	var req model.DeletePVCReq
	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	ns, err := utils.GetParamCustomName(ctx, "namespace")
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
	req.Namespace = ns
	req.Name = name
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.pvcService.DeletePVC(ctx, &req)
	})
}

func (h *K8sPVCHandler) CreatePVCByYaml(ctx *gin.Context) {
	var req model.CreatePVCByYamlReq
	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	req.ClusterID = clusterID
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.pvcService.CreatePVCByYaml(ctx, &req)
	})
}

func (h *K8sPVCHandler) UpdatePVCByYaml(ctx *gin.Context) {
	var req model.UpdatePVCByYamlReq
	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	ns, err := utils.GetParamCustomName(ctx, "namespace")
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
	req.Namespace = ns
	req.Name = name
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.pvcService.UpdatePVCByYaml(ctx, &req)
	})
}

func (h *K8sPVCHandler) ExpandPVC(ctx *gin.Context) {
	var req model.ExpandPVCReq
	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	ns, err := utils.GetParamCustomName(ctx, "namespace")
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
	req.Namespace = ns
	req.Name = name
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.pvcService.ExpandPVC(ctx, &req)
	})
}

func (h *K8sPVCHandler) GetPVCPods(ctx *gin.Context) {
	var req model.GetPVCPodsReq
	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	ns, err := utils.GetParamCustomName(ctx, "namespace")
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
	req.Namespace = ns
	req.Name = name
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.pvcService.GetPVCPods(ctx, &req)
	})
}

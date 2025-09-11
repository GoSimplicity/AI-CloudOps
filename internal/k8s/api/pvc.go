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

func (k *K8sPVCHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		// Unify to /clusters/:cluster_id/pvcs style
		k8sGroup.GET("/clusters/:cluster_id/pvcs", k.GetPVCList)
		k8sGroup.GET("/clusters/:cluster_id/pvcs/:namespace/:name", k.GetPVCDetails)
		k8sGroup.GET("/clusters/:cluster_id/pvcs/:namespace/:name/yaml", k.GetPVCYaml)
		k8sGroup.POST("/clusters/:cluster_id/pvcs", k.CreatePVC)
		k8sGroup.POST("/clusters/:cluster_id/pvcs/yaml", k.CreatePVCByYaml)
		k8sGroup.PUT("/clusters/:cluster_id/pvcs/:namespace/:name", k.UpdatePVC)
		k8sGroup.PUT("/clusters/:cluster_id/pvcs/:namespace/:name/yaml", k.UpdatePVCByYaml)
		k8sGroup.DELETE("/clusters/:cluster_id/pvcs/:namespace/:name", k.DeletePVC)
	}
}

func (k *K8sPVCHandler) GetPVCList(ctx *gin.Context) {
	var req model.GetPVCListReq
	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	req.ClusterID = clusterID
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.pvcService.GetPVCList(ctx, &req)
	})
}

func (k *K8sPVCHandler) GetPVCDetails(ctx *gin.Context) {
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
		return k.pvcService.GetPVC(ctx, &req)
	})
}

func (k *K8sPVCHandler) GetPVCYaml(ctx *gin.Context) {
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
		return k.pvcService.GetPVCYaml(ctx, &req)
	})
}

func (k *K8sPVCHandler) CreatePVC(ctx *gin.Context) {
	var req model.CreatePVCReq
	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	req.ClusterID = clusterID
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvcService.CreatePVC(ctx, &req)
	})
}

func (k *K8sPVCHandler) UpdatePVC(ctx *gin.Context) {
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
		return nil, k.pvcService.UpdatePVC(ctx, &req)
	})
}

func (k *K8sPVCHandler) DeletePVC(ctx *gin.Context) {
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
		return nil, k.pvcService.DeletePVC(ctx, &req)
	})
}

func (k *K8sPVCHandler) CreatePVCByYaml(ctx *gin.Context) {
	var req model.CreatePVCByYamlReq
	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	req.ClusterID = clusterID
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvcService.CreatePVCByYaml(ctx, &req)
	})
}

func (k *K8sPVCHandler) UpdatePVCByYaml(ctx *gin.Context) {
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
		return nil, k.pvcService.UpdatePVCByYaml(ctx, &req)
	})
}

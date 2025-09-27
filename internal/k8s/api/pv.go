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

type K8sPVHandler struct {
	pvService service.PVService
}

func NewK8sPVHandler(pvService service.PVService) *K8sPVHandler {
	return &K8sPVHandler{
		pvService: pvService,
	}
}

func (k *K8sPVHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/pv/:cluster_id/list", k.GetPVList)                   // 获取PV列表
		k8sGroup.GET("/pv/:cluster_id/:name/detail", k.GetPVDetails)        // 获取PV详情
		k8sGroup.GET("/pv/:cluster_id/:name/detail/yaml", k.GetPVYaml)      // 获取PV YAML
		k8sGroup.POST("/pv/:cluster_id/create", k.CreatePV)                 // 创建PV
		k8sGroup.POST("/pv/:cluster_id/create/yaml", k.CreatePVByYaml)      // 通过YAML创建PV
		k8sGroup.PUT("/pv/:cluster_id/:name/update", k.UpdatePV)            // 更新PV
		k8sGroup.PUT("/pv/:cluster_id/:name/update/yaml", k.UpdatePVByYaml) // 通过YAML更新PV
		k8sGroup.DELETE("/pv/:cluster_id/:name/delete", k.DeletePV)         // 删除PV
		k8sGroup.POST("/pv/:cluster_id/:name/reclaim", k.ReclaimPV)         // 回收PV
	}
}

func (k *K8sPVHandler) GetPVList(ctx *gin.Context) {
	var req model.GetPVListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.pvService.GetPVList(ctx, &req)
	})
}

func (k *K8sPVHandler) GetPVDetails(ctx *gin.Context) {
	var req model.GetPVDetailsReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
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
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.pvService.GetPV(ctx, req.ClusterID, req.Name)
	})
}

func (k *K8sPVHandler) GetPVYaml(ctx *gin.Context) {
	var req model.GetPVYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
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
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.pvService.GetPVYaml(ctx, req.ClusterID, req.Name)
	})
}

func (k *K8sPVHandler) CreatePV(ctx *gin.Context) {
	var req model.CreatePVReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvService.CreatePV(ctx, &req)
	})
}

func (k *K8sPVHandler) CreatePVByYaml(ctx *gin.Context) {
	var req model.CreatePVByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvService.CreatePVByYaml(ctx, &req)
	})
}

func (k *K8sPVHandler) UpdatePV(ctx *gin.Context) {
	var req model.UpdatePVReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
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
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvService.UpdatePV(ctx, &req)
	})
}

func (k *K8sPVHandler) UpdatePVByYaml(ctx *gin.Context) {
	var req model.UpdatePVByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
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
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvService.UpdatePVByYaml(ctx, &req)
	})
}

func (k *K8sPVHandler) DeletePV(ctx *gin.Context) {
	var req model.DeletePVReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
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
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvService.DeletePV(ctx, &req)
	})
}

func (k *K8sPVHandler) ReclaimPV(ctx *gin.Context) {
	var req model.ReclaimPVReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
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
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvService.ReclaimPV(ctx, &req)
	})
}

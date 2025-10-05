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

type K8sYamlTemplateHandler struct {
	yamlTemplateService service.YamlTemplateService
}

func NewK8sYamlTemplateHandler(yamlTemplateService service.YamlTemplateService) *K8sYamlTemplateHandler {
	return &K8sYamlTemplateHandler{
		yamlTemplateService: yamlTemplateService,
	}
}

func (h *K8sYamlTemplateHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/yaml_template/:cluster_id/list", h.GetYamlTemplateList)
		k8sGroup.POST("/yaml_template/:cluster_id/create", h.CreateYamlTemplate)
		k8sGroup.POST("/yaml_template/:cluster_id/check", h.CheckYamlTemplate)
		k8sGroup.POST("/yaml_template/:cluster_id/:id/update", h.UpdateYamlTemplate)
		k8sGroup.DELETE("/yaml_template/:cluster_id/:id/delete", h.DeleteYamlTemplate)
		k8sGroup.GET("/yaml_template/:cluster_id/:id/yaml", h.GetYamlTemplateDetail)
	}
}

func (h *K8sYamlTemplateHandler) GetYamlTemplateList(ctx *gin.Context) {
	var req model.YamlTemplateListReq

	clusterId, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, "缺少 'cluster_id' 参数")
		return
	}

	req.ClusterID = clusterId

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.yamlTemplateService.GetYamlTemplateList(ctx, &req)
	})
}

func (h *K8sYamlTemplateHandler) CreateYamlTemplate(ctx *gin.Context) {
	var req model.YamlTemplateCreateReq

	clusterId, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, "缺少 'cluster_id' 参数")
		return
	}
	uc := ctx.MustGet("user").(utils.UserClaims)

	req.ClusterID = clusterId
	req.UserID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.yamlTemplateService.CreateYamlTemplate(ctx, &req)
	})
}

func (h *K8sYamlTemplateHandler) UpdateYamlTemplate(ctx *gin.Context) {
	var req model.YamlTemplateUpdateReq

	clusterId, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, "缺少 'cluster_id' 参数")
		return
	}

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}
	uc := ctx.MustGet("user").(utils.UserClaims)

	req.ClusterID = clusterId
	req.UserID = uc.Uid
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.yamlTemplateService.UpdateYamlTemplate(ctx, &req)
	})
}

func (h *K8sYamlTemplateHandler) DeleteYamlTemplate(ctx *gin.Context) {
	var req model.YamlTemplateDeleteReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	clusterId, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, "缺少 'cluster_id' 参数")
		return
	}

	req.ID = id
	req.ClusterID = clusterId

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.yamlTemplateService.DeleteYamlTemplate(ctx, &req)
	})
}

func (h *K8sYamlTemplateHandler) CheckYamlTemplate(ctx *gin.Context) {
	var req model.YamlTemplateCheckReq

	clusterId, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, "缺少 'cluster_id' 参数")
		return
	}

	req.ClusterID = clusterId

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.yamlTemplateService.CheckYamlTemplate(ctx, &req)
	})
}

func (h *K8sYamlTemplateHandler) GetYamlTemplateDetail(ctx *gin.Context) {
	var req model.YamlTemplateDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	clusterId, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, "缺少 'cluster_id' 参数")
		return
	}

	req.ID = id
	req.ClusterID = clusterId

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.yamlTemplateService.GetYamlTemplateDetail(ctx, &req)
	})
}

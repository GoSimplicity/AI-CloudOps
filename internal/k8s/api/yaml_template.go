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
	"strconv"

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

func (k *K8sYamlTemplateHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/yaml_templates/list", k.GetYamlTemplateList)         // 获取 YAML 模板列表
		k8sGroup.POST("/yaml_templates/create", k.CreateYamlTemplate)       // 创建新的 YAML 模板
		k8sGroup.POST("/yaml_templates/check", k.CheckYamlTemplate)         // 检查 YAML 模板是否可用
		k8sGroup.POST("/yaml_templates/update", k.UpdateYamlTemplate)       // 更新指定 ID 的 YAML 模板
		k8sGroup.DELETE("/yaml_templates/delete/:id", k.DeleteYamlTemplate) // 删除指定 ID 的 YAML 模板
		k8sGroup.GET("/yaml_templates/:id/yaml", k.GetYamlTemplateDetail)
	}
}

// GetYamlTemplateList 获取 YAML 模板列表
func (k *K8sYamlTemplateHandler) GetYamlTemplateList(ctx *gin.Context) {
	clusterId := ctx.Query("cluster_id")
	if clusterId == "" {
		utils.BadRequestError(ctx, "缺少 'cluster_id' 参数")
		return
	}

	intClusterId, err := strconv.Atoi(clusterId)
	if err != nil {
		utils.BadRequestError(ctx, "'cluster_id' 参数必须为整数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.yamlTemplateService.GetYamlTemplateList(ctx, intClusterId)
	})
}

// CreateYamlTemplate 创建新的 YAML 模板
func (k *K8sYamlTemplateHandler) CreateYamlTemplate(ctx *gin.Context) {
	var req model.YamlTemplateCreateReq

	uc := ctx.MustGet("user").(utils.UserClaims)
	req.UserID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		template := &model.K8sYamlTemplate{
			Name:      req.Name,
			UserID:    req.UserID,
			Content:   req.Content,
			ClusterID: req.ClusterID,
		}
		return nil, k.yamlTemplateService.CreateYamlTemplate(ctx, template)
	})
}

// UpdateYamlTemplate 更新指定 ID 的 YAML 模板
func (k *K8sYamlTemplateHandler) UpdateYamlTemplate(ctx *gin.Context) {
	var req model.YamlTemplateUpdateReq

	uc := ctx.MustGet("user").(utils.UserClaims)
	req.UserID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		template := &model.K8sYamlTemplate{
			Model:     model.Model{ID: req.ID},
			Name:      req.Name,
			UserID:    req.UserID,
			Content:   req.Content,
			ClusterID: req.ClusterID,
		}
		return nil, k.yamlTemplateService.UpdateYamlTemplate(ctx, template)
	})
}

// DeleteYamlTemplate 删除指定 ID 的 YAML 模板
func (k *K8sYamlTemplateHandler) DeleteYamlTemplate(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	clusterId := ctx.Query("cluster_id")
	if clusterId == "" {
		utils.BadRequestError(ctx, "缺少 'cluster_id' 参数")
		return
	}

	intClusterId, err := strconv.Atoi(clusterId)
	if err != nil {
		utils.BadRequestError(ctx, "'cluster_id' 参数必须为整数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.yamlTemplateService.DeleteYamlTemplate(ctx, id, intClusterId)
	})
}

// CheckYamlTemplate 检查 YAML 模板
func (k *K8sYamlTemplateHandler) CheckYamlTemplate(ctx *gin.Context) {
	var req model.YamlTemplateCheckReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		template := &model.K8sYamlTemplate{
			Name:      req.Name,
			Content:   req.Content,
			ClusterID: req.ClusterID,
		}
		return nil, k.yamlTemplateService.CheckYamlTemplate(ctx, template)
	})
}

// GetYamlTemplateDetail 获取 YAML 模板详情
func (k *K8sYamlTemplateHandler) GetYamlTemplateDetail(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	clusterId := ctx.Query("cluster_id")
	if clusterId == "" {
		utils.BadRequestError(ctx, "缺少 'cluster_id' 参数")
		return
	}

	intClusterId, err := strconv.Atoi(clusterId)
	if err != nil {
		utils.BadRequestError(ctx, "'cluster_id' 参数必须为整数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.yamlTemplateService.GetYamlTemplateDetail(ctx, id, intClusterId)
	})
}

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
	"go.uber.org/zap"
)

type K8sYamlTemplateHandler struct {
	logger              *zap.Logger
	yamlTemplateService service.YamlTemplateService
}

func NewK8sYamlTemplateHandler(logger *zap.Logger, yamlTemplateService service.YamlTemplateService) *K8sYamlTemplateHandler {
	return &K8sYamlTemplateHandler{
		logger:              logger,
		yamlTemplateService: yamlTemplateService,
	}
}

func (k *K8sYamlTemplateHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	yamlTemplates := k8sGroup.Group("/yaml_templates")
	{
		yamlTemplates.GET("/list", k.GetYamlTemplateList)         // 获取 YAML 模板列表
		yamlTemplates.POST("/create", k.CreateYamlTemplate)       // 创建新的 YAML 模板
		yamlTemplates.POST("/check", k.CheckYamlTemplate)         // 检查 YAML 模板是否可用
		yamlTemplates.POST("/update", k.UpdateYamlTemplate)       // 更新指定 ID 的 YAML 模板
		yamlTemplates.DELETE("/delete/:id", k.DeleteYamlTemplate) // 删除指定 ID 的 YAML 模板
		yamlTemplates.GET("/:id/yaml", k.GetYamlTemplateDetail)
	}
}

// GetYamlTemplateList 获取 YAML 模板列表
// @Summary 获取 YAML 模板列表
// @Description 根据集群ID获取该集群下的所有YAML模板列表
// @Tags YAML模板管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sYamlTemplate} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/yaml_templates/list [get]
// @Security BearerAuth
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
// @Summary 创建新的 YAML 模板
// @Description 在指定集群中创建一个新的YAML模板
// @Tags YAML模板管理
// @Accept json
// @Produce json
// @Param template body model.YamlTemplateCreateReq true "YAML模板信息"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/yaml_templates/create [post]
// @Security BearerAuth
func (k *K8sYamlTemplateHandler) CreateYamlTemplate(ctx *gin.Context) {
	var req model.YamlTemplateCreateReq

	uc := ctx.MustGet("user").(utils.UserClaims)
	req.UserID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		template := &model.K8sYamlTemplate{
			Name:      req.Name,
			UserID:    req.UserID,
			Content:   req.Content,
			ClusterId: req.ClusterId,
		}
		return nil, k.yamlTemplateService.CreateYamlTemplate(ctx, template)
	})
}

// UpdateYamlTemplate 更新指定 ID 的 YAML 模板
// @Summary 更新 YAML 模板
// @Description 更新指定ID的YAML模板信息
// @Tags YAML模板管理
// @Accept json
// @Produce json
// @Param template body model.YamlTemplateUpdateReq true "YAML模板信息"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/yaml_templates/update [post]
// @Security BearerAuth
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
			ClusterId: req.ClusterId,
		}
		return nil, k.yamlTemplateService.UpdateYamlTemplate(ctx, template)
	})
}

// DeleteYamlTemplate 删除指定 ID 的 YAML 模板
// @Summary 删除 YAML 模板
// @Description 根据ID删除指定的YAML模板
// @Tags YAML模板管理
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Param cluster_id query int true "集群ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/yaml_templates/delete/{id} [delete]
// @Security BearerAuth
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
// @Summary 检查 YAML 模板
// @Description 验证YAML模板格式的正确性和可用性
// @Tags YAML模板管理
// @Accept json
// @Produce json
// @Param template body model.YamlTemplateCheckReq true "YAML模板信息"
// @Success 200 {object} utils.ApiResponse "检查成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/yaml_templates/check [post]
// @Security BearerAuth
func (k *K8sYamlTemplateHandler) CheckYamlTemplate(ctx *gin.Context) {
	var req model.YamlTemplateCheckReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		template := &model.K8sYamlTemplate{
			Name:      req.Name,
			Content:   req.Content,
			ClusterId: req.ClusterId,
		}
		return nil, k.yamlTemplateService.CheckYamlTemplate(ctx, template)
	})
}

// GetYamlTemplateDetail 获取 YAML 模板详情
// @Summary 获取 YAML 模板详情
// @Description 根据ID获取指定YAML模板的详细信息
// @Tags YAML模板管理
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Param cluster_id query int true "集群ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/yaml_templates/{id}/yaml [get]
// @Security BearerAuth
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

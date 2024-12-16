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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sYamlTemplateHandler struct {
	l                   *zap.Logger
	yamlTemplateService admin.YamlTemplateService
}

func NewK8sYamlTemplateHandler(l *zap.Logger, yamlTemplateService admin.YamlTemplateService) *K8sYamlTemplateHandler {
	return &K8sYamlTemplateHandler{
		l:                   l,
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
func (k *K8sYamlTemplateHandler) GetYamlTemplateList(ctx *gin.Context) {
	clusterId := ctx.Query("cluster_id")
	if clusterId == "" {
		apiresponse.BadRequestError(ctx, "缺少 'cluster_id' 参数")
		return
	}

	intClusterId, err := strconv.Atoi(clusterId)
	if err != nil {
		apiresponse.BadRequestError(ctx, "'cluster_id' 参数必须为整数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.yamlTemplateService.GetYamlTemplateList(ctx, intClusterId)
	})
}

// CreateYamlTemplate 创建新的 YAML 模板
func (k *K8sYamlTemplateHandler) CreateYamlTemplate(ctx *gin.Context) {
	var req model.K8sYamlTemplate

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.yamlTemplateService.CreateYamlTemplate(ctx, &req)
	})
}

// UpdateYamlTemplate 更新指定 ID 的 YAML 模板
func (k *K8sYamlTemplateHandler) UpdateYamlTemplate(ctx *gin.Context) {
	var req model.K8sYamlTemplate

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.yamlTemplateService.UpdateYamlTemplate(ctx, &req)
	})
}

// DeleteYamlTemplate 删除指定 ID 的 YAML 模板
func (k *K8sYamlTemplateHandler) DeleteYamlTemplate(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	clusterId := ctx.Query("cluster_id")
	if clusterId == "" {
		apiresponse.BadRequestError(ctx, "缺少 'cluster_id' 参数")
		return
	}

	intClusterId, err := strconv.Atoi(clusterId)
	if err != nil {
		apiresponse.BadRequestError(ctx, "'cluster_id' 参数必须为整数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.yamlTemplateService.DeleteYamlTemplate(ctx, id, intClusterId)
	})
}

func (k *K8sYamlTemplateHandler) CheckYamlTemplate(ctx *gin.Context) {
	var req model.K8sYamlTemplate

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.yamlTemplateService.CheckYamlTemplate(ctx, &req)
	})
}

func (k *K8sYamlTemplateHandler) GetYamlTemplateDetail(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	clusterId := ctx.Query("cluster_id")
	if clusterId == "" {
		apiresponse.BadRequestError(ctx, "缺少 'cluster_id' 参数")
		return
	}

	intClusterId, err := strconv.Atoi(clusterId)
	if err != nil {
		apiresponse.BadRequestError(ctx, "'cluster_id' 参数必须为整数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.yamlTemplateService.GetYamlTemplateDetail(ctx, id, intClusterId)
	})
}

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
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sYamlTaskHandler struct {
	l               *zap.Logger
	yamlTaskService admin.YamlTaskService
}

func NewK8sYamlTaskHandler(l *zap.Logger, yamlTaskService admin.YamlTaskService) *K8sYamlTaskHandler {
	return &K8sYamlTaskHandler{
		l:               l,
		yamlTaskService: yamlTaskService,
	}
}

func (k *K8sYamlTaskHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	yamlTasks := k8sGroup.Group("/yaml_tasks")
	{
		yamlTasks.GET("/list", k.GetYamlTaskList)         // 获取 YAML 任务列表
		yamlTasks.POST("/create", k.CreateYamlTask)       // 创建新的 YAML 任务
		yamlTasks.POST("/update", k.UpdateYamlTask)       // 更新指定 ID 的 YAML 任务
		yamlTasks.POST("/apply/:id", k.ApplyYamlTask)     // 应用指定 ID 的 YAML 任务
		yamlTasks.DELETE("/delete/:id", k.DeleteYamlTask) // 删除指定 ID 的 YAML 任务
	}
}

// GetYamlTaskList 获取 YAML 任务列表
// @Summary 获取 YAML 任务列表
// @Description 获取所有的 YAML 任务列表信息
// @Tags YAML任务管理
// @Accept json
// @Produce json
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sYamlTask} "获取 YAML 任务列表成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/yaml_tasks/list [get]
// @Security BearerAuth
func (k *K8sYamlTaskHandler) GetYamlTaskList(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.yamlTaskService.GetYamlTaskList(ctx)
	})
}

// CreateYamlTask 创建新的 YAML 任务
// @Summary 创建新的 YAML 任务
// @Description 根据提供的参数创建一个新的 YAML 任务
// @Tags YAML任务管理
// @Accept json
// @Produce json
// @Param body body model.K8sYamlTask true "YAML 任务信息"
// @Success 200 {object} utils.ApiResponse "创建 YAML 任务成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/yaml_tasks/create [post]
// @Security BearerAuth
func (k *K8sYamlTaskHandler) CreateYamlTask(ctx *gin.Context) {
	var req model.K8sYamlTask

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.yamlTaskService.CreateYamlTask(ctx, &req)
	})
}

// UpdateYamlTask 更新指定 ID 的 YAML 任务
// @Summary 更新 YAML 任务
// @Description 根据提供的参数更新指定的 YAML 任务信息
// @Tags YAML任务管理
// @Accept json
// @Produce json
// @Param body body model.K8sYamlTask true "YAML 任务更新信息"
// @Success 200 {object} utils.ApiResponse "更新 YAML 任务成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/yaml_tasks/update [post]
// @Security BearerAuth
func (k *K8sYamlTaskHandler) UpdateYamlTask(ctx *gin.Context) {
	var req model.K8sYamlTask

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.yamlTaskService.UpdateYamlTask(ctx, &req)
	})
}

// ApplyYamlTask 应用指定 ID 的 YAML 任务
// @Summary 应用 YAML 任务
// @Description 根据任务 ID 应用指定的 YAML 任务到 Kubernetes 集群
// @Tags YAML任务管理
// @Accept json
// @Produce json
// @Param id path int true "YAML 任务 ID"
// @Success 200 {object} utils.ApiResponse "应用 YAML 任务成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/yaml_tasks/apply/{id} [post]
// @Security BearerAuth
func (k *K8sYamlTaskHandler) ApplyYamlTask(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.yamlTaskService.ApplyYamlTask(ctx, id)
	})
}

// DeleteYamlTask 删除指定 ID 的 YAML 任务
// @Summary 删除 YAML 任务
// @Description 根据任务 ID 删除指定的 YAML 任务
// @Tags YAML任务管理
// @Accept json
// @Produce json
// @Param id path int true "YAML 任务 ID"
// @Success 200 {object} utils.ApiResponse "删除 YAML 任务成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/yaml_tasks/delete/{id} [delete]
// @Security BearerAuth
func (k *K8sYamlTaskHandler) DeleteYamlTask(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.yamlTaskService.DeleteYamlTask(ctx, id)
	})
}

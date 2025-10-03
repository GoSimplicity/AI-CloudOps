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

type K8sYamlTaskHandler struct {
	yamlTaskService service.YamlTaskService
}

func NewK8sYamlTaskHandler(yamlTaskService service.YamlTaskService) *K8sYamlTaskHandler {
	return &K8sYamlTaskHandler{
		yamlTaskService: yamlTaskService,
	}
}

func (h *K8sYamlTaskHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/yaml_task/:cluster_id/list", h.GetYamlTaskList)         // 获取 YAML 任务列表
		k8sGroup.POST("/yaml_task/:cluster_id/create", h.CreateYamlTask)       // 创建新的 YAML 任务
		k8sGroup.POST("/yaml_task/:cluster_id/:id/update", h.UpdateYamlTask)   // 更新指定 ID 的 YAML 任务
		k8sGroup.POST("/yaml_task/:cluster_id/:id/apply", h.ApplyYamlTask)     // 应用指定 ID 的 YAML 任务
		k8sGroup.DELETE("/yaml_task/:cluster_id/:id/delete", h.DeleteYamlTask) // 删除指定 ID 的 YAML 任务
	}
}

// GetYamlTaskList 获取 YAML 任务列表
func (h *K8sYamlTaskHandler) GetYamlTaskList(ctx *gin.Context) {
	var req model.YamlTaskListReq

	clusterId, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, "缺少 'cluster_id' 参数")
		return
	}

	req.ClusterID = clusterId

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.yamlTaskService.GetYamlTaskList(ctx, &req)
	})
}

// CreateYamlTask 创建新的 YAML 任务
func (h *K8sYamlTaskHandler) CreateYamlTask(ctx *gin.Context) {
	var req model.YamlTaskCreateReq

	clusterId, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, "缺少 'cluster_id' 参数")
		return
	}
	uc := ctx.MustGet("user").(utils.UserClaims)

	req.UserID = uc.Uid
	req.ClusterID = clusterId

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.yamlTaskService.CreateYamlTask(ctx, &req)
	})
}

// UpdateYamlTask 更新指定 ID 的 YAML 任务
func (h *K8sYamlTaskHandler) UpdateYamlTask(ctx *gin.Context) {
	var req model.YamlTaskUpdateReq

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

	req.UserID = uc.Uid
	req.ClusterID = clusterId
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.yamlTaskService.UpdateYamlTask(ctx, &req)
	})
}

// ApplyYamlTask 应用指定 ID 的 YAML 任务
func (h *K8sYamlTaskHandler) ApplyYamlTask(ctx *gin.Context) {
	var req model.YamlTaskExecuteReq

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
		return nil, h.yamlTaskService.ApplyYamlTask(ctx, &req)
	})
}

// DeleteYamlTask 删除指定 ID 的 YAML 任务
func (h *K8sYamlTaskHandler) DeleteYamlTask(ctx *gin.Context) {
	var req model.YamlTaskDeleteReq

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
		return nil, h.yamlTaskService.DeleteYamlTask(ctx, &req)
	})
}

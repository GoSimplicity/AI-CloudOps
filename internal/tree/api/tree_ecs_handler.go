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
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type TreeEcsHandler struct {
	ecsService service.TreeEcsService
}

func NewTreeEcsHandler(ecsService service.TreeEcsService) *TreeEcsHandler {
	return &TreeEcsHandler{
		ecsService: ecsService,
	}
}

func (h *TreeEcsHandler) RegisterRouters(server *gin.Engine) {
	ecsGroup := server.Group("/api/tree/ecs")
	{
		ecsGroup.GET("/list", h.ListEcsResources)
		// ecsGroup.GET("/instance_options", h.ListInstanceOptions) // 云资源特有功能，仅支持本地资源
		ecsGroup.GET("/detail/:id", h.GetEcsDetail)
		ecsGroup.POST("/create", h.CreateEcsResource)
		ecsGroup.PUT("/update/:id", h.UpdateEcs)
		ecsGroup.DELETE("/delete/:id", h.DeleteEcs)
		ecsGroup.POST("/start/:id", h.StartEcs)
		ecsGroup.POST("/stop/:id", h.StopEcs)
		ecsGroup.POST("/restart/:id", h.RestartEcs)
		// ecsGroup.POST("/resize/:id", h.ResizeEcs) // 云资源特有功能，仅支持本地资源
		// ecsGroup.POST("/reset_password/:id", h.ResetEcsPassword) // 云资源特有功能，仅支持本地资源
		// ecsGroup.POST("/renew/:id", h.RenewEcs) // 云资源特有功能，仅支持本地资源
	}
}

// ListEcsResources 获取ECS实例列表
// @Summary 获取ECS实例列表
// @Description 分页获取ECS实例列表，支持按条件筛选
// @Tags ECS管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param region query string false "地域"
// @Param status query string false "实例状态"
// @Param keyword query string false "搜索关键词"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/ecs/list [get]
func (h *TreeEcsHandler) ListEcsResources(ctx *gin.Context) {
	var req model.ListEcsResourcesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.ecsService.ListEcsResources(ctx, &req)
	})
}

// ListInstanceOptions 获取ECS实例规格列表 - 云资源特有功能，仅支持本地资源时暂不提供
/*
func (h *TreeEcsHandler) ListInstanceOptions(ctx *gin.Context) {
	var req model.ListEcsResourceOptionsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.ecsService.ListEcsResourceOptions(ctx, &req)
	})
}
*/

// GetEcsDetail 获取ECS实例详情
// @Summary 获取ECS实例详情
// @Description 根据ID获取ECS实例的详细信息
// @Tags ECS管理
// @Accept json
// @Produce json
// @Param id path int true "实例ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/ecs/detail/{id} [get]
func (h *TreeEcsHandler) GetEcsDetail(ctx *gin.Context) {
	var req model.GetEcsDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.ecsService.GetEcsDetail(ctx, &req)
	})
}

// CreateEcsResource 创建ECS实例
// @Summary 创建ECS实例
// @Description 创建新的ECS实例
// @Tags ECS管理
// @Accept json
// @Produce json
// @Param request body model.CreateEcsResourceReq true "创建ECS实例请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/ecs/create [post]
func (h *TreeEcsHandler) CreateEcsResource(ctx *gin.Context) {
	var req model.CreateEcsResourceReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.ecsService.CreateEcsResource(ctx, &req)
	})
}

// DeleteEcs 删除ECS实例
// @Summary 删除ECS实例
// @Description 删除指定的ECS实例
// @Tags ECS管理
// @Accept json
// @Produce json
// @Param id path int true "实例ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/ecs/delete/{id} [delete]
func (h *TreeEcsHandler) DeleteEcs(ctx *gin.Context) {
	var req model.DeleteEcsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.ecsService.DeleteEcs(ctx, &req)
	})
}

// StartEcs 启动ECS实例
// @Summary 启动ECS实例
// @Description 启动指定的ECS实例
// @Tags ECS管理
// @Accept json
// @Produce json
// @Param id path int true "实例ID"
// @Success 200 {object} utils.ApiResponse "启动成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/ecs/start/{id} [post]
func (h *TreeEcsHandler) StartEcs(ctx *gin.Context) {
	var req model.StartEcsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.ecsService.StartEcs(ctx, &req)
	})
}

// StopEcs 停止ECS实例
// @Summary 停止ECS实例
// @Description 停止指定的ECS实例
// @Tags ECS管理
// @Accept json
// @Produce json
// @Param id path int true "实例ID"
// @Success 200 {object} utils.ApiResponse "停止成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/ecs/stop/{id} [post]
func (h *TreeEcsHandler) StopEcs(ctx *gin.Context) {
	var req model.StopEcsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.ecsService.StopEcs(ctx, &req)
	})
}

// RestartEcs 重启ECS实例
// @Summary 重启ECS实例
// @Description 重启指定的ECS实例
// @Tags ECS管理
// @Accept json
// @Produce json
// @Param id path int true "实例ID"
// @Success 200 {object} utils.ApiResponse "重启成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/ecs/restart/{id} [post]
func (h *TreeEcsHandler) RestartEcs(ctx *gin.Context) {
	var req model.RestartEcsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.ecsService.RestartEcs(ctx, &req)
	})
}

// UpdateEcs 更新ECS实例
// @Summary 更新ECS实例
// @Description 更新指定ECS实例的配置信息
// @Tags ECS管理
// @Accept json
// @Produce json
// @Param id path int true "实例ID"
// @Param request body model.UpdateEcsReq true "更新ECS实例请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/ecs/update/{id} [put]
func (h *TreeEcsHandler) UpdateEcs(ctx *gin.Context) {
	var req model.UpdateEcsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.ecsService.UpdateEcs(ctx, &req)
	})
}

// ResizeEcs 调整ECS实例规格 - 云资源特有功能，仅支持本地资源时暂不提供
/*
func (h *TreeEcsHandler) ResizeEcs(ctx *gin.Context) {
	var req model.ResizeEcsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.ecsService.ResizeEcs(ctx, &req)
	})
}
*/

// ResetEcsPassword 重置ECS实例密码 - 云资源特有功能，仅支持本地资源时暂不提供
/*
func (h *TreeEcsHandler) ResetEcsPassword(ctx *gin.Context) {
	var req model.ResetEcsPasswordReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.ecsService.ResetEcsPassword(ctx, &req)
	})
}
*/

// RenewEcs 续费ECS实例 - 云资源特有功能，仅支持本地资源时暂不提供
/*
func (h *TreeEcsHandler) RenewEcs(ctx *gin.Context) {
	var req model.RenewEcsReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的实例ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.ecsService.RenewEcs(ctx, &req)
	})
}
*/

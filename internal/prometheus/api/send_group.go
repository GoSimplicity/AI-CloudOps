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
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertEventService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type SendGroupHandler struct {
	alertSendService alertEventService.AlertManagerSendService
}

func NewSendGroupHandler(alertSendService alertEventService.AlertManagerSendService) *SendGroupHandler {
	return &SendGroupHandler{
		alertSendService: alertSendService,
	}
}

func (s *SendGroupHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")

	sendGroups := monitorGroup.Group("/send_groups")
	{
		sendGroups.GET("/list", s.GetMonitorSendGroupList)
		sendGroups.GET("/detail/:id", s.GetMonitorSendGroup)
		sendGroups.POST("/create", s.CreateMonitorSendGroup)
		sendGroups.PUT("/update/:id", s.UpdateMonitorSendGroup)
		sendGroups.DELETE("/delete/:id", s.DeleteMonitorSendGroup)
	}
}

// GetMonitorSendGroupList 获取发送组列表
// @Summary 获取发送组列表
// @Description 获取所有发送组的分页列表
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/send_groups/list [get]
// @Security BearerAuth
func (s *SendGroupHandler) GetMonitorSendGroupList(ctx *gin.Context) {
	var req model.GetMonitorSendGroupListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return s.alertSendService.GetMonitorSendGroupList(ctx, &req)
	})
}

// CreateMonitorSendGroup 创建新的发送组
// @Summary 创建发送组
// @Description 创建新的告警发送组配置
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param request body model.CreateMonitorSendGroupReq true "创建发送组请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/send_groups/create [post]
// @Security BearerAuth
func (s *SendGroupHandler) CreateMonitorSendGroup(ctx *gin.Context) {
	var req model.CreateMonitorSendGroupReq

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid
	req.CreateUserName = uc.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, s.alertSendService.CreateMonitorSendGroup(ctx, &req)
	})
}

// UpdateMonitorSendGroup 更新现有的发送组
// @Summary 更新发送组
// @Description 更新指定的告警发送组配置
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path int true "发送组ID"
// @Param request body model.UpdateMonitorSendGroupReq true "更新发送组请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/send_groups/update/{id} [put]
// @Security BearerAuth
func (s *SendGroupHandler) UpdateMonitorSendGroup(ctx *gin.Context) {
	var req model.UpdateMonitorSendGroupReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, s.alertSendService.UpdateMonitorSendGroup(ctx, &req)
	})
}

// DeleteMonitorSendGroup 删除指定的发送组
// @Summary 删除发送组
// @Description 删除指定ID的告警发送组
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path int true "发送组ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/send_groups/delete/{id} [delete]
// @Security BearerAuth
func (s *SendGroupHandler) DeleteMonitorSendGroup(ctx *gin.Context) {
	var req model.DeleteMonitorSendGroupReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, s.alertSendService.DeleteMonitorSendGroup(ctx, &req)
	})
}

// GetMonitorSendGroup 获取指定的发送组详情
// @Summary 获取发送组详情
// @Description 根据ID获取指定发送组的详细信息
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path int true "发送组ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/send_groups/detail/{id} [get]
// @Security BearerAuth
func (s *SendGroupHandler) GetMonitorSendGroup(ctx *gin.Context) {
	var req model.GetMonitorSendGroupReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return s.alertSendService.GetMonitorSendGroup(ctx, &req)
	})
}

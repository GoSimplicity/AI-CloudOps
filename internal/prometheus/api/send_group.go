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
	alertEventService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
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

func (h *SendGroupHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")
	{
		monitorGroup.GET("/send_groups/list", h.GetMonitorSendGroupList)
		monitorGroup.GET("/send_groups/detail/:id", h.GetMonitorSendGroup)
		monitorGroup.POST("/send_groups/create", h.CreateMonitorSendGroup)
		monitorGroup.PUT("/send_groups/update/:id", h.UpdateMonitorSendGroup)
		monitorGroup.DELETE("/send_groups/delete/:id", h.DeleteMonitorSendGroup)
	}
}

// GetMonitorSendGroupList 获取发送组列表
func (h *SendGroupHandler) GetMonitorSendGroupList(ctx *gin.Context) {
	var req model.GetMonitorSendGroupListReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.alertSendService.GetMonitorSendGroupList(ctx, &req)
	})
}

// CreateMonitorSendGroup 创建新的发送组
func (h *SendGroupHandler) CreateMonitorSendGroup(ctx *gin.Context) {
	var req model.CreateMonitorSendGroupReq

	uc := ctx.MustGet("user").(jwt.UserClaims)
	req.UserID = uc.Uid
	req.CreateUserName = uc.Username

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.alertSendService.CreateMonitorSendGroup(ctx, &req)
	})
}

// UpdateMonitorSendGroup 更新现有的发送组
func (h *SendGroupHandler) UpdateMonitorSendGroup(ctx *gin.Context) {
	var req model.UpdateMonitorSendGroupReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.alertSendService.UpdateMonitorSendGroup(ctx, &req)
	})
}

// DeleteMonitorSendGroup 删除指定的发送组
func (h *SendGroupHandler) DeleteMonitorSendGroup(ctx *gin.Context) {
	var req model.DeleteMonitorSendGroupReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.alertSendService.DeleteMonitorSendGroup(ctx, &req)
	})
}

// GetMonitorSendGroup 获取指定的发送组详情
func (h *SendGroupHandler) GetMonitorSendGroup(ctx *gin.Context) {
	var req model.GetMonitorSendGroupReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.alertSendService.GetMonitorSendGroup(ctx, &req)
	})
}

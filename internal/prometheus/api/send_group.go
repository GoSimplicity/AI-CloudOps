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
	"go.uber.org/zap"
)

type SendGroupHandler struct {
	alertSendService alertEventService.AlertManagerSendService
	l                *zap.Logger
}

func NewSendGroupHandler(l *zap.Logger, alertSendService alertEventService.AlertManagerSendService) *SendGroupHandler {
	return &SendGroupHandler{
		l:                l,
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
func (s *SendGroupHandler) GetMonitorSendGroupList(ctx *gin.Context) {
	var req model.GetMonitorSendGroupListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return s.alertSendService.GetMonitorSendGroupList(ctx, &req)
	})
}

// CreateMonitorSendGroup 创建新的发送组
func (s *SendGroupHandler) CreateMonitorSendGroup(ctx *gin.Context) {
	var req model.CreateMonitorSendGroupReq

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, s.alertSendService.CreateMonitorSendGroup(ctx, &req)
	})
}

// UpdateMonitorSendGroup 更新现有的发送组
func (s *SendGroupHandler) UpdateMonitorSendGroup(ctx *gin.Context) {
	var req model.UpdateMonitorSendGroupReq

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id
	req.UserID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, s.alertSendService.UpdateMonitorSendGroup(ctx, &req)
	})
}

// DeleteMonitorSendGroup 删除指定的发送组
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

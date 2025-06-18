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
		sendGroups.GET("/:id", s.GetMonitorSendGroup)
		sendGroups.POST("/create", s.CreateMonitorSendGroup)
		sendGroups.POST("/update", s.UpdateMonitorSendGroup)
		sendGroups.DELETE("/:id", s.DeleteMonitorSendGroup)
		sendGroups.GET("/all", s.GetMonitorSendGroupAll)
	}
}

// GetMonitorSendGroupList 获取发送组列表
func (s *SendGroupHandler) GetMonitorSendGroupList(ctx *gin.Context) {
	var listReq model.ListReq

	utils.HandleRequest(ctx, &listReq, func() (interface{}, error) {
		return s.alertSendService.GetMonitorSendGroupList(ctx, &listReq)
	})
}

// CreateMonitorSendGroup 创建新的发送组
func (s *SendGroupHandler) CreateMonitorSendGroup(ctx *gin.Context) {
	var sendGroup model.MonitorSendGroup

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	sendGroup.UserID = uc.Uid

	utils.HandleRequest(ctx, &sendGroup, func() (interface{}, error) {
		return nil, s.alertSendService.CreateMonitorSendGroup(ctx, &sendGroup)
	})
}

// UpdateMonitorSendGroup 更新现有的发送组
func (s *SendGroupHandler) UpdateMonitorSendGroup(ctx *gin.Context) {
	var sendGroup model.MonitorSendGroup

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	sendGroup.UserID = uc.Uid

	utils.HandleRequest(ctx, &sendGroup, func() (interface{}, error) {
		return nil, s.alertSendService.UpdateMonitorSendGroup(ctx, &sendGroup)
	})
}

// DeleteMonitorSendGroup 删除指定的发送组
func (s *SendGroupHandler) DeleteMonitorSendGroup(ctx *gin.Context) {
	var req model.DeleteMonitorSendGroupRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, s.alertSendService.DeleteMonitorSendGroup(ctx, req.ID)
	})
}

func (s *SendGroupHandler) GetMonitorSendGroup(ctx *gin.Context) {
	var req model.GetMonitorSendGroupRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return s.alertSendService.GetMonitorSendGroup(ctx, req.ID)
	})
}

// GetMonitorSendGroupAll 获取所有发送组
func (s *SendGroupHandler) GetMonitorSendGroupAll(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return s.alertSendService.GetMonitorSendGroupAll(ctx)
	})
}

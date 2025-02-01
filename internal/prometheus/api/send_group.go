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
		sendGroups.GET("/total", s.GetMonitorSendGroupTotal)
		sendGroups.GET("/all", s.GetMonitorSendGroupAll)
	}
}

// GetMonitorSendGroupList 获取发送组列表
func (s *SendGroupHandler) GetMonitorSendGroupList(ctx *gin.Context) {
	var listReq model.ListReq

	if err := ctx.ShouldBindQuery(&listReq); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	list, err := s.alertSendService.GetMonitorSendGroupList(ctx, &listReq)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.SuccessWithData(ctx, list)
}

// CreateMonitorSendGroup 创建新的发送组
func (s *SendGroupHandler) CreateMonitorSendGroup(ctx *gin.Context) {
	var sendGroup model.MonitorSendGroup

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	if err := ctx.ShouldBindJSON(&sendGroup); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	sendGroup.UserID = uc.Uid

	if err := s.alertSendService.CreateMonitorSendGroup(ctx, &sendGroup); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.Success(ctx)
}

// UpdateMonitorSendGroup 更新现有的发送组
func (s *SendGroupHandler) UpdateMonitorSendGroup(ctx *gin.Context) {
	var sendGroup model.MonitorSendGroup

	if err := ctx.ShouldBind(&sendGroup); err != nil {
		utils.ErrorWithDetails(ctx, err, "参数错误")
		return
	}

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	sendGroup.UserID = uc.Uid

	if err := s.alertSendService.UpdateMonitorSendGroup(ctx, &sendGroup); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.Success(ctx)
}

// DeleteMonitorSendGroup 删除指定的发送组
func (s *SendGroupHandler) DeleteMonitorSendGroup(ctx *gin.Context) {
	id := ctx.Param("id")

	intId, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := s.alertSendService.DeleteMonitorSendGroup(ctx, intId); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.Success(ctx)
}

func (s *SendGroupHandler) GetMonitorSendGroup(ctx *gin.Context) {
	id := ctx.Param("id")

	intId, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	group, err := s.alertSendService.GetMonitorSendGroup(ctx, intId)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.SuccessWithData(ctx, group)
}

// GetMonitorSendGroupTotal 获取发送组总数
func (s *SendGroupHandler) GetMonitorSendGroupTotal(ctx *gin.Context) {
	total, err := s.alertSendService.GetMonitorSendGroupTotal(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	utils.SuccessWithData(ctx, total)
}

// GetMonitorSendGroupAll 获取所有发送组
func (s *SendGroupHandler) GetMonitorSendGroupAll(ctx *gin.Context) {
	groups, err := s.alertSendService.GetMonitorSendGroupAll(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	utils.SuccessWithData(ctx, groups)
}

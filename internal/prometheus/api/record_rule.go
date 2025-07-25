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

type RecordRuleHandler struct {
	alertRecordService alertEventService.AlertManagerRecordService
}

func NewRecordRuleHandler(alertRecordService alertEventService.AlertManagerRecordService) *RecordRuleHandler {
	return &RecordRuleHandler{
		alertRecordService: alertRecordService,
	}
}

func (r *RecordRuleHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")

	recordRules := monitorGroup.Group("/record_rules")
	{
		recordRules.GET("/list", r.GetMonitorRecordRuleList)
		recordRules.POST("/create", r.CreateMonitorRecordRule)
		recordRules.PUT("/update/:id", r.UpdateMonitorRecordRule)
		recordRules.DELETE("/delete/:id", r.DeleteMonitorRecordRule)
		recordRules.GET("/detail/:id", r.GetMonitorRecordRule)
	}
}

// GetMonitorRecordRuleList 获取预聚合规则列表
func (r *RecordRuleHandler) GetMonitorRecordRuleList(ctx *gin.Context) {
	var req model.GetMonitorRecordRuleListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return r.alertRecordService.GetMonitorRecordRuleList(ctx, &req)
	})
}

// CreateMonitorRecordRule 创建新的预聚合规则
func (r *RecordRuleHandler) CreateMonitorRecordRule(ctx *gin.Context) {
	var req model.CreateMonitorRecordRuleReq

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid
	req.CreateUserName = uc.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, r.alertRecordService.CreateMonitorRecordRule(ctx, &req)
	})
}

// UpdateMonitorRecordRule 更新现有的预聚合规则
func (r *RecordRuleHandler) UpdateMonitorRecordRule(ctx *gin.Context) {
	var req model.UpdateMonitorRecordRuleReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, r.alertRecordService.UpdateMonitorRecordRule(ctx, &req)
	})
}

// DeleteMonitorRecordRule 删除指定的预聚合规则
func (r *RecordRuleHandler) DeleteMonitorRecordRule(ctx *gin.Context) {
	var req model.DeleteMonitorRecordRuleReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, r.alertRecordService.DeleteMonitorRecordRule(ctx, &req)
	})
}

func (r *RecordRuleHandler) GetMonitorRecordRule(ctx *gin.Context) {
	var req model.GetMonitorRecordRuleReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return r.alertRecordService.GetMonitorRecordRule(ctx, &req)
	})
}

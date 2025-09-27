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
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CronJobHandler struct {
	logger      *zap.Logger
	cronService service.CronService
}

func NewCronJobHandler(logger *zap.Logger, cronService service.CronService) *CronJobHandler {
	return &CronJobHandler{
		logger:      logger,
		cronService: cronService,
	}
}

// RegisterRouters 注册路由
func (api *CronJobHandler) RegisterRouters(server *gin.Engine) {
	cronGroup := server.Group("/api/cron")
	{
		cronGroup.POST("/job/create", api.CreateCronJob)
		cronGroup.PUT("/job/:id/update", api.UpdateCronJob)
		cronGroup.DELETE("/job/:id/delete", api.DeleteCronJob)
		cronGroup.GET("/job/:id/detail", api.GetCronJob)
		cronGroup.GET("/job/list", api.GetCronJobList)
		cronGroup.POST("/job/:id/enable", api.EnableCronJob)
		cronGroup.POST("/job/:id/disable", api.DisableCronJob)
		cronGroup.POST("/job/:id/trigger", api.TriggerCronJob)
		cronGroup.POST("/validate-schedule", api.ValidateSchedule)
	}
}

// CreateCronJob 创建任务
func (api *CronJobHandler) CreateCronJob(ctx *gin.Context) {
	var req model.CreateCronJobReq

	user := ctx.MustGet("user").(utils.UserClaims)

	req.CreatedBy = user.Uid
	req.CreatedByName = user.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, api.cronService.CreateCronJob(ctx, &req)
	})
}

// UpdateCronJob 更新任务
func (api *CronJobHandler) UpdateCronJob(ctx *gin.Context) {
	var req model.UpdateCronJobReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, api.cronService.UpdateCronJob(ctx, &req)
	})
}

// DeleteCronJob 删除任务
func (api *CronJobHandler) DeleteCronJob(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, api.cronService.DeleteCronJob(ctx, id)
	})
}

// GetCronJob 获取任务详情
func (api *CronJobHandler) GetCronJob(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return api.cronService.GetCronJob(ctx, id)
	})
}

// GetCronJobList 获取任务列表
func (api *CronJobHandler) GetCronJobList(ctx *gin.Context) {
	var req model.GetCronJobListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return api.cronService.GetCronJobList(ctx, &req)
	})
}

// EnableCronJob 启用任务
func (api *CronJobHandler) EnableCronJob(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, api.cronService.EnableCronJob(ctx, id)
	})
}

// DisableCronJob 禁用任务
func (api *CronJobHandler) DisableCronJob(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, api.cronService.DisableCronJob(ctx, id)
	})
}

// TriggerCronJob 手动触发任务
func (api *CronJobHandler) TriggerCronJob(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, api.cronService.TriggerCronJob(ctx, id)
	})
}

// ValidateSchedule 验证调度表达式
func (api *CronJobHandler) ValidateSchedule(ctx *gin.Context) {
	var req model.ValidateScheduleReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return api.cronService.ValidateSchedule(ctx, &req)
	})
}

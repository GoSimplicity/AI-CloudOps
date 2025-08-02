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
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	service service.NotificationService
}

func NewNotificationHandler(service service.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		service: service,
	}
}

func (h *NotificationHandler) RegisterRouters(server *gin.Engine) {
	notificationGroup := server.Group("/api/workorder/notification")
	{
		notificationGroup.POST("/create", h.CreateNotification)
		notificationGroup.PUT("/update/:id", h.UpdateNotification)
		notificationGroup.DELETE("/delete/:id", h.DeleteNotification)
		notificationGroup.GET("/list", h.ListNotification)
		notificationGroup.GET("/detail/:id", h.DetailNotification)
		notificationGroup.GET("/logs", h.GetSendLogs)
		notificationGroup.POST("/test/send", h.TestSendNotification)
	}
}

// CreateNotification 创建通知配置
// @Summary 创建工单通知配置
// @Description 创建新的工单通知配置
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body model.CreateWorkorderNotificationReq true "创建通知配置请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/notification/create [post]
func (h *NotificationHandler) CreateNotification(ctx *gin.Context) {
	var req model.CreateWorkorderNotificationReq

	user := ctx.MustGet("user").(utils.UserClaims)
	req.UserID = user.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateNotification(ctx, &req)
	})

}

// UpdateNotification 更新通知配置
// @Summary 更新工单通知配置
// @Description 更新指定的工单通知配置信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "通知配置ID"
// @Param request body model.UpdateWorkorderNotificationReq true "更新通知配置请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/notification/update/{id} [put]
func (h *NotificationHandler) UpdateNotification(ctx *gin.Context) {
	var req model.UpdateWorkorderNotificationReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateNotification(ctx, &req)
	})
}

// DeleteNotification 删除通知配置
// @Summary 删除工单通知配置
// @Description 删除指定的工单通知配置
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "通知配置ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/notification/delete/{id} [delete]
func (h *NotificationHandler) DeleteNotification(ctx *gin.Context) {
	var req model.DeleteWorkorderNotificationReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteNotification(ctx, &req)
	})
}

// ListNotification 获取通知配置列表
// @Summary 获取工单通知配置列表
// @Description 分页获取工单通知配置列表
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Param name query string false "通知配置名称"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/notification/list [get]
func (h *NotificationHandler) ListNotification(ctx *gin.Context) {
	var req model.ListWorkorderNotificationReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListNotification(ctx, &req)
	})
}

// DetailNotification 获取通知配置详情
// @Summary 获取工单通知配置详情
// @Description 获取指定工单通知配置的详细信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "通知配置ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/notification/detail/{id} [get]
func (h *NotificationHandler) DetailNotification(ctx *gin.Context) {
	var req model.DetailWorkorderNotificationReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.DetailNotification(ctx, &req)
	})
}

// GetSendLogs 获取通知发送日志
// @Summary 获取工单通知发送日志
// @Description 获取工单通知的发送日志记录
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Param notificationId query int false "通知配置ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/notification/logs [get]
func (h *NotificationHandler) GetSendLogs(ctx *gin.Context) {
	var req model.ListWorkorderNotificationLogReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetSendLogs(ctx, &req)
	})
}

// TestSendNotification 测试发送通知
// @Summary 测试工单通知发送
// @Description 测试工单通知的发送功能
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body model.TestSendWorkorderNotificationReq true "测试发送通知请求参数"
// @Success 200 {object} utils.ApiResponse "发送成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/notification/test/send [post]
func (h *NotificationHandler) TestSendNotification(ctx *gin.Context) {
	var req model.TestSendWorkorderNotificationReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.TestSendNotification(ctx, &req)
	})
}


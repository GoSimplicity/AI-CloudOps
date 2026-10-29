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
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	service service.WorkorderNotificationService
}

func NewNotificationHandler(service service.WorkorderNotificationService) *NotificationHandler {
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
		notificationGroup.GET("/events", h.ListNotificationEvents)
		notificationGroup.GET("/events/:id/deliveries", h.ListNotificationEventDeliveries)
		notificationGroup.GET("/events/:id/resolutions", h.ListNotificationEventResolutions)
		notificationGroup.POST("/events/:id/resolve", h.ResolveNotificationDelivery)
		notificationGroup.POST("/events/:id/retry", h.RetryNotificationEvent)
		notificationGroup.POST("/events/:id/cancel", h.CancelScheduledNotification)
		notificationGroup.POST("/test/send", h.TestSendNotification)
		notificationGroup.GET("/channels", h.GetAvailableChannels)
		notificationGroup.POST("/send", h.SendNotificationManually)
	}
}

func (h *NotificationHandler) CreateNotification(ctx *gin.Context) {
	var req model.CreateWorkorderNotificationReq

	user := ctx.MustGet("user").(jwt.UserClaims)
	req.UserID = user.Uid

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateNotification(ctx.Request.Context(), &req)
	})

}

func (h *NotificationHandler) UpdateNotification(ctx *gin.Context) {
	var req model.UpdateWorkorderNotificationReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateNotification(ctx.Request.Context(), &req)
	})
}

func (h *NotificationHandler) DeleteNotification(ctx *gin.Context) {
	var req model.DeleteWorkorderNotificationReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteNotification(ctx.Request.Context(), &req)
	})
}

func (h *NotificationHandler) ListNotification(ctx *gin.Context) {
	var req model.ListWorkorderNotificationReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListNotification(ctx.Request.Context(), &req)
	})
}

func (h *NotificationHandler) DetailNotification(ctx *gin.Context) {
	var req model.DetailWorkorderNotificationReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}
	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.DetailNotification(ctx.Request.Context(), &req)
	})
}

func (h *NotificationHandler) GetSendLogs(ctx *gin.Context) {
	var req model.ListWorkorderNotificationLogReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetSendLogs(ctx.Request.Context(), &req)
	})
}

// TestSendNotification 测试发送通知
func (h *NotificationHandler) TestSendNotification(ctx *gin.Context) {
	var req model.TestSendWorkorderNotificationReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.TestSendNotification(ctx.Request.Context(), &req)
	})
}

func (h *NotificationHandler) GetAvailableChannels(ctx *gin.Context) {
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetAvailableChannels(), nil
	})
}

// SendNotificationManually 手动发送通知
func (h *NotificationHandler) SendNotificationManually(ctx *gin.Context) {
	var req model.ManualSendNotificationReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.SendNotificationByChannels(ctx.Request.Context(), req.Channels, req.Recipient, req.Subject, req.Content)
	})
}

// ListNotificationEvents 和 ListNotificationEventDeliveries 由全局JWT及API角色权限控制
func (h *NotificationHandler) ListNotificationEvents(ctx *gin.Context) {
	var req model.ListWorkorderNotificationEventsReq
	base.HandleRequest(ctx, &req, func() (interface{}, error) { return h.service.ListNotificationEvents(ctx.Request.Context(), &req) })
}
func (h *NotificationHandler) ListNotificationEventDeliveries(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil || id <= 0 {
		base.ErrorWithMessage(ctx, "无效的事件ID")
		return
	}
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.ListNotificationEventDeliveries(ctx.Request.Context(), uint64(id))
	})
}

// ResolveNotificationDelivery 使用独立API权限并记录经过鉴权的操作者
func (h *NotificationHandler) ResolveNotificationDelivery(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil || id <= 0 {
		base.ErrorWithMessage(ctx, "无效的事件ID")
		return
	}
	user := ctx.MustGet("user").(jwt.UserClaims)
	var req model.ResolveWorkorderNotificationDeliveryReq
	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.ResolveNotificationDelivery(ctx.Request.Context(), uint64(id), &req, user.Uid)
	})
}
func (h *NotificationHandler) ListNotificationEventResolutions(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil || id <= 0 {
		base.ErrorWithMessage(ctx, "无效的事件ID")
		return
	}
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.ListNotificationEventResolutions(ctx.Request.Context(), uint64(id))
	})
}

func (h *NotificationHandler) RetryNotificationEvent(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil || id <= 0 {
		base.ErrorWithMessage(ctx, "无效的事件ID")
		return
	}
	user := ctx.MustGet("user").(jwt.UserClaims)
	var req model.RetryWorkorderNotificationEventReq
	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.RetryNotificationEvent(ctx.Request.Context(), uint64(id), req.Reason, user.Uid)
	})
}

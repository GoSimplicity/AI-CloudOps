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
		notificationGroup.POST("/duplicate", h.DuplicateNotification)
	}
}

func (h *NotificationHandler) CreateNotification(ctx *gin.Context) {
	var req model.CreateWorkorderNotificationReq

	user := ctx.MustGet("user").(utils.UserClaims)
	req.UserID = user.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateNotification(ctx, &req)
	})

}

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

func (h *NotificationHandler) ListNotification(ctx *gin.Context) {
	var req model.ListWorkorderNotificationReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListNotification(ctx, &req)
	})
}

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

func (h *NotificationHandler) GetSendLogs(ctx *gin.Context) {
	var req model.ListWorkorderNotificationLogReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetSendLogs(ctx, &req)
	})
}

func (h *NotificationHandler) TestSendNotification(ctx *gin.Context) {
	var req model.TestSendWorkorderNotificationReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.TestSendNotification(ctx, &req)
	})
}

func (h *NotificationHandler) DuplicateNotification(ctx *gin.Context) {
	var req model.DuplicateWorkorderNotificationReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DuplicateNotification(ctx, &req)
	})
}

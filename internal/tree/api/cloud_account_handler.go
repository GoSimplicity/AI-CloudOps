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
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type CloudAccountHandler struct {
	service service.CloudAccountService
}

func NewCloudAccountHandler(service service.CloudAccountService) *CloudAccountHandler {
	return &CloudAccountHandler{
		service: service,
	}
}

func (h *CloudAccountHandler) RegisterRouters(server *gin.Engine) {
	accountGroup := server.Group("/api/tree/cloud/account")
	{
		// 基础操作
		accountGroup.GET("/list", h.GetCloudAccountList)
		accountGroup.GET("/:id/detail", h.GetCloudAccountDetail)
		accountGroup.POST("/create", h.CreateCloudAccount)
		accountGroup.PUT("/:id/update", h.UpdateCloudAccount)
		accountGroup.DELETE("/:id/delete", h.DeleteCloudAccount)
		accountGroup.PUT("/:id/status", h.UpdateCloudAccountStatus)
		accountGroup.POST("/:id/verify", h.VerifyCloudAccount)

		// 批量操作
		accountGroup.POST("/batch/delete", h.BatchDeleteCloudAccount)
		accountGroup.PUT("/batch/status", h.BatchUpdateCloudAccountStatus)

		// 导入导出
		accountGroup.POST("/import", h.ImportCloudAccount)
		accountGroup.POST("/export", h.ExportCloudAccount)
	}
}

// GetCloudAccountList 获取云账户列表
func (h *CloudAccountHandler) GetCloudAccountList(ctx *gin.Context) {
	var req model.GetCloudAccountListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetCloudAccountList(ctx, &req)
	})
}

// GetCloudAccountDetail 获取云账户详情
func (h *CloudAccountHandler) GetCloudAccountDetail(ctx *gin.Context) {
	var req model.GetCloudAccountDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的账户ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetCloudAccountDetail(ctx, &req)
	})
}

// CreateCloudAccount 创建云账户
func (h *CloudAccountHandler) CreateCloudAccount(ctx *gin.Context) {
	var req model.CreateCloudAccountReq

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateCloudAccount(ctx, &req, user.Uid, user.Username)
	})
}

// UpdateCloudAccount 更新云账户
func (h *CloudAccountHandler) UpdateCloudAccount(ctx *gin.Context) {
	var req model.UpdateCloudAccountReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的账户ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateCloudAccount(ctx, &req)
	})
}

// DeleteCloudAccount 删除云账户
func (h *CloudAccountHandler) DeleteCloudAccount(ctx *gin.Context) {
	var req model.DeleteCloudAccountReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的账户ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteCloudAccount(ctx, &req)
	})
}

// UpdateCloudAccountStatus 更新云账户状态
func (h *CloudAccountHandler) UpdateCloudAccountStatus(ctx *gin.Context) {
	var req model.UpdateCloudAccountStatusReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的账户ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateCloudAccountStatus(ctx, &req)
	})
}

// VerifyCloudAccount 验证云账户凭证
func (h *CloudAccountHandler) VerifyCloudAccount(ctx *gin.Context) {
	var req model.VerifyCloudAccountReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的账户ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.VerifyCloudAccount(ctx, &req)
	})
}

// BatchDeleteCloudAccount 批量删除云账户
func (h *CloudAccountHandler) BatchDeleteCloudAccount(ctx *gin.Context) {
	var req model.BatchDeleteCloudAccountReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.BatchDeleteCloudAccount(ctx, &req)
	})
}

// BatchUpdateCloudAccountStatus 批量更新云账户状态
func (h *CloudAccountHandler) BatchUpdateCloudAccountStatus(ctx *gin.Context) {
	var req model.BatchUpdateCloudAccountStatusReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.BatchUpdateCloudAccountStatus(ctx, &req)
	})
}

// ImportCloudAccount 导入云账户
func (h *CloudAccountHandler) ImportCloudAccount(ctx *gin.Context) {
	var req model.ImportCloudAccountReq

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ImportCloudAccount(ctx, &req, user.Uid, user.Username)
	})
}

// ExportCloudAccount 导出云账户
func (h *CloudAccountHandler) ExportCloudAccount(ctx *gin.Context) {
	var req model.ExportCloudAccountReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ExportCloudAccount(ctx, &req)
	})
}

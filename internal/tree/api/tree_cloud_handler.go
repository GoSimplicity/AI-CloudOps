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

type TreeCloudHandler struct {
	cloudService service.TreeCloudService
}

func NewTreeCloudHandler(cloudService service.TreeCloudService) *TreeCloudHandler {
	return &TreeCloudHandler{
		cloudService: cloudService,
	}
}

func (h *TreeCloudHandler) RegisterRouters(r gin.IRouter) {
	cloudGroup := r.Group("/api/tree/cloud")
	{
		// 云账号管理
		accounts := cloudGroup.Group("/accounts")
		{
			accounts.POST("/create", h.CreateCloudAccount)
			accounts.GET("/list", h.ListCloudAccounts)
			accounts.GET("/detail/:id", h.DetailCloudAccount)
			accounts.PUT("/update/:id", h.UpdateCloudAccount)
			accounts.DELETE("/delete/:id", h.DeleteCloudAccount)
			accounts.POST("/test/:id", h.TestCloudAccount)
		}

		// 云资源同步
		cloudGroup.POST("/sync", h.SyncCloudResources)
		cloudGroup.POST("/sync/:id", h.SyncCloudAccountResources)

		// 云账号统计
		cloudGroup.GET("/statistics", h.GetCloudAccountStatistics)
	}
}

// CreateCloudAccount 创建云账号
func (h *TreeCloudHandler) CreateCloudAccount(ctx *gin.Context) {
	var req model.CreateCloudAccountReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.cloudService.CreateCloudAccount(ctx, &req)
	})
}

// UpdateCloudAccount 更新云账号
func (h *TreeCloudHandler) UpdateCloudAccount(ctx *gin.Context) {
	var req model.UpdateCloudAccountReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "账号ID格式错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.cloudService.UpdateCloudAccount(ctx, &req)
	})
}

// DeleteCloudAccount 删除云账号
func (h *TreeCloudHandler) DeleteCloudAccount(ctx *gin.Context) {
	var req model.DeleteCloudAccountReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "账号ID格式错误")
		return
	}

	req.ID = id
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.cloudService.DeleteCloudAccount(ctx, id)
	})
}

// DetailCloudAccount 获取云账号详情
func (h *TreeCloudHandler) DetailCloudAccount(ctx *gin.Context) {
	var req model.GetCloudAccountReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "账号ID格式错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.cloudService.GetCloudAccount(ctx, req.ID)
	})
}

// ListCloudAccounts 获取云账号列表
func (h *TreeCloudHandler) ListCloudAccounts(ctx *gin.Context) {
	var req model.ListCloudAccountsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.cloudService.ListCloudAccounts(ctx, &req)
	})
}

// TestCloudAccount 测试云账号连接
func (h *TreeCloudHandler) TestCloudAccount(ctx *gin.Context) {
	var req model.TestCloudAccountReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "账号ID格式错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.cloudService.TestCloudAccount(ctx, req.ID)
	})
}

// SyncCloudResources 同步所有云资源
func (h *TreeCloudHandler) SyncCloudResources(ctx *gin.Context) {
	var req model.SyncCloudReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.cloudService.SyncCloudResources(ctx, &req)
	})
}

// SyncCloudAccountResources 同步指定云账号的资源
func (h *TreeCloudHandler) SyncCloudAccountResources(ctx *gin.Context) {
	var req model.SyncCloudAccountResourcesReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "账号ID格式错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.cloudService.SyncCloudAccountResources(ctx, &req)
	})
}

// GetCloudAccountStatistics 获取云账号统计信息
func (h *TreeCloudHandler) GetCloudAccountStatistics(ctx *gin.Context) {
	var req model.GetCloudAccountStatisticsReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.cloudService.GetCloudAccountStatistics(ctx, &req)
	})
}

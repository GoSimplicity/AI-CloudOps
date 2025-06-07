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

// RegisterRoutes 注册路由
func (h *TreeCloudHandler) RegisterRouters(r gin.IRouter) {
	cloudGroup := r.Group("/api/tree/cloud")
	{
		cloudGroup.POST("/accounts/create", h.CreateCloudAccount)
		cloudGroup.GET("/accounts/list", h.ListCloudAccounts)
		cloudGroup.GET("/accounts/detail/:id", h.DetailCloudAccount)
		cloudGroup.PUT("/accounts/update/:id", h.UpdateCloudAccount)
		cloudGroup.DELETE("/accounts/delete/:id", h.DeleteCloudAccount)
		cloudGroup.POST("/accounts/test/:id", h.TestCloudAccount)
		cloudGroup.POST("/sync", h.SyncCloudResources)
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
		utils.ErrorWithMessage(ctx, "账号ID格式错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.cloudService.UpdateCloudAccount(ctx, req.ID, &req)
	})
}

// DeleteCloudAccount 删除云账号
func (h *TreeCloudHandler) DeleteCloudAccount(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "账号ID格式错误")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.cloudService.DeleteCloudAccount(ctx, id)
	})
}

// GetCloudAccount 获取云账号详情
func (h *TreeCloudHandler) DetailCloudAccount(ctx *gin.Context) {
	var req model.GetCloudAccountReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "账号ID格式错误")
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
		utils.ErrorWithMessage(ctx, "账号ID格式错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.cloudService.TestCloudAccount(ctx, req.ID)
	})
}

// SyncCloudResources 同步云资源
func (h *TreeCloudHandler) SyncCloudResources(ctx *gin.Context) {
	var req model.SyncCloudReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.cloudService.SyncCloudResources(ctx, &req)
	})
}

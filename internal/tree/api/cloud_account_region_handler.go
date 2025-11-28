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
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type CloudAccountRegionHandler struct {
	service service.CloudAccountRegionService
}

func NewCloudAccountRegionHandler(service service.CloudAccountRegionService) *CloudAccountRegionHandler {
	return &CloudAccountRegionHandler{
		service: service,
	}
}

func (h *CloudAccountRegionHandler) RegisterRouters(server *gin.Engine) {
	regionGroup := server.Group("/api/tree/cloud/account/region")
	{
		regionGroup.GET("/list", h.GetCloudAccountRegionList)
		regionGroup.GET("/:id/detail", h.GetCloudAccountRegionDetail)
		regionGroup.POST("/create", h.CreateCloudAccountRegion)
		regionGroup.POST("/batch-create", h.BatchCreateCloudAccountRegion)
		regionGroup.PUT("/:id/update", h.UpdateCloudAccountRegion)
		regionGroup.DELETE("/:id/delete", h.DeleteCloudAccountRegion)
		regionGroup.PUT("/:id/status", h.UpdateCloudAccountRegionStatus)
		regionGroup.GET("/available-regions", h.GetAvailableRegions)
	}
}

// GetCloudAccountRegionList 获取云账号区域列表
func (h *CloudAccountRegionHandler) GetCloudAccountRegionList(ctx *gin.Context) {
	var req model.GetCloudAccountRegionListReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetCloudAccountRegionList(ctx, &req)
	})
}

// GetCloudAccountRegionDetail 获取云账号区域详情
func (h *CloudAccountRegionHandler) GetCloudAccountRegionDetail(ctx *gin.Context) {
	var req model.GetCloudAccountDetailReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的区域ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetCloudAccountRegionDetail(ctx, req.ID)
	})
}

// CreateCloudAccountRegion 创建云账号区域关联
func (h *CloudAccountRegionHandler) CreateCloudAccountRegion(ctx *gin.Context) {
	var req model.CreateCloudAccountRegionReq

	user := ctx.MustGet("user").(jwt.UserClaims)

	req.CreateUserID = user.Uid
	req.CreateUserName = user.Username

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateCloudAccountRegion(ctx, &req)
	})
}

// BatchCreateCloudAccountRegion 批量创建云账号区域关联
func (h *CloudAccountRegionHandler) BatchCreateCloudAccountRegion(ctx *gin.Context) {
	var req model.BatchCreateCloudAccountRegionReq

	user := ctx.MustGet("user").(jwt.UserClaims)

	req.CreateUserID = user.Uid
	req.CreateUserName = user.Username

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.BatchCreateCloudAccountRegion(ctx, &req)
	})
}

// UpdateCloudAccountRegion 更新云账号区域关联
func (h *CloudAccountRegionHandler) UpdateCloudAccountRegion(ctx *gin.Context) {
	var req model.UpdateCloudAccountRegionReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的区域ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateCloudAccountRegion(ctx, &req)
	})
}

// DeleteCloudAccountRegion 删除云账号区域关联
func (h *CloudAccountRegionHandler) DeleteCloudAccountRegion(ctx *gin.Context) {
	var req model.DeleteCloudAccountRegionReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的区域ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteCloudAccountRegion(ctx, &req)
	})
}

// UpdateCloudAccountRegionStatus 更新云账号区域状态
func (h *CloudAccountRegionHandler) UpdateCloudAccountRegionStatus(ctx *gin.Context) {
	var req model.UpdateCloudAccountRegionStatusReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的区域ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateCloudAccountRegionStatus(ctx, &req)
	})
}

// GetAvailableRegions 获取指定云厂商的可用区域列表
func (h *CloudAccountRegionHandler) GetAvailableRegions(ctx *gin.Context) {
	var req model.GetAvailableRegionsReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetAvailableRegions(ctx, &req)
	})
}

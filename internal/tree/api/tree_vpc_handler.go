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

type TreeVpcHandler struct {
	vpcService service.TreeVpcService
}

func NewTreeVpcHandler(vpcService service.TreeVpcService) *TreeVpcHandler {
	return &TreeVpcHandler{
		vpcService: vpcService,
	}
}

func (h *TreeVpcHandler) RegisterRouters(server *gin.Engine) {
	vpcGroup := server.Group("/api/tree/vpc")
	{
		vpcGroup.POST("/detail/:id", h.GetVpcDetail)
		vpcGroup.POST("/create", h.CreateVpcResource)
		vpcGroup.DELETE("/delete/:id", h.DeleteVpc)
		vpcGroup.POST("/list", h.ListVpcResources)
		vpcGroup.POST("/update/:id", h.UpdateVpc)
		vpcGroup.POST("/subnet/create", h.CreateSubnet)
		vpcGroup.DELETE("/subnet/delete/:id", h.DeleteSubnet)
		vpcGroup.POST("/subnet/list", h.ListSubnets)
		vpcGroup.POST("/subnet/detail/:id", h.GetSubnetDetail)
		vpcGroup.POST("/subnet/update/:id", h.UpdateSubnet)
		vpcGroup.POST("/peering/create", h.CreateVpcPeering)
		vpcGroup.DELETE("/peering/delete/:id", h.DeleteVpcPeering)
		vpcGroup.POST("/peering/list", h.ListVpcPeerings)
	}
}

// GetVpcDetail 获取VPC详情
func (h *TreeVpcHandler) GetVpcDetail(ctx *gin.Context) {
	var req model.GetVpcDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的VPC ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.vpcService.GetVpcDetail(ctx, &req)
	})
}

// CreateVpcResource 创建VPC资源
func (h *TreeVpcHandler) CreateVpcResource(ctx *gin.Context) {
	var req model.CreateVpcResourceReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.vpcService.CreateVpcResource(ctx, &req)
	})
}

// DeleteVpc 删除VPC
func (h *TreeVpcHandler) DeleteVpc(ctx *gin.Context) {
	var req model.DeleteVpcReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的VPC ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.vpcService.DeleteVpc(ctx, &req)
	})
}

// ListVpcResources 获取VPC列表
func (h *TreeVpcHandler) ListVpcResources(ctx *gin.Context) {
	var req model.ListVpcResourcesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.vpcService.ListVpcResources(ctx, &req)
	})
}

// UpdateVpc 更新VPC
func (h *TreeVpcHandler) UpdateVpc(ctx *gin.Context) {
	var req model.UpdateVpcReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的VPC ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.vpcService.UpdateVpc(ctx, &req)
	})
}

// CreateSubnet 创建子网
func (h *TreeVpcHandler) CreateSubnet(ctx *gin.Context) {
	var req model.CreateSubnetReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.vpcService.CreateSubnet(ctx, &req)
	})
}

// DeleteSubnet 删除子网
func (h *TreeVpcHandler) DeleteSubnet(ctx *gin.Context) {
	var req model.DeleteSubnetReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的子网 ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.vpcService.DeleteSubnet(ctx, &req)
	})
}

// ListSubnets 获取子网列表
func (h *TreeVpcHandler) ListSubnets(ctx *gin.Context) {
	var req model.ListSubnetsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.vpcService.ListSubnets(ctx, &req)
	})
}

// GetSubnetDetail 获取子网详情
func (h *TreeVpcHandler) GetSubnetDetail(ctx *gin.Context) {
	var req model.GetSubnetDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的子网 ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.vpcService.GetSubnetDetail(ctx, &req)
	})
}

// UpdateSubnet 更新子网
func (h *TreeVpcHandler) UpdateSubnet(ctx *gin.Context) {
	var req model.UpdateSubnetReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的子网 ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.vpcService.UpdateSubnet(ctx, &req)
	})
}

// CreateVpcPeering 创建VPC对等连接
func (h *TreeVpcHandler) CreateVpcPeering(ctx *gin.Context) {
	var req model.CreateVpcPeeringReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的VPC ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.vpcService.CreateVpcPeering(ctx, &req)
	})
}

// DeleteVpcPeering 删除VPC对等连接
func (h *TreeVpcHandler) DeleteVpcPeering(ctx *gin.Context) {
	var req model.DeleteVpcPeeringReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的VPC ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.vpcService.DeleteVpcPeering(ctx, &req)
	})
}

// ListVpcPeerings 获取VPC对等连接列表
func (h *TreeVpcHandler) ListVpcPeerings(ctx *gin.Context) {
	var req model.ListVpcPeeringsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.vpcService.ListVpcPeerings(ctx, &req)
	})
}

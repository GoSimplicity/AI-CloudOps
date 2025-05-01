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

type TreeHandler struct {
	service service.TreeService
}

func NewTreeHandler(service service.TreeService) *TreeHandler {
	return &TreeHandler{
		service: service,
	}
}

func (h *TreeHandler) RegisterRouters(server *gin.Engine) {
	treeGroup := server.Group("/api/tree")
	{
		treeGroup.GET("/list", h.GetTree)
		treeGroup.GET("/detail/:id", h.GetNodeDetail)
		treeGroup.GET("/children/:parentId", h.GetChildNodes)
		treeGroup.GET("/path/:nodeId", h.GetNodePath)

		treeGroup.POST("/create", h.CreateNode)
		treeGroup.POST("/update", h.UpdateNode)
		treeGroup.POST("/delete/:id", h.DeleteNode)

		treeGroup.POST("/bind_resource", h.BindResource)
		treeGroup.POST("/unbind_resource", h.UnbindResource)

		treeGroup.POST("/add_admin", h.AddNodeAdmin)
		treeGroup.POST("/remove_admin", h.RemoveNodeAdmin)
		treeGroup.POST("/add_member", h.AddNodeMember)
		treeGroup.POST("/remove_member", h.RemoveNodeMember)
	}
}

// GetTree 获取整个服务树
func (h *TreeHandler) GetTree(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetTree(ctx)
	})
}

// GetNodeDetail 获取节点详情
func (h *TreeHandler) GetNodeDetail(ctx *gin.Context) {
	nodeId, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetNodeById(ctx, nodeId)
	})
}

// GetChildNodes 获取子节点列表
func (h *TreeHandler) GetChildNodes(ctx *gin.Context) {
	parentId, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetChildNodes(ctx, parentId)
	})
}

// GetNodePath 获取节点路径
func (h *TreeHandler) GetNodePath(ctx *gin.Context) {
	nodeId, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetNodePath(ctx, nodeId)
	})
}

// CreateNode 创建节点
func (h *TreeHandler) CreateNode(ctx *gin.Context) {
	var req model.CreateNodeReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.CreateNode(ctx, &req)
	})
}

// UpdateNode 更新节点
func (h *TreeHandler) UpdateNode(ctx *gin.Context) {
	var req model.UpdateNodeReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateNode(ctx, &req)
	})
}

// DeleteNode 删除节点
func (h *TreeHandler) DeleteNode(ctx *gin.Context) {
	nodeId, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.DeleteNode(ctx, nodeId)
	})
}

// BindResource 绑定资源到节点
func (h *TreeHandler) BindResource(ctx *gin.Context) {
	var req model.ResourceBindingRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.BindResource(ctx, &req)
	})
}

// UnbindResource 解绑节点资源
func (h *TreeHandler) UnbindResource(ctx *gin.Context) {
	var req model.ResourceBindingRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UnbindResource(ctx, &req)
	})
}

// AddNodeAdmin 添加节点管理员
func (h *TreeHandler) AddNodeAdmin(ctx *gin.Context) {
	var req model.NodeAdminReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.AddNodeAdmin(ctx, &req)
	})
}

// RemoveNodeAdmin 移除节点管理员
func (h *TreeHandler) RemoveNodeAdmin(ctx *gin.Context) {
	var req model.NodeAdminReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.RemoveNodeAdmin(ctx, &req)
	})
}

// AddNodeMember 添加节点成员
func (h *TreeHandler) AddNodeMember(ctx *gin.Context) {
	var req model.NodeMemberReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.AddNodeMember(ctx, &req)
	})
}

// RemoveNodeMember 移除节点成员
func (h *TreeHandler) RemoveNodeMember(ctx *gin.Context) {
	var req model.NodeMemberReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.RemoveNodeMember(ctx, &req)
	})
}

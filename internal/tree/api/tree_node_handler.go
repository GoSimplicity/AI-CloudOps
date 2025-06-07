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

type TreeNodeHandler struct {
	service service.TreeNodeService
}

func NewTreeNodeHandler(service service.TreeNodeService) *TreeNodeHandler {
	return &TreeNodeHandler{
		service: service,
	}
}

func (h *TreeNodeHandler) RegisterRouters(server *gin.Engine) {
	treeGroup := server.Group("/api/tree/node")
	{
		// 树结构相关接口
		treeGroup.GET("/list", h.GetTreeList)
		treeGroup.GET("/detail/:id", h.GetNodeDetail)
		treeGroup.GET("/children/:id", h.GetChildNodes)
		treeGroup.GET("/statistics", h.GetTreeStatistics)

		// 节点管理接口
		treeGroup.POST("/node/create", h.CreateNode)
		treeGroup.PUT("/node/update/:id", h.UpdateNode)
		treeGroup.DELETE("/node/delete/:id", h.DeleteNode)
		treeGroup.PUT("/node/move/:id", h.MoveNode)
		treeGroup.PUT("/node/status/:id", h.UpdateNodeStatus)

		// 成员管理接口
		treeGroup.GET("/members/:id", h.GetNodeMembers)
		treeGroup.POST("/member/add", h.AddNodeMember)
		treeGroup.DELETE("/member/remove/:id", h.RemoveNodeMember)

		// 资源绑定接口
		treeGroup.GET("/resources/:id", h.GetNodeResources)
		treeGroup.POST("/resource/bind", h.BindResource)
		treeGroup.DELETE("/resource/unbind", h.UnbindResource)
	}
}

// GetTreeList 获取树节点列表
func (h *TreeNodeHandler) GetTreeList(ctx *gin.Context) {
	var req model.GetTreeListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetTreeList(ctx, &req)
	})
}

// GetNodeDetail 获取节点详情
func (h *TreeNodeHandler) GetNodeDetail(ctx *gin.Context) {
	var req model.GetNodeDetailReq
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetNodeDetail(ctx, req.ID)
	})
}

// GetChildNodes 获取子节点列表
func (h *TreeNodeHandler) GetChildNodes(ctx *gin.Context) {
	var req model.GetChildNodesReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetChildNodes(ctx, req.ID)
	})
}

// GetTreeStatistics 获取树统计信息
func (h *TreeNodeHandler) GetTreeStatistics(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetTreeStatistics(ctx)
	})
}

// CreateNode 创建节点
func (h *TreeNodeHandler) CreateNode(ctx *gin.Context) {
	var req model.CreateNodeReq

	user := ctx.MustGet("user").(utils.UserClaims)
	req.CreatorID = user.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateNode(ctx, &req)
	})
}

// UpdateNode 更新节点
func (h *TreeNodeHandler) UpdateNode(ctx *gin.Context) {
	var req model.UpdateNodeReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateNode(ctx, &req)
	})
}

// UpdateNodeStatus 更新节点状态
func (h *TreeNodeHandler) UpdateNodeStatus(ctx *gin.Context) {
	var req model.UpdateNodeStatusReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateNodeStatus(ctx, &req)
	})
}

// DeleteNode 删除节点
func (h *TreeNodeHandler) DeleteNode(ctx *gin.Context) {
	var req model.DeleteNodeReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteNode(ctx, req.ID)
	})
}

// MoveNode 移动节点
func (h *TreeNodeHandler) MoveNode(ctx *gin.Context) {
	var req model.MoveNodeReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.MoveNode(ctx, req.ID, req.NewParentID)
	})
}

// GetNodeMembers 获取节点成员
func (h *TreeNodeHandler) GetNodeMembers(ctx *gin.Context) {
	var req model.GetNodeMembersReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetNodeMembers(ctx, req.ID, req.Type)
	})
}

// AddNodeMember 添加节点成员
func (h *TreeNodeHandler) AddNodeMember(ctx *gin.Context) {
	var req model.AddNodeMemberReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.AddNodeMember(ctx, &req)
	})
}

// RemoveNodeMember 移除节点成员
func (h *TreeNodeHandler) RemoveNodeMember(ctx *gin.Context) {
	var req model.RemoveNodeMemberReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}

	req.NodeID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.RemoveNodeMember(ctx, &req)
	})
}

// GetNodeResources 获取节点资源
func (h *TreeNodeHandler) GetNodeResources(ctx *gin.Context) {
	var req model.GetNodeResourcesReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetNodeResources(ctx, req.ID)
	})
}

// BindResource 绑定资源
func (h *TreeNodeHandler) BindResource(ctx *gin.Context) {
	var req model.BindResourceReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.BindResource(ctx, &req)
	})
}

// UnbindResource 解绑资源
func (h *TreeNodeHandler) UnbindResource(ctx *gin.Context) {
	var req model.UnbindResourceReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UnbindResource(ctx, &req)
	})
}

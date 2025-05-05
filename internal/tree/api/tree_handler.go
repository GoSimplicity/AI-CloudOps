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
		// 树结构相关接口
		treeGroup.GET("/list", h.GetTreeList)
		treeGroup.GET("/detail/:id", h.GetNodeDetail)
		treeGroup.GET("/children/:parentId", h.GetChildNodes)
		treeGroup.GET("/path/:nodeId", h.GetNodePath)
		treeGroup.GET("/statistics", h.GetTreeStatistics)

		// 节点管理接口
		treeGroup.POST("/node/create", h.CreateNode)
		treeGroup.POST("/node/create_child", h.CreateChildNode)
		treeGroup.PUT("/node/update", h.UpdateNode)
		treeGroup.DELETE("/node/delete/:id", h.DeleteNode)

		// 资源绑定接口
		treeGroup.GET("/resources/:nodeId", h.GetNodeResources)
		treeGroup.POST("/resource/bind", h.BindResource)
		treeGroup.POST("/resource/unbind", h.UnbindResource)
		treeGroup.GET("/resource/types", h.GetResourceTypes)

		// 成员管理接口
		treeGroup.GET("/members/:nodeId", h.GetNodeMembers)
		treeGroup.POST("/member/add", h.AddNodeMember)
		treeGroup.POST("/member/remove", h.RemoveNodeMember)
		treeGroup.POST("/admin/add", h.AddNodeAdmin)
		treeGroup.POST("/admin/remove", h.RemoveNodeAdmin)
	}
}

// 获取整个服务树结构
func (h *TreeHandler) GetTreeList(ctx *gin.Context) {
	var req model.TreeNodeListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetTreeList(ctx, &req)
	})
}

// 获取节点详情
func (h *TreeHandler) GetNodeDetail(ctx *gin.Context) {
	var req model.TreeNodeDetailReq
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetNodeDetail(ctx, id)
	})
}

// 获取子节点列表
func (h *TreeHandler) GetChildNodes(ctx *gin.Context) {
	var req model.TreeNodeListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetChildNodes(ctx, req.ParentID)
	})
}

// 获取节点路径
func (h *TreeHandler) GetNodePath(ctx *gin.Context) {
	var req model.TreeNodePathReq
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetNodePath(ctx, id)
	})
}

// 获取服务树统计信息
func (h *TreeHandler) GetTreeStatistics(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetTreeStatistics(ctx)
	})
}

// 创建节点
func (h *TreeHandler) CreateNode(ctx *gin.Context) {
	var req model.TreeNodeCreateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.CreateNode(ctx, &req)
	})
}

// 创建子节点
func (h *TreeHandler) CreateChildNode(ctx *gin.Context) {
	var req model.TreeNodeCreateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.CreateChildNode(ctx, req.ParentID, &req)
	})
}

// 更新节点
func (h *TreeHandler) UpdateNode(ctx *gin.Context) {
	var req model.TreeNodeUpdateReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateNode(ctx, &req)
	})
}

// 删除节点
func (h *TreeHandler) DeleteNode(ctx *gin.Context) {
	var req model.TreeNodeDeleteReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteNode(ctx, id)
	})
}

// 获取节点绑定的资源
func (h *TreeHandler) GetNodeResources(ctx *gin.Context) {
	var req model.TreeNodeResourceReq
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetNodeResources(ctx, id)
	})
}

// 绑定资源
func (h *TreeHandler) BindResource(ctx *gin.Context) {
	var req model.TreeNodeResourceBindReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.BindResource(ctx, &req)
	})
}

// 解绑资源
func (h *TreeHandler) UnbindResource(ctx *gin.Context) {
	var req model.TreeNodeResourceUnbindReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UnbindResource(ctx, &req)
	})
}

// 获取可绑定的资源类型
func (h *TreeHandler) GetResourceTypes(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetResourceTypes(ctx)
	})
}

// 获取节点成员
func (h *TreeHandler) GetNodeMembers(ctx *gin.Context) {
	var req model.TreeNodeMemberReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetNodeMembers(ctx, &req)
	})
}

// 添加成员
func (h *TreeHandler) AddNodeMember(ctx *gin.Context) {
	var req model.TreeNodeMemberReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.AddNodeMember(ctx, &req)
	})
}

// 移除成员
func (h *TreeHandler) RemoveNodeMember(ctx *gin.Context) {
	var req model.TreeNodeMemberReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.RemoveNodeMember(ctx, req.NodeID, req.UserID)
	})
}

// 添加管理员
func (h *TreeHandler) AddNodeAdmin(ctx *gin.Context) {
	var req model.TreeNodeMemberReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.AddNodeAdmin(ctx, req.NodeID, req.UserID)
	})
}

// 移除管理员
func (h *TreeHandler) RemoveNodeAdmin(ctx *gin.Context) {
	var req model.TreeNodeMemberReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.RemoveNodeAdmin(ctx, req.NodeID, req.UserID)
	})
}

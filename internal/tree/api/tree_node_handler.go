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
		treeGroup.POST("/create", h.CreateNode)
		treeGroup.PUT("/update/:id", h.UpdateNode)
		treeGroup.DELETE("/delete/:id", h.DeleteNode)
		treeGroup.PUT("/move/:id", h.MoveNode)

		// 成员管理接口
		treeGroup.GET("/members/:id", h.GetNodeMembers)
		treeGroup.POST("/member/add", h.AddNodeMember)
		treeGroup.DELETE("/member/remove/:id", h.RemoveNodeMember)

		// 资源绑定接口
		treeGroup.POST("/resource/bind", h.BindResource)
		treeGroup.POST("/resource/unbind", h.UnbindResource)
	}
}

// GetChildNodes 获取直接子节点列表
// @Summary 获取直接子节点列表
// @Tags 资源树管理
// @Param id path int true "父节点ID"
// @Success 200 {object} utils.ApiResponse{data=[]model.TreeNode}
// @Router /api/tree/node/children/{id} [get]
func (h *TreeNodeHandler) GetChildNodes(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的父节点ID")
		return
	}
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetChildNodes(ctx, id)
	})
}

// GetTreeStatistics 获取服务树统计信息
// @Summary 获取服务树统计信息
// @Tags 资源树管理
// @Success 200 {object} utils.ApiResponse{data=model.TreeNodeStatisticsResp}
// @Router /api/tree/node/statistics [get]
func (h *TreeNodeHandler) GetTreeStatistics(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetTreeStatistics(ctx)
	})
}

// GetTreeList 获取树节点列表
// @Summary 获取树节点列表
// @Description 获取完整的树结构节点列表
// @Tags 资源树管理
// @Accept json
// @Produce json
// @Param keyword query string false "搜索关键词（匹配名称/描述）"
// @Success 200 {object} utils.ApiResponse{data=[]model.TreeNode} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/node/list [get]
func (h *TreeNodeHandler) GetTreeList(ctx *gin.Context) {
	var req model.GetTreeNodeListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetTreeList(ctx, &req)
	})
}

// GetNodeDetail 获取节点详情
// @Summary 获取节点详情
// @Description 根据节点ID获取节点的详细信息
// @Tags 资源树管理
// @Accept json
// @Produce json
// @Param id path int true "节点ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/node/detail/{id} [get]
func (h *TreeNodeHandler) GetNodeDetail(ctx *gin.Context) {
	var req model.GetTreeNodeDetailReq
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

// CreateNode 创建节点
// @Summary 创建节点
// @Description 创建新的树节点
// @Tags 资源树管理
// @Accept json
// @Produce json
// @Param request body model.CreateTreeNodeReq true "创建节点请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/node/create [post]
func (h *TreeNodeHandler) CreateNode(ctx *gin.Context) {
	var req model.CreateTreeNodeReq

	user := ctx.MustGet("user").(utils.UserClaims)

	req.CreateUserID = user.Uid
	req.CreateUserName = user.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateNode(ctx, &req)
	})
}

// UpdateNode 更新节点
// @Summary 更新节点
// @Description 更新指定节点的信息
// @Tags 资源树管理
// @Accept json
// @Produce json
// @Param id path int true "节点ID"
// @Param request body model.UpdateTreeNodeReq true "更新节点请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/node/update/{id} [put]
func (h *TreeNodeHandler) UpdateNode(ctx *gin.Context) {
	var req model.UpdateTreeNodeReq

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

// DeleteNode 删除节点
// @Summary 删除节点
// @Description 删除指定的树节点
// @Tags 资源树管理
// @Accept json
// @Produce json
// @Param id path int true "节点ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/node/delete/{id} [delete]
func (h *TreeNodeHandler) DeleteNode(ctx *gin.Context) {
	var req model.DeleteTreeNodeReq

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
// @Summary 移动节点
// @Description 将节点移动到新的父节点下
// @Tags 资源树管理
// @Accept json
// @Produce json
// @Param id path int true "节点ID"
// @Param request body model.MoveTreeNodeReq true "移动节点请求参数"
// @Success 200 {object} utils.ApiResponse "移动成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/node/move/{id} [put]
func (h *TreeNodeHandler) MoveNode(ctx *gin.Context) {
	var req model.MoveTreeNodeReq

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
// @Summary 获取节点成员
// @Description 获取指定节点的成员列表
// @Tags 资源树管理
// @Accept json
// @Produce json
// @Param id path int true "节点ID"
// @Param type query int false "成员类型(1:admin,2:member,省略/其他:all)"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/node/members/{id} [get]
func (h *TreeNodeHandler) GetNodeMembers(ctx *gin.Context) {
	var req model.GetTreeNodeMembersReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		// 将数值型的成员类型映射为服务层使用的语义化字符串
		memberType := "all"
		switch req.Type {
		case model.AdminRole:
			memberType = "admin"
		case model.MemberRole:
			memberType = "member"
		default:
			memberType = "all"
		}
		return h.service.GetNodeMembers(ctx, req.ID, memberType)
	})
}

// AddNodeMember 添加节点成员
// @Summary 添加节点成员
// @Description 向指定节点添加成员
// @Tags 资源树管理
// @Accept json
// @Produce json
// @Param request body model.AddTreeNodeMemberReq true "添加节点成员请求参数"
// @Success 200 {object} utils.ApiResponse "添加成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/node/member/add [post]
func (h *TreeNodeHandler) AddNodeMember(ctx *gin.Context) {
	var req model.AddTreeNodeMemberReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.AddNodeMember(ctx, &req)
	})
}

// RemoveNodeMember 移除节点成员
// @Summary 移除节点成员
// @Description 从指定节点移除成员
// @Tags 资源树管理
// @Accept json
// @Produce json
// @Param id path int true "节点ID"
// @Param request body model.RemoveTreeNodeMemberReq true "移除节点成员请求参数"
// @Success 200 {object} utils.ApiResponse "移除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/node/member/remove/{id} [delete]
func (h *TreeNodeHandler) RemoveNodeMember(ctx *gin.Context) {
	var req model.RemoveTreeNodeMemberReq

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

// BindResource 绑定资源
// @Summary 绑定资源
// @Description 将资源绑定到指定节点
// @Tags 资源树管理
// @Accept json
// @Produce json
// @Param request body model.BindTreeNodeResourceReq true "绑定资源请求参数"
// @Success 200 {object} utils.ApiResponse "绑定成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/node/resource/bind [post]
func (h *TreeNodeHandler) BindResource(ctx *gin.Context) {
	var req model.BindTreeNodeResourceReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.BindResource(ctx, &req)
	})
}

// UnbindResource 解绑资源
// @Summary 解绑资源
// @Description 将资源从节点解绑
// @Tags 资源树管理
// @Accept json
// @Produce json
// @Param request body model.UnbindTreeNodeResourceReq true "解绑资源请求参数"
// @Success 200 {object} utils.ApiResponse "解绑成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/tree/node/resource/unbind [delete]
func (h *TreeNodeHandler) UnbindResource(ctx *gin.Context) {
	var req model.UnbindTreeNodeResourceReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UnbindResource(ctx, &req)
	})
}

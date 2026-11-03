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
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
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

		treeGroup.POST("/create", h.CreateNode)
		treeGroup.PUT("/update/:id", h.UpdateNode)
		treeGroup.DELETE("/delete/:id", h.DeleteNode)
		treeGroup.PUT("/move/:id", h.MoveNode)

		treeGroup.GET("/members/:id", h.GetNodeMembers)
		treeGroup.POST("/member/add", h.AddNodeMember)
		treeGroup.DELETE("/member/remove/:id", h.RemoveNodeMember)

		treeGroup.POST("/resource/bind", h.BindResource)
		treeGroup.POST("/resource/unbind", h.UnbindResource)
	}
}

func (h *TreeNodeHandler) GetChildNodes(ctx *gin.Context) {
	id, err := base.GetParentParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的父节点ID")
		return
	}
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetChildNodes(ctx.Request.Context(), id)
	})
}

func (h *TreeNodeHandler) GetTreeStatistics(ctx *gin.Context) {
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetTreeStatistics(ctx.Request.Context())
	})
}

func (h *TreeNodeHandler) GetTreeList(ctx *gin.Context) {
	var req model.GetTreeNodeListReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetTreeList(ctx.Request.Context(), &req)
	})
}

func (h *TreeNodeHandler) GetNodeDetail(ctx *gin.Context) {
	var req model.GetTreeNodeDetailReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}
	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetNodeDetail(ctx.Request.Context(), req.ID)
	})
}

func (h *TreeNodeHandler) CreateNode(ctx *gin.Context) {
	var req model.CreateTreeNodeReq

	user := ctx.MustGet("user").(jwt.UserClaims)

	req.CreateUserID = user.Uid
	req.CreateUserName = user.Username

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateNode(ctx.Request.Context(), &req)
	})
}

func (h *TreeNodeHandler) UpdateNode(ctx *gin.Context) {
	var req model.UpdateTreeNodeReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}
	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateNode(ctx.Request.Context(), &req)
	})
}

func (h *TreeNodeHandler) DeleteNode(ctx *gin.Context) {
	var req model.DeleteTreeNodeReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}
	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteNode(ctx.Request.Context(), req.ID)
	})
}

func (h *TreeNodeHandler) MoveNode(ctx *gin.Context) {
	var req model.MoveTreeNodeReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.BadRequestWithDetails(ctx, nil, "无效的节点ID")
		return
	}
	req.ID = id
	contentType := ctx.ContentType()
	if contentType == binding.MIMEPOSTForm || contentType == binding.MIMEMultipartPOSTForm || contentType == "" {
		// 保留解析错误，避免Gin缓存读取忽略坏编码后使后续绑定误接受。
		var parseErr error
		if contentType == binding.MIMEMultipartPOSTForm {
			_, parseErr = ctx.MultipartForm()
		} else {
			parseErr = ctx.Request.ParseForm()
		}
		if parseErr != nil {
			base.BadRequestWithDetails(ctx, "无效的表单参数", "绑定数据失败")
			return
		}
		values, present := ctx.GetPostFormArray("new_parent_id")
		if !present && contentType != binding.MIMEMultipartPOSTForm {
			values, present = ctx.GetQueryArray("new_parent_id")
		}
		// Gin会把空串转成0；只有明确提供的整数0才表示虚拟根。
		if !present || len(values) == 0 || strings.TrimSpace(values[0]) == "" {
			base.BadRequestWithDetails(ctx, "新父节点ID不能为空", "绑定数据失败")
			return
		}
	}

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.MoveNode(ctx.Request.Context(), req.ID, *req.NewParentID)
	})
}

func (h *TreeNodeHandler) GetNodeMembers(ctx *gin.Context) {
	var req model.GetTreeNodeMembersReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}
	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
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
		return h.service.GetNodeMembers(ctx.Request.Context(), req.ID, memberType)
	})
}

func (h *TreeNodeHandler) AddNodeMember(ctx *gin.Context) {
	var req model.AddTreeNodeMemberReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.AddNodeMember(ctx.Request.Context(), &req)
	})
}

func (h *TreeNodeHandler) RemoveNodeMember(ctx *gin.Context) {
	var req model.RemoveTreeNodeMemberReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}

	req.NodeID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.RemoveNodeMember(ctx.Request.Context(), &req)
	})
}

func (h *TreeNodeHandler) BindResource(ctx *gin.Context) {
	var req model.BindTreeNodeResourceReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.BindResource(ctx.Request.Context(), &req)
	})
}

func (h *TreeNodeHandler) UnbindResource(ctx *gin.Context) {
	var req model.UnbindTreeNodeResourceReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UnbindResource(ctx.Request.Context(), &req)
	})
}

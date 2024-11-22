package api

import (
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
)

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

type TreeNodeHandler struct {
	service service.TreeNodeService
}

func NewTreeNodeHandler(service service.TreeNodeService) *TreeNodeHandler {
	return &TreeNodeHandler{
		service: service,
	}
}

func (t *TreeNodeHandler) RegisterRouters(server *gin.Engine) {
	treeNodeGroup := server.Group("/api/tree/node")

	// 树节点相关路由
	treeNodeGroup.GET("/listTreeNode", t.ListTreeNode)
	treeNodeGroup.GET("/selectTreeNode", t.SelectTreeNode)
	treeNodeGroup.GET("/getTopTreeNode", t.GetTopTreeNode)
	treeNodeGroup.GET("/listLeafTreeNode", t.ListLeafTreeNodes)
	treeNodeGroup.POST("/createTreeNode", t.CreateTreeNode)
	treeNodeGroup.DELETE("/deleteTreeNode/:id", t.DeleteTreeNode)
	treeNodeGroup.GET("/getChildrenTreeNode/:pid", t.GetChildrenTreeNode)
	treeNodeGroup.POST("/updateTreeNode", t.UpdateTreeNode)
}

func (t *TreeNodeHandler) ListTreeNode(ctx *gin.Context) {
	list, err := t.service.ListTreeNodes(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "获取树节点列表失败: "+err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

func (t *TreeNodeHandler) SelectTreeNode(ctx *gin.Context) {
	levelStr := ctx.DefaultQuery("level", "0")
	levelLtStr := ctx.DefaultQuery("levelLt", "0")

	level, err := strconv.Atoi(levelStr)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "level参数必须为有效的整数")
		return
	}

	levelLt, err := strconv.Atoi(levelLtStr)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "levelLt参数必须为有效的整数")
		return
	}

	nodes, err := t.service.SelectTreeNode(ctx, level, levelLt)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "查询指定层级的树节点失败: "+err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, nodes)
}

func (t *TreeNodeHandler) GetTopTreeNode(ctx *gin.Context) {
	nodes, err := t.service.GetTopTreeNode(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "获取顶层树节点失败: "+err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, nodes)
}

func (t *TreeNodeHandler) ListLeafTreeNodes(ctx *gin.Context) {
	list, err := t.service.ListLeafTreeNodes(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "获取叶子节点列表失败: "+err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

func (t *TreeNodeHandler) CreateTreeNode(ctx *gin.Context) {
	var req model.TreeNode

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "请求参数格式错误,请检查输入的树节点数据是否完整")
		return
	}

	if req.Title == "" {
		apiresponse.BadRequestWithDetails(ctx, "节点名称不能为空", "请提供有效的节点名称")
		return
	}

	if err := t.service.CreateTreeNode(ctx, &req); err != nil {
		apiresponse.ErrorWithMessage(ctx, "创建树节点失败: "+err.Error())
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeNodeHandler) DeleteTreeNode(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		apiresponse.BadRequestWithDetails(ctx, "节点ID不能为空", "请提供要删除的树节点ID")
		return
	}

	nodeId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "节点ID必须为有效的整数")
		return
	}

	if err := t.service.DeleteTreeNode(ctx, nodeId); err != nil {
		apiresponse.ErrorWithMessage(ctx, "删除树节点失败: "+err.Error())
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeNodeHandler) GetChildrenTreeNode(ctx *gin.Context) {
	pid := ctx.Param("pid")
	if pid == "" {
		apiresponse.BadRequestWithDetails(ctx, "父节点ID不能为空", "请提供有效的父节点ID")
		return
	}

	parentId, err := strconv.Atoi(pid)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "父节点ID必须为有效的整数")
		return
	}

	list, err := t.service.GetChildrenTreeNodes(ctx, parentId)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "获取子节点列表失败: "+err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

func (t *TreeNodeHandler) UpdateTreeNode(ctx *gin.Context) {
	var req model.TreeNode
	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "请求参数格式错误,请检查更新的树节点数据是否完整")
		return
	}

	if req.ID == 0 {
		apiresponse.BadRequestWithDetails(ctx, "节点ID不能为空", "请提供要更新的节点ID")
		return
	}

	if req.Title == "" {
		apiresponse.BadRequestWithDetails(ctx, "节点名称不能为空", "请提供有效的节点名称")
		return
	}

	if err := t.service.UpdateTreeNode(ctx, &req); err != nil {
		apiresponse.ErrorWithMessage(ctx, "更新树节点失败: "+err.Error())
		return
	}

	apiresponse.Success(ctx)
}

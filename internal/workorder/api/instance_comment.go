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
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type InstanceCommentHandler struct {
	commentService service.InstanceCommentService
}

func NewInstanceCommentHandler(commentService service.InstanceCommentService) *InstanceCommentHandler {
	return &InstanceCommentHandler{
		commentService: commentService,
	}
}

func (h *InstanceCommentHandler) RegisterRouters(server *gin.Engine) {
	commentGroup := server.Group("/api/workorder/instance/comment")
	{
		commentGroup.POST("/create", h.CreateInstanceComment)
		commentGroup.PUT("/update/:id", h.UpdateInstanceComment)
		commentGroup.DELETE("/delete/:id", h.DeleteInstanceComment)
		commentGroup.GET("/detail/:id", h.GetInstanceComment)
		commentGroup.GET("/list", h.ListInstanceComments)
		commentGroup.GET("/tree/:id", h.GetInstanceCommentsTree)
	}
}

// CreateInstanceComment 创建工单评论
// CreateInstanceComment 创建工单评论
func (h *InstanceCommentHandler) CreateInstanceComment(ctx *gin.Context) {
	var req model.CreateWorkorderInstanceCommentReq
	user := ctx.MustGet("user").(utils.UserClaims)

	req.OperatorID = user.Uid
	req.OperatorName = user.Username

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.commentService.CreateInstanceComment(ctx, &req)
	})
}

// UpdateInstanceComment 更新工单评论
// UpdateInstanceComment 更新工单评论
func (h *InstanceCommentHandler) UpdateInstanceComment(ctx *gin.Context) {
	var req model.UpdateWorkorderInstanceCommentReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.commentService.UpdateInstanceComment(ctx, &req, user.Uid)
	})
}

// DeleteInstanceComment 删除工单评论
// DeleteInstanceComment 删除工单评论
func (h *InstanceCommentHandler) DeleteInstanceComment(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.commentService.DeleteInstanceComment(ctx, id, user.Uid)
	})
}

// GetInstanceComment 获取工单评论详情
// GetInstanceComment 获取工单评论详情
func (h *InstanceCommentHandler) GetInstanceComment(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	utils.HandleRequest(ctx, nil, func() (any, error) {
		return h.commentService.GetInstanceComment(ctx, id)
	})
}

// ListInstanceComments 获取工单评论列表
// ListInstanceComments 获取工单评论列表
func (h *InstanceCommentHandler) ListInstanceComments(ctx *gin.Context) {
	var req model.ListWorkorderInstanceCommentReq

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.commentService.ListInstanceComments(ctx, &req)
	})
}

// GetInstanceCommentsTree 获取工单评论树结构
// GetInstanceCommentsTree 获取工单评论树结构
func (h *InstanceCommentHandler) GetInstanceCommentsTree(ctx *gin.Context) {
	var req model.GetInstanceCommentsTreeReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.commentService.GetInstanceCommentsTree(ctx, req.ID)
	})
}

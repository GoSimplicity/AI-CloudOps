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
	"strconv"

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
		commentGroup.GET("/tree/:instanceId", h.GetInstanceCommentsTree)
	}
}

// CreateInstanceComment 创建工单评论
// @Summary 创建工单评论
// @Description 为指定工单实例创建新的评论
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body model.CreateWorkorderInstanceCommentReq true "创建评论请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/comment/create [post]
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
// @Summary 更新工单评论
// @Description 更新指定的工单评论内容
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "评论 ID"
// @Param request body model.UpdateWorkorderInstanceCommentReq true "更新评论请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/comment/update/{id} [put]
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
// @Summary 删除工单评论
// @Description 删除指定的工单评论
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "评论 ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/comment/delete/{id} [delete]
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
// @Summary 获取工单评论详情
// @Description 获取指定工单评论的详细信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "评论 ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/comment/detail/{id} [get]
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
// @Summary 获取工单评论列表
// @Description 分页获取工单评论列表
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Param instanceId query int false "工单实例ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/comment/list [get]
// ListInstanceComments 获取工单评论列表
func (h *InstanceCommentHandler) ListInstanceComments(ctx *gin.Context) {
	var req model.ListWorkorderInstanceCommentReq

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.commentService.ListInstanceComments(ctx, &req)
	})
}

// GetInstanceCommentsTree 获取工单评论树结构
// @Summary 获取工单评论树结构
// @Description 获取指定工单实例的评论树结构，包含父子关系
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param instanceId path int true "工单实例ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/comment/tree/{instanceId} [get]
// GetInstanceCommentsTree 获取工单评论树结构
func (h *InstanceCommentHandler) GetInstanceCommentsTree(ctx *gin.Context) {
	instanceIdStr := ctx.Param("instanceId")
	instanceId, err := strconv.Atoi(instanceIdStr)
	if err != nil {
		utils.ErrorWithMessage(ctx, "实例ID格式无效")
		return
	}

	utils.HandleRequest(ctx, nil, func() (any, error) {
		return h.commentService.GetInstanceCommentsTree(ctx, instanceId)
	})
}

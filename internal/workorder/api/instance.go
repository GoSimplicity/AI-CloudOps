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

type InstanceHandler struct {
	service           service.InstanceService
	flowService       service.InstanceFlowService
	commentService    service.InstanceCommentService
	attachmentService service.InstanceAttachmentService
}

func NewInstanceHandler(
	service service.InstanceService,
	flowService service.InstanceFlowService,
	commentService service.InstanceCommentService,
	attachmentService service.InstanceAttachmentService,
) *InstanceHandler {
	return &InstanceHandler{
		service:           service,
		flowService:       flowService,
		commentService:    commentService,
		attachmentService: attachmentService,
	}
}

func (h *InstanceHandler) RegisterRouters(server *gin.Engine) {
	instanceGroup := server.Group("/api/workorder/instance")
	{
		// 基础CRUD操作
		instanceGroup.POST("/create", h.CreateInstance)
		instanceGroup.PUT("/update/:id", h.UpdateInstance)
		instanceGroup.DELETE("/delete/:id", h.DeleteInstance)
		instanceGroup.GET("/list", h.ListInstance)
		instanceGroup.GET("/detail/:id", h.DetailInstance)
		instanceGroup.POST("/transfer/:id", h.TransferInstance)

		// 用户相关
		instanceGroup.GET("/my", h.GetMyInstances)
		instanceGroup.GET("/overdue", h.GetOverdueInstances)

		// 流程操作
		instanceGroup.POST("/action/:id", h.ProcessInstanceFlow)

		// 评论功能
		instanceGroup.POST("/comment/:id", h.CommentInstance)
		instanceGroup.GET("/comments/:id", h.GetInstanceComments)

		// TODO: 附件功能暂不支持
		// // 附件功能
		// instanceGroup.POST("/attachment/:id", h.UploadAttachment)
		// instanceGroup.DELETE("/:id/attachment/:aid", h.DeleteAttachment)
		// instanceGroup.GET("/attachments/:id", h.GetInstanceAttachments)
		// instanceGroup.DELETE("/attachments/batch/:id", h.BatchDeleteAttachments)

		// 流程查看
		instanceGroup.GET("/flows/:id", h.GetInstanceFlows)
		instanceGroup.GET("/process/:pid/definition", h.GetProcessDefinition)
	}
}

// CreateInstance 创建工单实例
func (h *InstanceHandler) CreateInstance(ctx *gin.Context) {
	var req model.CreateInstanceReq
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.CreateInstance(ctx, &req, user.Uid, user.Username)
	})
}

// UpdateInstance 更新工单实例
func (h *InstanceHandler) UpdateInstance(ctx *gin.Context) {
	var req model.UpdateInstanceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateInstance(ctx, &req, user.Uid)
	})
}

// DeleteInstance 删除工单实例
func (h *InstanceHandler) DeleteInstance(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.DeleteInstance(ctx, id, user.Uid)
	})
}

// DetailInstance 获取工单实例详情
func (h *InstanceHandler) DetailInstance(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetInstance(ctx, id)
	})
}

// ListInstance 获取工单实例列表
func (h *InstanceHandler) ListInstance(ctx *gin.Context) {
	var req model.ListInstanceReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListInstance(ctx, &req)
	})
}

// ProcessInstanceFlow 处理工单流程
func (h *InstanceHandler) ProcessInstanceFlow(ctx *gin.Context) {
	var req model.InstanceActionReq

	user := ctx.MustGet("user").(utils.UserClaims)
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.InstanceID = id
		return nil, h.flowService.ProcessInstanceFlow(ctx, &req, user.Uid, user.Username)
	})
}

// GetInstanceFlows 获取工单流程记录
func (h *InstanceHandler) GetInstanceFlows(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.flowService.GetInstanceFlows(ctx, id)
	})
}

// GetProcessDefinition 获取流程定义
func (h *InstanceHandler) GetProcessDefinition(ctx *gin.Context) {
	processIDStr := ctx.Param("pid")
	processID, err := strconv.Atoi(processIDStr)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的流程ID")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.flowService.GetProcessDefinition(ctx, processID)
	})
}

// CommentInstance 添加工单评论
func (h *InstanceHandler) CommentInstance(ctx *gin.Context) {
	var req model.InstanceCommentReq
	user := ctx.MustGet("user").(utils.UserClaims)
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.InstanceID = id
		return nil, h.commentService.CommentInstance(ctx, &req, user.Uid, user.Username)
	})
}

// GetInstanceComments 获取工单评论
func (h *InstanceHandler) GetInstanceComments(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.commentService.GetInstanceComments(ctx, id)
	})
}

// UploadAttachment 上传附件
func (h *InstanceHandler) UploadAttachment(ctx *gin.Context) {
	user := ctx.MustGet("user").(utils.UserClaims)
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		utils.ErrorWithMessage(ctx, "文件上传失败")
		return
	}
	defer file.Close()

	//TODO: 这里需要实现文件保存逻辑，返回文件路径
	filePath := "/uploads/" + header.Filename
	fileType := header.Header.Get("Content-Type")

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.attachmentService.UploadAttachment(ctx, id, header.Filename, header.Size, filePath, fileType, user.Uid, user.Username)
	})
}

// DeleteAttachment 删除附件
func (h *InstanceHandler) DeleteAttachment(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	attachmentIDStr := ctx.Param("aid")
	attachmentID, err := strconv.Atoi(attachmentIDStr)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的附件ID")
		return
	}

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.attachmentService.DeleteAttachment(ctx, id, attachmentID, user.Uid)
	})
}

// GetInstanceAttachments 获取工单附件列表
func (h *InstanceHandler) GetInstanceAttachments(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.attachmentService.GetInstanceAttachments(ctx, id)
	})
}

// BatchDeleteAttachments 批量删除附件
func (h *InstanceHandler) BatchDeleteAttachments(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	var req struct {
		AttachmentIDs []int `json:"attachment_ids" binding:"required"`
	}

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.attachmentService.BatchDeleteAttachments(ctx, id, req.AttachmentIDs, user.Uid)
	})
}

// GetMyInstances 获取我的工单
func (h *InstanceHandler) GetMyInstances(ctx *gin.Context) {
	var req model.MyInstanceReq
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetMyInstances(ctx, user.Uid, &req)
	})
}

// GetOverdueInstances 获取超时工单
func (h *InstanceHandler) GetOverdueInstances(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetOverdueInstances(ctx)
	})
}

// TransferInstance 转移工单
func (h *InstanceHandler) TransferInstance(ctx *gin.Context) {
	var req model.TransferInstanceReq

	user := ctx.MustGet("user").(utils.UserClaims)
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.TransferInstance(ctx, id, user.Uid, req.AssigneeID, req.Comment)
	})
}

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
	service service.InstanceService
}

func NewInstanceHandler(service service.InstanceService) *InstanceHandler {
	return &InstanceHandler{
		service: service,
	}
}

func (h *InstanceHandler) RegisterRouters(server *gin.Engine) {
	instanceGroup := server.Group("/api/workorder/instance")
	{
		// 基础CRUD操作
		instanceGroup.POST("/", h.CreateInstance)
		instanceGroup.PUT("/:id", h.UpdateInstance)
		instanceGroup.DELETE("/:id", h.DeleteInstance)
		instanceGroup.GET("/", h.ListInstance)
		instanceGroup.GET("/:id", h.GetInstance)
		instanceGroup.PATCH("/batch/status", h.BatchUpdateInstanceStatus)

		// 用户相关
		instanceGroup.GET("/my", h.GetMyInstances)
		instanceGroup.GET("/overdue", h.GetOverdueInstances)

		// 流程操作
		instanceGroup.POST("/:id/action", h.ProcessInstanceFlow)
		instanceGroup.POST("/:id/transfer", h.TransferInstance)

		// 评论功能
		instanceGroup.POST("/:id/comment", h.CommentInstance)
		instanceGroup.GET("/:id/comments", h.GetInstanceComments)

		// 附件功能
		instanceGroup.POST("/:id/attachment", h.UploadAttachment)
		instanceGroup.DELETE("/:id/attachment/:aid", h.DeleteAttachment)
		instanceGroup.GET("/:id/attachments", h.GetInstanceAttachments)
		instanceGroup.DELETE("/:id/attachments/batch", h.BatchDeleteAttachments)

		// 流程查看
		instanceGroup.GET("/:id/flows", h.GetInstanceFlows)
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

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.ID = id
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

// GetInstance 获取工单实例详情
func (h *InstanceHandler) GetInstance(ctx *gin.Context) {
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

// BatchUpdateInstanceStatus 批量更新工单状态
func (h *InstanceHandler) BatchUpdateInstanceStatus(ctx *gin.Context) {
	var req struct {
		IDs    []int `json:"ids" binding:"required"`
		Status int8  `json:"status" binding:"required"`
	}

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.BatchUpdateInstanceStatus(ctx, req.IDs, req.Status, user.Uid)
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
		return nil, h.service.ProcessInstanceFlow(ctx, &req, user.Uid, user.Username)
	})
}

// GetInstanceFlows 获取工单流程记录
func (h *InstanceHandler) GetInstanceFlows(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetInstanceFlows(ctx, id)
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
		return h.service.GetProcessDefinition(ctx, processID)
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
		return nil, h.service.CommentInstance(ctx, &req, user.Uid, user.Username)
	})
}

// GetInstanceComments 获取工单评论
func (h *InstanceHandler) GetInstanceComments(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetInstanceComments(ctx, id)
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

	// 这里需要实现文件保存逻辑，返回文件路径
	// 示例代码，实际应该根据具体的文件存储服务实现
	filePath := "/uploads/" + header.Filename // 简化示例
	fileType := header.Header.Get("Content-Type")

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.UploadAttachment(ctx, id, header.Filename, header.Size, filePath, fileType, user.Uid, user.Username)
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
		return nil, h.service.DeleteAttachment(ctx, id, attachmentID, user.Uid)
	})
}

// GetInstanceAttachments 获取工单附件列表
func (h *InstanceHandler) GetInstanceAttachments(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetInstanceAttachments(ctx, id)
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
		return nil, h.service.BatchDeleteAttachments(ctx, id, req.AttachmentIDs, user.Uid)
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
	var req struct {
		ToUserID int    `json:"to_user_id" binding:"required"`
		Comment  string `json:"comment"`
	}
	user := ctx.MustGet("user").(utils.UserClaims)
	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.TransferInstance(ctx, id, user.Uid, req.ToUserID, req.Comment)
	})
}

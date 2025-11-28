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
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
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
		instanceGroup.POST("/create", h.CreateInstance)
		instanceGroup.POST("/create-from-template/:id", h.CreateInstanceFromTemplate)
		instanceGroup.PUT("/update/:id", h.UpdateInstance)
		instanceGroup.DELETE("/delete/:id", h.DeleteInstance)
		instanceGroup.GET("/list", h.ListInstance)
		instanceGroup.GET("/detail/:id", h.DetailInstance)
		instanceGroup.POST("/submit/:id", h.SubmitInstance)
		instanceGroup.POST("/assign/:id", h.AssignInstance)
		instanceGroup.POST("/approve/:id", h.ApproveInstance)
		instanceGroup.POST("/reject/:id", h.RejectInstance)
		instanceGroup.POST("/cancel/:id", h.CancelInstance)
		instanceGroup.POST("/complete/:id", h.CompleteInstance)
		instanceGroup.POST("/return/:id", h.ReturnInstance)
		instanceGroup.GET("/actions/:id", h.GetAvailableActions)
		instanceGroup.GET("/current-step/:id", h.GetCurrentStep)
	}
}

// CreateInstance 创建工单实例
func (h *InstanceHandler) CreateInstance(ctx *gin.Context) {
	var req model.CreateWorkorderInstanceReq
	user := ctx.MustGet("user").(jwt.UserClaims)

	req.OperatorID = user.Uid
	req.OperatorName = user.Username

	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.CreateInstance(ctx, &req)
	})
}

// CreateInstanceFromTemplate 从模板创建工单实例
func (h *InstanceHandler) CreateInstanceFromTemplate(ctx *gin.Context) {
	var req model.CreateWorkorderInstanceFromTemplateReq
	user := ctx.MustGet("user").(jwt.UserClaims)

	templateID, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的模板ID")
		return
	}

	req.OperatorID = user.Uid
	req.OperatorName = user.Username

	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.CreateInstanceFromTemplate(ctx, templateID, &req)
	})
}

// UpdateInstance 更新工单实例
func (h *InstanceHandler) UpdateInstance(ctx *gin.Context) {
	var req model.UpdateWorkorderInstanceReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.UpdateInstance(ctx, &req)
	})
}

// DeleteInstance 删除工单实例
func (h *InstanceHandler) DeleteInstance(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.service.DeleteInstance(ctx, id)
	})
}

// DetailInstance 获取工单实例详情
func (h *InstanceHandler) DetailInstance(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.service.GetInstance(ctx, id)
	})
}

// ListInstance 获取工单实例列表
func (h *InstanceHandler) ListInstance(ctx *gin.Context) {
	var req model.ListWorkorderInstanceReq

	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.service.ListInstance(ctx, &req)
	})
}

// SubmitInstance 提交工单
func (h *InstanceHandler) SubmitInstance(ctx *gin.Context) {
	var req model.SubmitWorkorderInstanceReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(jwt.UserClaims)

	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.SubmitInstance(ctx, req.ID, user.Uid, user.Username)
	})
}

// AssignInstance 指派工单
func (h *InstanceHandler) AssignInstance(ctx *gin.Context) {
	var req model.AssignWorkorderInstanceReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(jwt.UserClaims)

	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.AssignInstance(ctx, req.ID, req.AssigneeID, user.Uid, user.Username)
	})
}

// ApproveInstance 审批通过工单
func (h *InstanceHandler) ApproveInstance(ctx *gin.Context) {
	var req model.ApproveWorkorderInstanceReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(jwt.UserClaims)

	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.ApproveInstance(ctx, req.ID, user.Uid, user.Username, req.Comment)
	})
}

// RejectInstance 拒绝工单
func (h *InstanceHandler) RejectInstance(ctx *gin.Context) {
	var req model.RejectWorkorderInstanceReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(jwt.UserClaims)

	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.RejectInstance(ctx, req.ID, user.Uid, user.Username, req.Comment)
	})
}

// CancelInstance 取消工单
func (h *InstanceHandler) CancelInstance(ctx *gin.Context) {
	var req model.CancelWorkorderInstanceReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(jwt.UserClaims)

	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.CancelInstance(ctx, req.ID, user.Uid, user.Username, req.Comment)
	})
}

// CompleteInstance 完成工单
func (h *InstanceHandler) CompleteInstance(ctx *gin.Context) {
	var req model.CompleteWorkorderInstanceReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(jwt.UserClaims)

	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.CompleteInstance(ctx, req.ID, user.Uid, user.Username, req.Comment)
	})
}

// ReturnInstance 退回工单
func (h *InstanceHandler) ReturnInstance(ctx *gin.Context) {
	var req model.ReturnWorkorderInstanceReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(jwt.UserClaims)

	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.ReturnInstance(ctx, req.ID, user.Uid, user.Username, req.Comment)
	})
}

// GetAvailableActions 获取可执行动作
func (h *InstanceHandler) GetAvailableActions(ctx *gin.Context) {
	var req model.GetAvailableActionsReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id

	user := ctx.MustGet("user").(jwt.UserClaims)

	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.service.GetAvailableActions(ctx, req.ID, user.Uid)
	})
}

// GetCurrentStep 获取当前步骤
func (h *InstanceHandler) GetCurrentStep(ctx *gin.Context) {
	var req model.GetCurrentStepReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.service.GetCurrentStep(ctx, req.ID)
	})
}

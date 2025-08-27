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
// @Summary 创建工单实例
// @Description 创建新的工单实例
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body model.CreateWorkorderInstanceReq true "创建工单实例请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/create [post]
func (h *InstanceHandler) CreateInstance(ctx *gin.Context) {
	var req model.CreateWorkorderInstanceReq
	user := ctx.MustGet("user").(utils.UserClaims)

	req.OperatorID = user.Uid
	req.OperatorName = user.Username

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.CreateInstance(ctx, &req)
	})
}

// CreateInstanceFromTemplate 从模板创建工单实例
// @Summary 从模板创建工单实例
// @Description 基于工单模板创建新的工单实例
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param templateId path int true "模板ID"
// @Param request body model.CreateWorkorderInstanceFromTemplateReq true "创建工单实例请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/create-from-template/{templateId} [post]
func (h *InstanceHandler) CreateInstanceFromTemplate(ctx *gin.Context) {
	var req model.CreateWorkorderInstanceFromTemplateReq
	user := ctx.MustGet("user").(utils.UserClaims)

	templateID, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的模板ID")
		return
	}

	req.OperatorID = user.Uid
	req.OperatorName = user.Username

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.CreateInstanceFromTemplate(ctx, templateID, &req)
	})
}

// UpdateInstance 更新工单实例
// @Summary 更新工单实例
// @Description 更新指定工单实例的信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "工单实例ID"
// @Param request body model.UpdateWorkorderInstanceReq true "更新工单实例请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/update/{id} [put]
func (h *InstanceHandler) UpdateInstance(ctx *gin.Context) {
	var req model.UpdateWorkorderInstanceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.UpdateInstance(ctx, &req)
	})
}

// DeleteInstance 删除工单实例
// @Summary 删除工单实例
// @Description 删除指定的工单实例
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "工单实例ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/delete/{id} [delete]
func (h *InstanceHandler) DeleteInstance(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	utils.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.service.DeleteInstance(ctx, id)
	})
}

// DetailInstance 获取工单实例详情
// @Summary 获取工单实例详情
// @Description 根据ID获取工单实例的详细信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "工单实例ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/detail/{id} [get]
func (h *InstanceHandler) DetailInstance(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	utils.HandleRequest(ctx, nil, func() (any, error) {
		return h.service.GetInstance(ctx, id)
	})
}

// ListInstance 获取工单实例列表
// @Summary 获取工单实例列表
// @Description 分页获取工单实例列表
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param status query string false "工单状态"
// @Param keyword query string false "搜索关键词"
// @Success 200 {object} utils.ApiResponse{data=[]model.WorkorderInstance} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/list [get]
func (h *InstanceHandler) ListInstance(ctx *gin.Context) {
	var req model.ListWorkorderInstanceReq

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return h.service.ListInstance(ctx, &req)
	})
}

// SubmitInstance 提交工单
// @Summary 提交工单
// @Description 将工单实例提交审批
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "工单实例ID"
// @Param request body model.SubmitWorkorderInstanceReq true "提交工单请求参数"
// @Success 200 {object} utils.ApiResponse "提交成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/submit/{id} [post]
func (h *InstanceHandler) SubmitInstance(ctx *gin.Context) {
	var req model.SubmitWorkorderInstanceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.SubmitInstance(ctx, req.ID, user.Uid, user.Username)
	})
}

// AssignInstance 指派工单
// @Summary 指派工单
// @Description 将工单实例指派给指定处理人
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "工单实例ID"
// @Param request body model.AssignWorkorderInstanceReq true "指派请求参数"
// @Success 200 {object} utils.ApiResponse "指派成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/assign/{id} [post]
func (h *InstanceHandler) AssignInstance(ctx *gin.Context) {
	var req model.AssignWorkorderInstanceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.AssignInstance(ctx, req.ID, req.AssigneeID, user.Uid, user.Username)
	})
}

// ApproveInstance 审批通过工单
// @Summary 审批通过工单
// @Description 审批通过指定的工单实例
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "工单实例ID"
// @Param request body model.ApproveWorkorderInstanceReq true "审批意见"
// @Success 200 {object} utils.ApiResponse "审批成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/approve/{id} [post]
func (h *InstanceHandler) ApproveInstance(ctx *gin.Context) {
	var req model.ApproveWorkorderInstanceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.ApproveInstance(ctx, req.ID, user.Uid, user.Username, req.Comment)
	})
}

// RejectInstance 拒绝工单
// @Summary 拒绝工单
// @Description 拒绝指定的工单实例
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "工单实例ID"
// @Param request body model.RejectWorkorderInstanceReq true "拒绝原因"
// @Success 200 {object} utils.ApiResponse "拒绝成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/reject/{id} [post]
func (h *InstanceHandler) RejectInstance(ctx *gin.Context) {
	var req model.RejectWorkorderInstanceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.RejectInstance(ctx, req.ID, user.Uid, user.Username, req.Comment)
	})
}

// CancelInstance 取消工单
// @Summary 取消工单
// @Description 取消指定的工单实例
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "工单实例ID"
// @Param request body model.CancelWorkorderInstanceReq true "取消原因"
// @Success 200 {object} utils.ApiResponse "取消成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/cancel/{id} [post]
func (h *InstanceHandler) CancelInstance(ctx *gin.Context) {
	var req model.CancelWorkorderInstanceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.CancelInstance(ctx, req.ID, user.Uid, user.Username, req.Comment)
	})
}

// CompleteInstance 完成工单
// @Summary 完成工单
// @Description 完成指定的工单实例
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "工单实例ID"
// @Param request body model.CompleteWorkorderInstanceReq true "完成说明"
// @Success 200 {object} utils.ApiResponse "完成成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/complete/{id} [post]
func (h *InstanceHandler) CompleteInstance(ctx *gin.Context) {
	var req model.CompleteWorkorderInstanceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.CompleteInstance(ctx, req.ID, user.Uid, user.Username, req.Comment)
	})
}

// ReturnInstance 退回工单
// @Summary 退回工单
// @Description 退回指定的工单实例
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "工单实例ID"
// @Param request body model.ReturnWorkorderInstanceReq true "退回原因"
// @Success 200 {object} utils.ApiResponse "退回成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/return/{id} [post]
func (h *InstanceHandler) ReturnInstance(ctx *gin.Context) {
	var req model.ReturnWorkorderInstanceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.service.ReturnInstance(ctx, req.ID, user.Uid, user.Username, req.Comment)
	})
}

// GetAvailableActions 获取可执行动作
// @Summary 获取可执行动作
// @Description 获取当前工单实例可执行的动作列表
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "工单实例ID"
// @Success 200 {object} utils.ApiResponse{data=[]string} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/actions/{id} [get]
func (h *InstanceHandler) GetAvailableActions(ctx *gin.Context) {
	var req model.GetAvailableActionsReq
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id

	user := ctx.MustGet("user").(utils.UserClaims)

	utils.HandleRequest(ctx, nil, func() (any, error) {
		return h.service.GetAvailableActions(ctx, req.ID, user.Uid)
	})
}

// GetCurrentStep 获取当前步骤
// @Summary 获取当前步骤
// @Description 获取工单实例当前流程步骤信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path int true "工单实例ID"
// @Success 200 {object} utils.ApiResponse{data=model.ProcessStep} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/workorder/instance/current-step/{id} [get]
func (h *InstanceHandler) GetCurrentStep(ctx *gin.Context) {
	var req model.GetCurrentStepReq
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的工单ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, nil, func() (any, error) {
		return h.service.GetCurrentStep(ctx, req.ID)
	})
}

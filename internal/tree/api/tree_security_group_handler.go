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

type TreeSecurityGroupHandler struct {
	securityGroupService service.TreeSecurityGroupService
}

func NewTreeSecurityGroupHandler(securityGroupService service.TreeSecurityGroupService) *TreeSecurityGroupHandler {
	return &TreeSecurityGroupHandler{
		securityGroupService: securityGroupService,
	}
}

func (h *TreeSecurityGroupHandler) RegisterRouters(server *gin.Engine) {
	securityGroupGroup := server.Group("/api/tree/security_group")
	{
		securityGroupGroup.POST("/create", h.CreateSecurityGroup)
		securityGroupGroup.DELETE("/delete/:id", h.DeleteSecurityGroup)
		securityGroupGroup.POST("/list", h.ListSecurityGroups)
		securityGroupGroup.POST("/detail/:id", h.GetSecurityGroupDetail)
		securityGroupGroup.POST("/update/:id", h.UpdateSecurityGroup)
		securityGroupGroup.POST("/add_rule/:id", h.AddSecurityGroupRule)
		securityGroupGroup.DELETE("/remove_rule/:id", h.RemoveSecurityGroupRule)
		securityGroupGroup.POST("/bind_instance/:id", h.BindInstanceToSecurityGroup)
		securityGroupGroup.POST("/unbind_instance/:id", h.UnbindInstanceFromSecurityGroup)
	}
}

// CreateSecurityGroup 创建安全组
func (h *TreeSecurityGroupHandler) CreateSecurityGroup(ctx *gin.Context) {
	var req model.CreateSecurityGroupReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.securityGroupService.CreateSecurityGroup(ctx, &req)
	})
}

// DeleteSecurityGroup 删除安全组
func (h *TreeSecurityGroupHandler) DeleteSecurityGroup(ctx *gin.Context) {
	var req model.DeleteSecurityGroupReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的安全组ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.securityGroupService.DeleteSecurityGroup(ctx, &req)
	})
}

// ListSecurityGroups 获取安全组列表
func (h *TreeSecurityGroupHandler) ListSecurityGroups(ctx *gin.Context) {
	var req model.ListSecurityGroupsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.securityGroupService.ListSecurityGroups(ctx, &req)
	})
}

// GetSecurityGroupDetail 获取安全组详情
func (h *TreeSecurityGroupHandler) GetSecurityGroupDetail(ctx *gin.Context) {
	var req model.GetSecurityGroupDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的安全组ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.securityGroupService.GetSecurityGroupDetail(ctx, &req)
	})
}

// UpdateSecurityGroup 更新安全组
func (h *TreeSecurityGroupHandler) UpdateSecurityGroup(ctx *gin.Context) {
	var req model.UpdateSecurityGroupReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的安全组ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.securityGroupService.UpdateSecurityGroup(ctx, &req)
	})
}

// AddSecurityGroupRule 添加安全组规则
func (h *TreeSecurityGroupHandler) AddSecurityGroupRule(ctx *gin.Context) {
	var req model.AddSecurityGroupRuleReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的安全组ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.securityGroupService.AddSecurityGroupRule(ctx, &req)
	})
}

// RemoveSecurityGroupRule 删除安全组规则
func (h *TreeSecurityGroupHandler) RemoveSecurityGroupRule(ctx *gin.Context) {
	var req model.RemoveSecurityGroupRuleReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的安全组ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.securityGroupService.RemoveSecurityGroupRule(ctx, &req)
	})
}

// BindInstanceToSecurityGroup 绑定实例到安全组
func (h *TreeSecurityGroupHandler) BindInstanceToSecurityGroup(ctx *gin.Context) {
	var req model.BindInstanceToSecurityGroupReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的安全组ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.securityGroupService.BindInstanceToSecurityGroup(ctx, &req)
	})
}

// UnbindInstanceFromSecurityGroup 解绑实例从安全组
func (h *TreeSecurityGroupHandler) UnbindInstanceFromSecurityGroup(ctx *gin.Context) {
	var req model.UnbindInstanceFromSecurityGroupReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的安全组ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.securityGroupService.UnbindInstanceFromSecurityGroup(ctx, &req)
	})
}

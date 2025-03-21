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
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type RoleHandler struct {
	svc    service.RoleService
	apiSvc service.ApiService
	l      *zap.Logger
}

func NewRoleHandler(svc service.RoleService, apiSvc service.ApiService, l *zap.Logger) *RoleHandler {
	return &RoleHandler{
		svc:    svc,
		apiSvc: apiSvc,
		l:      l,
	}
}

func (r *RoleHandler) RegisterRouters(server *gin.Engine) {
	roleGroup := server.Group("/api/roles")

	roleGroup.POST("/list", r.ListRoles)
	roleGroup.POST("/create", r.CreateRole)
	roleGroup.POST("/update", r.UpdateRole)
	roleGroup.POST("/delete", r.DeleteRole)
	roleGroup.POST("/user/roles", r.GetUserRoles)
}

// ListRoles 获取角色列表
func (r *RoleHandler) ListRoles(c *gin.Context) {
	var req model.ListRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		r.l.Error("绑定请求参数失败", zap.Error(err))
		utils.ErrorWithMessage(c, err.Error())
		return
	}

	// 调用service获取角色列表
	resp, err := r.svc.ListRoles(c.Request.Context(), req.PageNumber, req.PageSize)
	if err != nil {
		r.l.Error("获取角色列表失败", zap.Error(err))
		utils.ErrorWithMessage(c, err.Error())
		return
	}

	utils.SuccessWithData(c, gin.H{
		"items": resp.Items,
		"total": resp.Total,
	})
}

// CreateRole 创建角色
func (r *RoleHandler) CreateRole(c *gin.Context) {
	var req model.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		r.l.Error("绑定请求参数失败", zap.Error(err))
		utils.ErrorWithMessage(c, err.Error())
		return
	}

	// 创建角色
	if err := r.svc.CreateRole(c.Request.Context(), &req); err != nil {
		r.l.Error("创建角色失败", zap.Error(err))
		utils.ErrorWithMessage(c, err.Error())
		return
	}

	utils.Success(c)
}

// UpdateRole 更新角色
func (r *RoleHandler) UpdateRole(c *gin.Context) {
	var req model.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		r.l.Error("绑定请求参数失败", zap.Error(err))
		utils.ErrorWithMessage(c, err.Error())
		return
	}

	// 更新角色基本信息
	if err := r.svc.UpdateRole(c.Request.Context(), &req); err != nil {
		r.l.Error("更新角色失败", zap.Error(err))
		utils.ErrorWithMessage(c, err.Error())
		return
	}

	utils.Success(c)
}

// DeleteRole 删除角色
func (r *RoleHandler) DeleteRole(c *gin.Context) {
	var req model.DeleteRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		r.l.Error("绑定请求参数失败", zap.Error(err))
		utils.ErrorWithMessage(c, err.Error())
		return
	}

	if err := r.svc.DeleteRole(c.Request.Context(), &req); err != nil {
		r.l.Error("删除角色失败", zap.Error(err))
		utils.ErrorWithMessage(c, err.Error())
		return
	}

	utils.Success(c)
}

// GetUserRoles 获取用户角色
func (r *RoleHandler) GetUserRoles(c *gin.Context) {
	var req model.ListUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		r.l.Error("绑定请求参数失败", zap.Error(err))
		utils.ErrorWithMessage(c, err.Error())
		return
	}

	// 从URL参数中获取用户ID
	userId, err := strconv.Atoi(c.Query("user_id"))
	if err != nil {
		r.l.Error("解析用户ID失败", zap.Error(err))
		utils.ErrorWithMessage(c, "无效的用户ID")
		return
	}

	resp, err := r.svc.GetUserRoles(c.Request.Context(), userId, req.PageNumber, req.PageSize)
	if err != nil {
		r.l.Error("获取用户角色失败", zap.Error(err))
		utils.ErrorWithMessage(c, err.Error())
		return
	}

	utils.SuccessWithData(c, resp)
}

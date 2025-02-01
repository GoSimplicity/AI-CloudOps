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
	svc           service.RoleService
	apiSvc        service.ApiService
	permissionSvc service.PermissionService
	l             *zap.Logger
}

func NewRoleHandler(svc service.RoleService, apiSvc service.ApiService, permissionSvc service.PermissionService, l *zap.Logger) *RoleHandler {
	return &RoleHandler{
		svc:           svc,
		apiSvc:        apiSvc,
		permissionSvc: permissionSvc,
		l:             l,
	}
}

func (r *RoleHandler) RegisterRouters(server *gin.Engine) {
	roleGroup := server.Group("/api/roles")

	roleGroup.POST("/list", r.ListRoles)
	roleGroup.POST("/create", r.CreateRole)
	roleGroup.POST("/update", r.UpdateRole)
	roleGroup.DELETE("/:id", r.DeleteRole)
	roleGroup.GET("/user/:id", r.GetUserRoles)
	roleGroup.GET("/:id", r.GetRoles)
}

// ListRoles 获取角色列表
func (r *RoleHandler) ListRoles(c *gin.Context) {
	var req model.ListRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		r.l.Error("绑定请求参数失败", zap.Error(err))
		utils.Error(c)
		return
	}

	// 调用service获取角色列表
	roles, total, err := r.svc.ListRoles(c.Request.Context(), req.PageNumber, req.PageSize)
	if err != nil {
		r.l.Error("获取角色列表失败", zap.Error(err))
		utils.Error(c)
		return
	}

	utils.SuccessWithData(c, gin.H{
		"list":  roles,
		"total": total,
	})
}

// CreateRole 创建角色
func (r *RoleHandler) CreateRole(c *gin.Context) {
	var req model.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		r.l.Error("绑定请求参数失败", zap.Error(err))
		utils.Error(c)
		return
	}

	// 构建角色对象
	role := &model.Role{
		Name:      req.Name,
		Desc:      req.Description,
		RoleType:  int8(req.RoleType),
		IsDefault: int8(req.IsDefault),
	}

	// 创建角色并分配权限
	if err := r.svc.CreateRole(c.Request.Context(), role, req.ApiIds); err != nil {
		r.l.Error("创建角色失败", zap.Error(err))
		utils.Error(c)
		return
	}

	utils.Success(c)
}

// UpdateRole 更新角色
func (r *RoleHandler) UpdateRole(c *gin.Context) {
	var req model.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		r.l.Error("绑定请求参数失败", zap.Error(err))
		utils.Error(c)
		return
	}

	// 构建角色对象
	role := &model.Role{
		ID:        req.Id,
		Name:      req.Name,
		Desc:      req.Description,
		RoleType:  int8(req.RoleType),
		IsDefault: int8(req.IsDefault),
	}

	// 更新角色基本信息
	if err := r.svc.UpdateRole(c.Request.Context(), role); err != nil {
		r.l.Error("更新角色失败", zap.Error(err))
		utils.Error(c)
		return
	}

	// 更新角色权限
	if err := r.permissionSvc.AssignRole(c.Request.Context(), role.ID, req.ApiIds); err != nil {
		r.l.Error("更新权限失败", zap.Error(err))
		utils.Error(c)
		return
	}

	utils.Success(c)
}

// DeleteRole 删除角色
func (r *RoleHandler) DeleteRole(c *gin.Context) {
	// 从URL参数中获取角色ID
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		r.l.Error("解析ID失败", zap.Error(err))
		utils.Error(c)
		return
	}

	if err := r.svc.DeleteRole(c.Request.Context(), id); err != nil {
		r.l.Error("删除角色失败", zap.Error(err))
		utils.Error(c)
		return
	}

	utils.Success(c)
}

// UpdateUserRole 更新用户角色
func (r *RoleHandler) UpdateUserRole(c *gin.Context) {
	var req model.UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		r.l.Error("绑定请求参数失败", zap.Error(err))
		utils.Error(c)
		return
	}

	// 分配用户角色和权限
	if err := r.permissionSvc.AssignRoleToUser(c.Request.Context(), req.UserId, req.RoleIds, req.ApiIds); err != nil {
		r.l.Error("分配API权限失败", zap.Error(err))
		utils.Error(c)
		return
	}

	utils.Success(c)
}

// GetUserRoles 获取用户角色
func (r *RoleHandler) GetUserRoles(c *gin.Context) {
	// 从URL参数中获取用户ID
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		r.l.Error("解析ID失败", zap.Error(err))
		utils.Error(c)
		return
	}

	role, err := r.svc.GetUserRole(c.Request.Context(), id)
	if err != nil {
		r.l.Error("获取用户角色失败", zap.Error(err))
		utils.Error(c)
		return
	}

	utils.SuccessWithData(c, role)
}

// GetRoles 获取角色详情
func (r *RoleHandler) GetRoles(c *gin.Context) {
	// 从URL参数中获取角色ID
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		r.l.Error("解析ID失败", zap.Error(err))
		utils.Error(c)
		return
	}

	role, err := r.svc.GetRole(c.Request.Context(), id)
	if err != nil {
		r.l.Error("获取角色失败", zap.Error(err))
		utils.Error(c)
		return
	}

	utils.SuccessWithData(c, role)
}

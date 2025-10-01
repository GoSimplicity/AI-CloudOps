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
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	svc service.RoleService
}

func NewRoleHandler(svc service.RoleService) *RoleHandler {
	return &RoleHandler{
		svc: svc,
	}
}

func (h *RoleHandler) RegisterRouters(server *gin.Engine) {
	roleGroup := server.Group("/api/role")
	{
		// 角色管理
		roleGroup.GET("/list", h.ListRoles)
		roleGroup.POST("/create", h.CreateRole)
		roleGroup.PUT("/update/:id", h.UpdateRole)
		roleGroup.DELETE("/delete/:id", h.DeleteRole)
		roleGroup.GET("/detail/:id", h.GetRoleDetail)

		// 角色权限管理
		roleGroup.POST("/assign-apis", h.AssignApisToRole)
		roleGroup.POST("/revoke-apis", h.RevokeApisFromRole)
		roleGroup.GET("/apis/:id", h.GetRoleApis)

		// 用户角色管理
		roleGroup.POST("/assign_users", h.AssignRolesToUser)
		roleGroup.POST("/revoke_users", h.RevokeRolesFromUser)
		roleGroup.GET("/users/:id", h.GetRoleUsers)
		roleGroup.GET("/user_roles/:id", h.GetUserRoles)

		// 权限检查
		roleGroup.POST("/check_permission", h.CheckUserPermission)
		roleGroup.GET("/user_permissions/:id", h.GetUserPermissions)
	}
}

// ListRoles 获取角色列表
func (h *RoleHandler) ListRoles(ctx *gin.Context) {
	var req model.ListRolesRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.ListRoles(ctx, &req)
	})
}

// CreateRole 创建角色
func (h *RoleHandler) CreateRole(ctx *gin.Context) {
	var req model.CreateRoleRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.CreateRole(ctx, &req)
	})
}

// UpdateRole 更新角色
func (h *RoleHandler) UpdateRole(ctx *gin.Context) {
	var req model.UpdateRoleRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.UpdateRole(ctx, &req)
	})
}

// DeleteRole 删除角色
func (h *RoleHandler) DeleteRole(ctx *gin.Context) {
	var req model.DeleteRoleRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.DeleteRole(ctx, req.ID)
	})
}

// GetRoleDetail 获取角色详情
func (h *RoleHandler) GetRoleDetail(ctx *gin.Context) {
	var req model.GetRoleRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetRoleByID(ctx, id)
	})
}

// AssignApisToRole 为角色分配API权限
func (h *RoleHandler) AssignApisToRole(ctx *gin.Context) {
	var req model.AssignRoleApiRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.AssignApisToRole(ctx, req.RoleID, req.ApiIds)
	})
}

// RevokeApisFromRole 撤销角色的API权限
func (h *RoleHandler) RevokeApisFromRole(ctx *gin.Context) {
	var req model.RevokeRoleApiRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.RevokeApisFromRole(ctx, req.RoleID, req.ApiIds)
	})
}

// GetRoleApis 获取角色的API权限列表
func (h *RoleHandler) GetRoleApis(ctx *gin.Context) {
	var req model.GetRoleApiRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetRoleApis(ctx, id)
	})
}

// AssignRolesToUser 为用户分配角色
func (h *RoleHandler) AssignRolesToUser(ctx *gin.Context) {
	var req model.AssignRolesToUserRequest

	user := ctx.MustGet("user").(utils.UserClaims)

	req.UserID = user.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.AssignRolesToUser(ctx, req.UserID, req.RoleIds, 0)
	})
}

// RevokeRolesFromUser 撤销用户角色
func (h *RoleHandler) RevokeRolesFromUser(ctx *gin.Context) {
	var req model.RevokeRolesFromUserRequest

	user := ctx.MustGet("user").(utils.UserClaims)

	req.UserID = user.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.RevokeRolesFromUser(ctx, req.UserID, req.RoleIds)
	})
}

// GetRoleUsers 获取角色下的用户列表
func (h *RoleHandler) GetRoleUsers(ctx *gin.Context) {
	var req model.GetRoleUsersRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetRoleUsers(ctx, id)
	})
}

// GetUserRoles 获取用户的角色列表
func (h *RoleHandler) GetUserRoles(ctx *gin.Context) {
	var req model.GetUserRolesRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetUserRoles(ctx, req.ID)
	})
}

// CheckUserPermission 检查用户权限
func (h *RoleHandler) CheckUserPermission(ctx *gin.Context) {
	var req model.CheckUserPermissionRequest

	user := ctx.MustGet("user").(utils.UserClaims)
	req.UserID = user.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.CheckUserPermission(ctx, req.UserID, req.Method, req.Path)
	})
}

// GetUserPermissions 获取用户的所有权限
func (h *RoleHandler) GetUserPermissions(ctx *gin.Context) {
	var req model.GetUserPermissionsRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetUserPermissions(ctx, req.ID)
	})
}

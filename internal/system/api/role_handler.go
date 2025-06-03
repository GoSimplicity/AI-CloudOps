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

func (r *RoleHandler) RegisterRouters(server *gin.Engine) {
	roleGroup := server.Group("/api/role")
	{
		// 角色管理
		roleGroup.POST("/list", r.ListRoles)
		roleGroup.POST("/create", r.CreateRole)
		roleGroup.POST("/update", r.UpdateRole)
		roleGroup.POST("/delete", r.DeleteRole)
		roleGroup.GET("/detail/:id", r.GetRoleDetail)

		// 角色权限管理
		roleGroup.POST("/assign-apis", r.AssignApisToRole)
		roleGroup.POST("/revoke-apis", r.RevokeApisFromRole)
		roleGroup.GET("/apis/:id", r.GetRoleApis)

		// 用户角色管理
		roleGroup.POST("/assign_users", r.AssignRolesToUser)
		roleGroup.POST("/revoke_users", r.RevokeRolesFromUser)
		roleGroup.GET("/users/:id", r.GetRoleUsers)
		roleGroup.GET("/user_roles/:id", r.GetUserRoles)

		// 权限检查
		roleGroup.POST("/check_permission", r.CheckUserPermission)
		roleGroup.GET("/user_permissions/:id", r.GetUserPermissions)
	}
}

// ListRoles 获取角色列表
func (r *RoleHandler) ListRoles(ctx *gin.Context) {
	var req model.ListRolesRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return r.svc.ListRoles(ctx, &req)
	})
}

// CreateRole 创建角色
func (r *RoleHandler) CreateRole(ctx *gin.Context) {
	var req model.CreateRoleRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return r.svc.CreateRole(ctx, &req)
	})
}

// UpdateRole 更新角色
func (r *RoleHandler) UpdateRole(ctx *gin.Context) {
	var req model.UpdateRoleRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return r.svc.UpdateRole(ctx, &req)
	})
}

// DeleteRole 删除角色
func (r *RoleHandler) DeleteRole(ctx *gin.Context) {
	var req model.DeleteRoleRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, r.svc.DeleteRole(ctx, req.ID)
	})
}

// GetRoleDetail 获取角色详情
func (r *RoleHandler) GetRoleDetail(ctx *gin.Context) {
	var req model.GetRoleRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return r.svc.GetRoleByID(ctx, id)
	})
}

// AssignApisToRole 为角色分配API权限
func (r *RoleHandler) AssignApisToRole(ctx *gin.Context) {
	var req model.AssignRoleApiRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, r.svc.AssignApisToRole(ctx, req.RoleID, req.ApiIds)
	})
}

// RevokeApisFromRole 撤销角色的API权限
func (r *RoleHandler) RevokeApisFromRole(ctx *gin.Context) {
	var req model.RevokeRoleApiRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, r.svc.RevokeApisFromRole(ctx, req.RoleID, req.ApiIds)
	})
}

// GetRoleApis 获取角色的API权限列表
func (r *RoleHandler) GetRoleApis(ctx *gin.Context) {
	var req model.GetRoleApiRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return r.svc.GetRoleApis(ctx, id)
	})
}

// AssignRolesToUser 为用户分配角色
func (r *RoleHandler) AssignRolesToUser(ctx *gin.Context) {
	var req model.AssignRolesToUserRequest

	user := ctx.MustGet("user").(utils.UserClaims)

	req.UserID = user.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, r.svc.AssignRolesToUser(ctx, req.UserID, req.RoleIds, 0)
	})
}

// RevokeRolesFromUser 撤销用户角色
func (r *RoleHandler) RevokeRolesFromUser(ctx *gin.Context) {
	var req model.RevokeRolesFromUserRequest

	user := ctx.MustGet("user").(utils.UserClaims)

	req.UserID = user.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, r.svc.RevokeRolesFromUser(ctx, req.UserID, req.RoleIds)
	})
}

// GetRoleUsers 获取角色下的用户列表
func (r *RoleHandler) GetRoleUsers(ctx *gin.Context) {
	var req model.GetRoleUsersRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return r.svc.GetRoleUsers(ctx, id)
	})
}

// GetUserRoles 获取用户的角色列表
func (r *RoleHandler) GetUserRoles(ctx *gin.Context) {
	var req model.GetUserRolesRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return r.svc.GetUserRoles(ctx, req.ID)
	})
}

// CheckUserPermission 检查用户权限
func (r *RoleHandler) CheckUserPermission(ctx *gin.Context) {
	var req model.CheckUserPermissionRequest

	user := ctx.MustGet("user").(utils.UserClaims)
	req.UserID = user.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return r.svc.CheckUserPermission(ctx, req.UserID, req.Method, req.Path)
	})
}

// GetUserPermissions 获取用户的所有权限
func (r *RoleHandler) GetUserPermissions(ctx *gin.Context) {
	var req model.GetUserPermissionsRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return r.svc.GetUserPermissions(ctx, req.ID)
	})
}

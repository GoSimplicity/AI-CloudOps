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
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

		roleGroup.POST("/check_permission", h.CheckUserPermission)
		roleGroup.GET("/user_permissions/:id", h.GetUserPermissions)
	}
}

func (h *RoleHandler) ListRoles(ctx *gin.Context) {
	var req model.ListRolesRequest

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.ListRoles(ctx.Request.Context(), &req)
	})
}

func (h *RoleHandler) CreateRole(ctx *gin.Context) {
	var req model.CreateRoleRequest

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.CreateRole(ctx.Request.Context(), &req)
	})
}

func (h *RoleHandler) UpdateRole(ctx *gin.Context) {
	var req model.UpdateRoleRequest
	id, err := base.GetParamID(ctx)
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}
	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.UpdateRole(ctx.Request.Context(), &req)
	})
}

func (h *RoleHandler) DeleteRole(ctx *gin.Context) {
	var req model.DeleteRoleRequest
	id, err := base.GetParamID(ctx)
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}
	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.DeleteRole(ctx.Request.Context(), req.ID)
	})
}

func (h *RoleHandler) GetRoleDetail(ctx *gin.Context) {
	var req model.GetRoleRequest

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetRoleByID(ctx.Request.Context(), id)
	})
}

// AssignApisToRole 为角色分配API权限
func (h *RoleHandler) AssignApisToRole(ctx *gin.Context) {
	var req model.AssignRoleApiRequest

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.AssignApisToRole(ctx.Request.Context(), req.RoleID, req.ApiIds)
	})
}

// RevokeApisFromRole 撤销角色的API权限
func (h *RoleHandler) RevokeApisFromRole(ctx *gin.Context) {
	var req model.RevokeRoleApiRequest

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.RevokeApisFromRole(ctx.Request.Context(), req.RoleID, req.ApiIds)
	})
}

// GetRoleApis 获取角色的API权限列表
func (h *RoleHandler) GetRoleApis(ctx *gin.Context) {
	var req model.GetRoleApiRequest

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetRoleApis(ctx.Request.Context(), id)
	})
}

// AssignRolesToUser 为用户分配角色
func (h *RoleHandler) AssignRolesToUser(ctx *gin.Context) {
	var req model.AssignRolesToUserRequest

	user := ctx.MustGet("user").(jwt.UserClaims)

	if err := base.BindRequest(ctx, &req); err != nil {
		base.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}
	if !h.guardSuperAdminRoles(ctx, req.UserID, req.RoleIds) {
		return
	}

	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.svc.AssignRolesToUser(ctx.Request.Context(), req.UserID, req.RoleIds, user.Uid)
	})
}

// RevokeRolesFromUser 撤销用户角色
func (h *RoleHandler) RevokeRolesFromUser(ctx *gin.Context) {
	var req model.RevokeRolesFromUserRequest

	if err := base.BindRequest(ctx, &req); err != nil {
		base.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}
	if !h.guardSuperAdminRoles(ctx, req.UserID, req.RoleIds) {
		return
	}

	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.svc.RevokeRolesFromUser(ctx.Request.Context(), req.UserID, req.RoleIds)
	})
}

// guardSuperAdminRoles 授予或撤销内置超管角色只能由超管操作，防止持有角色分配
// 权限的普通用户把自己（或他人）提升为超管、或撤销超管。roleIds 为空时 DAO 会撤销
// 目标用户的全部角色，因此按目标用户当前角色判断。返回 false 时已写出响应。
func (h *RoleHandler) guardSuperAdminRoles(ctx *gin.Context, userID int, roleIds []int) bool {
	if permission.IsSuperAdmin(ctx) {
		return true
	}
	touches := false
	if len(roleIds) == 0 {
		roles, err := h.svc.GetUserRoles(ctx.Request.Context(), userID)
		if err != nil {
			base.ErrorWithMessage(ctx, "获取用户角色失败")
			return false
		}
		if roles != nil {
			for _, role := range roles.Items {
				touches = touches || permission.IsSuperAdminIdentity(role)
			}
		}
	}
	for _, id := range roleIds {
		role, err := h.svc.GetRoleByID(ctx.Request.Context(), id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			base.ErrorWithMessage(ctx, "获取角色失败")
			return false
		}
		touches = touches || permission.IsSuperAdminIdentity(role)
	}
	if touches {
		base.ForbiddenError(ctx, "仅超级管理员可授予或撤销超级管理员角色")
		return false
	}
	return true
}

func (h *RoleHandler) GetRoleUsers(ctx *gin.Context) {
	var req model.GetRoleUsersRequest

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetRoleUsers(ctx.Request.Context(), id)
	})
}

func (h *RoleHandler) GetUserRoles(ctx *gin.Context) {
	var req model.GetUserRolesRequest

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetUserRoles(ctx.Request.Context(), req.ID)
	})
}

func (h *RoleHandler) CheckUserPermission(ctx *gin.Context) {
	var req model.CheckUserPermissionRequest

	user := ctx.MustGet("user").(jwt.UserClaims)
	req.UserID = user.Uid

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.CheckUserPermission(ctx.Request.Context(), req.UserID, req.Method, req.Path)
	})
}

func (h *RoleHandler) GetUserPermissions(ctx *gin.Context) {
	var req model.GetUserPermissionsRequest

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetUserPermissions(ctx.Request.Context(), req.ID)
	})
}

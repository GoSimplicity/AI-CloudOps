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
		roleGroup.GET("/list", r.ListRoles)
		roleGroup.POST("/create", r.CreateRole)
		roleGroup.PUT("/update/:id", r.UpdateRole)
		roleGroup.DELETE("/delete/:id", r.DeleteRole)
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
// @Summary 获取角色列表
// @Description 分页获取系统中的角色列
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param name query string false "角色名称模糊搜索"
// @Success 200 {object} utils.ApiResponse{data=[]model.Role} "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/role/list [get]
func (r *RoleHandler) ListRoles(ctx *gin.Context) {
	var req model.ListRolesRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return r.svc.ListRoles(ctx, &req)
	})
}

// CreateRole 创建角色
// @Summary 创建角色
// @Description 创建新的系统角色
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param request body model.CreateRoleRequest true "创建角色请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/role/create [post]
func (r *RoleHandler) CreateRole(ctx *gin.Context) {
	var req model.CreateRoleRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return r.svc.CreateRole(ctx, &req)
	})
}

// UpdateRole 更新角色
// @Summary 更新角色
// @Description 更新指定角色的信息
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param id path int true "角色ID"
// @Param request body model.UpdateRoleRequest true "更新角色请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/role/update/{id} [put]
func (r *RoleHandler) UpdateRole(ctx *gin.Context) {
	var req model.UpdateRoleRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return r.svc.UpdateRole(ctx, &req)
	})
}

// DeleteRole 删除角色
// @Summary 删除角色
// @Description 根据ID删除指定的角色
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param id path int true "角色ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/role/delete/{id} [delete]
func (r *RoleHandler) DeleteRole(ctx *gin.Context) {
	var req model.DeleteRoleRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, r.svc.DeleteRole(ctx, req.ID)
	})
}

// GetRoleDetail 获取角色详情
// @Summary 获取角色详情
// @Description 根据ID获取指定角色的详细信息
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param id path int true "角色ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/role/detail/{id} [get]
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
// @Summary 为角色分配API权限
// @Description 为指定角色分配多个API权限
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param request body model.AssignRoleApiRequest true "分配API权限请求参数"
// @Success 200 {object} utils.ApiResponse "分配成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/role/assign-apis [post]
func (r *RoleHandler) AssignApisToRole(ctx *gin.Context) {
	var req model.AssignRoleApiRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, r.svc.AssignApisToRole(ctx, req.RoleID, req.ApiIds)
	})
}

// RevokeApisFromRole 撤销角色的API权限
// @Summary 撤销角色API权限
// @Description 撤销指定角色的多个API权限
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param request body model.RevokeRoleApiRequest true "撤销API权限请求参数"
// @Success 200 {object} utils.ApiResponse "撤销成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/role/revoke-apis [post]
func (r *RoleHandler) RevokeApisFromRole(ctx *gin.Context) {
	var req model.RevokeRoleApiRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, r.svc.RevokeApisFromRole(ctx, req.RoleID, req.ApiIds)
	})
}

// GetRoleApis 获取角色的API权限列表
// @Summary 获取角色API权限
// @Description 获取指定角色的所有API权限列表
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param id path int true "角色ID"
// @Success 200 {object} utils.ApiResponse{data=[]model.Api} "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/role/apis/{id} [get]
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
// @Summary 为用户分配角色
// @Description 为指定用户分配多个角色
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param request body model.AssignRolesToUserRequest true "为用户分配角色请求参数"
// @Success 200 {object} utils.ApiResponse "分配成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/role/assign_users [post]
func (r *RoleHandler) AssignRolesToUser(ctx *gin.Context) {
	var req model.AssignRolesToUserRequest

	user := ctx.MustGet("user").(utils.UserClaims)

	req.UserID = user.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, r.svc.AssignRolesToUser(ctx, req.UserID, req.RoleIds, 0)
	})
}

// RevokeRolesFromUser 撤销用户角色
// @Summary 撤销用户角色
// @Description 撤销指定用户的多个角色
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param request body model.RevokeRolesFromUserRequest true "撤销用户角色请求参数"
// @Success 200 {object} utils.ApiResponse "撤销成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/role/revoke_users [post]
func (r *RoleHandler) RevokeRolesFromUser(ctx *gin.Context) {
	var req model.RevokeRolesFromUserRequest

	user := ctx.MustGet("user").(utils.UserClaims)

	req.UserID = user.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, r.svc.RevokeRolesFromUser(ctx, req.UserID, req.RoleIds)
	})
}

// GetRoleUsers 获取角色下的用户列表
// @Summary 获取角色用户列表
// @Description 获取指定角色下的所有用户列表
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param id path int true "角色ID"
// @Success 200 {object} utils.ApiResponse{data=[]model.User} "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/role/users/{id} [get]
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
// @Summary 获取用户角色列表
// @Description 获取指定用户的所有角色列表
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} utils.ApiResponse{data=[]model.Role} "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/role/user_roles/{id} [get]
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
// @Summary 检查用户权限
// @Description 检查用户对指定API路径和方法是否有访问权限
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param request body model.CheckUserPermissionRequest true "检查权限请求参数"
// @Success 200 {object} utils.ApiResponse{data=bool} "检查成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/role/check_permission [post]
func (r *RoleHandler) CheckUserPermission(ctx *gin.Context) {
	var req model.CheckUserPermissionRequest

	user := ctx.MustGet("user").(utils.UserClaims)
	req.UserID = user.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return r.svc.CheckUserPermission(ctx, req.UserID, req.Method, req.Path)
	})
}

// GetUserPermissions 获取用户的所有权限
// @Summary 获取用户所有权限
// @Description 获取指定用户的所有API权限列表
// @Tags 系统管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} utils.ApiResponse{data=[]model.Api} "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/role/user_permissions/{id} [get]
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

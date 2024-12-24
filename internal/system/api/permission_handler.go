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

type PermissionHandler struct {
	svc service.PermissionService
}

func NewPermissionHandler(svc service.PermissionService) *PermissionHandler {
	return &PermissionHandler{
		svc: svc,
	}
}

func (h *PermissionHandler) RegisterRouters(server *gin.Engine) {
	permissionGroup := server.Group("/api/permissions")

	permissionGroup.POST("/user/assign", h.AssignUserRole)
	permissionGroup.POST("/users/assign", h.AssignUsersRole)
}

// AssignUserRole 为单个用户分配角色和权限
func (h *PermissionHandler) AssignUserRole(c *gin.Context) {
	var r model.AssignUserRoleRequest
	// 绑定请求参数
	if err := c.ShouldBindJSON(&r); err != nil {
		utils.Error(c)
		return
	}

	// 调用服务层分配角色和权限
	if err := h.svc.AssignRoleToUser(c.Request.Context(), r.UserId, r.RoleIds, r.ApiIds); err != nil {
		utils.ErrorWithMessage(c, err.Error())
		return
	}

	utils.Success(c)
}

// AssignUsersRole 批量为用户分配角色和权限
func (h *PermissionHandler) AssignUsersRole(c *gin.Context) {
	var r model.AssignUsersRoleRequest
	// 绑定请求参数
	if err := c.ShouldBindJSON(&r); err != nil {
		utils.Error(c)
		return
	}

	// 调用服务层批量分配角色和权限
	if err := h.svc.AssignRoleToUsers(c.Request.Context(), r.UserIds, r.RoleIds); err != nil {
		utils.Error(c)
		return
	}

	utils.Success(c)
}

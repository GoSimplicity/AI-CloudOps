package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
)

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

type AuthRoleHandler struct {
	roleService service.AuthRoleService
}

func NewAuthRoleHandler(roleService service.AuthRoleService) *AuthRoleHandler {
	return &AuthRoleHandler{
		roleService: roleService,
	}
}

func (r *AuthRoleHandler) RegisterRouters(server *gin.Engine) {
	authGroup := server.Group("/api/auth")

	// 权限管理相关路由
	authGroup.GET("/role/list", r.GetAllRoleList)
	authGroup.POST("/role/create", r.CreateRole)
	authGroup.POST("/role/update", r.UpdateRole)
	authGroup.POST("/role/status", r.SetRoleStatus)
	authGroup.DELETE("/role/:id", r.DeleteRole)
}

func (r *AuthRoleHandler) GetAllRoleList(ctx *gin.Context) {
	roles, err := r.roleService.GetAllRoleList(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, roles)
}

func (r *AuthRoleHandler) CreateRole(ctx *gin.Context) {
	var req model.Role

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	err := r.roleService.CreateRole(ctx, req)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "创建成功")
}

func (r *AuthRoleHandler) UpdateRole(ctx *gin.Context) {
	var req model.Role

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	err := r.roleService.UpdateRole(ctx, req)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "更新成功")
}

func (r *AuthRoleHandler) SetRoleStatus(ctx *gin.Context) {
	var req model.Role

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	err := r.roleService.SetRoleStatus(ctx, req.ID, req.Status)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "更新成功")
}

func (r *AuthRoleHandler) DeleteRole(ctx *gin.Context) {
	id := ctx.Param("id")

	err := r.roleService.DeleteRole(ctx, id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "删除成功")
}

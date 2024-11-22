package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
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

type AuthMenuHandler struct {
	menuService service.AuthMenuService
}

func NewAuthMenuHandler(menuService service.AuthMenuService) *AuthMenuHandler {
	return &AuthMenuHandler{
		menuService: menuService,
	}
}

func (m *AuthMenuHandler) RegisterRouters(server *gin.Engine) {
	authGroup := server.Group("/api/auth")

	// 菜单相关路由
	authGroup.GET("/menu/list", m.GetMenuList)
	authGroup.GET("/menu/all", m.GetAllMenuList)
	authGroup.POST("/menu/update", m.UpdateMenu)
	authGroup.POST("/menu/update_status", m.UpdateMenuStatus)
	authGroup.POST("/menu/create", m.CreateMenu)
	authGroup.DELETE("/menu/:id", m.DeleteMenu)
}

func (m *AuthMenuHandler) GetMenuList(ctx *gin.Context) {
	uc := ctx.MustGet("user").(ijwt.UserClaims)

	roles, err := m.menuService.GetMenuList(ctx, uc.Uid)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, roles)
}

func (m *AuthMenuHandler) GetAllMenuList(ctx *gin.Context) {
	menus, err := m.menuService.GetAllMenuList(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, menus)
}

func (m *AuthMenuHandler) UpdateMenu(ctx *gin.Context) {
	var req model.Menu

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	if err := m.menuService.UpdateMenu(ctx, req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "更新成功")
}

func (m *AuthMenuHandler) UpdateMenuStatus(ctx *gin.Context) {
	var req struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	if err := m.menuService.UpdateMenuStatus(ctx, req.ID, req.Status); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "更新成功")
}

func (m *AuthMenuHandler) CreateMenu(ctx *gin.Context) {
	var req model.Menu

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	if err := m.menuService.CreateMenu(ctx, req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "创建成功")
}

func (m *AuthMenuHandler) DeleteMenu(ctx *gin.Context) {
	id := ctx.Param("id")

	if err := m.menuService.DeleteMenu(ctx, id); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "删除成功")
}

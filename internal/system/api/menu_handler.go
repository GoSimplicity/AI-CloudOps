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
)

type MenuHandler struct {
	svc service.MenuService
}

func NewMenuHandler(svc service.MenuService) *MenuHandler {
	return &MenuHandler{
		svc: svc,
	}
}

func (m *MenuHandler) RegisterRouters(server *gin.Engine) {
	menuGroup := server.Group("/api/menus")

	menuGroup.POST("/list", m.ListMenus)
	menuGroup.POST("/create", m.CreateMenu)
	menuGroup.POST("/update", m.UpdateMenu)
	menuGroup.DELETE("/:id", m.DeleteMenu)
	menuGroup.POST("/update_related", m.UpdateUserMenu)
}

// ListMenus 获取菜单列表
func (m *MenuHandler) ListMenus(c *gin.Context) {
	var req model.ListMenusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMessage(c, "参数错误")
		return
	}

	// 调用service层获取菜单列表
	menus, _, err := m.svc.GetMenus(c.Request.Context(), req.PageNumber, req.PageSize)
	if err != nil {
		utils.ErrorWithMessage(c, "获取菜单列表失败")
		return
	}

	utils.SuccessWithData(c, menus)
}

// CreateMenu 创建菜单
func (m *MenuHandler) CreateMenu(c *gin.Context) {
	var req model.CreateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMessage(c, "参数错误")
		return
	}
	menu := &model.Menu{
		Name:      req.Name,
		Path:      req.Path,
		Component: req.Component,
		ParentID:  req.ParentId,
		Hidden:    int8(req.Hidden),
		RouteName: req.RouteName,
		Redirect:  req.Redirect,
		Meta:      req.Meta,
		Children:  req.Children,
	}

	if err := m.svc.CreateMenu(c.Request.Context(), menu); err != nil {
		utils.ErrorWithMessage(c, "创建菜单失败")
		return
	}

	utils.SuccessWithMessage(c, "创建成功")
}

// UpdateMenu 更新菜单
func (m *MenuHandler) UpdateMenu(c *gin.Context) {
	var req model.UpdateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMessage(c, "参数错误")
		return
	}

	menu := &model.Menu{
		ID:        req.Id,
		Name:      req.Name,
		Path:      req.Path,
		Component: req.Component,
		ParentID:  req.ParentId,
		Hidden:    int8(req.Hidden),
		RouteName: req.RouteName,
	}

	if err := m.svc.UpdateMenu(c.Request.Context(), menu); err != nil {
		utils.ErrorWithMessage(c, "更新菜单失败")
		return
	}

	utils.SuccessWithMessage(c, "更新成功")
}

// DeleteMenu 删除菜单
func (m *MenuHandler) DeleteMenu(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.ErrorWithMessage(c, "参数错误")
		return
	}

	if err := m.svc.DeleteMenu(c.Request.Context(), id); err != nil {
		utils.ErrorWithMessage(c, "删除菜单失败")
		return
	}

	utils.SuccessWithMessage(c, "删除成功")
}

// AddUserMenu 添加用户菜单关联
func (m *MenuHandler) UpdateUserMenu(c *gin.Context) {
	var req model.UpdateUserMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMessage(c, "参数错误")
		return
	}

	if err := m.svc.UpdateUserMenu(c.Request.Context(), req.UserId, req.MenuIds); err != nil {
		utils.ErrorWithMessage(c, "更新用户菜单关联失败")
		return
	}

	utils.SuccessWithMessage(c, "更新成功")
}

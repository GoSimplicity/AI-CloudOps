package service

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"sort"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
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

type AuthMenuService interface {
	GetMenuList(ctx context.Context, uid int) ([]*model.Menu, error)
	GetAllMenuList(ctx context.Context) ([]*model.Menu, error)
	UpdateMenu(ctx context.Context, menu model.Menu) error
	UpdateMenuStatus(ctx context.Context, menuID int, status string) error
	CreateMenu(ctx context.Context, menu model.Menu) error
	DeleteMenu(ctx context.Context, id string) error
}

type authMenuService struct {
	menuDao dao.AuthMenuDAO
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewAuthMenuService(menuDao dao.AuthMenuDAO, l *zap.Logger, userDao userDao.UserDAO) AuthMenuService {
	return &authMenuService{
		menuDao: menuDao,
		l:       l,
		userDao: userDao,
	}
}

// GetMenuList 根据用户ID获取菜单列表，支持按角色过滤菜单
func (m *authMenuService) GetMenuList(ctx context.Context, uid int) ([]*model.Menu, error) {
	// 获取用户信息
	user, err := m.userDao.GetUserByID(ctx, uid)
	if err != nil {
		m.l.Error("GetUserByID failed", zap.Error(err))
		return nil, err
	}

	// 父菜单映射和子菜单唯一性检查
	fatherMenuMap := make(map[int]*model.Menu)
	uniqueChildMap := make(map[int]struct{})

	// 遍历用户角色，处理菜单
	for _, role := range user.Roles {
		if role.Status == "0" {
			// 跳过禁用的角色
			continue
		}

		// 处理角色的菜单
		for _, menu := range role.Menus {
			if menu.Status == "0" && role.RoleValue != "super" {
				// 非超级管理员跳过禁用菜单
				continue
			}

			// 设置菜单元数据
			m.setMenuMeta(menu)

			// 父菜单处理
			if menu.Pid == 0 {
				fatherMenuMap[menu.ID] = menu
			} else {
				// 处理子菜单并附加到父菜单
				if err := m.attachToFatherMenu(ctx, menu, fatherMenuMap, uniqueChildMap); err != nil {
					m.l.Error("attachToFatherMenu failed", zap.Error(err))
					continue
				}
			}
		}
	}

	// 对菜单进行排序并返回
	return m.sortedMenuList(fatherMenuMap), nil
}

// GetAllMenuList 获取所有菜单列表
func (m *authMenuService) GetAllMenuList(ctx context.Context) ([]*model.Menu, error) {
	// 从数据库获取所有菜单
	menus, err := m.menuDao.GetAllMenus(ctx)
	if err != nil {
		m.l.Error("GetAllMenus failed", zap.Error(err))
		return nil, err
	}

	// 设置每个菜单的元数据
	for _, menu := range menus {
		m.setMenuMeta(menu)
	}

	return menus, nil
}

// UpdateMenu 更新菜单信息
func (m *authMenuService) UpdateMenu(ctx context.Context, menu model.Menu) error {
	return m.menuDao.UpdateMenu(ctx, &menu)
}

func (m *authMenuService) UpdateMenuStatus(ctx context.Context, menuID int, status string) error {
	return m.menuDao.UpdateMenuStatus(ctx, menuID, status)
}

// CreateMenu 创建新菜单
func (m *authMenuService) CreateMenu(ctx context.Context, menu model.Menu) error {
	return m.menuDao.CreateMenu(ctx, &menu)
}

// DeleteMenu 删除菜单
func (m *authMenuService) DeleteMenu(ctx context.Context, id string) error {
	return m.menuDao.DeleteMenu(ctx, id)
}

func (m *authMenuService) attachToFatherMenu(ctx context.Context, menu *model.Menu, fatherMenuMap map[int]*model.Menu, uniqueChildMap map[int]struct{}) error {
	// 获取父菜单
	fatherMenu, err := m.menuDao.GetMenuByID(ctx, int(menu.Pid))
	if err != nil {
		return err
	}

	// 设置父菜单的元数据
	m.setMenuMeta(fatherMenu)

	// 确保子菜单唯一性
	if _, exists := uniqueChildMap[menu.ID]; !exists {
		uniqueChildMap[menu.ID] = struct{}{}

		// 添加子菜单到父菜单
		if existingFather, ok := fatherMenuMap[fatherMenu.ID]; ok {
			existingFather.Children = append(existingFather.Children, menu)
		} else {
			fatherMenu.Children = append(fatherMenu.Children, menu)
			fatherMenuMap[fatherMenu.ID] = fatherMenu
		}
	}

	return nil
}

// sortedMenuList 根据ID对菜单进行排序并返回列表
func (m *authMenuService) sortedMenuList(fatherMenuMap map[int]*model.Menu) []*model.Menu {
	finalMenus := make([]*model.Menu, 0, len(fatherMenuMap))
	finalMenuIds := make([]int, 0, len(fatherMenuMap))

	for id := range fatherMenuMap {
		finalMenuIds = append(finalMenuIds, int(id))
	}

	sort.Ints(finalMenuIds)

	for _, id := range finalMenuIds {
		finalMenus = append(finalMenus, fatherMenuMap[int(id)])
	}

	return finalMenus
}

// setMenuMeta 设置菜单的元数据信息
func (m *authMenuService) setMenuMeta(menu *model.Menu) {
	menu.Meta = &model.MenuMeta{
		Icon:            menu.Icon,
		Title:           menu.Title,
		ShowMenu:        menu.Show,
		HideMenu:        !menu.Show,
		IgnoreKeepAlive: true,
	}

	menu.Key = menu.ID
	menu.Value = menu.ID
}

// getMenusByIDs 根据菜单ID列表获取对应的菜单对象
func (m *authMenuService) getMenusByIDs(ctx context.Context, menuIds []int) ([]*model.Menu, error) {
	menus := make([]*model.Menu, 0)

	for _, menuId := range menuIds {
		// 根据ID获取菜单信息
		menu, err := m.menuDao.GetMenuByID(ctx, int(menuId))
		if err != nil {
			m.l.Error("GetMenuByID failed", zap.Error(err))
			return nil, err
		}

		menus = append(menus, menu)
	}

	return menus, nil
}

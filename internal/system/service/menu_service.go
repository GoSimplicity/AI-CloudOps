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

package service

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type MenuService interface {
	GetMenus(ctx context.Context, pageNum, pageSize int, isTree bool) ([]*model.Menu, int, error)
	CreateMenu(ctx context.Context, menu *model.Menu) error
	GetMenuById(ctx context.Context, id int) (*model.Menu, error)
	UpdateMenu(ctx context.Context, menu *model.Menu) error
	DeleteMenu(ctx context.Context, id int) error
	GetMenuTree(ctx context.Context) ([]*model.Menu, error)
}

type menuService struct {
	menuDao dao.MenuDAO
	l       *zap.Logger
}

func NewMenuService(menuDao dao.MenuDAO, l *zap.Logger) MenuService {
	return &menuService{
		menuDao: menuDao,
		l:       l,
	}
}

// GetMenus 获取菜单列表,支持分页和树形结构
func (m *menuService) GetMenus(ctx context.Context, pageNum, pageSize int, isTree bool) ([]*model.Menu, int, error) {
	if pageNum < 1 || pageSize < 1 {
		m.l.Warn("分页参数无效", zap.Int("页码", pageNum), zap.Int("每页数量", pageSize))
		return nil, 0, errors.New("分页参数无效")
	}

	// 如果需要树形结构,则调用GetMenuTree
	if isTree {
		menus, err := m.menuDao.GetMenuTree(ctx)
		if err != nil {
			m.l.Error("获取菜单树失败", zap.Error(err))
			return nil, 0, err
		}
		return menus, len(menus), nil
	}

	return m.menuDao.ListMenus(ctx, pageNum, pageSize)
}

// CreateMenu 创建新菜单
func (m *menuService) CreateMenu(ctx context.Context, menu *model.Menu) error {
	if menu == nil {
		m.l.Warn("菜单不能为空")
		return errors.New("菜单不能为空")
	}

	return m.menuDao.CreateMenu(ctx, menu)
}

// GetMenuById 根据ID获取菜单
func (m *menuService) GetMenuById(ctx context.Context, id int) (*model.Menu, error) {
	if id <= 0 {
		m.l.Warn("菜单ID无效", zap.Int("ID", id))
		return nil, errors.New("菜单ID无效")
	}

	return m.menuDao.GetMenuById(ctx, id)
}

// UpdateMenu 更新菜单信息
func (m *menuService) UpdateMenu(ctx context.Context, menu *model.Menu) error {
	if menu == nil {
		m.l.Warn("菜单不能为空")
		return errors.New("菜单不能为空")
	}

	return m.menuDao.UpdateMenu(ctx, menu)
}

// DeleteMenu 删除指定ID的菜单
func (m *menuService) DeleteMenu(ctx context.Context, id int) error {
	if id <= 0 {
		m.l.Warn("菜单ID无效", zap.Int("ID", id))
		return errors.New("菜单ID无效")
	}

	return m.menuDao.DeleteMenu(ctx, id)
}

// GetMenuTree 获取菜单树形结构
func (m *menuService) GetMenuTree(ctx context.Context) ([]*model.Menu, error) {
	return m.menuDao.GetMenuTree(ctx)
}

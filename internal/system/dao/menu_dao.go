package dao

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
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

type AuthMenuDAO interface {
	// UpdateMenus 更新菜单
	UpdateMenus(ctx context.Context, menus []*model.Menu) error
	// UpdateMenuStatus 更新菜单状态
	UpdateMenuStatus(ctx context.Context, menuID int, status string) error
	// UpdateMenu 更新菜单
	UpdateMenu(ctx context.Context, menu *model.Menu) error
	// CreateMenu 创建菜单
	CreateMenu(ctx context.Context, menu *model.Menu) error
	// GetAllMenus 获取所有菜单
	GetAllMenus(ctx context.Context) ([]*model.Menu, error)
	// GetMenuByID 根据ID获取菜单
	GetMenuByID(ctx context.Context, id int) (*model.Menu, error)
	// GetMenuByFatherID 根据父亲ID获取菜单
	GetMenuByFatherID(ctx context.Context, id int) (*model.Menu, error)
	// DeleteMenu 通过ID删除菜单
	DeleteMenu(ctx context.Context, menuID string) error
}

type authMenuDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewAuthMenuDAO(db *gorm.DB, l *zap.Logger) AuthMenuDAO {
	return &authMenuDAO{
		db: db,
		l:  l,
	}
}

func (m *authMenuDAO) UpdateMenus(ctx context.Context, menus []*model.Menu) error {
	tx := m.db.WithContext(ctx).Begin() // 开始事务

	// 遍历每个菜单项，逐个更新
	for _, menu := range menus {
		if err := tx.Model(&menu).Updates(menu).Error; err != nil {
			tx.Rollback() // 出错时回滚
			m.l.Error("failed to update menu", zap.Error(err))
			return err
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		m.l.Error("failed to commit transaction for updating menus", zap.Error(err))
		return err
	}

	return nil
}

func (m *authMenuDAO) UpdateMenu(ctx context.Context, menu *model.Menu) error {
	if err := m.db.WithContext(ctx).Model(menu).Updates(map[string]interface{}{
		"title":     menu.Title,
		"name":      menu.Name,
		"path":      menu.Path,
		"type":      menu.Type,
		"orderNo":   menu.OrderNo,
		"icon":      menu.Icon,
		"component": menu.Component,
		"show":      menu.Show,
		"status":    menu.Status,
	}).Error; err != nil {
		m.l.Error("failed to update menu", zap.Int("menuID", menu.ID), zap.Error(err))
		return err
	}

	return nil
}

func (m *authMenuDAO) UpdateMenuStatus(ctx context.Context, menuID int, status string) error {
	if err := m.db.WithContext(ctx).Model(&model.Menu{}).Where("id = ?", menuID).Update("status", status).Error; err != nil {
		m.l.Error("failed to update menu status", zap.Int("menu_id", menuID), zap.String("status", status), zap.Error(err))
		return err
	}

	m.l.Info("menu status updated successfully", zap.Int("menu_id", menuID), zap.String("status", status))

	return nil
}

func (m *authMenuDAO) CreateMenu(ctx context.Context, menu *model.Menu) error {
	if err := m.db.WithContext(ctx).Create(menu).Error; err != nil {
		m.l.Error("failed to create menu", zap.Error(err))
		return err
	}

	return nil
}

func (m *authMenuDAO) GetMenuByID(ctx context.Context, id int) (*model.Menu, error) {
	var menu model.Menu

	if err := m.db.WithContext(ctx).Where("pid = ?", id).First(&menu).Error; err != nil {
		m.l.Error("failed to get menu by ID", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &menu, nil
}

func (m *authMenuDAO) GetMenuByFatherID(ctx context.Context, id int) (*model.Menu, error) {
	var menu model.Menu

	if err := m.db.WithContext(ctx).Where("pid = ?", id).First(&menu).Error; err != nil {
		m.l.Error("failed to get menu by ID", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &menu, nil
}

func (m *authMenuDAO) GetAllMenus(ctx context.Context) ([]*model.Menu, error) {
	var menus []*model.Menu

	if err := m.db.WithContext(ctx).Find(&menus).Error; err != nil {
		m.l.Error("failed to get all menus", zap.Error(err))
		return nil, err
	}

	return menus, nil
}

func (m *authMenuDAO) DeleteMenu(ctx context.Context, menuID string) error {
	if err := m.db.WithContext(ctx).Where("id = ?", menuID).Delete(&model.Menu{}).Error; err != nil {
		m.l.Error("failed to delete menu", zap.String("menuID", menuID), zap.Error(err))
		return err
	}

	return nil
}

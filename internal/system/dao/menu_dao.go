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

package dao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrMenuNotFound = errors.New("菜单不存在")
	ErrInvalidMenu  = errors.New("无效的菜单参数")
)

type MenuDAO interface {
	CreateMenu(ctx context.Context, menu *model.Menu) error
	GetMenuById(ctx context.Context, id int) (*model.Menu, error)
	UpdateMenu(ctx context.Context, menu *model.Menu) error
	DeleteMenu(ctx context.Context, id int) error
	ListMenus(ctx context.Context, page, pageSize int) ([]*model.Menu, int, error)
	GetMenuTree(ctx context.Context) ([]*model.Menu, error)
}

type menuDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewMenuDAO(db *gorm.DB, l *zap.Logger) MenuDAO {
	return &menuDAO{
		db: db,
		l:  l,
	}
}

// CreateMenu 创建菜单
func (m *menuDAO) CreateMenu(ctx context.Context, menu *model.Menu) error {
	if menu == nil {
		return ErrInvalidMenu
	}

	// 检查必填字段
	if menu.Name == "" {
		return errors.New("菜单名称不能为空")
	}

	if menu.Path == "" {
		return errors.New("菜单路径不能为空")
	}

	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查父菜单是否存在
		if menu.ParentID != 0 {
			var count int64
			if err := tx.Model(&model.Menu{}).Where("id = ? AND is_deleted = ?", menu.ParentID, 0).Count(&count).Error; err != nil {
				return fmt.Errorf("检查父菜单失败: %v", err)
			}
			if count == 0 {
				return errors.New("父菜单不存在")
			}
		}

		// 检查同级菜单名称是否重复
		var count int64
		if err := tx.Model(&model.Menu{}).Where("name = ? AND parent_id = ? AND is_deleted = ?", menu.Name, menu.ParentID, 0).Count(&count).Error; err != nil {
			return fmt.Errorf("检查菜单名称失败: %v", err)
		}
		if count > 0 {
			return errors.New("同级菜单名称已存在")
		}

		now := time.Now().Unix()
		menu.CreateTime = now
		menu.UpdateTime = now

		return tx.Create(menu).Error
	})
}

// GetMenuById 根据ID获取菜单
func (m *menuDAO) GetMenuById(ctx context.Context, id int) (*model.Menu, error) {
	if id <= 0 {
		return nil, errors.New("无效的菜单ID")
	}

	var menu model.Menu
	if err := m.db.WithContext(ctx).Where("id = ? AND is_deleted = ?", id, 0).First(&menu).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMenuNotFound
		}
		return nil, fmt.Errorf("查询菜单失败: %v", err)
	}

	return &menu, nil
}

// UpdateMenu 更新菜单
func (m *menuDAO) UpdateMenu(ctx context.Context, menu *model.Menu) error {
	if menu == nil {
		return errors.New("菜单对象不能为空")
	}
	if menu.ID <= 0 {
		return errors.New("无效的菜单ID")
	}
	if menu.Name == "" {
		return errors.New("菜单名称不能为空")
	}
	if menu.Path == "" {
		return errors.New("菜单路径不能为空")
	}

	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查菜单是否存在
		var exists bool
		if err := tx.Model(&model.Menu{}).Select("1").Where("id = ? AND is_deleted = ?", menu.ID, 0).Find(&exists).Error; err != nil {
			return fmt.Errorf("检查菜单是否存在失败: %v", err)
		}
		if !exists {
			return ErrMenuNotFound
		}

		// 检查是否有子菜单
		var childCount int64
		if err := tx.Model(&model.Menu{}).Where("parent_id = ? AND is_deleted = ?", menu.ID, 0).Count(&childCount).Error; err != nil {
			return fmt.Errorf("检查子菜单失败: %v", err)
		}

		// 获取原菜单信息
		var oldMenu model.Menu
		if err := tx.Where("id = ? AND is_deleted = ?", menu.ID, 0).First(&oldMenu).Error; err != nil {
			return fmt.Errorf("获取原菜单信息失败: %v", err)
		}

		// 如果有子菜单且尝试修改父级菜单ID,则不允许修改
		if childCount > 0 && oldMenu.ParentID != menu.ParentID {
			return errors.New("当前菜单存在子菜单,不能修改父级菜单")
		}

		// 检查父菜单是否存在且不能将菜单设置为自己的子菜单
		if menu.ParentID != 0 {
			if menu.ParentID == menu.ID {
				return errors.New("不能将菜单设置为自己的子菜单")
			}
			var count int64
			if err := tx.Model(&model.Menu{}).Where("id = ? AND is_deleted = ?", menu.ParentID, 0).Count(&count).Error; err != nil {
				return fmt.Errorf("检查父菜单失败: %v", err)
			}
			if count == 0 {
				return errors.New("父菜单不存在")
			}
		}

		// 检查同级菜单名称是否重复
		var count int64
		if err := tx.Model(&model.Menu{}).Where("name = ? AND parent_id = ? AND id != ? AND is_deleted = ?",
			menu.Name, menu.ParentID, menu.ID, 0).Count(&count).Error; err != nil {
			return fmt.Errorf("检查菜单名称失败: %v", err)
		}
		if count > 0 {
			return errors.New("同级菜单名称已存在")
		}

		// 更新菜单信息
		updates := map[string]interface{}{
			"name":        menu.Name,
			"parent_id":   menu.ParentID,
			"path":        menu.Path,
			"component":   menu.Component,
			"icon":        menu.Icon,
			"sort_order":  menu.SortOrder,
			"route_name":  menu.RouteName,
			"hidden":      menu.Hidden,
			"update_time": time.Now().Unix(),
		}

		result := tx.Model(&model.Menu{}).
			Where("id = ? AND is_deleted = ?", menu.ID, 0).
			Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("更新菜单失败: %v", result.Error)
		}
		if result.RowsAffected == 0 {
			return errors.New("菜单不存在或已被删除")
		}

		return nil
	})
}

// DeleteMenu 删除菜单
func (m *menuDAO) DeleteMenu(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("无效的菜单ID")
	}

	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查菜单是否存在
		var exists bool
		if err := tx.Model(&model.Menu{}).Select("1").Where("id = ? AND is_deleted = ?", id, 0).Find(&exists).Error; err != nil {
			return fmt.Errorf("检查菜单是否存在失败: %v", err)
		}
		if !exists {
			return ErrMenuNotFound
		}

		// 检查是否有子菜单
		var count int64
		if err := tx.Model(&model.Menu{}).Where("parent_id = ? AND is_deleted = ?", id, 0).Count(&count).Error; err != nil {
			return fmt.Errorf("检查子菜单失败: %v", err)
		}
		if count > 0 {
			return errors.New("存在子菜单,不能删除")
		}

		// 软删除菜单
		updates := map[string]interface{}{
			"is_deleted":  1,
			"update_time": time.Now().Unix(),
		}
		result := tx.Model(&model.Menu{}).Where("id = ? AND is_deleted = ?", id, 0).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("删除菜单失败: %v", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrMenuNotFound
		}
		return nil
	})
}

// ListMenus 分页获取菜单列表
func (m *menuDAO) ListMenus(ctx context.Context, page, pageSize int) ([]*model.Menu, int, error) {
	if page <= 0 {
		return nil, 0, errors.New("页码必须大于0")
	}
	if pageSize <= 0 || pageSize > 100 {
		return nil, 0, errors.New("每页条数必须在1-100之间")
	}

	var menus []*model.Menu
	var total int64

	db := m.db.WithContext(ctx).Model(&model.Menu{}).Where("is_deleted = ?", 0)

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取菜单总数失败: %v", err)
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Order("sort_order ASC, id DESC").Find(&menus).Error; err != nil {
		return nil, 0, fmt.Errorf("查询菜单列表失败: %v", err)
	}

	return menus, int(total), nil
}

// GetMenuTree 获取菜单树形结构
func (m *menuDAO) GetMenuTree(ctx context.Context) ([]*model.Menu, error) {
	// 预分配合适的初始容量
	menus := make([]*model.Menu, 0, 50)

	// 使用索引字段优化查询
	if err := m.db.WithContext(ctx).
		Select("id, name, parent_id, path, component, icon, sort_order, route_name, hidden, create_time, update_time").
		Where("is_deleted = ?", 0).
		Order("sort_order ASC, id ASC").
		Find(&menus).Error; err != nil {
		return nil, fmt.Errorf("查询菜单列表失败: %v", err)
	}

	// 预分配map容量
	menuMap := make(map[int]*model.Menu, len(menus))
	rootMenus := make([]*model.Menu, 0, len(menus)/3) // 假设大约1/3的菜单是根菜单

	// 第一次遍历,建立ID到菜单的映射
	for _, menu := range menus {
		if menu == nil {
			continue
		}
		menu.Children = make([]*model.Menu, 0, 4) // 预分配子菜单切片,假设平均4个子菜单
		menuMap[menu.ID] = menu
	}

	// 第二次遍历,构建树形结构
	for _, menu := range menus {
		if menu == nil {
			continue
		}
		if menu.ParentID == 0 {
			rootMenus = append(rootMenus, menu)
		} else {
			if parent, exists := menuMap[menu.ParentID]; exists {
				parent.Children = append(parent.Children, menu)
			} else {
				// 如果找不到父节点,作为根节点处理
				rootMenus = append(rootMenus, menu)
			}
		}
	}

	return rootMenus, nil
}

package menu

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MenuDAO interface {
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

func (m *menuDAO) UpdateMenus(ctx context.Context, menus []*model.Menu) error {
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

func (m *menuDAO) UpdateMenu(ctx context.Context, menu *model.Menu) error {
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

func (m *menuDAO) UpdateMenuStatus(ctx context.Context, menuID int, status string) error {
	if err := m.db.WithContext(ctx).Model(&model.Menu{}).Where("id = ?", menuID).Update("status", status).Error; err != nil {
		m.l.Error("failed to update menu status", zap.Int("menu_id", menuID), zap.String("status", status), zap.Error(err))
		return err
	}

	m.l.Info("menu status updated successfully", zap.Int("menu_id", menuID), zap.String("status", status))

	return nil
}

func (m *menuDAO) CreateMenu(ctx context.Context, menu *model.Menu) error {
	if err := m.db.WithContext(ctx).Create(menu).Error; err != nil {
		m.l.Error("failed to create menu", zap.Error(err))
		return err
	}

	return nil
}

func (m *menuDAO) GetMenuByID(ctx context.Context, id int) (*model.Menu, error) {
	var menu model.Menu

	if err := m.db.WithContext(ctx).Where("pid = ?", id).First(&menu).Error; err != nil {
		m.l.Error("failed to get menu by ID", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &menu, nil
}

func (m *menuDAO) GetMenuByFatherID(ctx context.Context, id int) (*model.Menu, error) {
	var menu model.Menu

	if err := m.db.WithContext(ctx).Where("pid = ?", id).First(&menu).Error; err != nil {
		m.l.Error("failed to get menu by ID", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &menu, nil
}

func (m *menuDAO) GetAllMenus(ctx context.Context) ([]*model.Menu, error) {
	var menus []*model.Menu

	if err := m.db.WithContext(ctx).Find(&menus).Error; err != nil {
		m.l.Error("failed to get all menus", zap.Error(err))
		return nil, err
	}

	return menus, nil
}

func (m *menuDAO) DeleteMenu(ctx context.Context, menuID string) error {
	if err := m.db.WithContext(ctx).Where("id = ?", menuID).Delete(&model.Menu{}).Error; err != nil {
		m.l.Error("failed to delete menu", zap.String("menuID", menuID), zap.Error(err))
		return err
	}

	return nil
}

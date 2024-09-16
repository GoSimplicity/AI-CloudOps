package auth

import (
	"context"
	"github.com/GoSimplicity/CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuthDAO interface {
	// GetRoleByRoleValue 通过角色值获取角色
	GetRoleByRoleValue(ctx context.Context, roleValue int) (*model.Role, error)
	// GetRoleByRoleID 通过角色ID获取角色
	GetRoleByRoleID(ctx context.Context, roleID int) (*model.Role, error)
	// CreateRole 创建角色
	CreateRole(ctx context.Context, role *model.Role) error
	// UpdateRole 更新角色
	UpdateRole(ctx context.Context, role *model.Role) error
	// UpdateRoleStatus 更新角色状态
	UpdateRoleStatus(ctx context.Context, id int, status string) error
	// GetApisByRoleID 通过角色ID获取API
	GetApisByRoleID(ctx context.Context, roleID int) ([]*model.Api, error)
	// GetAllRoles 获取所有角色
	GetAllRoles(ctx context.Context) ([]*model.Role, error)
	// UpdateMenus 更新菜单
	UpdateMenus(ctx context.Context, menus []*model.Menu) error
	// UpdateApis 更新API
	UpdateApis(ctx context.Context, apis []*model.Api) error
	// DeleteRole 删除角色
	DeleteRole(ctx context.Context, roleId string) error
	// GetAllApis 获取所有API
	GetAllApis(ctx context.Context) ([]*model.Api, error)
	// GetApiByID 通过ID获取API
	GetApiByID(ctx context.Context, apiID int) (*model.Api, error)
	// GetApiByTitle 通过标题获取API
	GetApiByTitle(ctx context.Context, title string) (*model.Api, error)
	// DeleteApi 通过ID删除API
	DeleteApi(ctx context.Context, apiID string) error
	// CreateApi 创建API
	CreateApi(ctx context.Context, api *model.Api) error
	// UpdateApi 更新API
	UpdateApi(ctx context.Context, api *model.Api) error
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

type authDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewAuthDAO(db *gorm.DB, l *zap.Logger) AuthDAO {
	return &authDAO{
		db: db,
		l:  l,
	}
}

func (a *authDAO) GetRoleByRoleValue(ctx context.Context, roleValue int) (*model.Role, error) {
	var role model.Role
	if err := a.db.WithContext(ctx).Where("role_value = ?", roleValue).First(&role).Error; err != nil {
		a.l.Error("failed to get role by roleValue", zap.Int("roleValue", roleValue), zap.Error(err))
		return nil, err
	}
	return &role, nil
}

func (a *authDAO) GetRoleByRoleID(ctx context.Context, roleID int) (*model.Role, error) {
	var role model.Role

	if err := a.db.WithContext(ctx).Where("id = ?", roleID).First(&role).Error; err != nil {
		a.l.Error("failed to get role by roleID", zap.Int("roleID", roleID), zap.Error(err))
		return nil, err
	}

	return &role, nil
}

func (a *authDAO) CreateRole(ctx context.Context, role *model.Role) error {
	if err := a.db.WithContext(ctx).Create(role).Error; err != nil {
		a.l.Error("failed to create role", zap.Error(err))
		return err
	}

	return nil
}

func (a *authDAO) UpdateRole(ctx context.Context, role *model.Role) error {
	if err := a.db.WithContext(ctx).Model(role).Where("id = ?", role.ID).Updates(map[string]interface{}{
		"role_name":  role.RoleName,
		"role_value": role.RoleValue,
		"remark":     role.Remark,
		"status":     role.Status,
	}).Error; err != nil {
		a.l.Error("failed to update role", zap.Error(err))
		return err
	}

	return nil
}

func (a *authDAO) UpdateRoleStatus(ctx context.Context, id int, status string) error {
	if err := a.db.WithContext(ctx).Model(model.Role{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": status,
	}).Error; err != nil {
		a.l.Error("update role status failed", zap.Error(err))
		return err
	}

	return nil
}

func (a *authDAO) UpdateMenus(ctx context.Context, menus []*model.Menu) error {
	tx := a.db.WithContext(ctx).Begin() // 开始事务

	// 遍历每个菜单项，逐个更新
	for _, menu := range menus {
		if err := tx.Model(&menu).Updates(menu).Error; err != nil {
			tx.Rollback() // 出错时回滚
			a.l.Error("failed to update menu", zap.Error(err))
			return err
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		a.l.Error("failed to commit transaction for updating menus", zap.Error(err))
		return err
	}

	return nil
}

func (a *authDAO) GetAllRoles(ctx context.Context) ([]*model.Role, error) {
	var roles []*model.Role

	if err := a.db.WithContext(ctx).Find(&roles).Error; err != nil {
		a.l.Error("failed to get all roles", zap.Error(err))
		return nil, err
	}

	return roles, nil
}

func (a *authDAO) UpdateApis(ctx context.Context, apis []*model.Api) error {
	tx := a.db.WithContext(ctx).Begin() // 开始事务

	// 遍历每个API项，逐个更新
	for _, api := range apis {
		if err := tx.Model(&api).Updates(api).Error; err != nil {
			tx.Rollback() // 出错时回滚
			a.l.Error("failed to update api", zap.Error(err))
			return err
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		a.l.Error("failed to commit transaction for updating apis", zap.Error(err))
		return err
	}

	return nil
}

func (a *authDAO) DeleteRole(ctx context.Context, id string) error {
	if err := a.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Role{}).Error; err != nil {
		a.l.Error("failed to delete role", zap.Error(err))
		return err
	}

	return nil
}

// GetApisByRoleID 根据角色ID获取API列表
func (a *authDAO) GetApisByRoleID(ctx context.Context, roleID int) ([]*model.Api, error) {
	var apis []*model.Api

	// 使用联表查询，假设角色和API的关联表为 `role_apis`
	err := a.db.WithContext(ctx).
		Table("role_apis").
		Select("apis.*").
		Joins("join apis on role_apis.api_id = apis.id").
		Where("role_apis.role_id = ?", roleID).
		Find(&apis).Error
	if err != nil {
		return nil, err
	}

	return apis, nil
}

func (a *authDAO) GetAllApis(ctx context.Context) ([]*model.Api, error) {
	var apis []*model.Api

	if err := a.db.WithContext(ctx).Find(&apis).Error; err != nil {
		a.l.Error("failed to get all APIs", zap.Error(err))
		return nil, err
	}

	return apis, nil
}

func (a *authDAO) GetApiByID(ctx context.Context, apiID int) (*model.Api, error) {
	var api model.Api

	if err := a.db.WithContext(ctx).Where("id = ?", apiID).First(&api).Error; err != nil {
		a.l.Error("failed to get API by ID", zap.Int("apiID", apiID), zap.Error(err))
		return nil, err
	}

	return &api, nil
}

func (a *authDAO) GetApiByTitle(ctx context.Context, title string) (*model.Api, error) {
	var api model.Api

	if err := a.db.WithContext(ctx).Where("title = ?", title).First(&api).Error; err != nil {
		a.l.Error("failed to get API by title", zap.String("title", title), zap.Error(err))
		return nil, err
	}

	return &api, nil
}

func (a *authDAO) DeleteApi(ctx context.Context, apiID string) error {
	if err := a.db.WithContext(ctx).Where("id = ?", apiID).Delete(&model.Api{}).Error; err != nil {
		a.l.Error("failed to delete API", zap.String("apiID", apiID), zap.Error(err))
		return err
	}

	return nil
}

func (a *authDAO) CreateApi(ctx context.Context, api *model.Api) error {
	if err := a.db.WithContext(ctx).Create(api).Error; err != nil {
		a.l.Error("failed to create API", zap.Error(err))
		return err
	}

	return nil
}

func (a *authDAO) UpdateApi(ctx context.Context, api *model.Api) error {
	if err := a.db.WithContext(ctx).Model(api).Updates(api).Error; err != nil {
		a.l.Error("failed to update API", zap.Int("apiID", api.ID), zap.Error(err))
		return err
	}

	return nil
}

func (a *authDAO) UpdateMenu(ctx context.Context, menu *model.Menu) error {
	if err := a.db.WithContext(ctx).Model(menu).Updates(map[string]interface{}{
		"title":     menu.Title,
		"name":      menu.Name,
		"path":      menu.Path,
		"component": menu.Component,
		"show":      menu.Show,
	}).Error; err != nil {
		a.l.Error("failed to update menu", zap.Int("menuID", menu.ID), zap.Error(err))
		return err
	}

	return nil
}

func (a *authDAO) CreateMenu(ctx context.Context, menu *model.Menu) error {
	if err := a.db.WithContext(ctx).Create(menu).Error; err != nil {
		a.l.Error("failed to create menu", zap.Error(err))
		return err
	}
	return nil
}

func (a *authDAO) GetMenuByID(ctx context.Context, id int) (*model.Menu, error) {
	var menu model.Menu

	if err := a.db.WithContext(ctx).Where("pid = ?", id).First(&menu).Error; err != nil {
		a.l.Error("failed to get menu by ID", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &menu, nil
}

func (a *authDAO) GetMenuByFatherID(ctx context.Context, id int) (*model.Menu, error) {
	var menu model.Menu

	if err := a.db.WithContext(ctx).Where("pid = ?", id).First(&menu).Error; err != nil {
		a.l.Error("failed to get menu by ID", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &menu, nil
}

func (a *authDAO) GetAllMenus(ctx context.Context) ([]*model.Menu, error) {
	var menus []*model.Menu

	if err := a.db.WithContext(ctx).Find(&menus).Error; err != nil {
		a.l.Error("failed to get all menus", zap.Error(err))
		return nil, err
	}

	return menus, nil
}

func (a *authDAO) DeleteMenu(ctx context.Context, menuID string) error {
	if err := a.db.WithContext(ctx).Where("id = ?", menuID).Delete(&model.Menu{}).Error; err != nil {
		a.l.Error("failed to delete menu", zap.String("menuID", menuID), zap.Error(err))
		return err
	}

	return nil
}

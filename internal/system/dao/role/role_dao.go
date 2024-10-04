package role

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RoleDAO interface {
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
	// DeleteRole 删除角色
	DeleteRole(ctx context.Context, roleId string) error
}

type roleDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewRoleDAO(db *gorm.DB, l *zap.Logger) RoleDAO {
	return &roleDAO{
		db: db,
		l:  l,
	}
}

func (r *roleDAO) GetRoleByRoleValue(ctx context.Context, roleValue int) (*model.Role, error) {
	var role model.Role
	if err := r.db.WithContext(ctx).Where("role_value = ?", roleValue).First(&role).Error; err != nil {
		r.l.Error("failed to get role by roleValue", zap.Int("roleValue", roleValue), zap.Error(err))
		return nil, err
	}
	return &role, nil
}

func (r *roleDAO) GetRoleByRoleID(ctx context.Context, roleID int) (*model.Role, error) {
	var role model.Role

	if err := r.db.WithContext(ctx).Where("id = ?", roleID).First(&role).Error; err != nil {
		r.l.Error("failed to get role by roleID", zap.Int("roleID", roleID), zap.Error(err))
		return nil, err
	}

	return &role, nil
}

func (r *roleDAO) CreateRole(ctx context.Context, role *model.Role) error {
	if err := r.db.WithContext(ctx).Create(role).Error; err != nil {
		r.l.Error("failed to create role", zap.Error(err))
		return err
	}

	return nil
}

func (r *roleDAO) UpdateRole(ctx context.Context, role *model.Role) error {
	if err := r.db.WithContext(ctx).Model(role).Where("id = ?", role.ID).Updates(map[string]interface{}{
		"role_name":  role.RoleName,
		"role_value": role.RoleValue,
		"remark":     role.Remark,
		"status":     role.Status,
	}).Error; err != nil {
		r.l.Error("failed to update role", zap.Error(err))
		return err
	}

	return nil
}

func (r *roleDAO) UpdateRoleStatus(ctx context.Context, id int, status string) error {
	if err := r.db.WithContext(ctx).Model(model.Role{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": status,
	}).Error; err != nil {
		r.l.Error("update role status failed", zap.Error(err))
		return err
	}

	return nil
}

func (r *roleDAO) GetAllRoles(ctx context.Context) ([]*model.Role, error) {
	var roles []*model.Role

	if err := r.db.WithContext(ctx).Find(&roles).Error; err != nil {
		r.l.Error("failed to get all roles", zap.Error(err))
		return nil, err
	}

	return roles, nil
}

func (r *roleDAO) DeleteRole(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Role{}).Error; err != nil {
		r.l.Error("failed to delete role", zap.Error(err))
		return err
	}

	return nil
}

// GetApisByRoleID 根据角色ID获取API列表
func (r *roleDAO) GetApisByRoleID(ctx context.Context, roleID int) ([]*model.Api, error) {
	var apis []*model.Api

	// 使用联表查询，假设角色和API的关联表为 `role_apis`
	err := r.db.WithContext(ctx).
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

package api

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ApiDAO interface {
	// GetApisByRoleID 通过角色ID获取API
	GetApisByRoleID(ctx context.Context, roleID int) ([]*model.Api, error)
	// UpdateApis 更新API
	UpdateApis(ctx context.Context, apis []*model.Api) error
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
}

type apiDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewApiDAO(db *gorm.DB, l *zap.Logger) ApiDAO {
	return &apiDAO{
		db: db,
		l:  l,
	}
}

func (a *apiDAO) UpdateApis(ctx context.Context, apis []*model.Api) error {
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

// GetApisByRoleID 根据角色ID获取API列表
func (a *apiDAO) GetApisByRoleID(ctx context.Context, roleID int) ([]*model.Api, error) {
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

func (a *apiDAO) GetAllApis(ctx context.Context) ([]*model.Api, error) {
	var apis []*model.Api

	if err := a.db.WithContext(ctx).Find(&apis).Error; err != nil {
		a.l.Error("failed to get all APIs", zap.Error(err))
		return nil, err
	}

	return apis, nil
}

func (a *apiDAO) GetApiByID(ctx context.Context, apiID int) (*model.Api, error) {
	var api model.Api

	if err := a.db.WithContext(ctx).Where("id = ?", apiID).First(&api).Error; err != nil {
		a.l.Error("failed to get API by ID", zap.Int("apiID", apiID), zap.Error(err))
		return nil, err
	}

	return &api, nil
}

func (a *apiDAO) GetApiByTitle(ctx context.Context, title string) (*model.Api, error) {
	var api model.Api

	if err := a.db.WithContext(ctx).Where("title = ?", title).First(&api).Error; err != nil {
		a.l.Error("failed to get API by title", zap.String("title", title), zap.Error(err))
		return nil, err
	}

	return &api, nil
}

func (a *apiDAO) DeleteApi(ctx context.Context, apiID string) error {
	if err := a.db.WithContext(ctx).Where("id = ?", apiID).Delete(&model.Api{}).Error; err != nil {
		a.l.Error("failed to delete API", zap.String("apiID", apiID), zap.Error(err))
		return err
	}

	return nil
}

func (a *apiDAO) CreateApi(ctx context.Context, api *model.Api) error {
	if err := a.db.WithContext(ctx).Create(api).Error; err != nil {
		a.l.Error("failed to create API", zap.Error(err))
		return err
	}

	return nil
}

func (a *apiDAO) UpdateApi(ctx context.Context, api *model.Api) error {
	if err := a.db.WithContext(ctx).Model(api).Updates(api).Error; err != nil {
		a.l.Error("failed to update API", zap.Int("apiID", api.ID), zap.Error(err))
		return err
	}

	return nil
}

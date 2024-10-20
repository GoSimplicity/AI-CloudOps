package ali_resource

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AliResourceDAO interface {
	// Create 在数据库中创建一个新的阿里云资源记录
	Create(ctx context.Context, config *model.TerraformConfig) (int, error)
	// Get 从数据库中获取指定 ID 的阿里云资源记录
	Get(ctx context.Context, id int) (*model.TerraformConfig, error)
	// Update 更新数据库中指定 ID 的阿里云资源记录
	Update(ctx context.Context, id int, updatedConfig *model.TerraformConfig) error
	// Delete 删除数据库中指定 ID 的阿里云资源记录
	Delete(ctx context.Context, id int) error
}

type aliResourceDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewAliResourceDAO(db *gorm.DB, logger *zap.Logger) AliResourceDAO {
	return &aliResourceDAO{
		db: db,
		l:  logger,
	}
}

func (dao *aliResourceDAO) Create(ctx context.Context, config *model.TerraformConfig) (int, error) {
	if config == nil {
		return 0, errors.New("config cannot be nil")
	}

	// 使用 GORM 插入到数据库
	if err := dao.db.WithContext(ctx).Create(config).Error; err != nil {
		dao.l.Error("Failed to create resource record", zap.Error(err))
		return 0, fmt.Errorf("failed to create resource: %w", err)
	}

	dao.l.Info("Resource created successfully", zap.Int("id", config.ID))
	return config.ID, nil
}

// Get 从数据库中获取指定 ID 的阿里云资源记录
func (dao *aliResourceDAO) Get(ctx context.Context, id int) (*model.TerraformConfig, error) {
	var config model.TerraformConfig

	if err := dao.db.WithContext(ctx).First(&config, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dao.l.Warn("Resource not found", zap.Int("id", id))
			return nil, fmt.Errorf("resource with ID %d not found", id)
		}
		dao.l.Error("Failed to retrieve resource", zap.Int("id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to get resource with ID %d: %w", id, err)
	}

	dao.l.Info("Resource retrieved successfully", zap.Int("id", id))
	return &config, nil
}

// Update 更新数据库中指定 ID 的阿里云资源记录
func (dao *aliResourceDAO) Update(ctx context.Context, id int, updatedConfig *model.TerraformConfig) error {
	if updatedConfig == nil {
		return errors.New("updatedConfig cannot be nil")
	}

	// 确保更新的配置 ID 与目标 ID 一致
	updatedConfig.ID = id

	// 使用 Select 只更新需要的字段，防止覆盖其他字段
	if err := dao.db.WithContext(ctx).Model(&model.TerraformConfig{}).Where("id = ?", id).Updates(updatedConfig).Error; err != nil {
		dao.l.Error("Failed to update resource", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("failed to update resource with ID %d: %w", id, err)
	}

	// 检查是否有记录被更新
	var rowsAffected int64
	dao.db.WithContext(ctx).Model(&model.TerraformConfig{}).Where("id = ?", id).Count(&rowsAffected)
	if rowsAffected == 0 {
		dao.l.Warn("No resource found to update", zap.Int("id", id))
		return fmt.Errorf("no resource found with ID %d to update", id)
	}

	dao.l.Info("Resource updated successfully", zap.Int("id", id))
	return nil
}

// Delete 删除数据库中指定 ID 的阿里云资源记录
func (dao *aliResourceDAO) Delete(ctx context.Context, id int) error {
	// 执行软删除
	if err := dao.db.WithContext(ctx).Delete(&model.TerraformConfig{}, id).Error; err != nil {
		dao.l.Error("Failed to delete resource", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("failed to delete resource with ID %d: %w", id, err)
	}

	dao.l.Info("Resource deleted successfully", zap.Int("id", id))
	return nil
}

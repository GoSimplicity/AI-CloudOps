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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CloudAccountRegionDAO interface {
	Create(ctx context.Context, region *model.CloudAccountRegion) error
	BatchCreate(ctx context.Context, regions []*model.CloudAccountRegion) error
	Update(ctx context.Context, region *model.CloudAccountRegion) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.CloudAccountRegion, error)
	GetList(ctx context.Context, req *model.GetCloudAccountRegionListReq) ([]*model.CloudAccountRegion, int64, error)
	GetByCloudAccountID(ctx context.Context, cloudAccountID int) ([]*model.CloudAccountRegion, error)
	GetByCloudAccountAndRegion(ctx context.Context, cloudAccountID int, region string) (*model.CloudAccountRegion, error)
	UpdateStatus(ctx context.Context, id int, status model.CloudAccountRegionStatus) error
	ClearDefaultRegion(ctx context.Context, cloudAccountID int) error
	GetResourceCountByRegion(ctx context.Context, regionID int) (int64, error)
}

type cloudAccountRegionDAO struct {
	logger *zap.Logger
	db     *gorm.DB
}

func NewCloudAccountRegionDAO(db *gorm.DB, logger *zap.Logger) CloudAccountRegionDAO {
	return &cloudAccountRegionDAO{
		logger: logger,
		db:     db,
	}
}

// Create 创建云账号区域关联
func (d *cloudAccountRegionDAO) Create(ctx context.Context, region *model.CloudAccountRegion) error {
	if err := d.db.WithContext(ctx).Create(region).Error; err != nil {
		d.logger.Error("创建云账号区域关联失败", zap.Error(err))
		return err
	}

	return nil
}

// BatchCreate 批量创建云账号区域关联
func (d *cloudAccountRegionDAO) BatchCreate(ctx context.Context, regions []*model.CloudAccountRegion) error {
	if err := d.db.WithContext(ctx).Create(regions).Error; err != nil {
		d.logger.Error("批量创建云账号区域关联失败", zap.Error(err))
		return err
	}

	return nil
}

// Update 更新云账号区域关联
func (d *cloudAccountRegionDAO) Update(ctx context.Context, region *model.CloudAccountRegion) error {
	if err := d.db.WithContext(ctx).Model(region).Updates(region).Error; err != nil {
		d.logger.Error("更新云账号区域关联失败", zap.Error(err))
		return err
	}

	return nil
}

// Delete 删除云账号区域关联
func (d *cloudAccountRegionDAO) Delete(ctx context.Context, id int) error {
	if err := d.db.WithContext(ctx).Delete(&model.CloudAccountRegion{}, id).Error; err != nil {
		d.logger.Error("删除云账号区域关联失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	return nil
}

// GetByID 根据ID获取云账号区域关联详情
func (d *cloudAccountRegionDAO) GetByID(ctx context.Context, id int) (*model.CloudAccountRegion, error) {
	var region model.CloudAccountRegion

	err := d.db.WithContext(ctx).Preload("CloudAccount").Where("id = ?", id).First(&region).Error
	if err != nil {
		d.logger.Error("根据ID获取云账号区域关联详情失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &region, nil
}

// GetList 获取云账号区域关联列表
func (d *cloudAccountRegionDAO) GetList(ctx context.Context, req *model.GetCloudAccountRegionListReq) ([]*model.CloudAccountRegion, int64, error) {
	var regions []*model.CloudAccountRegion
	var total int64

	query := d.db.WithContext(ctx).Model(&model.CloudAccountRegion{})

	// 添加查询条件
	if req.CloudAccountID != 0 {
		query = query.Where("cloud_account_id = ?", req.CloudAccountID)
	}

	if req.Region != "" {
		query = query.Where("region = ?", req.Region)
	}

	if req.Status != 0 {
		query = query.Where("status = ?", req.Status)
	}

	if req.Search != "" {
		query = query.Where("region LIKE ? OR region_name LIKE ?", "%"+req.Search+"%", "%"+req.Search+"%")
	}

	// 计算总数
	err := query.Count(&total).Error
	if err != nil {
		d.logger.Error("获取云账号区域关联总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	err = query.
		Preload("CloudAccount").
		Order("created_at DESC").
		Limit(req.Size).
		Offset(offset).
		Find(&regions).Error
	if err != nil {
		d.logger.Error("获取云账号区域关联列表失败", zap.Error(err))
		return nil, 0, err
	}

	return regions, total, nil
}

// GetByCloudAccountID 根据云账号ID获取区域列表
func (d *cloudAccountRegionDAO) GetByCloudAccountID(ctx context.Context, cloudAccountID int) ([]*model.CloudAccountRegion, error) {
	var regions []*model.CloudAccountRegion

	err := d.db.WithContext(ctx).
		Where("cloud_account_id = ?", cloudAccountID).
		Order("is_default DESC, created_at ASC").
		Find(&regions).Error
	if err != nil {
		d.logger.Error("根据云账号ID获取区域列表失败", zap.Error(err), zap.Int("cloudAccountID", cloudAccountID))
		return nil, err
	}

	return regions, nil
}

// GetByCloudAccountAndRegion 根据云账号ID和区域获取区域关联
func (d *cloudAccountRegionDAO) GetByCloudAccountAndRegion(ctx context.Context, cloudAccountID int, region string) (*model.CloudAccountRegion, error) {
	var regionItem model.CloudAccountRegion

	err := d.db.WithContext(ctx).
		Where("cloud_account_id = ? AND region = ?", cloudAccountID, region).
		First(&regionItem).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		d.logger.Error("根据云账号ID和区域获取区域关联失败", zap.Error(err))
		return nil, err
	}

	return &regionItem, nil
}

// UpdateStatus 更新云账号区域状态
func (d *cloudAccountRegionDAO) UpdateStatus(ctx context.Context, id int, status model.CloudAccountRegionStatus) error {
	if err := d.db.WithContext(ctx).
		Model(&model.CloudAccountRegion{}).
		Where("id = ?", id).
		Update("status", status).Error; err != nil {
		d.logger.Error("更新云账号区域状态失败", zap.Error(err), zap.Int("id", id), zap.Int8("status", int8(status)))
		return err
	}

	return nil
}

// ClearDefaultRegion 清除指定云账号的所有默认区域标记
func (d *cloudAccountRegionDAO) ClearDefaultRegion(ctx context.Context, cloudAccountID int) error {
	if err := d.db.WithContext(ctx).
		Model(&model.CloudAccountRegion{}).
		Where("cloud_account_id = ? AND is_default = ?", cloudAccountID, true).
		Update("is_default", false).Error; err != nil {
		d.logger.Error("清除默认区域失败", zap.Error(err), zap.Int("cloudAccountID", cloudAccountID))
		return err
	}

	return nil
}

// GetResourceCountByRegion 获取指定区域下的资源数量
func (d *cloudAccountRegionDAO) GetResourceCountByRegion(ctx context.Context, regionID int) (int64, error) {
	var count int64

	err := d.db.WithContext(ctx).
		Model(&model.TreeCloudResource{}).
		Where("cloud_account_region_id = ?", regionID).
		Count(&count).Error
	if err != nil {
		d.logger.Error("获取区域资源数量失败", zap.Error(err), zap.Int("regionID", regionID))
		return 0, err
	}

	return count, nil
}

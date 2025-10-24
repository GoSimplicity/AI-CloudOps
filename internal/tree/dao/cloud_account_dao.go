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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CloudAccountDAO interface {
	Create(ctx context.Context, account *model.CloudAccount) error
	CreateWithTransaction(ctx context.Context, fn func(tx interface{}) error) error
	CreateInTransaction(ctx context.Context, account *model.CloudAccount, tx interface{}) error
	CreateRegionInTransaction(ctx context.Context, region *model.CloudAccountRegion, tx interface{}) error
	Update(ctx context.Context, account *model.CloudAccount) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.CloudAccount, error)
	GetList(ctx context.Context, req *model.GetCloudAccountListReq) ([]*model.CloudAccount, int64, error)
	UpdateStatus(ctx context.Context, id int, status model.CloudAccountStatus) error
	GetByProviderAndRegion(ctx context.Context, provider model.CloudProvider, region string) ([]*model.CloudAccount, error)
}

type cloudAccountDAO struct {
	logger *zap.Logger
	db     *gorm.DB
}

func NewCloudAccountDAO(db *gorm.DB, logger *zap.Logger) CloudAccountDAO {
	return &cloudAccountDAO{
		logger: logger,
		db:     db,
	}
}

// Create 创建云账户
func (d *cloudAccountDAO) Create(ctx context.Context, account *model.CloudAccount) error {
	if err := d.db.WithContext(ctx).Create(account).Error; err != nil {
		d.logger.Error("创建云账户失败", zap.Error(err))
		return err
	}

	return nil
}

// Update 更新云账户
func (d *cloudAccountDAO) Update(ctx context.Context, account *model.CloudAccount) error {
	if err := d.db.WithContext(ctx).Model(account).Updates(account).Error; err != nil {
		d.logger.Error("更新云账户失败", zap.Error(err))
		return err
	}

	return nil
}

// Delete 删除云账户
func (d *cloudAccountDAO) Delete(ctx context.Context, id int) error {
	if err := d.db.WithContext(ctx).Delete(&model.CloudAccount{}, id).Error; err != nil {
		d.logger.Error("删除云账户失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	return nil
}

// GetByID 根据ID获取云账户详情
func (d *cloudAccountDAO) GetByID(ctx context.Context, id int) (*model.CloudAccount, error) {
	var account model.CloudAccount

	err := d.db.WithContext(ctx).
		Preload("Regions").
		Preload("CloudResources").
		Where("id = ?", id).
		First(&account).Error
	if err != nil {
		d.logger.Error("根据ID获取云账户详情失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &account, nil
}

// GetList 获取云账户列表
func (d *cloudAccountDAO) GetList(ctx context.Context, req *model.GetCloudAccountListReq) ([]*model.CloudAccount, int64, error) {
	var accounts []*model.CloudAccount
	var total int64

	query := d.db.WithContext(ctx).Model(&model.CloudAccount{})

	// 添加查询条件
	if req.Provider != 0 {
		query = query.Where("provider = ?", req.Provider)
	}

	if req.Status != 0 {
		query = query.Where("status = ?", req.Status)
	}

	if req.Search != "" {
		query = query.Where("name LIKE ? OR account_name LIKE ?", "%"+req.Search+"%", "%"+req.Search+"%")
	}

	// 计算总数
	err := query.Count(&total).Error
	if err != nil {
		d.logger.Error("获取云账户总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	err = query.
		Order("created_at DESC").
		Limit(req.Size).
		Offset(offset).
		Find(&accounts).Error
	if err != nil {
		d.logger.Error("获取云账户列表失败", zap.Error(err))
		return nil, 0, err
	}

	return accounts, total, nil
}

// UpdateStatus 更新云账户状态
func (d *cloudAccountDAO) UpdateStatus(ctx context.Context, id int, status model.CloudAccountStatus) error {
	if err := d.db.WithContext(ctx).
		Model(&model.CloudAccount{}).
		Where("id = ?", id).
		Update("status", status).Error; err != nil {
		d.logger.Error("更新云账户状态失败", zap.Error(err), zap.Int("id", id), zap.Int8("status", int8(status)))
		return err
	}

	return nil
}

// GetByProviderAndRegion 根据云厂商和区域获取云账户列表
func (d *cloudAccountDAO) GetByProviderAndRegion(ctx context.Context, provider model.CloudProvider, region string) ([]*model.CloudAccount, error) {
	var accounts []*model.CloudAccount

	query := d.db.WithContext(ctx).Where("provider = ?", provider)
	// 注：这里需要根据新的数据结构调整查询逻辑
	// 现在Region信息存储在 CloudAccountRegion 表中
	if region != "" {
		query = query.
			Joins("JOIN cl_cloud_account_region ON cl_cloud_account.id = cl_cloud_account_region.cloud_account_id").
			Where("cl_cloud_account_region.region = ?", region)
	}

	err := query.Find(&accounts).Error
	if err != nil {
		d.logger.Error("根据云厂商和区域获取云账户列表失败", zap.Error(err))
		return nil, err
	}

	return accounts, nil
}

// CreateWithTransaction 使用事务创建云账户
func (d *cloudAccountDAO) CreateWithTransaction(ctx context.Context, fn func(tx interface{}) error) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// CreateInTransaction 在事务中创建云账户
func (d *cloudAccountDAO) CreateInTransaction(ctx context.Context, account *model.CloudAccount, tx interface{}) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok {
		return errors.New("事务类型转换失败")
	}

	if err := gormTx.WithContext(ctx).Create(account).Error; err != nil {
		d.logger.Error("在事务中创建云账户失败", zap.Error(err))
		return err
	}

	return nil
}

// CreateRegionInTransaction 在事务中创建区域关联
func (d *cloudAccountDAO) CreateRegionInTransaction(ctx context.Context, region *model.CloudAccountRegion, tx interface{}) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok {
		return errors.New("事务类型转换失败")
	}

	if err := gormTx.WithContext(ctx).Create(region).Error; err != nil {
		d.logger.Error("在事务中创建区域关联失败", zap.Error(err))
		return err
	}

	return nil
}

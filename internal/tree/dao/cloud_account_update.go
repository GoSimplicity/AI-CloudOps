package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrCloudAccountRegionInUse = errors.New("地域仍有关联云资源，请先处理资源关联再移除")

// UpdateAccountAndRegions 同一父账户行锁内更新配置及地域差量，保留未改地域身份
func (d *cloudAccountDAO) UpdateAccountAndRegions(ctx context.Context, update *model.CloudAccount, fields []string, regions []model.CreateCloudAccountRegionItem) error {
	if update == nil || update.ID <= 0 {
		return fmt.Errorf("云账户ID无效")
	}
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account model.CloudAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, update.ID).Error; err != nil {
			return err
		}
		if len(fields) > 0 {
			if err := tx.Model(&model.CloudAccount{}).Where("id = ?", update.ID).Select(fields).Updates(update).Error; err != nil {
				return err
			}
		}
		if regions == nil {
			return nil
		}
		if len(regions) == 0 {
			return fmt.Errorf("账户至少需要一个地域")
		}
		var existing []model.CloudAccountRegion
		if err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).Where("cloud_account_id = ?", account.ID).Find(&existing).Error; err != nil {
			return err
		}
		selected := make(map[string]model.CreateCloudAccountRegionItem, len(regions))
		for _, region := range regions {
			selected[region.Region] = region
		}
		byCode := make(map[string]model.CloudAccountRegion, len(existing))
		for _, region := range existing {
			byCode[region.Region] = region
			if _, keep := selected[region.Region]; keep || region.DeletedAt != 0 {
				continue
			}
			var references int64
			if err := tx.Model(&model.TreeCloudResource{}).Where("cloud_account_region_id = ?", region.ID).Count(&references).Error; err != nil {
				return err
			}
			if references > 0 {
				return ErrCloudAccountRegionInUse
			}
			if err := tx.Where("id = ? AND cloud_account_id = ?", region.ID, account.ID).Delete(&model.CloudAccountRegion{}).Error; err != nil {
				return err
			}
		}
		for _, item := range regions {
			region, exists := byCode[item.Region]
			if !exists {
				created := &model.CloudAccountRegion{CloudAccountID: account.ID, Region: item.Region, RegionName: item.RegionName, Description: item.Description, IsDefault: item.IsDefault, Status: model.CloudAccountRegionEnabled, CreateUserID: account.CreateUserID, CreateUserName: account.CreateUserName}
				if err := tx.Omit("CloudAccount").Create(created).Error; err != nil {
					return err
				}
				continue
			}
			if region.DeletedAt == 0 && region.RegionName == item.RegionName && region.Description == item.Description && region.IsDefault == item.IsDefault {
				continue
			}
			values := map[string]interface{}{"region_name": item.RegionName, "description": item.Description, "is_default": item.IsDefault, "deleted_at": 0}
			if err := tx.Unscoped().Model(&model.CloudAccountRegion{}).Where("id = ? AND cloud_account_id = ?", region.ID, account.ID).Updates(values).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, ErrCloudAccountRegionInUse) {
		return databaseLogger.WrapDatabaseError("云账户更新事务失败", err)
	}
	return err
}

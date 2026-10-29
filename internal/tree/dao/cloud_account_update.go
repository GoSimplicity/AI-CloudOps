package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
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
		if update.ExpectedCredentialVersion != nil {
			if account.CredentialVersion != *update.ExpectedCredentialVersion {
				return errors.New("账户凭据已变更，请刷新后重试")
			}
			update.CredentialVersion = account.CredentialVersion + 1
			fields = append(fields, "credential_version")
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
			if err := treeUtils.ValidateCloudRegion(account.Provider, region.Region); err != nil {
				return err
			}
			selected[region.Region] = region
		}
		byCode := make(map[string]model.CloudAccountRegion, len(existing))
		for _, region := range existing {
			byCode[region.Region] = region
			if _, keep := selected[region.Region]; keep || region.DeletedAt != 0 {
				continue
			}
			references, err := countCloudResources(tx, "cloud_account_region_id", region.ID)
			if err != nil {
				return err
			}
			if references > 0 {
				return ErrCloudAccountRegionInUse
			}
			if err := tx.Where("id = ? AND cloud_account_id = ?", region.ID, account.ID).Delete(&model.CloudAccountRegion{}).Error; err != nil {
				return err
			}
		}
		if account.Provider == model.ProviderHuawei {
			for _, item := range regions {
				project := item.ProjectID
				if project == nil {
					if old, ok := byCode[item.Region]; ok {
						project = old.ProjectID
					}
				}
				if project == nil || !treeUtils.ValidHuaweiProjectID(*project) {
					return errors.New("华为云地域必须指定项目ID")
				}
			}
		}
		for _, item := range regions {
			region, exists := byCode[item.Region]
			if !exists {
				created := &model.CloudAccountRegion{CloudAccountID: account.ID, Region: item.Region, RegionName: item.RegionName, Description: item.Description, ProjectID: item.ProjectID, IsDefault: item.IsDefault, Status: model.CloudAccountRegionEnabled, CreateUserID: account.CreateUserID, CreateUserName: account.CreateUserName}
				if err := tx.Omit("CloudAccount").Create(created).Error; err != nil {
					return err
				}
				continue
			}
			projectChanged := item.ProjectID != nil && (region.ProjectID == nil || *item.ProjectID != *region.ProjectID)
			if region.DeletedAt == 0 && region.RegionName == item.RegionName && region.Description == item.Description && region.IsDefault == item.IsDefault && !projectChanged {
				continue
			}
			values := map[string]interface{}{"region_name": item.RegionName, "description": item.Description, "is_default": item.IsDefault, "deleted_at": 0}
			if item.ProjectID != nil {
				values["project_id"] = *item.ProjectID
			}
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

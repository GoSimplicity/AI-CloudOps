package dao

import (
	"context"
	"errors"
	"sort"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrCloudAccountInUse = errors.New("云账户仍有关联云资源")

func countCloudResources(tx *gorm.DB, field string, id int) (int64, error) {
	if field != "cloud_account_id" && field != "cloud_account_region_id" {
		return 0, errors.New("资源范围无效")
	}
	var ecs, managed int64
	if err := tx.Model(&model.TreeCloudResource{}).Where(field+" = ?", id).Count(&ecs).Error; err != nil {
		return 0, err
	}
	if err := tx.Model(&model.CloudManagedResource{}).Where(field+" = ?", id).Count(&managed).Error; err != nil {
		return 0, err
	}
	return ecs + managed, nil
}

// Account -> region is the same lock order as the sync commit transaction.
func (d *cloudAccountDAO) deleteAccountsSafely(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return errors.New("账户列表不能为空")
	}
	ordered := append([]int(nil), ids...)
	sort.Ints(ordered)
	for i, id := range ordered {
		if id <= 0 || (i > 0 && id == ordered[i-1]) {
			return errors.New("账户ID无效或重复")
		}
	}
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var accounts []model.CloudAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", ordered).Order("id ASC").Find(&accounts).Error; err != nil {
			return err
		}
		if len(accounts) != len(ordered) {
			return errors.New("部分云账户不存在")
		}
		var regions []model.CloudAccountRegion
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("cloud_account_id IN ?", ordered).Order("id ASC").Find(&regions).Error; err != nil {
			return err
		}
		for _, account := range accounts {
			count, err := countCloudResources(tx, "cloud_account_id", account.ID)
			if err != nil {
				return err
			}
			if count > 0 {
				return ErrCloudAccountInUse
			}
		}
		if err := tx.Where("cloud_account_id IN ?", ordered).Delete(&model.CloudAccountRegion{}).Error; err != nil {
			return err
		}
		result := tx.Where("id IN ?", ordered).Delete(&model.CloudAccount{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != int64(len(ordered)) {
			return errors.New("账户删除冲突")
		}
		return nil
	})
}

func (d *cloudAccountRegionDAO) deleteRegionSafely(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("地域ID无效")
	}
	var initial model.CloudAccountRegion
	if err := d.db.WithContext(ctx).Select("id, cloud_account_id").First(&initial, id).Error; err != nil {
		return err
	}
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account model.CloudAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, initial.CloudAccountID).Error; err != nil {
			return err
		}
		var region model.CloudAccountRegion
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND cloud_account_id = ?", id, account.ID).First(&region).Error; err != nil {
			return err
		}
		count, err := countCloudResources(tx, "cloud_account_region_id", id)
		if err != nil {
			return err
		}
		if count > 0 {
			return ErrCloudAccountRegionInUse
		}
		result := tx.Delete(&model.CloudAccountRegion{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("地域删除冲突")
		}
		return nil
	})
}

func (d *cloudAccountRegionDAO) createRegionsSafely(ctx context.Context, regions []*model.CloudAccountRegion) error {
	if len(regions) == 0 {
		return errors.New("地域列表不能为空")
	}
	seen := map[int]bool{}
	ids := make([]int, 0)
	for _, region := range regions {
		if region == nil || region.CloudAccountID <= 0 {
			return errors.New("地域账户无效")
		}
		if !seen[region.CloudAccountID] {
			seen[region.CloudAccountID] = true
			ids = append(ids, region.CloudAccountID)
		}
	}
	sort.Ints(ids)
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var accounts []model.CloudAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", ids).Order("id ASC").Find(&accounts).Error; err != nil {
			return err
		}
		if len(accounts) != len(ids) {
			return errors.New("地域所属账户不存在")
		}
		return tx.Omit("CloudAccount").Create(regions).Error
	})
}

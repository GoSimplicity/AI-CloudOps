package dao

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm/clause"
	"time"
)

func (d *managedCloudDAO) HasUnresolvedHistory(ctx context.Context, accountID, regionID int, kind string) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.CloudManagedSyncHistory{}).Where("cloud_account_id = ? AND cloud_account_region_id = ? AND kind = ? AND resolved_at IS NULL AND status IN (?, ?)", accountID, regionID, kind, "unknown", "running").Count(&count).Error
	return count > 0, err
}

// Called only inside ApplyRegionSync after the account, region and lease locks.
func (d *managedCloudDAO) ResolveHistory(ctx context.Context, req *model.ResolveCloudManagedHistoryReq) error {
	var row model.CloudManagedSyncHistory
	if err := d.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND cloud_account_id = ? AND cloud_account_region_id = ? AND kind = ?", req.HistoryID, req.CloudAccountID, req.CloudAccountRegionID, req.Kind).First(&row).Error; err != nil {
		return errors.New("同步历史不存在或不属于该范围")
	}
	if row.ResolvedAt != nil {
		return errors.New("该历史已经完成核对")
	}
	// A success may still need acknowledgement after a lost HTTP response. Never
	// rewrite the original outcome or fabricate a successful sync from review.
	if row.Status != "success" && row.Status != "unknown" && row.Status != "running" {
		return errors.New("该状态不需要未知结果核对")
	}
	now := time.Now().UTC()
	result := d.db.WithContext(ctx).Model(&model.CloudManagedSyncHistory{}).Where("id = ? AND resolved_at IS NULL", row.ID).Updates(map[string]any{"resolved_at": now, "resolved_by": req.OperatorID, "resolution_reason": req.Reason})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("核对状态已变化")
	}
	return nil
}

package dao

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm/clause"
	"time"
)

// The enclosing fenced transaction serializes this with region data commits.
// Inserting a missing reviewed tombstone prevents delayed delivery from gaining
// the operation's original execution grant.
func (d *managedCloudDAO) ReviewOperation(ctx context.Context, candidate *model.CloudManagedOperation, reason string) error {
	if err := d.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(candidate).Error; err != nil {
		return err
	}
	var row model.CloudManagedOperation
	if err := d.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("`key` = ?", candidate.Key).First(&row).Error; err != nil {
		return err
	}
	if row.RequestHash != candidate.RequestHash || row.OperatorID != candidate.OperatorID || row.CloudAccountID != candidate.CloudAccountID || row.CloudAccountRegionID != candidate.CloudAccountRegionID || row.Kind != candidate.Kind {
		return errors.New("操作ID已用于其他请求范围或资源")
	}
	if row.ReviewedAt != nil {
		return errors.New("该操作已记录核对，不允许覆盖")
	}
	previous := row.State
	if row.ClaimToken == candidate.ClaimToken {
		previous = "not_recorded"
	} else if row.State != "running" && row.State != "completed" {
		return errors.New("操作状态不允许核对")
	}
	now := time.Now().UTC()
	update := d.db.WithContext(ctx).Model(&model.CloudManagedOperation{}).Where("`key` = ? AND reviewed_at IS NULL", candidate.Key).Updates(map[string]any{"state": "reviewed", "previous_state": previous, "reviewed_at": now, "reviewed_by": candidate.ReviewedBy, "review_reason": reason, "updated_at": now})
	if update.Error != nil {
		return update.Error
	}
	if update.RowsAffected != 1 {
		return errors.New("操作核对状态已变化")
	}
	// Older persisted operation rows may lack the optional ID snapshot. The
	// reviewed request has already matched the immutable request hash.
	if len(row.ResourceIDs) == 0 {
		if err := d.db.WithContext(ctx).Model(&model.CloudManagedOperation{}).Where("`key` = ?", candidate.Key).Update("resource_ids", candidate.ResourceIDs).Error; err != nil {
			return err
		}
	}
	return d.db.WithContext(ctx).Model(&model.CloudManagedSyncHistory{}).Where("cloud_account_id = ? AND cloud_account_region_id = ? AND kind = ? AND client_operation_id = ? AND operator_id = ? AND resolved_at IS NULL", candidate.CloudAccountID, candidate.CloudAccountRegionID, candidate.Kind, candidate.OperationID, candidate.OperatorID).Updates(map[string]any{"resolved_at": now, "resolved_by": candidate.ReviewedBy, "resolution_reason": reason}).Error
}

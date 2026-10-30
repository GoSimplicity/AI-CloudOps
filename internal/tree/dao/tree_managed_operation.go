package dao

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm/clause"
	"time"
)

// A unique insert/token grants a reservation. Cloud reads additionally require
// fenced durable admission; duplicates never acquire another reservation.
func (d *managedCloudDAO) BeginOperation(ctx context.Context, row *model.CloudManagedOperation) (bool, *model.CloudManagedOperation, error) {
	result := d.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(row)
	if result.Error != nil {
		return false, nil, result.Error
	}
	var previous model.CloudManagedOperation
	if err := d.db.WithContext(ctx).Where("`key` = ?", row.Key).First(&previous).Error; err != nil {
		return false, nil, err
	}
	if previous.RequestHash != row.RequestHash || previous.OperatorID != row.OperatorID || previous.OperationID != row.OperationID {
		return false, nil, errors.New("操作ID已用于其他请求")
	}
	// MySQL clientFoundRows changes duplicate-insert row counts. Only the
	// independently generated persisted reservation token grants execution.
	return previous.ClaimToken == row.ClaimToken, &previous, nil
}
func (d *managedCloudDAO) CompleteOperation(ctx context.Context, key string, payload []byte) error {
	result := d.db.WithContext(ctx).Model(&model.CloudManagedOperation{}).Where("`key` = ? AND state = ?", key, "running").Updates(map[string]any{"state": "completed", "result": payload, "updated_at": time.Now().UTC()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("操作结果未确认")
	}
	return nil
}
func (d *managedCloudDAO) GetOperation(ctx context.Context, key string, owner int) (*model.CloudManagedOperation, error) {
	var row model.CloudManagedOperation
	if err := d.db.WithContext(ctx).Where("`key` = ? AND operator_id = ?", key, owner).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Called inside the region commit transaction with the same account->region->
// lease->operation lock order as manual review.
func (d *managedCloudDAO) CheckOperationClaim(ctx context.Context, key, token string) error {
	var row model.CloudManagedOperation
	if err := d.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("`key` = ? AND claim_token = ? AND state = ?", key, token, "running").First(&row).Error; err != nil {
		return errors.New("操作已核对或领取状态失效")
	}
	return nil
}

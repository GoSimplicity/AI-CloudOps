package config

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Enumeration is a locator, never permission to deliver an old operation.
// Each publisher and the eventual SQL lease revalidate its current operation.
type ProjectionRecoveryJournalDAO interface {
	CurrentProjectionRecoveryHighWater(context.Context) (uint64, error)
	ListCurrentUnprojectedPoolOperations(context.Context, uint64, uint64, int) ([]PoolReservation, error)
	ReadPoolProjectionSourceVersion(context.Context, int8, int) (uint64, error)
	ReadAMPoolSourceVersion(context.Context, int) (AMSourceVersion, error)
}

func NewProjectionRecoveryJournalDAO(logger *zap.Logger, db *gorm.DB) ProjectionRecoveryJournalDAO {
	return &monitorConfigDAO{l: logger, db: db}
}
func (d *monitorConfigDAO) CurrentProjectionRecoveryHighWater(ctx context.Context) (uint64, error) {
	var result uint64
	err := d.db.WithContext(ctx).Model(&model.MonitorConfigPoolOperation{}).Select("COALESCE(MAX(operation_id),0)").Scan(&result).Error
	return result, journalError(ctx, err)
}
func (d *monitorConfigDAO) ListCurrentUnprojectedPoolOperations(ctx context.Context, after, through uint64, limit int) ([]PoolReservation, error) {
	if limit < 1 || limit > 100 {
		return nil, ErrJournalInvalid
	}
	if after >= through {
		return []PoolReservation{}, ctx.Err()
	}
	var rows []model.MonitorConfigPoolOperation
	err := d.db.WithContext(ctx).Table("cl_monitor_config_pool_operations AS op").Select("op.*").Joins("JOIN cl_monitor_config_pool_controls AS control ON control.last_committed_operation_id=op.operation_id AND control.family=op.family AND control.pool_id=op.pool_id").Where("op.operation_id > ? AND op.operation_id <= ? AND op.phase = ?", after, through, model.ConfigPublicationCommitted).Order("op.operation_id ASC").Limit(limit).Scan(&rows).Error
	if err != nil {
		return nil, journalError(ctx, err)
	}
	result := make([]PoolReservation, 0, len(rows))
	for _, row := range rows {
		if row.OperationID <= after || row.OperationID > through {
			return nil, ErrJournalConflict
		}
		// A corrupt payload is an individual poison item, not a broken page.
		// Return only its locator; LoadLastCommitted/lease validate all content.
		ref := PoolReservation{OperationID: row.OperationID, Family: row.Family, PoolID: row.PoolID, Generation: row.Generation}
		copy(ref.Nonce[:], row.Nonce)
		result = append(result, ref)
		after = row.OperationID
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
func (d *monitorConfigDAO) ReadPoolProjectionSourceVersion(ctx context.Context, family int8, poolID int) (uint64, error) {
	if !validScrapePublicationScope(scrapePublicationScope(family)) || poolID <= 0 {
		return 0, ErrJournalInvalid
	}
	var row model.MonitorConfigPoolControl
	err := d.db.WithContext(ctx).Where("family = ? AND pool_id = ?", family, poolID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, journalError(ctx, err)
	}
	return row.ReservedGeneration, ctx.Err()
}
func (d *monitorConfigDAO) ReadAMPoolSourceVersion(ctx context.Context, poolID int) (AMSourceVersion, error) {
	if poolID <= 0 {
		return AMSourceVersion{}, ErrJournalInvalid
	}
	var rows []model.MonitorConfigPoolControl
	err := d.db.WithContext(ctx).Where("pool_id = ? AND family IN ?", poolID, []int8{2, 5}).Order("family ASC").Find(&rows).Error
	if err != nil {
		return AMSourceVersion{}, journalError(ctx, err)
	}
	result := AMSourceVersion{PoolID: poolID}
	for _, row := range rows {
		if row.Family == 2 {
			result.FullCounter = row.ReservedGeneration
			result.FullCommittedOperationID = cloneUint64(row.LastCommittedOperationID)
		} else {
			result.SubsetCounter = row.ReservedGeneration
			result.SubsetCommittedOperationID = cloneUint64(row.LastCommittedOperationID)
		}
	}
	return result, ctx.Err()
}

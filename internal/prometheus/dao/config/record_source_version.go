package config

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type RecordSourceVersion struct {
	PoolID  int
	Counter uint64
}

// Capture durable attempt versions before reading any pool/rule source. An
// absent pool is version zero; a later writer's reservation makes its CAS fail.
func (d *monitorConfigDAO) ReadRecordSourceVersions(ctx context.Context) ([]RecordSourceVersion, error) {
	result := []RecordSourceVersion{}
	seen := map[int]bool{}
	var expected int64 = -1
	var scanned int64
	for page := 0; ; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var total int64
		query := d.db.WithContext(ctx).Model(&model.MonitorConfigPoolControl{}).Where("family = ?", model.ConfigPublicationFamilyRecordRule)
		if err := query.Count(&total).Error; err != nil {
			return nil, journalError(ctx, err)
		}
		if total < 0 || (expected >= 0 && total != expected) {
			return nil, ErrJournalConflict
		}
		expected = total
		if total == 0 {
			return result, nil
		}
		var rows []model.MonitorConfigPoolControl
		if err := query.Order("pool_id ASC").Offset(page * 100).Limit(100).Find(&rows).Error; err != nil {
			return nil, journalError(ctx, err)
		}
		if int64(len(rows)) > expected-scanned || len(rows) == 0 {
			return nil, ErrJournalConflict
		}
		for _, row := range rows {
			if row.Family != model.ConfigPublicationFamilyRecordRule || row.PoolID <= 0 || seen[row.PoolID] {
				return nil, ErrJournalConflict
			}
			seen[row.PoolID] = true
			result = append(result, RecordSourceVersion{PoolID: row.PoolID, Counter: row.ReservedGeneration})
			scanned++
		}
		if scanned == expected {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return result, nil
		}
		if len(rows) < 100 {
			return nil, ErrJournalConflict
		}
	}
}
func (d *monitorConfigDAO) ReserveRecordPoolOperation(ctx context.Context, req PoolOperationRequest, expectedCounter uint64) (PoolReservation, error) {
	if req.Family != model.ConfigPublicationFamilyRecordRule {
		return PoolReservation{}, ErrJournalInvalid
	}
	return d.reservePoolOperation(ctx, req, &expectedCounter)
}

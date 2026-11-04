package config

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

func validAMFamily(family int8) bool { return family == 2 || family == 5 }

type amLockedControls struct {
	full, subset *model.MonitorConfigPoolControl
}

func lockAMControls(tx *gorm.DB, poolID int, create bool) (amLockedControls, error) {
	var result amLockedControls
	var err error
	lock := lockExistingPoolControl
	if create {
		lock = lockPoolControl
	}
	result.full, err = lock(tx, 2, poolID)
	if err != nil {
		return result, err
	}
	result.subset, err = lock(tx, 5, poolID)
	return result, err
}
func (c amLockedControls) own(family int8) *model.MonitorConfigPoolControl {
	if family == 2 {
		return c.full
	}
	return c.subset
}
func (c amLockedControls) matchesSource(source AMSourceVersion, ownFamily int8, ownGeneration uint64) bool {
	full, subset := source.FullCounter, source.SubsetCounter
	if ownFamily == 2 {
		full = ownGeneration
	}
	if ownFamily == 5 {
		subset = ownGeneration
	}
	if c.full.PoolID != source.PoolID || c.subset.PoolID != source.PoolID || c.full.ReservedGeneration != full || c.subset.ReservedGeneration != subset {
		return false
	}
	return sameOptionalID(c.full.LastCommittedOperationID, source.FullCommittedOperationID) && sameOptionalID(c.subset.LastCommittedOperationID, source.SubsetCommittedOperationID)
}
func (c amLockedControls) committed() AMCommittedVector {
	return AMCommittedVector{FullOperationID: cloneUint64(c.full.LastCommittedOperationID), SubsetOperationID: cloneUint64(c.subset.LastCommittedOperationID)}
}
func (c amLockedControls) matchesCommitted(vector AMCommittedVector) bool {
	return sameOptionalID(c.full.LastCommittedOperationID, vector.FullOperationID) && sameOptionalID(c.subset.LastCommittedOperationID, vector.SubsetOperationID)
}
func (d *monitorConfigDAO) ReadAMSourceVersions(ctx context.Context) ([]AMSourceVersion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Single complete query: these controls have no source bodies or secrets.
	var controls []model.MonitorConfigPoolControl
	if err := d.db.WithContext(ctx).Where("family IN ?", []int8{2, 5}).Order("pool_id ASC,family ASC").Find(&controls).Error; err != nil {
		return nil, journalError(ctx, err)
	}
	byPool := map[int]*AMSourceVersion{}
	result := []AMSourceVersion{}
	for _, control := range controls {
		if control.PoolID <= 0 || !validAMFamily(control.Family) {
			return nil, ErrJournalConflict
		}
		current := byPool[control.PoolID]
		if current == nil {
			current = &AMSourceVersion{PoolID: control.PoolID}
			byPool[control.PoolID] = current
		}
		if control.Family == 2 {
			current.FullCounter = control.ReservedGeneration
			current.FullCommittedOperationID = cloneUint64(control.LastCommittedOperationID)
		} else {
			current.SubsetCounter = control.ReservedGeneration
			current.SubsetCommittedOperationID = cloneUint64(control.LastCommittedOperationID)
		}
	}
	for _, control := range controls {
		if current := byPool[control.PoolID]; current != nil {
			result = append(result, *current)
			delete(byPool, control.PoolID)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
func (d *monitorConfigDAO) ReserveAMPoolOperation(ctx context.Context, req PoolOperationRequest, source AMSourceVersion) (PoolReservation, error) {
	if !validAMFamily(req.Family) || source.PoolID != req.PoolID {
		return PoolReservation{}, ErrJournalInvalid
	}
	return d.reservePoolOperationWithAM(ctx, req, nil, &source)
}
func validateAMCommittedVector(tx *gorm.DB, op model.MonitorConfigPoolOperation, vector AMCommittedVector) error {
	if !validAMFamily(op.Family) {
		return ErrJournalConflict
	}
	own := vector.FullOperationID
	if op.Family == 5 {
		own = vector.SubsetOperationID
	}
	if own == nil || *own != op.OperationID {
		return ErrJournalConflict
	}
	for family, id := range map[int8]*uint64{2: vector.FullOperationID, 5: vector.SubsetOperationID} {
		if id == nil {
			continue
		}
		if *id == 0 {
			return ErrJournalConflict
		}
		var other model.MonitorConfigPoolOperation
		if err := tx.Where("operation_id = ?", *id).First(&other).Error; err != nil {
			return err
		}
		if other.Family != family || other.PoolID != op.PoolID || !operationStateValid(other) || other.CommittedAt == nil || other.Manifest == nil {
			return ErrJournalConflict
		}
	}
	return nil
}
func amProjectionError(ctx context.Context, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return errors.Join(err, ctx.Err())
	}
	return journalError(ctx, err)
}

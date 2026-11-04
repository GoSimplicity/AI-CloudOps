package config

import (
	"bytes"
	"context"
	"errors"
	"math"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func journalError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	for _, known := range []error{ErrJournalInvalid, ErrJournalConflict, ErrJournalNotFound, ErrJournalUncertain, ErrJournalCounterExhausted} {
		if errors.Is(err, known) {
			return errors.Join(known, ctx.Err())
		}
	}
	if errors.Is(err, ErrConfigNamespaceConflict) || errors.Is(err, ErrGeneratedConfigConflict) || errors.Is(err, ErrInvalidConfigWrite) {
		return errors.Join(ErrJournalConflict, ctx.Err())
	}
	if configWriteLockRetry(err) {
		return errors.Join(ErrJournalWrite, ErrConfigWriteRetry, ctx.Err())
	}
	if errors.Is(err, context.Canceled) {
		return errors.Join(ErrJournalWrite, context.Canceled)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.Join(ErrJournalWrite, context.DeadlineExceeded)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.Join(ErrJournalNotFound, ctx.Err())
	}
	return errors.Join(ErrJournalWrite, ctx.Err())
}
func (d *monitorConfigDAO) journalTransaction(ctx context.Context, work func(*gorm.DB) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	complete := false
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := work(tx); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		complete = true
		return nil
	})
	if err != nil && complete {
		return errors.Join(ErrJournalUncertain, ctx.Err())
	}
	return err
}
func lockPoolControl(tx *gorm.DB, family int8, poolID int) (*model.MonitorConfigPoolControl, error) {
	control := model.MonitorConfigPoolControl{Family: family, PoolID: poolID}
	if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "family"}, {Name: "pool_id"}}, DoUpdates: clause.Assignments(map[string]interface{}{"pool_id": gorm.Expr("pool_id")})}).Create(&control).Error; err != nil {
		return nil, err
	}
	var current model.MonitorConfigPoolControl
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("family = ? AND pool_id = ?", family, poolID).First(&current).Error; err != nil {
		return nil, err
	}
	return &current, nil
}
func lockExistingPoolControl(tx *gorm.DB, family int8, poolID int) (*model.MonitorConfigPoolControl, error) {
	var control model.MonitorConfigPoolControl
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("family = ? AND pool_id = ?", family, poolID).First(&control).Error; err != nil {
		return nil, err
	}
	return &control, nil
}

func operationStateValid(op model.MonitorConfigPoolOperation) bool {
	if op.CommittedAt == nil {
		return op.Manifest == nil && op.SourceHash == nil && op.ProjectedAt == nil && (op.Phase == model.ConfigPublicationReserved || op.Phase == model.ConfigPublicationFailed || op.Phase == model.ConfigPublicationSuperseded)
	}
	if op.Manifest == nil || len(op.SourceHash) != 32 {
		return false
	}
	switch op.Phase {
	case model.ConfigPublicationCommitted:
		return op.ProjectedAt == nil
	case model.ConfigPublicationProjected:
		return op.ProjectedAt != nil
	case model.ConfigPublicationSuperseded:
		return true
	}
	return false
}

func operationReservation(op model.MonitorConfigPoolOperation) (PoolReservation, error) {
	if op.OperationID == 0 || !validPublicationScope(op.Family, op.PoolID) || op.Generation == 0 || len(op.Nonce) != 16 {
		return PoolReservation{}, ErrJournalConflict
	}
	ref := PoolReservation{OperationID: op.OperationID, Family: op.Family, PoolID: op.PoolID, Generation: op.Generation}
	copy(ref.Nonce[:], op.Nonce)
	return ref, nil
}
func operationMatches(op model.MonitorConfigPoolOperation, ref PoolReservation) bool {
	actual, err := operationReservation(op)
	return err == nil && actual == ref
}
func (d *monitorConfigDAO) ReservePoolOperation(ctx context.Context, req PoolOperationRequest) (PoolReservation, error) {
	return d.reservePoolOperation(ctx, req, nil)
}

func (d *monitorConfigDAO) reservePoolOperation(ctx context.Context, req PoolOperationRequest, expectedCounter *uint64) (PoolReservation, error) {
	return d.reservePoolOperationWithAM(ctx, req, expectedCounter, nil)
}
func (d *monitorConfigDAO) reservePoolOperationWithAM(ctx context.Context, req PoolOperationRequest, expectedCounter *uint64, amSource *AMSourceVersion) (PoolReservation, error) {
	if !validPublicationScope(req.Family, req.PoolID) || !nonzeroNonce(req.Nonce) || req.RecoveryOfOperationID != nil && *req.RecoveryOfOperationID == 0 {
		return PoolReservation{}, ErrJournalInvalid
	}
	req.RecoveryOfOperationID = cloneUint64(req.RecoveryOfOperationID)
	var result PoolReservation
	err := d.journalTransaction(ctx, func(tx *gorm.DB) error {
		var control *model.MonitorConfigPoolControl
		var err error
		var pair amLockedControls
		if amSource != nil {
			pair, err = lockAMControls(tx, req.PoolID, true)
			control = pair.own(req.Family)
		} else {
			control, err = lockPoolControl(tx, req.Family, req.PoolID)
		}
		if err != nil {
			return err
		}
		var existing model.MonitorConfigPoolOperation
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("nonce = ?", req.Nonce[:]).First(&existing).Error
		if err == nil {
			if existing.Family != req.Family || existing.PoolID != req.PoolID || existing.Generation > control.ReservedGeneration || !operationStateValid(existing) || !sameOptionalID(existing.RecoveryOfOperationID, req.RecoveryOfOperationID) {
				return ErrJournalConflict
			}
			result, err = operationReservation(existing)
			return err
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if amSource != nil && !pair.matchesSource(*amSource, 0, 0) {
			return ErrJournalConflict
		}
		if expectedCounter != nil && control.ReservedGeneration != *expectedCounter {
			return ErrJournalConflict
		}
		if control.ReservedGeneration == math.MaxUint64 {
			return ErrJournalCounterExhausted
		}
		if req.RecoveryOfOperationID != nil {
			if control.LastCommittedOperationID == nil || *control.LastCommittedOperationID != *req.RecoveryOfOperationID {
				return ErrJournalConflict
			}
			var recovered model.MonitorConfigPoolOperation
			if err := tx.Where("operation_id = ?", *req.RecoveryOfOperationID).First(&recovered).Error; err != nil {
				return err
			}
			if recovered.Family != req.Family || recovered.PoolID != req.PoolID || recovered.CommittedAt == nil || recovered.Manifest == nil {
				return ErrJournalConflict
			}
		}
		if control.CurrentAttemptOperationID != nil {
			if err := tx.Model(&model.MonitorConfigPoolOperation{}).Where("operation_id = ? AND phase = ? AND manifest IS NULL", *control.CurrentAttemptOperationID, model.ConfigPublicationReserved).Updates(map[string]interface{}{"phase": model.ConfigPublicationSuperseded, "failure_code": "new_attempt"}).Error; err != nil {
				return err
			}
		}
		generation := control.ReservedGeneration + 1
		op := model.MonitorConfigPoolOperation{Family: req.Family, PoolID: req.PoolID, Generation: generation, Nonce: append([]byte(nil), req.Nonce[:]...), Phase: model.ConfigPublicationReserved, RecoveryOfOperationID: cloneUint64(req.RecoveryOfOperationID)}
		if err := tx.Create(&op).Error; err != nil {
			return err
		}
		if err := tx.Model(control).Updates(map[string]interface{}{"reserved_generation": generation, "current_attempt_operation_id": op.OperationID}).Error; err != nil {
			return err
		}
		result, err = operationReservation(op)
		return err
	})
	if err != nil {
		return PoolReservation{}, journalError(ctx, err)
	}
	return result, nil
}
func (d *monitorConfigDAO) loadReceipt(tx *gorm.DB, op model.MonitorConfigPoolOperation) (PoolReceipt, error) {
	ref, err := operationReservation(op)
	if err != nil {
		return PoolReceipt{}, err
	}
	if !operationStateValid(op) || op.CommittedAt == nil {
		return PoolReceipt{}, ErrJournalConflict
	}
	entries, err := decodePoolManifest(op.Manifest)
	if err != nil {
		return PoolReceipt{}, err
	}
	result := PoolReceipt{Reservation: ref, Phase: op.Phase, Artifacts: make([]ArtifactReceipt, 0, len(entries))}
	copy(result.SourceHash[:], op.SourceHash)
	for _, entry := range entries {
		var event model.MonitorConfigArtifactEvent
		if err := tx.Where("event_id = ?", entry.EventID).First(&event).Error; err != nil {
			return PoolReceipt{}, err
		}
		if event.OperationID == nil || *event.OperationID != op.OperationID || !familyAllowsKind(op.Family, event.ConfigType) {
			return PoolReceipt{}, ErrJournalConflict
		}
		artifact, err := artifactReceipt(event, entry.SourceID)
		if err != nil {
			return PoolReceipt{}, err
		}
		result.Artifacts = append(result.Artifacts, artifact)
	}
	var count int64
	if err := tx.Model(&model.MonitorConfigArtifactEvent{}).Where("operation_id = ?", op.OperationID).Count(&count).Error; err != nil {
		return PoolReceipt{}, err
	}
	if count != int64(len(entries)) {
		return PoolReceipt{}, ErrJournalConflict
	}
	return result, nil
}
func (d *monitorConfigDAO) LoadOperationByNonce(ctx context.Context, nonce [16]byte) (PoolOperationSnapshot, error) {
	if !nonzeroNonce(nonce) {
		return PoolOperationSnapshot{}, ErrJournalInvalid
	}
	var op model.MonitorConfigPoolOperation
	if err := d.db.WithContext(ctx).Where("nonce = ?", nonce[:]).First(&op).Error; err != nil {
		return PoolOperationSnapshot{}, journalError(ctx, err)
	}
	ref, err := operationReservation(op)
	if err != nil {
		return PoolOperationSnapshot{}, err
	}
	if !operationStateValid(op) {
		return PoolOperationSnapshot{}, ErrJournalConflict
	}
	result := PoolOperationSnapshot{Reservation: ref, Phase: op.Phase, FailureCode: op.FailureCode}
	if op.Manifest != nil {
		receipt, err := d.loadReceipt(d.db.WithContext(ctx), op)
		if err != nil {
			return PoolOperationSnapshot{}, journalError(ctx, err)
		}
		hash := receipt.SourceHash
		result.SourceHash = &hash
		result.Receipt = &receipt
	} else if op.CommittedAt != nil {
		return PoolOperationSnapshot{}, ErrJournalConflict
	}
	return result, nil
}
func (d *monitorConfigDAO) loadLastPoolReceipt(ctx context.Context, family int8, poolID int, projected bool) (*PoolReceipt, error) {
	if !validPublicationScope(family, poolID) {
		return nil, ErrJournalInvalid
	}
	var control model.MonitorConfigPoolControl
	if err := d.db.WithContext(ctx).Where("family = ? AND pool_id = ?", family, poolID).First(&control).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, journalError(ctx, err)
	}
	id := control.LastCommittedOperationID
	if projected {
		id = control.LastProjectedOperationID
	}
	if id == nil {
		return nil, nil
	}
	var op model.MonitorConfigPoolOperation
	if err := d.db.WithContext(ctx).Where("operation_id = ?", *id).First(&op).Error; err != nil {
		return nil, journalError(ctx, err)
	}
	if op.Family != family || op.PoolID != poolID || op.Generation > control.ReservedGeneration || projected && op.ProjectedAt == nil {
		return nil, ErrJournalConflict
	}
	receipt, err := d.loadReceipt(d.db.WithContext(ctx), op)
	if err != nil {
		return nil, journalError(ctx, err)
	}
	return &receipt, nil
}
func (d *monitorConfigDAO) LoadLastCommitted(ctx context.Context, family int8, poolID int) (*PoolReceipt, error) {
	return d.loadLastPoolReceipt(ctx, family, poolID, false)
}
func (d *monitorConfigDAO) LoadLastProjected(ctx context.Context, family int8, poolID int) (*PoolReceipt, error) {
	return d.loadLastPoolReceipt(ctx, family, poolID, true)
}
func reservationValid(ref PoolReservation) bool {
	return ref.OperationID > 0 && ref.Generation > 0 && validPublicationScope(ref.Family, ref.PoolID) && nonzeroNonce(ref.Nonce)
}
func sameHash(hash [32]byte, stored []byte) bool {
	return len(stored) == 32 && bytes.Equal(hash[:], stored)
}

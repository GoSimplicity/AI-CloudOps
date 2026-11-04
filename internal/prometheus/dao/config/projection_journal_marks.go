package config

import (
	"context"
	"reflect"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *monitorConfigDAO) MarkArtifactDeliveryIfCurrent(ctx context.Context, receipt ArtifactReceipt, state string) (bool, error) {
	key := configNamespaceKey{receipt.ConfigType, receipt.InstanceKey}
	if receipt.OperationID == nil && receipt.SourceID != 0 {
		return false, ErrJournalInvalid
	}
	if receipt.EventID == 0 || !validConfigNamespace(key) || (state != model.ConfigProjectionApplied && state != model.ConfigProjectionSuperseded) {
		return false, ErrJournalInvalid
	}
	// Discover routing before the transaction, then revalidate the immutable op
	// after control -> namespace locks. Ordinary receipts never acquire control.
	var observed model.MonitorConfigPoolOperation
	if receipt.OperationID != nil {
		if err := d.db.WithContext(ctx).Where("operation_id = ?", *receipt.OperationID).First(&observed).Error; err != nil {
			return false, journalError(ctx, err)
		}
		if _, err := operationReservation(observed); err != nil {
			return false, err
		}
	}
	changed := false
	err := d.journalTransaction(ctx, func(tx *gorm.DB) error {
		currentOperation := true
		if receipt.OperationID != nil {
			control, err := lockExistingPoolControl(tx, observed.Family, observed.PoolID)
			if err != nil {
				return err
			}
			currentOperation = control.LastCommittedOperationID != nil && *control.LastCommittedOperationID == observed.OperationID
		}
		if err := lockConfigNamespacesTx(tx, []configNamespaceKey{key}); err != nil {
			return err
		}
		if receipt.OperationID != nil {
			var locked model.MonitorConfigPoolOperation
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("operation_id = ?", observed.OperationID).First(&locked).Error; err != nil {
				return err
			}
			ref, err := operationReservation(observed)
			if err != nil || !operationMatches(locked, ref) || locked.Manifest == nil || locked.CommittedAt == nil {
				return ErrJournalConflict
			}
			issued, err := d.loadReceipt(tx, locked)
			if err != nil {
				return err
			}
			found := false
			for _, artifact := range issued.Artifacts {
				if artifact.EventID == receipt.EventID {
					if !reflect.DeepEqual(artifact, receipt) {
						return ErrJournalConflict
					}
					found = true
					break
				}
			}
			if !found {
				return ErrJournalConflict
			}
		}
		var event model.MonitorConfigArtifactEvent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("event_id = ?", receipt.EventID).First(&event).Error; err != nil {
			return err
		}
		actual, err := artifactReceipt(event, receipt.SourceID)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(actual, receipt) {
			return ErrJournalConflict
		}
		if event.ProjectionPolicy == model.ConfigProjectionNotApplicable {
			return nil
		}
		latest, err := latestArtifactEvent(tx, key)
		if err != nil {
			return err
		}
		current := latest != nil && latest.EventID == event.EventID && currentOperation
		if state == model.ConfigProjectionApplied && !current || state == model.ConfigProjectionSuperseded && current {
			return nil
		}
		if event.DeliveryState == state {
			changed = true
			return nil
		}
		if event.DeliveryState != model.ConfigProjectionPending {
			return nil
		}
		updates := map[string]interface{}{"delivery_state": state, "failure_code": ""}
		if state == model.ConfigProjectionApplied {
			updates["projected_at"] = time.Now()
		}
		result := tx.Model(&event).Where("delivery_state = ?", model.ConfigProjectionPending).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		changed = result.RowsAffected == 1
		return nil
	})
	if err != nil {
		return false, journalError(ctx, err)
	}
	return changed, nil
}
func (d *monitorConfigDAO) MarkPoolProjectedIfCurrent(ctx context.Context, receipt PoolReceipt) (bool, error) {
	if !reservationValid(receipt.Reservation) {
		return false, ErrJournalInvalid
	}
	keys := make([]configNamespaceKey, 0, len(receipt.Artifacts))
	for _, artifact := range receipt.Artifacts {
		key := configNamespaceKey{artifact.ConfigType, artifact.InstanceKey}
		if !validConfigNamespace(key) {
			return false, ErrJournalInvalid
		}
		keys = append(keys, key)
	}
	changed := false
	err := d.journalTransaction(ctx, func(tx *gorm.DB) error {
		ref := receipt.Reservation
		control, err := lockExistingPoolControl(tx, ref.Family, ref.PoolID)
		if err != nil {
			return err
		}
		if control.LastCommittedOperationID == nil || *control.LastCommittedOperationID != ref.OperationID {
			return nil
		}
		if err := lockConfigNamespacesTx(tx, keys); err != nil {
			return err
		}
		var op model.MonitorConfigPoolOperation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("operation_id = ?", ref.OperationID).First(&op).Error; err != nil {
			return err
		}
		if !operationMatches(op, ref) || !sameHash(receipt.SourceHash, op.SourceHash) {
			return ErrJournalConflict
		}
		stored, err := d.loadReceipt(tx, op)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(stored.Artifacts, receipt.Artifacts) {
			return ErrJournalConflict
		}
		for _, artifact := range stored.Artifacts {
			latest, err := latestArtifactEvent(tx, configNamespaceKey{artifact.ConfigType, artifact.InstanceKey})
			if err != nil {
				return err
			}
			if latest == nil || latest.EventID != artifact.EventID || latest.DeliveryState != model.ConfigProjectionApplied {
				return nil
			}
		}
		if op.Phase == model.ConfigPublicationProjected && op.ProjectedAt != nil && control.LastProjectedOperationID != nil && *control.LastProjectedOperationID == op.OperationID {
			changed = true
			return nil
		}
		if op.Phase != model.ConfigPublicationCommitted {
			return nil
		}
		now := time.Now()
		result := tx.Model(&op).Where("phase = ?", model.ConfigPublicationCommitted).Updates(map[string]interface{}{"phase": model.ConfigPublicationProjected, "projected_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrJournalConflict
		}
		if err := tx.Model(control).Updates(map[string]interface{}{"last_projected_operation_id": op.OperationID}).Error; err != nil {
			return err
		}
		changed = true
		return nil
	})
	if err != nil {
		return false, journalError(ctx, err)
	}
	return changed, nil
}

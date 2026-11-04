package config

import (
	"context"
	"reflect"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Coverage grants no raw/ACK authority. It can only certify that current
// projected full/subset manifests already cover a desired read-only view.
type AMCoverageRequest struct {
	PoolID    int
	Receipts  []ProjectionReceipt
	Artifacts []ArtifactReceipt
}
type AMCoverageSnapshot struct {
	Receipts  []ProjectionReceipt
	Artifacts []ProjectionMaterial
}

func (d *monitorConfigDAO) WithAMCoverage(ctx context.Context, req AMCoverageRequest, callback func(AMCoverageSnapshot) error) (bool, error) {
	if req.PoolID <= 0 || callback == nil || len(req.Receipts) == 0 || len(req.Receipts) > 2 {
		return false, ErrJournalInvalid
	}
	receipts := make([]ProjectionReceipt, len(req.Receipts))
	keys := []configNamespaceKey{}
	seenFamily := map[int8]bool{}
	seenKey := map[configNamespaceKey]bool{}
	for i, receipt := range req.Receipts {
		if !validAMFamily(receipt.Reservation.Family) || receipt.Reservation.PoolID != req.PoolID || seenFamily[receipt.Reservation.Family] {
			return false, ErrJournalInvalid
		}
		seenFamily[receipt.Reservation.Family] = true
		receipts[i] = cloneProjectionReceipt(receipt)
	}
	artifacts := make([]ArtifactReceipt, 0, len(req.Artifacts))
	for _, artifact := range req.Artifacts {
		key := configNamespaceKey{artifact.ConfigType, artifact.InstanceKey}
		if !validConfigNamespace(key) || seenKey[key] || (key.kind != 2 && key.kind != 5) || artifact.Action != model.ConfigProjectionRender {
			return false, ErrJournalInvalid
		}
		seenKey[key] = true
		keys = append(keys, key)
		artifacts = append(artifacts, artifact)
	}
	matched := false
	err := d.journalTransaction(ctx, func(tx *gorm.DB) error {
		pair, err := lockAMControls(tx, req.PoolID, false)
		if err != nil {
			return err
		}
		if err := lockConfigNamespacesTx(tx, keys); err != nil {
			return err
		}
		issued := map[uint64]ArtifactReceipt{}
		snapshot := AMCoverageSnapshot{Receipts: []ProjectionReceipt{}, Artifacts: []ProjectionMaterial{}}
		for _, expected := range receipts {
			ref := expected.Reservation
			own := pair.own(ref.Family)
			if own.LastCommittedOperationID == nil || *own.LastCommittedOperationID != ref.OperationID {
				return ErrJournalConflict
			}
			var op model.MonitorConfigPoolOperation
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("operation_id = ?", ref.OperationID).First(&op).Error; err != nil {
				return err
			}
			actual, err := d.loadProjectionReceipt(tx, op)
			if err != nil {
				return err
			}
			if actual.AMControls == nil || actual.Phase != model.ConfigPublicationProjected || !equalProjectionPayload(actual, expected) {
				return ErrJournalConflict
			}
			snapshot.Receipts = append(snapshot.Receipts, actual)
			for _, artifact := range actual.Artifacts {
				issued[artifact.EventID] = artifact
			}
		}
		for _, expected := range artifacts {
			actual, ok := issued[expected.EventID]
			if !ok || !reflect.DeepEqual(actual, expected) {
				return ErrJournalConflict
			}
			event, err := latestArtifactEvent(tx, configNamespaceKey{expected.ConfigType, expected.InstanceKey})
			if err != nil {
				return err
			}
			if event == nil || event.DeliveryState != model.ConfigProjectionApplied {
				return ErrJournalConflict
			}
			material, err := leaseMaterial(tx, expected, false)
			if err != nil {
				return err
			}
			snapshot.Artifacts = append(snapshot.Artifacts, material)
		}
		if err := callback(snapshot); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		matched = true
		return nil
	})
	if err != nil {
		return false, amProjectionError(ctx, err)
	}
	return matched, nil
}

package config

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ProjectionLease cannot be implemented outside this package. Its authority ends
// with the callback transaction; retaining the Go value does not retain a lease.
type ProjectionLease interface {
	Snapshot() (ProjectionLeaseSnapshot, error)
	RecheckCurrent(context.Context) error
	MarkComplete(ProjectionAckProof) (bool, error)
	projectionLease()
}
type ProjectionMaterial struct {
	Artifact ArtifactReceipt
	Binding  ProjectionBinding
	Body     []byte
}
type ProjectionLeaseSnapshot struct {
	Pool      *ProjectionReceipt
	Artifacts []ProjectionMaterial
}
type sqlProjectionLease struct {
	mu       sync.Mutex
	active   bool
	ctx      context.Context
	tx       *gorm.DB
	dao      *monitorConfigDAO
	pool     *ProjectionReceipt
	ordinary *ArtifactReceipt
	snapshot ProjectionLeaseSnapshot
}

func (*sqlProjectionLease) projectionLease() {}
func (l *sqlProjectionLease) usable(ctx context.Context) error {
	if !l.active {
		return ErrJournalConflict
	}
	if err := l.ctx.Err(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}
func (l *sqlProjectionLease) Snapshot() (ProjectionLeaseSnapshot, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.usable(l.ctx); err != nil {
		return ProjectionLeaseSnapshot{}, err
	}
	raw, err := json.Marshal(l.snapshot)
	if err != nil {
		return ProjectionLeaseSnapshot{}, ErrJournalConflict
	}
	var result ProjectionLeaseSnapshot
	if json.Unmarshal(raw, &result) != nil {
		return ProjectionLeaseSnapshot{}, ErrJournalConflict
	}
	return result, nil
}
func (l *sqlProjectionLease) RecheckCurrent(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.usable(ctx); err != nil {
		return err
	}
	err := l.recheck(ctx)
	if err != nil {
		l.active = false
	}
	return err
}
func (l *sqlProjectionLease) recheck(ctx context.Context) error {
	tx := l.tx.WithContext(ctx)
	var artifacts []ArtifactReceipt
	if l.pool != nil {
		ref := l.pool.Reservation
		var control *model.MonitorConfigPoolControl
		var err error
		if l.pool.AMControls != nil {
			pair, pairErr := lockAMControls(tx, ref.PoolID, false)
			if pairErr != nil {
				return pairErr
			}
			if !pair.matchesCommitted(*l.pool.AMControls) {
				return ErrJournalConflict
			}
			control = pair.own(ref.Family)
		} else {
			control, err = lockExistingPoolControl(tx, ref.Family, ref.PoolID)
		}
		if err != nil {
			return err
		}
		if control.LastCommittedOperationID == nil || *control.LastCommittedOperationID != ref.OperationID {
			return ErrJournalConflict
		}
		var op model.MonitorConfigPoolOperation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("operation_id = ?", ref.OperationID).First(&op).Error; err != nil {
			return err
		}
		if !operationMatches(op, ref) {
			return ErrJournalConflict
		}
		receipt, err := l.dao.loadProjectionReceipt(tx, op)
		if err != nil {
			return err
		}
		if !equalProjectionPayload(receipt, *l.pool) || !receipt.CompleteControls {
			return ErrJournalConflict
		}
		// Intent proofs are persisted, not inferred from an owner digest. Recheck
		// their original row and generated lineage under the same namespace locks.
		proofEvents := map[uint64]model.MonitorConfigArtifactEvent{}
		for _, artifact := range receipt.Artifacts {
			var event model.MonitorConfigArtifactEvent
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("event_id = ?", artifact.EventID).First(&event).Error; err != nil {
				return err
			}
			proofEvents[artifact.SourceID] = event
		}
		for _, control := range receipt.Controls {
			if control.Intents == nil {
				continue
			}
			request := &WebhookIntentsControl{RequireEmpty: true}
			for _, change := range control.Intents.Add {
				proof := ProjectionProofRef{EventID: change.ProofEventID}
				if change.ProofSourceID != 0 {
					proof = ProjectionProofRef{SourceID: change.ProofSourceID}
				}
				request.Add = append(request.Add, IntentOwnerChange{Owner: change.Owner, Proof: proof, AliasCurrentOwner: change.AliasCurrentOwner})
			}
			for _, change := range control.Intents.Remove {
				proof := ProjectionProofRef{EventID: change.ProofEventID}
				if change.ProofSourceID != 0 {
					proof = ProjectionProofRef{SourceID: change.ProofSourceID}
				}
				request.Remove = append(request.Remove, IntentOwnerChange{Owner: change.Owner, Proof: proof, AliasCurrentOwner: change.AliasCurrentOwner})
			}
			proofPlan := &preparedProjectionPlan{
				request:  GeneratedProjectionPlanRequest{GeneratedPlanRequest: GeneratedPlanRequest{Reservation: ref}},
				controls: []ProjectionControl{{ControlID: control.ControlID, Kind: ProjectionWebhookIntents, Intents: request}},
			}
			_, err := l.dao.resolveProjectionControls(tx, proofPlan, nil, proofEvents)
			if err != nil {
				return err
			}
		}
		artifacts = receipt.Artifacts
	} else {
		artifacts = []ArtifactReceipt{*l.ordinary}
	}
	materials := make([]ProjectionMaterial, 0, len(artifacts))
	for _, artifact := range artifacts {
		material, err := leaseMaterial(tx, artifact, l.ordinary != nil)
		if err != nil {
			return err
		}
		materials = append(materials, material)
	}
	l.snapshot = ProjectionLeaseSnapshot{Artifacts: materials}
	if l.pool != nil {
		copy := cloneProjectionReceipt(*l.pool)
		l.snapshot.Pool = &copy
	}
	return ctx.Err()
}
func leaseMaterial(tx *gorm.DB, receipt ArtifactReceipt, ordinary bool) (ProjectionMaterial, error) {
	key := configNamespaceKey{receipt.ConfigType, receipt.InstanceKey}
	event, err := latestArtifactEvent(tx, key)
	if err != nil {
		return ProjectionMaterial{}, err
	}
	if event == nil || event.EventID != receipt.EventID {
		return ProjectionMaterial{}, ErrJournalConflict
	}
	actual, err := artifactReceipt(*event, receipt.SourceID)
	if err != nil || !reflect.DeepEqual(actual, receipt) {
		return ProjectionMaterial{}, ErrJournalConflict
	}
	if ordinary && (receipt.OperationID != nil || receipt.SourceID != 0 ||
		receipt.ProjectionPolicy != model.ConfigProjectionLineageOnly ||
		(receipt.Action != model.ConfigProjectionInvalidate && receipt.Action != model.ConfigProjectionRetire)) {
		return ProjectionMaterial{}, ErrJournalInvalid
	}
	root, err := trustedBindingRoot(tx, event, key)
	if err != nil {
		return ProjectionMaterial{}, err
	}
	if root == nil || root.Binding == nil {
		return ProjectionMaterial{}, ErrJournalConflict
	}
	binding, err := ParseProjectionBinding(*root.Binding)
	if err != nil {
		return ProjectionMaterial{}, err
	}
	rows, err := currentConfigNamespace(tx, key)
	if err != nil {
		return ProjectionMaterial{}, err
	}
	var row *model.MonitorConfig
	if len(receipt.NewOwnerDigest) == 0 {
		if len(rows) != 0 {
			return ProjectionMaterial{}, ErrJournalConflict
		}
	} else {
		if len(rows) != 1 || receipt.ConfigRowID == nil || rows[0].ID != *receipt.ConfigRowID ||
			!bytes.Equal(ownerDigest(rows[0]), receipt.NewOwnerDigest) || !bytes.Equal(bodyDigest(rows[0]), receipt.BodyDigest) {
			return ProjectionMaterial{}, ErrJournalConflict
		}
		row = rows[0]
	}
	result := ProjectionMaterial{Artifact: actual, Binding: binding}
	if !ordinary && receipt.Action == model.ConfigProjectionRender {
		if row == nil {
			return ProjectionMaterial{}, ErrJournalConflict
		}
		result.Body = []byte(row.ConfigContent)
	}
	return result, nil
}
func projectionLeaseKeys(receipt ProjectionReceipt) ([]configNamespaceKey, error) {
	keys := make([]configNamespaceKey, 0, len(receipt.Artifacts))
	for _, artifact := range receipt.Artifacts {
		key := configNamespaceKey{artifact.ConfigType, artifact.InstanceKey}
		if !validConfigNamespace(key) {
			return nil, ErrJournalInvalid
		}
		keys = append(keys, key)
	}
	for _, control := range receipt.Controls {
		if control.Intents != nil {
			for _, changes := range [][]IssuedIntentOwnerChange{control.Intents.Add, control.Intents.Remove} {
				for _, change := range changes {
					key := configNamespaceKey{change.Owner.ConfigType, change.Owner.InstanceIP}
					if !validConfigNamespace(key) {
						return nil, ErrJournalInvalid
					}
					keys = append(keys, key)
				}
			}
		}
	}
	return keys, nil
}
func (d *monitorConfigDAO) WithPoolProjectionLease(ctx context.Context, receipt ProjectionReceipt, callback func(ProjectionLease) error) error {
	if callback == nil || !reservationValid(receipt.Reservation) || receipt.ManifestVersion != 2 || !receipt.CompleteControls {
		return ErrJournalInvalid
	}
	receipt = cloneProjectionReceipt(receipt)
	keys, err := projectionLeaseKeys(receipt)
	if err != nil {
		return err
	}
	err = d.journalTransaction(ctx, func(tx *gorm.DB) error {
		if receipt.AMControls != nil {
			if _, err := lockAMControls(tx, receipt.Reservation.PoolID, false); err != nil {
				return err
			}
		} else if _, err := lockExistingPoolControl(tx, receipt.Reservation.Family, receipt.Reservation.PoolID); err != nil {
			return err
		}
		if err := lockConfigNamespacesTx(tx, keys); err != nil {
			return err
		}
		lease := &sqlProjectionLease{active: true, ctx: ctx, tx: tx, dao: d, pool: &receipt}
		defer func() { lease.mu.Lock(); lease.active = false; lease.mu.Unlock() }()
		if err := lease.RecheckCurrent(ctx); err != nil {
			return err
		}
		if err := callback(lease); err != nil {
			return err
		}
		return lease.RecheckCurrent(ctx)
	})
	return journalError(ctx, err)
}
func (d *monitorConfigDAO) WithArtifactProjectionLease(ctx context.Context, receipt ArtifactReceipt, callback func(ProjectionLease) error) error {
	key := configNamespaceKey{receipt.ConfigType, receipt.InstanceKey}
	if callback == nil || !validConfigNamespace(key) || receipt.OperationID != nil || receipt.SourceID != 0 {
		return ErrJournalInvalid
	}
	// Clone the receipt before acquiring locks; the caller cannot mutate routing.
	raw, err := json.Marshal(receipt)
	if err != nil {
		return ErrJournalInvalid
	}
	var copy ArtifactReceipt
	if json.Unmarshal(raw, &copy) != nil {
		return ErrJournalInvalid
	}
	err = d.journalTransaction(ctx, func(tx *gorm.DB) error {
		if err := lockConfigNamespacesTx(tx, []configNamespaceKey{key}); err != nil {
			return err
		}
		lease := &sqlProjectionLease{active: true, ctx: ctx, tx: tx, dao: d, ordinary: &copy}
		defer func() { lease.mu.Lock(); lease.active = false; lease.mu.Unlock() }()
		if err := lease.RecheckCurrent(ctx); err != nil {
			return err
		}
		if err := callback(lease); err != nil {
			return err
		}
		return lease.RecheckCurrent(ctx)
	})
	return journalError(ctx, err)
}
func (l *sqlProjectionLease) MarkComplete(proof ProjectionAckProof) (marked bool, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// A failed completion attempt never leaves a reusable authority handle.
	defer func() {
		if err != nil {
			l.active = false
		}
	}()

	if err := l.usable(l.ctx); err != nil {
		return false, err
	}
	if l.pool == nil {
		return false, ErrJournalInvalid
	}
	expected, err := projectionAckProof(*l.pool)
	if err != nil || !reflect.DeepEqual(proof, expected) {
		return false, ErrJournalConflict
	}

	if err := l.recheck(l.ctx); err != nil {
		return false, err
	}
	now := time.Now()
	ref := l.pool.Reservation
	for _, artifact := range l.pool.Artifacts {
		var event model.MonitorConfigArtifactEvent
		if err := l.tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("event_id = ?", artifact.EventID).First(&event).Error; err != nil {
			return false, err
		}
		if event.DeliveryState == model.ConfigProjectionApplied {
			continue
		}
		if event.DeliveryState != model.ConfigProjectionPending {
			return false, ErrJournalConflict
		}
		update := l.tx.Model(&event).Where("delivery_state = ?", model.ConfigProjectionPending).Updates(map[string]interface{}{
			"delivery_state": model.ConfigProjectionApplied,
			"projected_at":   now,
			"failure_code":   "",
		})
		if update.Error != nil {
			return false, update.Error
		}
		if update.RowsAffected != 1 {
			return false, ErrJournalConflict
		}
	}
	var op model.MonitorConfigPoolOperation
	if err := l.tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("operation_id = ?", ref.OperationID).First(&op).Error; err != nil {
		return false, err
	}
	if op.Phase == model.ConfigPublicationProjected {
		control, err := lockExistingPoolControl(l.tx, ref.Family, ref.PoolID)
		if err != nil {
			return false, err
		}
		if op.ProjectedAt == nil || control.LastProjectedOperationID == nil || *control.LastProjectedOperationID != op.OperationID {
			return false, ErrJournalConflict
		}
		return true, nil
	}
	if op.Phase != model.ConfigPublicationCommitted {
		return false, ErrJournalConflict
	}
	if err := l.tx.Model(&op).Updates(map[string]interface{}{"phase": model.ConfigPublicationProjected, "projected_at": now}).Error; err != nil {
		return false, err
	}
	update := l.tx.Model(&model.MonitorConfigPoolControl{}).
		Where("family = ? AND pool_id = ? AND last_committed_operation_id = ?", ref.Family, ref.PoolID, ref.OperationID).
		Update("last_projected_operation_id", ref.OperationID)
	if update.Error != nil {
		return false, update.Error
	}
	if update.RowsAffected != 1 {
		return false, ErrJournalConflict
	}
	return true, nil
}
func (d *monitorConfigDAO) MarkProjectionCompleteIfCurrent(ctx context.Context, receipt ProjectionReceipt, proof ProjectionAckProof) (bool, error) {
	marked := false
	err := d.WithPoolProjectionLease(ctx, receipt, func(lease ProjectionLease) error {
		var markErr error
		marked, markErr = lease.MarkComplete(proof)
		return markErr
	})
	if err != nil {
		return false, err
	}
	return marked, nil
}

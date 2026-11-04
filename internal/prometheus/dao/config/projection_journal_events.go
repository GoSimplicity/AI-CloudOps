package config

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func latestArtifactEvent(tx *gorm.DB, key configNamespaceKey) (*model.MonitorConfigArtifactEvent, error) {
	var event model.MonitorConfigArtifactEvent
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("config_type = ? AND instance_key = ?", key.kind, []byte(key.raw)).Order("event_id DESC").First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}
func artifactReceipt(event model.MonitorConfigArtifactEvent, sourceID uint64) (ArtifactReceipt, error) {
	key := configNamespaceKey{event.ConfigType, string(event.InstanceKey)}
	if event.EventID == 0 || !validConfigNamespace(key) {
		return ArtifactReceipt{}, ErrJournalConflict
	}
	for _, digest := range [][]byte{event.OldOwnerDigest, event.NewOwnerDigest, event.BodyDigest} {
		if digest != nil && len(digest) != 32 {
			return ArtifactReceipt{}, ErrJournalConflict
		}
	}
	result := ArtifactReceipt{SourceID: sourceID, EventID: event.EventID, OperationID: cloneUint64(event.OperationID), ConfigType: event.ConfigType, InstanceKey: string(event.InstanceKey), Action: event.Action, ProjectionPolicy: event.ProjectionPolicy, ConfigRowID: cloneInt(event.ConfigRowID), OldOwnerDigest: append([]byte(nil), event.OldOwnerDigest...), NewOwnerDigest: append([]byte(nil), event.NewOwnerDigest...), BodyDigest: append([]byte(nil), event.BodyDigest...), BindingEventID: cloneUint64(event.BindingEventID)}
	if event.Binding != nil {
		binding, err := ParseProjectionBinding(*event.Binding)
		if err != nil {
			return ArtifactReceipt{}, ErrJournalConflict
		}
		bindingKey, _ := bindingNamespace(binding)
		if bindingKey != key {
			return ArtifactReceipt{}, ErrJournalConflict
		}
		result.Binding = &binding
	}
	switch event.ProjectionPolicy {
	case model.ConfigProjectionGenerated:
		if result.Binding == nil || event.OperationID == nil {
			return ArtifactReceipt{}, ErrJournalConflict
		}
	case model.ConfigProjectionLineageOnly:
		if event.BindingEventID == nil || event.Binding != nil {
			return ArtifactReceipt{}, ErrJournalConflict
		}
	case model.ConfigProjectionNotApplicable:
		if event.Binding != nil || event.BindingEventID != nil {
			return ArtifactReceipt{}, ErrJournalConflict
		}
	default:
		return ArtifactReceipt{}, ErrJournalConflict
	}
	if event.Action != model.ConfigProjectionRender && event.Action != model.ConfigProjectionRetire && event.Action != model.ConfigProjectionInvalidate {
		return ArtifactReceipt{}, ErrJournalConflict
	}
	return result, nil
}

// Only a committed generated root in the same exact namespace authorizes lineage.
// Do not search behind a broken/N/A latest event and resurrect an older grant.
func trustedBindingRoot(tx *gorm.DB, event *model.MonitorConfigArtifactEvent, key configNamespaceKey) (*model.MonitorConfigArtifactEvent, error) {
	if event == nil {
		return nil, nil
	}
	current := event
	seen := map[uint64]bool{}
	for depth := 0; depth < 128; depth++ {
		if current.EventID == 0 || seen[current.EventID] || current.ConfigType != key.kind || string(current.InstanceKey) != key.raw {
			return nil, nil
		}
		seen[current.EventID] = true
		if _, err := artifactReceipt(*current, 0); err != nil {
			return nil, nil
		}
		if current.ProjectionPolicy == model.ConfigProjectionGenerated {
			var op model.MonitorConfigPoolOperation
			// Payload is immutable after business commit. No control lock may be acquired
			// by this namespace-only ordinary path.
			if err := tx.Where("operation_id = ?", *current.OperationID).First(&op).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, nil
				}
				return nil, err
			}
			if !operationStateValid(op) || op.CommittedAt == nil || !familyAllowsKind(op.Family, current.ConfigType) {
				return nil, nil
			}
			entries, err := decodePoolManifest(op.Manifest)
			if err != nil {
				return nil, nil
			}
			for _, entry := range entries {
				if entry.EventID == current.EventID {
					return current, nil
				}
			}
			return nil, nil
		}
		if current.ProjectionPolicy != model.ConfigProjectionLineageOnly || current.BindingEventID == nil || *current.BindingEventID >= current.EventID {
			return nil, nil
		}
		var previous model.MonitorConfigArtifactEvent
		if err := tx.Where("event_id = ?", *current.BindingEventID).First(&previous).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			return nil, err
		}
		current = &previous
	}
	return nil, nil
}
func appendOrdinaryArtifactEvent(tx *gorm.DB, key configNamespaceKey, before, after *model.MonitorConfig) error {
	latest, err := latestArtifactEvent(tx, key)
	if err != nil {
		return err
	}
	policy := model.ConfigProjectionNotApplicable
	var root *model.MonitorConfigArtifactEvent
	if latest != nil && digestEqual(latest.NewOwnerDigest, ownerDigest(before)) {
		root, err = trustedBindingRoot(tx, latest, key)
		if err != nil {
			return err
		}
		rows, err := currentConfigNamespace(tx, key)
		if err != nil {
			return err
		}
		// A historical duplicate is not proof of sole ownership. A vacant namespace
		// after explicit repair is safe; a new/same row must be its only live occupant.
		unambiguous := len(rows) == 0 && after == nil || len(rows) == 1 && after != nil && rows[0].ID == after.ID
		if root != nil && unambiguous {
			policy = model.ConfigProjectionLineageOnly
		}
	}
	event := model.MonitorConfigArtifactEvent{ConfigType: key.kind, InstanceKey: []byte(key.raw), Action: model.ConfigProjectionInvalidate, ProjectionPolicy: policy, OldOwnerDigest: ownerDigest(before), NewOwnerDigest: ownerDigest(after), DeliveryState: model.ConfigProjectionApplied}
	if after != nil {
		id := after.ID
		event.ConfigRowID = &id
		event.BodyDigest = bodyDigest(after)
	} else if before != nil {
		id := before.ID
		event.ConfigRowID = &id
		event.BodyDigest = bodyDigest(before)
	}
	if policy == model.ConfigProjectionLineageOnly {
		id := root.EventID
		event.BindingEventID = &id
		event.DeliveryState = model.ConfigProjectionPending
	}
	return tx.Create(&event).Error
}
func appendConfigTransition(tx *gorm.DB, before, after *model.MonitorConfig) error {
	if before != nil && after != nil && configKey(before) == configKey(after) {
		return appendOrdinaryArtifactEvent(tx, configKey(before), before, after)
	}
	if before != nil {
		if err := appendOrdinaryArtifactEvent(tx, configKey(before), before, nil); err != nil {
			return err
		}
	}
	if after != nil {
		return appendOrdinaryArtifactEvent(tx, configKey(after), nil, after)
	}
	return nil
}

func (d *monitorConfigDAO) MarkPoolOperationFailed(ctx context.Context, ref PoolReservation, code string) (bool, error) {
	if !reservationValid(ref) {
		return false, ErrJournalInvalid
	}
	switch code {
	case "sql_failed", "render_failed", "cancelled", "source_failed", "owner_conflict":
	default:
		return false, ErrJournalInvalid
	}
	changed := false
	err := d.journalTransaction(ctx, func(tx *gorm.DB) error {
		control, err := lockExistingPoolControl(tx, ref.Family, ref.PoolID)
		if err != nil {
			return err
		}
		var op model.MonitorConfigPoolOperation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("operation_id = ?", ref.OperationID).First(&op).Error; err != nil {
			return err
		}
		if !operationMatches(op, ref) {
			return ErrJournalConflict
		}
		if op.Phase != model.ConfigPublicationReserved || op.Manifest != nil || control.CurrentAttemptOperationID == nil || *control.CurrentAttemptOperationID != ref.OperationID {
			return nil
		}
		result := tx.Model(&op).Where("phase = ? AND manifest IS NULL", model.ConfigPublicationReserved).Updates(map[string]interface{}{"phase": model.ConfigPublicationFailed, "failure_code": code})
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

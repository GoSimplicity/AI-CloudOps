package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type preparedPlanItem struct {
	item    GeneratedPlanItem
	input   *model.MonitorConfig
	inputID int
}

func prepareGeneratedPlan(req GeneratedPlanRequest) ([]preparedPlanItem, []configNamespaceKey, string, error) {
	if !reservationValid(req.Reservation) {
		return nil, nil, "", ErrJournalInvalid
	}
	prepared := make([]preparedPlanItem, 0, len(req.Items))
	keys := make([]configNamespaceKey, 0, len(req.Items))
	seenIDs := map[uint64]bool{}
	seenKeys := map[configNamespaceKey]bool{}
	for _, input := range req.Items {
		if input.SourceID == 0 || seenIDs[input.SourceID] {
			return nil, nil, "", ErrJournalInvalid
		}
		seenIDs[input.SourceID] = true
		item := input
		item.Binding = cloneBinding(input.Binding)
		if input.Expected != nil {
			owner := *input.Expected
			item.Expected = &owner
		}
		var key configNamespaceKey
		inputID := 0
		switch item.Action {
		case model.ConfigProjectionRender:
			if input.Desired == nil || input.Binding == nil {
				return nil, nil, "", ErrJournalInvalid
			}
			row := *input.Desired
			if row.DeletedAt != 0 {
				return nil, nil, "", ErrJournalInvalid
			}
			inputID = row.ID
			if item.Expected == nil {
				if row.ID < 0 {
					return nil, nil, "", ErrJournalInvalid
				}
				row.Model = model.Model{}
			} else {
				if row.ID == 0 {
					row.ID = item.Expected.ID
				}
				if row.DeletedAt != 0 {
					return nil, nil, "", ErrJournalInvalid
				}
			}
			if row.Status == 0 {
				row.Status = model.ConfigStatusActive
			}
			if row.Status != model.ConfigStatusActive && row.Status != model.ConfigStatusInactive {
				return nil, nil, "", ErrJournalInvalid
			}
			digest := bodyDigest(&row)
			hash := hex.EncodeToString(digest)
			if row.ConfigHash != "" && row.ConfigHash != hash {
				return nil, nil, "", ErrJournalInvalid
			}
			row.ConfigHash = hash
			if row.PoolID != req.Reservation.PoolID || !familyAllowsKind(req.Reservation.Family, row.ConfigType) || !generatedRowNameValid(&row) {
				return nil, nil, "", ErrJournalInvalid
			}
			if item.Expected != nil && !validGeneratedUpdate(GeneratedConfigUpdate{Expected: *item.Expected, Desired: row}) {
				return nil, nil, "", ErrJournalInvalid
			}
			key = configKey(&row)
			bindingKey, err := bindingNamespace(*item.Binding)
			if err != nil || bindingKey != key {
				return nil, nil, "", ErrJournalInvalid
			}
			item.Desired = &row
		case model.ConfigProjectionRetire:
			if item.Expected == nil || item.Desired != nil || !validGeneratedOwner(*item.Expected) || item.Expected.PoolID != req.Reservation.PoolID || !familyAllowsKind(req.Reservation.Family, item.Expected.ConfigType) {
				return nil, nil, "", ErrJournalInvalid
			}
			key = configNamespaceKey{item.Expected.ConfigType, item.Expected.InstanceIP}
			ownerRow := &model.MonitorConfig{Model: model.Model{ID: item.Expected.ID}, PoolID: item.Expected.PoolID, ConfigType: item.Expected.ConfigType, InstanceIP: item.Expected.InstanceIP, Name: item.Expected.Name}
			if !generatedRowNameValid(ownerRow) {
				return nil, nil, "", ErrJournalInvalid
			}
			if item.Binding != nil {
				bindingKey, err := bindingNamespace(*item.Binding)
				if err != nil || bindingKey != key {
					return nil, nil, "", ErrJournalInvalid
				}
			}
		default:
			return nil, nil, "", ErrJournalInvalid
		}
		if !validConfigNamespace(key) || seenKeys[key] {
			return nil, nil, "", ErrJournalInvalid
		}
		seenKeys[key] = true
		keys = append(keys, key)
		prepared = append(prepared, preparedPlanItem{item: item, input: input.Desired, inputID: inputID})
	}
	sort.Slice(prepared, func(i, j int) bool { return prepared[i].item.SourceID < prepared[j].item.SourceID })
	// Ignore DB-assigned identity/timestamps for create retries after known success.
	fingerprintItems := make([]GeneratedPlanItem, 0, len(prepared))
	for _, entry := range prepared {
		item := entry.item
		if item.Desired != nil {
			copy := *item.Desired
			copy.CreatedAt = time.Time{}
			copy.UpdatedAt = time.Time{}
			item.Desired = &copy
		}
		fingerprintItems = append(fingerprintItems, item)
	}
	raw, err := json.Marshal(fingerprintItems)
	if err != nil {
		return nil, nil, "", ErrJournalInvalid
	}
	digest := sha256.Sum256(raw)
	return prepared, keys, hex.EncodeToString(digest[:]), nil
}

func (d *monitorConfigDAO) CommitGeneratedPlan(ctx context.Context, req GeneratedPlanRequest) (PoolReceipt, error) {
	return d.commitGeneratedPlan(ctx, req, nil)
}

func (d *monitorConfigDAO) commitGeneratedPlan(ctx context.Context, req GeneratedPlanRequest, projection *preparedProjectionPlan) (PoolReceipt, error) {
	prepared, keys, fingerprint, err := prepareGeneratedPlan(req)
	if err != nil {
		return PoolReceipt{}, err
	}
	if projection != nil {
		keys = append(keys, projection.keys...)
		fingerprint = hex.EncodeToString(projection.digest[:])
	}
	var receipt PoolReceipt
	created := map[uint64]*model.MonitorConfig{}
	err = d.journalTransaction(ctx, func(tx *gorm.DB) error {
		var control *model.MonitorConfigPoolControl
		var err error
		var amPair amLockedControls
		if projection != nil && projection.amSource != nil {
			amPair, err = lockAMControls(tx, req.Reservation.PoolID, false)
			control = amPair.own(req.Reservation.Family)
		} else {
			control, err = lockExistingPoolControl(tx, req.Reservation.Family, req.Reservation.PoolID)
		}
		if err != nil {
			return err
		}
		if err := lockConfigNamespacesTx(tx, keys); err != nil {
			return err
		}
		var op model.MonitorConfigPoolOperation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("operation_id = ?", req.Reservation.OperationID).First(&op).Error; err != nil {
			return err
		}
		if !operationMatches(op, req.Reservation) {
			return ErrJournalConflict
		}
		if op.Manifest != nil {
			if !sameHash(req.SourceHash, op.SourceHash) {
				return ErrJournalConflict
			}
			decoded, err := decodeProjectionManifest(op.Manifest)
			if err != nil || (projection == nil && decoded.Version != 1) || (projection != nil && decoded.Version != 2) {
				return ErrJournalConflict
			}
			if projection != nil && decoded.PlanDigest != fingerprint {
				return ErrJournalConflict
			}
			entries, err := decodePoolManifest(op.Manifest)
			if err != nil || len(entries) != len(prepared) {
				return ErrJournalConflict
			}
			for _, entry := range entries {
				if entry.PlanDigest != fingerprint {
					return ErrJournalConflict
				}
			}
			receipt, err = d.loadReceipt(tx, op)
			if err != nil {
				return err
			}
			for _, entry := range prepared {
				if entry.item.Expected == nil && entry.item.Desired != nil && entry.inputID != 0 {
					found := false
					for _, artifact := range receipt.Artifacts {
						if artifact.SourceID == entry.item.SourceID && artifact.ConfigRowID != nil && *artifact.ConfigRowID == entry.inputID {
							found = true
						}
					}
					if !found {
						return ErrJournalConflict
					}
				}
			}
			return nil
		}
		if projection != nil && projection.amSource != nil && !amPair.matchesSource(*projection.amSource, req.Reservation.Family, req.Reservation.Generation) {
			return ErrJournalConflict
		}
		if op.Phase != model.ConfigPublicationReserved || op.CommittedAt != nil || control.CurrentAttemptOperationID == nil || *control.CurrentAttemptOperationID != op.OperationID {
			return ErrJournalConflict
		}
		for _, entry := range prepared {
			if entry.item.Expected == nil && entry.inputID != 0 {
				return ErrJournalInvalid
			}
		}
		if projection != nil {
			if err := validateScrapeRuleHistoryTx(tx, req.Reservation.Family, projection.history); err != nil {
				return err
			}
			for _, proof := range projection.superseded {
				if proof.Origin == RecordSupersededAcceptedArtifact && (op.RecoveryOfOperationID == nil || *op.RecoveryOfOperationID != proof.AcceptedOperationID) {
					return ErrJournalConflict
				}
			}
			if err := d.validateScrapeRuleReplacementsTx(tx, projection.superseded); err != nil {
				return err
			}

		}
		manifest := make([]poolManifestEntry, 0, len(prepared))
		events := map[uint64]model.MonitorConfigArtifactEvent{}
		for _, entry := range prepared {
			event, row, err := commitGeneratedItemTx(tx, op.OperationID, entry.item)
			if err != nil {
				return err
			}
			if err := tx.Create(&event).Error; err != nil {
				return err
			}
			events[entry.item.SourceID] = event
			if row != nil {
				created[entry.item.SourceID] = row
			}
			manifest = append(manifest, poolManifestEntry{SourceID: entry.item.SourceID, EventID: event.EventID, PlanDigest: fingerprint})
		}
		var payload interface{} = manifest
		if projection != nil {
			controls, err := d.resolveProjectionControls(tx, projection, manifest, events)
			if err != nil {
				return err
			}
			envelope := projectionManifestEnvelope{Version: 2, Artifacts: manifest, Controls: controls, PlanDigest: fingerprint}
			if req.Reservation.Family == 4 {
				envelope.RecordSuperseded = projection.superseded
			} else if req.Reservation.Family == 3 {
				envelope.RuleSuperseded = projection.superseded
			} else if req.Reservation.Family == 1 {
				envelope.PromSuperseded = projection.superseded
			} else if validAMFamily(req.Reservation.Family) {
				envelope.AMSuperseded = projection.superseded
			} else if len(projection.superseded) > 0 {
				return ErrJournalInvalid
			}
			if projection.amControls != nil {
				vector := *projection.amControls
				envelope.AMControls = &vector
			}
			payload = envelope
			if err := markScrapeRuleReplacementsTx(tx, projection.superseded); err != nil {
				return err
			}
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return ErrJournalInvalid
		}
		text := string(encoded)
		now := time.Now()
		committed := tx.Model(&op).Where("phase = ? AND manifest IS NULL", model.ConfigPublicationReserved).Updates(map[string]interface{}{"phase": model.ConfigPublicationCommitted, "source_hash": req.SourceHash[:], "manifest": text, "committed_at": now, "failure_code": ""})
		if committed.Error != nil {
			return committed.Error
		}
		if committed.RowsAffected != 1 {
			return ErrJournalConflict
		}
		if err := tx.Model(control).Updates(map[string]interface{}{"last_committed_operation_id": op.OperationID}).Error; err != nil {
			return err
		}
		op.Phase = model.ConfigPublicationCommitted
		op.SourceHash = append([]byte(nil), req.SourceHash[:]...)
		op.Manifest = &text
		op.CommittedAt = &now
		receipt, err = d.loadReceipt(tx, op)
		return err
	})
	if err != nil {
		return PoolReceipt{}, journalError(ctx, err)
	}
	for _, entry := range prepared {
		if entry.input != nil {
			if row := created[entry.item.SourceID]; row != nil {
				*entry.input = *row
			} else if entry.item.Expected == nil {
				for _, artifact := range receipt.Artifacts {
					if artifact.SourceID == entry.item.SourceID && artifact.ConfigRowID != nil {
						entry.input.ID = *artifact.ConfigRowID
					}
				}
			}
		}
	}
	return receipt, nil
}

func commitGeneratedItemTx(tx *gorm.DB, operationID uint64, item GeneratedPlanItem) (model.MonitorConfigArtifactEvent, *model.MonitorConfig, error) {
	var key configNamespaceKey
	if item.Desired != nil {
		key = configKey(item.Desired)
	} else {
		key = configNamespaceKey{item.Expected.ConfigType, item.Expected.InstanceIP}
	}
	rows, err := currentConfigNamespace(tx, key)
	if err != nil {
		return model.MonitorConfigArtifactEvent{}, nil, err
	}
	var before, after *model.MonitorConfig
	if item.Action == model.ConfigProjectionRender {
		desired := *item.Desired
		if item.Expected == nil {
			if len(rows) != 0 {
				return model.MonitorConfigArtifactEvent{}, nil, ErrJournalConflict
			}
			if err := tx.Create(&desired).Error; err != nil {
				return model.MonitorConfigArtifactEvent{}, nil, err
			}
			after = &desired
		} else {
			if len(rows) != 1 || (GeneratedConfigOwnerOf(rows[0]) != *item.Expected && !generatedDesiredMatches(*rows[0], desired)) {
				return model.MonitorConfigArtifactEvent{}, nil, ErrJournalConflict
			}
			old := *rows[0]
			before = &old
			result := generatedOwnerQuery(tx.Model(&model.MonitorConfig{}), *item.Expected).Updates(map[string]interface{}{"name": desired.Name, "pool_id": desired.PoolID, "config_type": desired.ConfigType, "instance_ip": desired.InstanceIP, "config_content": desired.ConfigContent, "config_hash": desired.ConfigHash, "status": desired.Status, "last_generated_time": desired.LastGeneratedTime})
			if result.Error != nil {
				return model.MonitorConfigArtifactEvent{}, nil, result.Error
			}
			if result.RowsAffected > 1 {
				return model.MonitorConfigArtifactEvent{}, nil, ErrJournalConflict
			}
			after, err = currentConfigByID(tx, desired.ID)
			if err != nil {
				return model.MonitorConfigArtifactEvent{}, nil, err
			}
			if !generatedDesiredMatches(*after, desired) {
				return model.MonitorConfigArtifactEvent{}, nil, ErrJournalConflict
			}
		}
	} else {
		if len(rows) != 0 {
			if len(rows) != 1 || GeneratedConfigOwnerOf(rows[0]) != *item.Expected {
				return model.MonitorConfigArtifactEvent{}, nil, ErrJournalConflict
			}
			old := *rows[0]
			before = &old
		} else {
			var retired model.MonitorConfig
			if err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).First(&retired, item.Expected.ID).Error; err != nil {
				return model.MonitorConfigArtifactEvent{}, nil, err
			}
			if retired.DeletedAt == 0 || GeneratedConfigOwnerOf(&retired) != *item.Expected {
				return model.MonitorConfigArtifactEvent{}, nil, ErrJournalConflict
			}
			before = &retired
		}
	}
	event := model.MonitorConfigArtifactEvent{OperationID: &operationID, ConfigType: key.kind, InstanceKey: []byte(key.raw), Action: item.Action, ProjectionPolicy: model.ConfigProjectionGenerated, OldOwnerDigest: ownerDigest(before), NewOwnerDigest: ownerDigest(after), DeliveryState: model.ConfigProjectionPending}
	if item.Binding != nil {
		binding, err := canonicalBinding(item.Binding)
		if err != nil {
			return event, nil, err
		}
		event.Binding = binding
	} else {
		latest, err := latestArtifactEvent(tx, key)
		if err != nil {
			return event, nil, err
		}
		if latest == nil || !(digestEqual(latest.NewOwnerDigest, ownerDigest(before)) || latest.NewOwnerDigest == nil && digestEqual(latest.OldOwnerDigest, ownerDigest(before))) {
			return event, nil, ErrJournalConflict
		}
		root, err := trustedBindingRoot(tx, latest, key)
		if err != nil {
			return event, nil, err
		}
		if root == nil {
			return event, nil, ErrJournalConflict
		}
		event.ProjectionPolicy = model.ConfigProjectionLineageOnly
		event.BindingEventID = &root.EventID
	}
	if after != nil {
		id := after.ID
		event.ConfigRowID = &id
		event.BodyDigest = bodyDigest(after)
	} else {
		id := before.ID
		event.ConfigRowID = &id
		event.BodyDigest = bodyDigest(before)
		if before.DeletedAt == 0 {
			result := generatedOwnerQuery(tx, *item.Expected).Delete(&model.MonitorConfig{})
			if result.Error != nil {
				return event, nil, result.Error
			}
			if result.RowsAffected != 1 {
				return event, nil, ErrJournalConflict
			}
		}
	}
	return event, after, nil
}

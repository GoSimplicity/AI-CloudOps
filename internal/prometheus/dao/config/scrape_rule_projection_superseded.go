package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"sort"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

func (d *monitorConfigDAO) getSupersededScrapeRuleArtifacts(ctx context.Context, scope scrapePublicationScope, previous ProjectionReceipt) ([]RecordSupersededArtifact, error) {
	if !validScrapePublicationScope(scope) {
		return nil, ErrJournalInvalid
	}
	result := []RecordSupersededArtifact{}
	if previous.ManifestVersion != 2 || !previous.CompleteControls || previous.Reservation.Family != int8(scope) {
		return result, nil
	}
	for _, artifact := range previous.Artifacts {
		if (artifact.Action != model.ConfigProjectionRetire && artifact.Action != model.ConfigProjectionRender) || (previous.Phase == model.ConfigPublicationProjected && artifact.Action == model.ConfigProjectionRetire) {
			continue
		}
		proof, err := d.readAcceptedScrapeRuleReplacement(d.db.WithContext(ctx), previous.Reservation.OperationID, artifact.EventID)
		if err != nil {
			return nil, journalError(ctx, err)
		}
		if proof != nil {
			result = append(result, *proof)
		}
	}
	return result, nil
}

// Both preflight and the business transaction use this exact predicate. The
// transaction caller already owns A's control and the shared namespace guard;
// replacement pool control is never locked after a namespace.
func (d *monitorConfigDAO) readAcceptedScrapeRuleReplacement(tx *gorm.DB, operationID, eventID uint64) (*RecordSupersededArtifact, error) {
	var oldOp model.MonitorConfigPoolOperation
	if err := tx.Where("operation_id = ?", operationID).First(&oldOp).Error; err != nil {
		return nil, err
	}
	if !validScrapePublicationScope(scrapePublicationScope(oldOp.Family)) || (oldOp.Phase != model.ConfigPublicationCommitted && oldOp.Phase != model.ConfigPublicationProjected) {
		return nil, nil
	}
	old, err := d.loadProjectionReceipt(tx, oldOp)
	if err != nil {
		return nil, err
	}
	if old.ManifestVersion != 2 || !old.CompleteControls {
		return nil, nil
	}
	var retirement *ArtifactReceipt
	for i := range old.Artifacts {
		if old.Artifacts[i].EventID == eventID {
			retirement = &old.Artifacts[i]
		}
	}
	if retirement == nil || (retirement.Action != model.ConfigProjectionRetire && retirement.Action != model.ConfigProjectionRender) || retirement.ConfigRowID == nil || retirement.ConfigType != oldOp.Family {
		return nil, nil
	}
	var oldRow model.MonitorConfig
	if err := tx.Unscoped().First(&oldRow, *retirement.ConfigRowID).Error; err != nil {
		return nil, err
	}
	if oldRow.DeletedAt == 0 || oldRow.PoolID != oldOp.PoolID || oldRow.InstanceIP != retirement.InstanceKey || !generatedRowNameValid(&oldRow) || !digestEqual(ownerDigest(&oldRow), scrapeRuleAcceptedOwnerDigest(*retirement)) {
		return nil, nil
	}
	proof, err := d.readProjectedScrapeRuleReplacement(tx, GeneratedConfigOwnerOf(&oldRow))
	if err != nil || proof == nil {
		return nil, err
	}
	proof.Origin = RecordSupersededAcceptedArtifact
	proof.RetiredDeletedAt = uint64(oldRow.DeletedAt)
	proof.AcceptedOperationID = operationID
	proof.AcceptedEventID = eventID
	proof.AcceptedAction = retirement.Action
	return proof, nil
}

func (d *monitorConfigDAO) readProjectedScrapeRuleReplacement(tx *gorm.DB, retired GeneratedConfigOwner) (*RecordSupersededArtifact, error) {
	latest, err := latestArtifactEvent(tx, configNamespaceKey{retired.ConfigType, retired.InstanceIP})
	if err != nil {
		return nil, err
	}
	if latest == nil || latest.OperationID == nil || latest.ConfigRowID == nil || latest.ProjectionPolicy != model.ConfigProjectionGenerated || latest.Action != model.ConfigProjectionRender || latest.DeliveryState != model.ConfigProjectionApplied {
		return nil, nil
	}
	rows, err := currentConfigNamespace(tx, configNamespaceKey{retired.ConfigType, retired.InstanceIP})
	if err != nil {
		return nil, err
	}
	if len(rows) != 1 {
		return nil, nil
	}
	row := rows[0]
	if row.ID != *latest.ConfigRowID || row.PoolID == retired.PoolID || !generatedRowNameValid(row) || !digestEqual(ownerDigest(row), latest.NewOwnerDigest) || !digestEqual(bodyDigest(row), latest.BodyDigest) {
		return nil, nil
	}
	if (row.Status != model.ConfigStatusActive && row.Status != model.ConfigStatusInactive) || row.ConfigHash != hex.EncodeToString(bodyDigest(row)) {
		return nil, nil
	}
	var replacementOp model.MonitorConfigPoolOperation
	if err := tx.Where("operation_id = ?", *latest.OperationID).First(&replacementOp).Error; err != nil {
		return nil, err
	}
	if replacementOp.Family != retired.ConfigType || replacementOp.PoolID != row.PoolID || replacementOp.Phase != model.ConfigPublicationProjected || replacementOp.ProjectedAt == nil {
		return nil, nil
	}
	replacement, err := d.loadProjectionReceipt(tx, replacementOp)
	if err != nil {
		return nil, err
	}
	if replacement.ManifestVersion != 2 || !replacement.CompleteControls {
		return nil, nil
	}
	found := false
	for _, artifact := range replacement.Artifacts {
		if artifact.EventID == latest.EventID && artifact.Action == model.ConfigProjectionRender && artifact.ConfigRowID != nil && *artifact.ConfigRowID == row.ID && artifact.Binding != nil && artifact.Binding.ConfigType == retired.ConfigType && artifact.Binding.InstanceIP == row.InstanceIP {
			found = true
		}
	}
	if !found {
		return nil, nil
	}
	return &RecordSupersededArtifact{RetiredOwner: retired, ReplacementOperationID: replacementOp.OperationID, ReplacementEventID: latest.EventID, ReplacementOwner: GeneratedConfigOwnerOf(row), ReplacementBodyDigest: sha256.Sum256([]byte(row.ConfigContent)), ReplacementHash: row.ConfigHash, ReplacementStatus: row.Status}, nil
}

func (d *monitorConfigDAO) validateScrapeRuleReplacementsTx(tx *gorm.DB, proofs []RecordSupersededArtifact) error {
	for _, expected := range proofs {
		var actual *RecordSupersededArtifact
		var err error
		if expected.Origin == RecordSupersededLegacyOwner {
			actual, err = d.readLegacyScrapeRuleReplacement(tx, expected.RetiredOwner)
		} else {
			actual, err = d.readAcceptedScrapeRuleReplacement(tx, expected.AcceptedOperationID, expected.AcceptedEventID)
		}
		if err != nil {
			return err
		}
		if actual == nil || !reflect.DeepEqual(*actual, expected) {
			return ErrJournalConflict
		}
	}
	return nil
}
func markScrapeRuleReplacementsTx(tx *gorm.DB, proofs []RecordSupersededArtifact) error {
	for _, proof := range proofs {
		if proof.Origin == RecordSupersededLegacyOwner {
			continue
		}
		var event model.MonitorConfigArtifactEvent
		if err := tx.Where("event_id = ? AND operation_id = ?", proof.AcceptedEventID, proof.AcceptedOperationID).First(&event).Error; err != nil {
			return err
		}
		if event.Action != proof.AcceptedAction || (event.Action != model.ConfigProjectionRetire && event.Action != model.ConfigProjectionRender) {
			return ErrJournalConflict
		}
		if event.DeliveryState == model.ConfigProjectionApplied || event.DeliveryState == model.ConfigProjectionSuperseded {
			continue
		}
		if event.DeliveryState != model.ConfigProjectionPending {
			return ErrJournalConflict
		}
		result := tx.Model(&event).Where("delivery_state = ?", model.ConfigProjectionPending).Update("delivery_state", model.ConfigProjectionSuperseded)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrJournalConflict
		}
	}
	return nil
}

func prepareScrapeRuleReplacements(req RecordProjectionPlanRequest) ([]RecordSupersededArtifact, error) {
	result := append([]RecordSupersededArtifact{}, req.Superseded...)
	seen := map[string]bool{}
	itemKeys := map[string]bool{}
	for _, item := range req.Items {
		if item.Desired != nil {
			itemKeys[item.Desired.InstanceIP] = true
		} else if item.Expected != nil {
			itemKeys[item.Expected.InstanceIP] = true
		}
	}
	for _, proof := range result {
		a, b := proof.RetiredOwner, proof.ReplacementOwner
		if proof.ReplacementOperationID == 0 || proof.ReplacementEventID == 0 || !validGeneratedOwner(a) || !validGeneratedOwner(b) || a.ConfigType != req.Reservation.Family || b.ConfigType != req.Reservation.Family || a.PoolID != req.Reservation.PoolID || b.PoolID == a.PoolID || a.InstanceIP != b.InstanceIP || seen[a.InstanceIP] || itemKeys[a.InstanceIP] {
			return nil, ErrJournalInvalid
		}
		for _, owner := range []GeneratedConfigOwner{a, b} {
			row := &model.MonitorConfig{Model: model.Model{ID: owner.ID}, PoolID: owner.PoolID, ConfigType: owner.ConfigType, InstanceIP: owner.InstanceIP, Name: owner.Name}
			if !generatedRowNameValid(row) {
				return nil, ErrJournalInvalid
			}
		}
		if proof.ReplacementStatus != model.ConfigStatusActive && proof.ReplacementStatus != model.ConfigStatusInactive {
			return nil, ErrJournalInvalid
		}
		if proof.RetiredDeletedAt == 0 {
			return nil, ErrJournalInvalid
		}
		switch proof.Origin {
		case RecordSupersededAcceptedArtifact:
			if proof.AcceptedOperationID == 0 || proof.AcceptedEventID == 0 || (proof.AcceptedAction != model.ConfigProjectionRender && proof.AcceptedAction != model.ConfigProjectionRetire) {
				return nil, ErrJournalInvalid
			}
		case RecordSupersededLegacyOwner:
			if proof.AcceptedOperationID != 0 || proof.AcceptedEventID != 0 || proof.AcceptedAction != "" {
				return nil, ErrJournalInvalid
			}
		default:
			return nil, ErrJournalInvalid
		}
		seen[a.InstanceIP] = true
	}
	sort.Slice(result, func(i, j int) bool { return result[i].RetiredOwner.InstanceIP < result[j].RetiredOwner.InstanceIP })
	return result, nil
}

// Historical proof validation deliberately does not require B's row to remain
// current forever. Its immutable issued artifact and completed operation prove
// the replacement that was checked under the business transaction's NS lock.
func (d *monitorConfigDAO) validateScrapeRuleReplacementHistory(tx *gorm.DB, recovery model.MonitorConfigPoolOperation, proofs []RecordSupersededArtifact) error {
	for _, proof := range proofs {
		if !validScrapePublicationScope(scrapePublicationScope(recovery.Family)) || proof.RetiredOwner.ConfigType != recovery.Family || proof.RetiredOwner.PoolID != recovery.PoolID {
			return ErrJournalConflict
		}
		if proof.Origin == RecordSupersededAcceptedArtifact {
			if recovery.RecoveryOfOperationID == nil || *recovery.RecoveryOfOperationID != proof.AcceptedOperationID || proof.ReplacementEventID <= proof.AcceptedEventID {
				return ErrJournalConflict
			}
			oldOp, old, event, err := d.loadScrapeRuleIssuedArtifact(tx, proof.AcceptedOperationID, proof.AcceptedEventID)
			if err != nil {
				return err
			}
			if oldOp.Family != recovery.Family || oldOp.PoolID != recovery.PoolID || old.Action != proof.AcceptedAction || old.ConfigRowID == nil || *old.ConfigRowID != proof.RetiredOwner.ID || old.InstanceKey != proof.RetiredOwner.InstanceIP {
				return ErrJournalConflict
			}
			oldRow := scrapeRuleOwnerModel(proof.RetiredOwner)
			if !digestEqual(scrapeRuleAcceptedOwnerDigest(old), ownerDigest(oldRow)) || (old.Action == model.ConfigProjectionRetire && len(old.NewOwnerDigest) != 0) {
				return ErrJournalConflict
			}
			if event.DeliveryState != model.ConfigProjectionSuperseded && event.DeliveryState != model.ConfigProjectionApplied {
				return ErrJournalConflict
			}
		} else if proof.Origin == RecordSupersededLegacyOwner {
			var retired model.MonitorConfig
			if err := tx.Unscoped().First(&retired, proof.RetiredOwner.ID).Error; err != nil {
				return err
			}
			if uint64(retired.DeletedAt) != proof.RetiredDeletedAt || retired.DeletedAt == 0 || GeneratedConfigOwnerOf(&retired) != proof.RetiredOwner || !generatedRowNameValid(&retired) {
				return ErrJournalConflict
			}
		} else {
			return ErrJournalConflict
		}
		replacementOp, replacement, replacementEvent, err := d.loadScrapeRuleIssuedArtifact(tx, proof.ReplacementOperationID, proof.ReplacementEventID)
		if err != nil {
			return err
		}
		if replacementOp.Family != recovery.Family || replacementOp.Phase != model.ConfigPublicationProjected || replacementOp.ProjectedAt == nil || replacementOp.PoolID != proof.ReplacementOwner.PoolID || replacementOp.PoolID == recovery.PoolID || replacement.Action != model.ConfigProjectionRender || replacement.ProjectionPolicy != model.ConfigProjectionGenerated || replacementEvent.DeliveryState != model.ConfigProjectionApplied {
			return ErrJournalConflict
		}
		if replacement.ConfigRowID == nil || *replacement.ConfigRowID != proof.ReplacementOwner.ID || replacement.InstanceKey != proof.RetiredOwner.InstanceIP || replacement.Binding == nil || replacement.Binding.ConfigType != recovery.Family || replacement.Binding.InstanceIP != proof.RetiredOwner.InstanceIP {
			return ErrJournalConflict
		}
		if !digestEqual(replacement.NewOwnerDigest, ownerDigest(scrapeRuleOwnerModel(proof.ReplacementOwner))) || !digestEqual(replacement.BodyDigest, proof.ReplacementBodyDigest[:]) || proof.ReplacementHash != hex.EncodeToString(proof.ReplacementBodyDigest[:]) {
			return ErrJournalConflict
		}
	}
	return nil
}
func scrapeRuleOwnerModel(owner GeneratedConfigOwner) *model.MonitorConfig {
	return &model.MonitorConfig{Model: model.Model{ID: owner.ID}, PoolID: owner.PoolID, ConfigType: owner.ConfigType, InstanceIP: owner.InstanceIP, Name: owner.Name}
}
func (d *monitorConfigDAO) loadScrapeRuleIssuedArtifact(tx *gorm.DB, opID, eventID uint64) (model.MonitorConfigPoolOperation, ArtifactReceipt, model.MonitorConfigArtifactEvent, error) {
	var op model.MonitorConfigPoolOperation
	var event model.MonitorConfigArtifactEvent
	if err := tx.Where("operation_id = ?", opID).First(&op).Error; err != nil {
		return op, ArtifactReceipt{}, event, err
	}
	if !validScrapePublicationScope(scrapePublicationScope(op.Family)) {
		return op, ArtifactReceipt{}, event, ErrJournalConflict
	}
	base, err := d.loadReceipt(tx, op)
	if err != nil {
		return op, ArtifactReceipt{}, event, err
	}
	envelope, err := decodeProjectionManifest(op.Manifest)
	if err != nil || envelope.Version != 2 || len(envelope.Controls) != 2 {
		return op, ArtifactReceipt{}, event, ErrJournalConflict
	}
	members, ack := false, false
	for _, control := range envelope.Controls {
		if control.Kind == ProjectionMembersFull && control.Members != nil {
			members = true
		} else if control.Kind == ProjectionACKFull && control.Ack != nil && control.Ack.LegacySHA != nil && sameHash(*control.Ack.LegacySHA, op.SourceHash) {
			ack = true
		} else {
			return op, ArtifactReceipt{}, event, ErrJournalConflict
		}
	}
	if !members || !ack {
		return op, ArtifactReceipt{}, event, ErrJournalConflict
	}
	desired := map[string]bool{}
	for _, artifact := range base.Artifacts {
		if artifact.Action == model.ConfigProjectionRender {
			if artifact.Binding == nil || artifact.Binding.ConfigType != op.Family {
				return op, ArtifactReceipt{}, event, ErrJournalConflict
			}
			desired[artifact.Binding.InstanceIP] = true
		}
	}
	for _, control := range envelope.Controls {
		if control.Members != nil {
			seen := map[string]bool{}
			for _, ip := range control.Members.DesiredIPs {
				if !desired[ip] || seen[ip] {
					return op, ArtifactReceipt{}, event, ErrJournalConflict
				}
				seen[ip] = true
			}
			if len(seen) != len(desired) {
				return op, ArtifactReceipt{}, event, ErrJournalConflict
			}
		}
	}

	for _, artifact := range base.Artifacts {
		if artifact.EventID == eventID {
			if err := tx.Where("event_id = ?", eventID).First(&event).Error; err != nil {
				return op, ArtifactReceipt{}, event, err
			}
			return op, artifact, event, nil
		}
	}
	return op, ArtifactReceipt{}, event, ErrJournalConflict
}

func scrapeRuleAcceptedOwnerDigest(artifact ArtifactReceipt) []byte {
	if artifact.Action == model.ConfigProjectionRender {
		return artifact.NewOwnerDigest
	}
	return artifact.OldOwnerDigest
}

func (d *monitorConfigDAO) readLegacyScrapeRuleReplacement(tx *gorm.DB, owner GeneratedConfigOwner) (*RecordSupersededArtifact, error) {
	var row model.MonitorConfig
	if err := tx.Unscoped().First(&row, owner.ID).Error; err != nil {
		return nil, err
	}
	if row.DeletedAt == 0 || GeneratedConfigOwnerOf(&row) != owner || !validScrapePublicationScope(scrapePublicationScope(owner.ConfigType)) || !generatedRowNameValid(&row) {
		return nil, nil
	}
	proof, err := d.readProjectedScrapeRuleReplacement(tx, owner)
	if err != nil || proof == nil {
		return nil, err
	}
	proof.Origin = RecordSupersededLegacyOwner
	proof.RetiredDeletedAt = uint64(row.DeletedAt)
	return proof, nil
}
func (d *monitorConfigDAO) getLegacySupersededScrapeRuleArtifacts(ctx context.Context, scope scrapePublicationScope, poolID int, owners []GeneratedConfigOwner) ([]RecordSupersededArtifact, error) {
	if !validScrapePublicationScope(scope) {
		return nil, ErrJournalInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if poolID <= 0 || len(owners) > GeneratedConfigQueryLimit {
		return nil, ErrInvalidGeneratedConfigQuery
	}
	result := []RecordSupersededArtifact{}
	seen := map[int]bool{}
	for _, owner := range owners {
		if !validGeneratedOwner(owner) || owner.PoolID != poolID || owner.ConfigType != int8(scope) || seen[owner.ID] {
			return nil, ErrInvalidGeneratedConfigQuery
		}
		seen[owner.ID] = true
		proof, err := d.readLegacyScrapeRuleReplacement(d.db.WithContext(ctx), owner)
		if err != nil {
			return nil, journalError(ctx, err)
		}
		if proof != nil {
			result = append(result, *proof)
		}
	}
	return result, nil
}

func (d *monitorConfigDAO) GetSupersededAlertRuleArtifacts(ctx context.Context, previous ProjectionReceipt) ([]ScrapeRuleSupersededArtifact, error) {
	return d.getSupersededScrapeRuleArtifacts(ctx, scrapeAlertPublication, previous)
}
func (d *monitorConfigDAO) GetLegacySupersededAlertRuleArtifacts(ctx context.Context, poolID int, owners []GeneratedConfigOwner) ([]ScrapeRuleSupersededArtifact, error) {
	return d.getLegacySupersededScrapeRuleArtifacts(ctx, scrapeAlertPublication, poolID, owners)
}

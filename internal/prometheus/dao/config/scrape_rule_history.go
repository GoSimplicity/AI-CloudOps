package config

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

func (d *monitorConfigDAO) getProjectedScrapeRuleRetirements(ctx context.Context, scope ScrapeRuleScope, poolID int, owners []GeneratedConfigOwner) ([]GeneratedConfigOwner, error) {
	if !validScrapeRuleScope(scope) {
		return nil, ErrJournalInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if poolID <= 0 || len(owners) > GeneratedConfigQueryLimit {
		return nil, ErrInvalidGeneratedConfigQuery
	}
	if len(owners) == 0 {
		return []GeneratedConfigOwner{}, nil
	}
	ids := make([]int, 0, len(owners))
	byID := map[int]GeneratedConfigOwner{}
	keys := make([][]byte, 0, len(owners))
	for _, owner := range owners {
		if !validGeneratedOwner(owner) || owner.PoolID != poolID || owner.ConfigType != int8(scope) || byID[owner.ID].ID != 0 {
			return nil, ErrInvalidGeneratedConfigQuery
		}
		byID[owner.ID] = owner
		ids = append(ids, owner.ID)
		keys = append(keys, []byte(owner.InstanceIP))
	}
	var events []model.MonitorConfigArtifactEvent
	err := d.db.WithContext(ctx).Where("config_type = ? AND instance_key IN ? AND config_row_id IN ? AND action IN ? AND delivery_state IN ? AND operation_id IS NOT NULL", int8(scope), keys, ids, []string{model.ConfigProjectionRetire, model.ConfigProjectionRender}, []string{model.ConfigProjectionApplied, model.ConfigProjectionSuperseded}).Find(&events).Error
	if err != nil {
		return nil, generatedConfigReadError(ctx)
	}
	operations := map[uint64]*ProjectionReceipt{}
	completed := map[int]bool{}
	for _, event := range events {
		if event.OperationID == nil || event.ConfigRowID == nil {
			continue
		}
		owner, ok := byID[*event.ConfigRowID]
		if !ok || string(event.InstanceKey) != owner.InstanceIP {
			continue
		}
		snapshot := &model.MonitorConfig{Model: model.Model{ID: owner.ID}, PoolID: owner.PoolID, ConfigType: owner.ConfigType, InstanceIP: owner.InstanceIP, Name: owner.Name}
		attested := event.OldOwnerDigest
		if event.Action == model.ConfigProjectionRender {
			attested = event.NewOwnerDigest
		}
		if !digestEqual(attested, ownerDigest(snapshot)) || (event.Action == model.ConfigProjectionRetire && len(event.NewOwnerDigest) != 0) {
			continue
		}
		var original model.MonitorConfigPoolOperation
		if err := d.db.WithContext(ctx).Where("operation_id = ?", *event.OperationID).First(&original).Error; err != nil {
			return nil, generatedConfigReadError(ctx)
		}
		if original.Family != int8(scope) || original.PoolID != poolID {
			continue
		}
		if event.Action == model.ConfigProjectionRender || original.Phase != model.ConfigPublicationProjected || event.DeliveryState != model.ConfigProjectionApplied {
			resolved, err := d.scrapeRuleRetirementResolvedByRecovery(d.db.WithContext(ctx), original, event, owner)
			if err != nil {
				return nil, generatedConfigReadError(ctx)
			}
			if resolved {
				completed[owner.ID] = true
			}
			continue
		}
		receipt, loaded := operations[*event.OperationID]
		if !loaded {
			var op model.MonitorConfigPoolOperation
			if err := d.db.WithContext(ctx).Where("operation_id = ?", *event.OperationID).First(&op).Error; err != nil {
				return nil, generatedConfigReadError(ctx)
			}
			if op.Family != int8(scope) || op.PoolID != poolID || op.Phase != model.ConfigPublicationProjected || op.ProjectedAt == nil {
				operations[*event.OperationID] = nil
				continue
			}
			value, err := d.loadProjectionReceipt(d.db.WithContext(ctx), op)
			if err != nil {
				return nil, generatedConfigReadError(ctx)
			}
			if value.ManifestVersion != 2 || !value.CompleteControls {
				operations[*event.OperationID] = nil
				continue
			}
			receipt = &value
			operations[*event.OperationID] = receipt
		}
		if receipt == nil {
			continue
		}
		for _, artifact := range receipt.Artifacts {
			if artifact.EventID == event.EventID && artifact.Action == model.ConfigProjectionRetire && artifact.ConfigRowID != nil && *artifact.ConfigRowID == owner.ID && artifact.InstanceKey == owner.InstanceIP {
				completed[owner.ID] = true
			}
		}
	}
	remaining := map[int]GeneratedConfigOwner{}
	for _, owner := range owners {
		if !completed[owner.ID] {
			remaining[owner.ID] = owner
		}
	}
	if len(remaining) > 0 {
		resolved, err := d.scrapeRuleLegacyOwnersResolvedByManifests(d.db.WithContext(ctx), int8(scope), poolID, remaining)
		if err != nil {
			return nil, generatedConfigReadError(ctx)
		}
		for id := range resolved {
			completed[id] = true
		}
	}
	result := []GeneratedConfigOwner{}
	for _, owner := range owners {
		if completed[owner.ID] {
			result = append(result, owner)
		}
	}
	return result, nil
}

func (d *monitorConfigDAO) scrapeRuleRetirementResolvedByRecovery(tx *gorm.DB, old model.MonitorConfigPoolOperation, event model.MonitorConfigArtifactEvent, owner GeneratedConfigOwner) (bool, error) {
	var operations []model.MonitorConfigPoolOperation
	if err := tx.Where("family = ? AND pool_id = ? AND recovery_of_operation_id = ? AND phase IN ? AND manifest IS NOT NULL", old.Family, old.PoolID, old.OperationID, []string{model.ConfigPublicationCommitted, model.ConfigPublicationProjected}).Order("operation_id ASC").Find(&operations).Error; err != nil {
		return false, err
	}
	for _, operation := range operations {
		receipt, err := d.loadProjectionReceipt(tx, operation)
		if err != nil {
			return false, err
		}
		if receipt.ManifestVersion != 2 || !receipt.CompleteControls {
			continue
		}
		envelope, err := decodeProjectionManifest(operation.Manifest)
		if err != nil {
			return false, err
		}
		proofs, err := scopedRuleSuperseded(envelope, operation.Family)
		if err != nil {
			return false, err
		}
		for _, proof := range proofs {
			if proof.AcceptedOperationID == old.OperationID && proof.AcceptedEventID == event.EventID && proof.AcceptedAction == event.Action && proof.RetiredOwner == owner {
				return true, nil
			}
		}
	}
	return false, nil
}

// Legacy owners have no generated event or accepted operation to use as an
// index. Their first typed control-only operation durably records the evidence.
func (d *monitorConfigDAO) scrapeRuleLegacyOwnersResolvedByManifests(tx *gorm.DB, family int8, poolID int, owners map[int]GeneratedConfigOwner) (map[int]bool, error) {
	var operations []model.MonitorConfigPoolOperation
	if err := tx.Where("family = ? AND pool_id = ? AND phase IN ? AND manifest LIKE ?", family, poolID, []string{model.ConfigPublicationCommitted, model.ConfigPublicationProjected}, scrapeRuleSidePattern(family)).Order("operation_id ASC").Find(&operations).Error; err != nil {
		return nil, err
	}
	result := map[int]bool{}
	for _, operation := range operations {
		envelope, err := decodeProjectionManifest(operation.Manifest)
		if err != nil {
			return nil, err
		}
		proofs, err := scopedRuleSuperseded(envelope, operation.Family)
		if err != nil {
			return nil, err
		}
		relevant := false
		for _, proof := range proofs {
			if proof.Origin == RecordSupersededLegacyOwner && owners[proof.RetiredOwner.ID] == proof.RetiredOwner {
				relevant = true
			}
		}
		if !relevant {
			continue
		}
		receipt, err := d.loadProjectionReceipt(tx, operation)
		if err != nil {
			return nil, err
		}
		if receipt.ManifestVersion != 2 || !receipt.CompleteControls {
			return nil, ErrJournalConflict
		}
		for _, proof := range proofs {
			if proof.Origin == RecordSupersededLegacyOwner && owners[proof.RetiredOwner.ID] == proof.RetiredOwner {
				result[proof.RetiredOwner.ID] = true
			}
		}
	}
	return result, nil
}

func (d *monitorConfigDAO) GetProjectedAlertRuleRetirements(ctx context.Context, poolID int, owners []GeneratedConfigOwner) ([]GeneratedConfigOwner, error) {
	return d.getProjectedScrapeRuleRetirements(ctx, ScrapeAlertRule, poolID, owners)
}

func scrapeRuleSidePattern(family int8) string {
	if family == 3 {
		return `%"rule_superseded"%`
	}
	return `%"record_superseded"%`
}

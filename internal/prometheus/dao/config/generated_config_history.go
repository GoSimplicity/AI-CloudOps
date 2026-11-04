package config

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GeneratedConfigHistoryReader locates historical owners at exact namespaces.
// It never filters PoolID/status/deletion state: ambiguity must remain visible.
// A returned row is a locator, not permission to delete an artifact.
type GeneratedConfigHistoryReader interface {
	GetLegacySupersededRecordArtifacts(context.Context, int, []GeneratedConfigOwner) ([]RecordSupersededArtifact, error)
	GetSupersededRecordArtifacts(context.Context, ProjectionReceipt) ([]RecordSupersededArtifact, error)
	GetGeneratedConfigHistoryCandidates(context.Context, GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error)
	GetProjectedRecordRetirements(context.Context, int, []GeneratedConfigOwner) ([]GeneratedConfigOwner, error)
	GetGeneratedConfigHistoryPage(context.Context, GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error)
}

func NewGeneratedConfigHistoryReader(logger *zap.Logger, db *gorm.DB) GeneratedConfigHistoryReader {
	return &monitorConfigDAO{l: logger, db: db}
}
func (d *monitorConfigDAO) GetGeneratedConfigHistoryPage(ctx context.Context, req GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	if !validGeneratedConfigPage(req.ConfigType, req.Page, req.Size) || len(req.InstanceKeys) > GeneratedConfigQueryLimit {
		return nil, 0, ErrInvalidGeneratedConfigQuery
	}
	for _, key := range req.InstanceKeys {
		if !validConfigNamespace(configNamespaceKey{req.ConfigType, key}) {
			return nil, 0, ErrInvalidGeneratedConfigQuery
		}
	}
	if len(req.InstanceKeys) == 0 {
		return []*model.MonitorConfig{}, 0, nil
	}
	keys := append([]string(nil), req.InstanceKeys...)
	query := d.db.WithContext(ctx).Unscoped().Model(&model.MonitorConfig{}).Where("config_type = ?", req.ConfigType)
	if d.db.Dialector.Name() == "mysql" {
		query = query.Where("BINARY instance_ip IN ?", keys)
	} else {
		query = query.Where("instance_ip COLLATE BINARY IN ?", keys)
	}
	return d.readGeneratedConfigPage(ctx, query, req.Page, req.Size)
}

// Pool-scoped rows locate otherwise unindexed tombstones. Callers must query
// the entire exact namespace before using any returned owner as authority.
func (d *monitorConfigDAO) GetGeneratedConfigHistoryCandidates(ctx context.Context, req GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	if req.PoolID <= 0 || !validGeneratedConfigPage(req.ConfigType, req.Page, req.Size) {
		return nil, 0, ErrInvalidGeneratedConfigQuery
	}
	query := d.db.WithContext(ctx).Unscoped().Model(&model.MonitorConfig{}).Where("pool_id = ? AND config_type = ?", req.PoolID, req.ConfigType)
	return d.readGeneratedConfigPage(ctx, query, req.Page, req.Size)
}

// Only an applied retirement inside a fully projected v2 operation proves that
// an old owner no longer needs cleanup. A raw key may now belong to a new pool.
func (d *monitorConfigDAO) GetProjectedRecordRetirements(ctx context.Context, poolID int, owners []GeneratedConfigOwner) ([]GeneratedConfigOwner, error) {
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
		if !validGeneratedOwner(owner) || owner.PoolID != poolID || owner.ConfigType != 4 || byID[owner.ID].ID != 0 {
			return nil, ErrInvalidGeneratedConfigQuery
		}
		byID[owner.ID] = owner
		ids = append(ids, owner.ID)
		keys = append(keys, []byte(owner.InstanceIP))
	}
	var events []model.MonitorConfigArtifactEvent
	err := d.db.WithContext(ctx).Where("config_type = ? AND instance_key IN ? AND config_row_id IN ? AND action IN ? AND delivery_state IN ? AND operation_id IS NOT NULL", 4, keys, ids, []string{model.ConfigProjectionRetire, model.ConfigProjectionRender}, []string{model.ConfigProjectionApplied, model.ConfigProjectionSuperseded}).Find(&events).Error
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
		if original.Family != 4 || original.PoolID != poolID {
			continue
		}
		if event.Action == model.ConfigProjectionRender || original.Phase != model.ConfigPublicationProjected || event.DeliveryState != model.ConfigProjectionApplied {
			resolved, err := d.recordRetirementResolvedByRecovery(d.db.WithContext(ctx), original, event, owner)
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
			if op.Family != model.ConfigPublicationFamilyRecordRule || op.PoolID != poolID || op.Phase != model.ConfigPublicationProjected || op.ProjectedAt == nil {
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
		resolved, err := d.recordLegacyOwnersResolvedByManifests(d.db.WithContext(ctx), poolID, remaining)
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

func (d *monitorConfigDAO) recordRetirementResolvedByRecovery(tx *gorm.DB, old model.MonitorConfigPoolOperation, event model.MonitorConfigArtifactEvent, owner GeneratedConfigOwner) (bool, error) {
	var operations []model.MonitorConfigPoolOperation
	if err := tx.Where("family = ? AND pool_id = ? AND recovery_of_operation_id = ? AND phase IN ? AND manifest IS NOT NULL", 4, old.PoolID, old.OperationID, []string{model.ConfigPublicationCommitted, model.ConfigPublicationProjected}).Order("operation_id ASC").Find(&operations).Error; err != nil {
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
		for _, proof := range envelope.RecordSuperseded {
			if proof.AcceptedOperationID == old.OperationID && proof.AcceptedEventID == event.EventID && proof.AcceptedAction == event.Action && proof.RetiredOwner == owner {
				return true, nil
			}
		}
	}
	return false, nil
}

// Legacy owners have no generated event or accepted operation to use as an
// index. Their first typed control-only operation durably records the evidence.
func (d *monitorConfigDAO) recordLegacyOwnersResolvedByManifests(tx *gorm.DB, poolID int, owners map[int]GeneratedConfigOwner) (map[int]bool, error) {
	var operations []model.MonitorConfigPoolOperation
	if err := tx.Where("family = ? AND pool_id = ? AND phase IN ? AND manifest LIKE ?", 4, poolID, []string{model.ConfigPublicationCommitted, model.ConfigPublicationProjected}, `%"record_superseded"%`).Order("operation_id ASC").Find(&operations).Error; err != nil {
		return nil, err
	}
	result := map[int]bool{}
	for _, operation := range operations {
		envelope, err := decodeProjectionManifest(operation.Manifest)
		if err != nil {
			return nil, err
		}
		relevant := false
		for _, proof := range envelope.RecordSuperseded {
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
		for _, proof := range envelope.RecordSuperseded {
			if proof.Origin == RecordSupersededLegacyOwner && owners[proof.RetiredOwner.ID] == proof.RetiredOwner {
				result[proof.RetiredOwner.ID] = true
			}
		}
	}
	return result, nil
}

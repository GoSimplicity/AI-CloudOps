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
	return d.getProjectedScrapeRuleRetirements(ctx, scrapeRecordPublication, poolID, owners)
}

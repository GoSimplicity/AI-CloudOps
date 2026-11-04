package config

import (
	"context"
	"errors"
	"math"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

const GeneratedConfigQueryLimit = 100

var ErrInvalidGeneratedConfigQuery = errors.New("生成配置查询参数无效")
var ErrGeneratedConfigQuery = errors.New("生成配置元数据读取失败")

// Pool candidates include every undeleted status and name. PoolID is a locator,
// not proof that a row belongs to a generator; ownership remains the caller's job.
type GeneratedConfigPoolQuery struct {
	PoolID     int
	ConfigType int8
	Page       int
	Size       int
}

// Namespace queries intentionally have no PoolID or status filter. Foreign,
// custom and duplicate rows at the exact raw keys must remain visible.
type GeneratedConfigKeysQuery struct {
	ConfigType   int8
	InstanceKeys []string
	Page         int
	Size         int
}

func validGeneratedConfigPage(kind int8, page, size int) bool {
	return kind >= model.ConfigTypePrometheus && kind <= model.ConfigTypeWebhookFile && page > 0 && size > 0 && size <= GeneratedConfigQueryLimit && (page-1) <= math.MaxInt/size
}
func generatedConfigReadError(ctx context.Context) error {
	return errors.Join(ErrGeneratedConfigQuery, ctx.Err())
}

func (d *monitorConfigDAO) GetGeneratedConfigPoolCandidates(ctx context.Context, req GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	if req.PoolID <= 0 || !validGeneratedConfigPage(req.ConfigType, req.Page, req.Size) {
		return nil, 0, ErrInvalidGeneratedConfigQuery
	}
	query := d.db.WithContext(ctx).Model(&model.MonitorConfig{}).Where("pool_id = ? AND config_type = ?", req.PoolID, req.ConfigType)
	return d.readGeneratedConfigPage(ctx, query, req.Page, req.Size)
}

func (d *monitorConfigDAO) GetGeneratedConfigNamespacePage(ctx context.Context, req GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	if !validGeneratedConfigPage(req.ConfigType, req.Page, req.Size) || len(req.InstanceKeys) > GeneratedConfigQueryLimit {
		return nil, 0, ErrInvalidGeneratedConfigQuery
	}
	for _, key := range req.InstanceKeys {
		if key == "" {
			return nil, 0, ErrInvalidGeneratedConfigQuery
		}
	}
	if len(req.InstanceKeys) == 0 {
		return []*model.MonitorConfig{}, 0, nil
	}
	keys := append([]string(nil), req.InstanceKeys...)
	query := d.db.WithContext(ctx).Model(&model.MonitorConfig{}).Where("config_type = ?", req.ConfigType)
	// Match the byte identity used by Redis and immutable SQL owner guards, even
	// when the DB column uses CI/NOCASE or space-padding comparison semantics.
	if d.db.Dialector.Name() == "mysql" {
		query = query.Where("BINARY instance_ip IN ?", keys)
	} else {
		query = query.Where("instance_ip COLLATE BINARY IN ?", keys)
	}
	return d.readGeneratedConfigPage(ctx, query, req.Page, req.Size)
}

// COUNT and page reads are intentionally separate. Consumers must validate all
// pages and observed totals; this is not a transactionally consistent snapshot.
func (d *monitorConfigDAO) readGeneratedConfigPage(ctx context.Context, query *gorm.DB, page, size int) ([]*model.MonitorConfig, int64, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, generatedConfigReadError(ctx)
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*model.MonitorConfig{}, 0, nil
	}
	var rows []*model.MonitorConfig
	if err := query.Order("id ASC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return nil, 0, generatedConfigReadError(ctx)
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

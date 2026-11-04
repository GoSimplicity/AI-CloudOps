package config

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrGeneratedConfigConflict = errors.New("生成配置归属已变化，请重新读取后重试")
	ErrGeneratedConfigWrite    = errors.New("生成配置条件写入失败")
)

// GeneratedConfigOwner is copied at preflight, not reconstructed from desired
// output. In particular an Alertmanager file rename retains its old name here.
type GeneratedConfigOwner struct {
	ID         int
	PoolID     int
	ConfigType int8
	InstanceIP string
	Name       string
}
type GeneratedConfigUpdate struct {
	Expected GeneratedConfigOwner
	Desired  model.MonitorConfig
}

func GeneratedConfigOwnerOf(row *model.MonitorConfig) GeneratedConfigOwner {
	if row == nil {
		return GeneratedConfigOwner{}
	}
	return GeneratedConfigOwner{ID: row.ID, PoolID: row.PoolID, ConfigType: row.ConfigType, InstanceIP: row.InstanceIP, Name: row.Name}
}
func validGeneratedOwner(owner GeneratedConfigOwner) bool {
	return owner.ID > 0 && owner.PoolID > 0 && owner.ConfigType >= 1 && owner.ConfigType <= 5 && owner.Name != "" && owner.InstanceIP != ""
}
func validGeneratedUpdate(change GeneratedConfigUpdate) bool {
	before, after := change.Expected, change.Desired
	return validGeneratedOwner(before) && after.ID == before.ID && after.PoolID == before.PoolID && after.ConfigType == before.ConfigType && after.InstanceIP == before.InstanceIP && after.Name != "" && (after.Status == model.ConfigStatusActive || after.Status == model.ConfigStatusInactive)
}

// MySQL's usual CI collation is weaker than the publisher's byte-exact canonical
// ownership check. Override it explicitly; SQLite's override also handles NOCASE.
func generatedOwnerQuery(tx *gorm.DB, owner GeneratedConfigOwner) *gorm.DB {
	query := tx.Where("id = ? AND deleted_at = 0 AND pool_id = ? AND config_type = ?", owner.ID, owner.PoolID, owner.ConfigType)
	if tx.Dialector.Name() == "mysql" {
		return query.Where("BINARY name = BINARY ? AND BINARY instance_ip = BINARY ?", owner.Name, owner.InstanceIP)
	}
	return query.Where("name COLLATE BINARY = ? AND instance_ip COLLATE BINARY = ?", owner.Name, owner.InstanceIP)
}
func generatedDesiredMatches(current, desired model.MonitorConfig) bool {
	return current.DeletedAt == 0 && current.ID == desired.ID && current.PoolID == desired.PoolID && current.ConfigType == desired.ConfigType && current.InstanceIP == desired.InstanceIP && current.Name == desired.Name && current.ConfigContent == desired.ConfigContent && current.ConfigHash == desired.ConfigHash && current.Status == desired.Status && current.LastGeneratedTime == desired.LastGeneratedTime
}
func generatedWriteError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrConfigNamespaceConflict) || errors.Is(err, ErrInvalidConfigWrite) {
		return errors.Join(ErrGeneratedConfigConflict, ctx.Err())
	}
	if errors.Is(err, ErrConfigWriteUncertain) {
		return errors.Join(ErrGeneratedConfigWrite, ErrConfigWriteUncertain, ctx.Err())
	}
	if errors.Is(err, ErrGeneratedConfigConflict) {
		return errors.Join(ErrGeneratedConfigConflict, ctx.Err())
	}
	if configWriteLockRetry(err) {
		return errors.Join(ErrGeneratedConfigWrite, ErrConfigWriteRetry, ctx.Err())
	}
	if errors.Is(err, context.Canceled) {
		return errors.Join(ErrGeneratedConfigWrite, context.Canceled)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.Join(ErrGeneratedConfigWrite, context.DeadlineExceeded)
	}
	return errors.Join(ErrGeneratedConfigWrite, ctx.Err())
}

// UpdateGeneratedMonitorConfigs rolls back the whole batch on ownership drift.
// It does not alter ordinary configuration CRUD or create-if-absent behavior.
func (d *monitorConfigDAO) UpdateGeneratedMonitorConfigs(ctx context.Context, changes []GeneratedConfigUpdate) error {
	if len(changes) == 0 {
		return nil
	}
	keys := make([]configNamespaceKey, 0, len(changes))
	seen := map[int]bool{}
	for _, change := range changes {
		if !validGeneratedUpdate(change) || seen[change.Expected.ID] {
			return ErrGeneratedConfigConflict
		}
		seen[change.Expected.ID] = true
		keys = append(keys, configNamespaceKey{change.Expected.ConfigType, change.Expected.InstanceIP})
	}
	err := d.withConfigNamespaces(ctx, keys, func(tx *gorm.DB) error {
		for _, change := range changes {
			if !validGeneratedUpdate(change) {
				return ErrGeneratedConfigConflict
			}
			namespace, err := currentConfigNamespace(tx, configNamespaceKey{change.Expected.ConfigType, change.Expected.InstanceIP})
			if err != nil {
				return err
			}
			if len(namespace) != 1 || (GeneratedConfigOwnerOf(namespace[0]) != change.Expected && !generatedDesiredMatches(*namespace[0], change.Desired)) {
				return ErrGeneratedConfigConflict
			}
			before := *namespace[0]
			next := change.Desired
			result := generatedOwnerQuery(tx.Model(&model.MonitorConfig{}), change.Expected).Updates(map[string]any{"name": next.Name, "pool_id": next.PoolID, "config_type": next.ConfigType, "instance_ip": next.InstanceIP, "config_content": next.ConfigContent, "config_hash": next.ConfigHash, "status": next.Status, "last_generated_time": next.LastGeneratedTime})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 1 {
				after, err := currentConfigByID(tx, change.Expected.ID)
				if err != nil {
					return err
				}
				if err := appendConfigTransition(tx, &before, after); err != nil {
					return err
				}
				continue
			}
			if result.RowsAffected != 0 {
				return ErrGeneratedConfigConflict
			}
			// MySQL can report changed rows rather than matched rows. Zero is accepted
			// only for a still-owned, fully applied desired artifact (including a rename).
			var current model.MonitorConfig
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, change.Expected.ID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrGeneratedConfigConflict
				}
				return err
			}
			if !generatedDesiredMatches(current, next) {
				return ErrGeneratedConfigConflict
			}
			if err := appendConfigTransition(tx, &before, &current); err != nil {
				return err
			}
		}
		return nil
	})
	return generatedWriteError(ctx, err)
}

// GetGeneratedMonitorConfigByID includes a tombstone solely for generator retry
// ownership validation. Ordinary configuration reads keep their active-only view.
func (d *monitorConfigDAO) GetGeneratedMonitorConfigByID(ctx context.Context, id int) (*model.MonitorConfig, error) {
	var row model.MonitorConfig
	if err := d.db.WithContext(ctx).Unscoped().First(&row, id).Error; err != nil {
		return nil, generatedWriteError(ctx, err)
	}
	return &row, nil
}

// DeleteGeneratedMonitorConfig gates retirement before destructive Redis cleanup.
// A previous delete is idempotent only for the identical owner tombstone with no
// active replacement at its cache address. This is not a DB/Redis transaction.
func (d *monitorConfigDAO) DeleteGeneratedMonitorConfig(ctx context.Context, expected GeneratedConfigOwner) error {
	if !validGeneratedOwner(expected) {
		return ErrGeneratedConfigConflict
	}
	key := configNamespaceKey{expected.ConfigType, expected.InstanceIP}
	err := d.withConfigNamespaces(ctx, []configNamespaceKey{key}, func(tx *gorm.DB) error {
		rows, err := currentConfigNamespace(tx, key)
		if err != nil {
			return err
		}
		if len(rows) != 0 {
			if len(rows) != 1 || GeneratedConfigOwnerOf(rows[0]) != expected {
				return ErrGeneratedConfigConflict
			}
			result := generatedOwnerQuery(tx, expected).Delete(&model.MonitorConfig{})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrGeneratedConfigConflict
			}
			return appendConfigTransition(tx, rows[0], nil)
		}
		var retired model.MonitorConfig
		if err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).First(&retired, expected.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrGeneratedConfigConflict
			}
			return err
		}
		if retired.DeletedAt == 0 || GeneratedConfigOwnerOf(&retired) != expected {
			return ErrGeneratedConfigConflict
		}
		// The full namespace is still locked and empty, including inactive rows.
		return nil
	})
	return generatedWriteError(ctx, err)
}

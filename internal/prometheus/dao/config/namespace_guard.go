package config

import (
	"context"
	"errors"
	"sort"
	"unicode/utf8"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInvalidConfigWrite      = errors.New("配置写入参数无效")
	ErrConfigNamespaceConflict = errors.New("配置命名空间已被占用或归属发生变化，请重新读取后重试")
	ErrConfigWrite             = errors.New("配置写入失败，请核查后重试")
	ErrConfigWriteRetry        = errors.New("配置写入发生锁竞争，请重新读取后重试")
	ErrConfigWriteUncertain    = errors.New("配置提交结果不确定，请核查后重试")
)

type configNamespaceKey struct {
	kind int8
	raw  string
}

func validConfigKey(raw string) bool {
	return raw != "" && utf8.ValidString(raw) && utf8.RuneCountInString(raw) <= model.MonitorConfigKeyMaxRunes
}
func validConfigNamespace(key configNamespaceKey) bool {
	return key.kind >= 1 && key.kind <= 5 && validConfigKey(key.raw)
}
func configKey(row *model.MonitorConfig) configNamespaceKey {
	return configNamespaceKey{row.ConfigType, row.InstanceIP}
}
func configPatchKey(source *model.MonitorConfig, patch *model.MonitorConfig) configNamespaceKey {
	key := configKey(source)
	if patch.ConfigType != 0 {
		key.kind = patch.ConfigType
	}
	if patch.InstanceIP != "" {
		key.raw = patch.InstanceIP
	}
	return key
}
func validConfigPatch(row *model.MonitorConfig) bool {
	return row != nil && row.ID > 0 && (row.ConfigType == 0 || row.ConfigType >= 1 && row.ConfigType <= 5) && (row.InstanceIP == "" || validConfigKey(row.InstanceIP))
}

func configWriteLockRetry(err error) bool {
	var driverErr *mysqlDriver.MySQLError
	return errors.As(err, &driverErr) && (driverErr.Number == 1213 || driverErr.Number == 1205)
}

func configWriteError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	for _, known := range []error{ErrInvalidConfigWrite, ErrConfigNamespaceConflict, ErrConfigWriteUncertain} {
		if errors.Is(err, known) {
			return errors.Join(known, ctx.Err())
		}
	}
	if configWriteLockRetry(err) {
		return errors.Join(ErrConfigWriteRetry, ctx.Err())
	}
	if errors.Is(err, context.Canceled) {
		return errors.Join(ErrConfigWrite, context.Canceled)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.Join(ErrConfigWrite, context.DeadlineExceeded)
	}
	return errors.Join(ErrConfigWrite, ctx.Err())
}

// All writers acquire permanent lock addresses in the same byte-exact order,
// before any data-row lock. This protects SQL only, not later Redis operations.
func (d *monitorConfigDAO) withConfigNamespaces(ctx context.Context, keys []configNamespaceKey, write func(*gorm.DB) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := orderedConfigNamespaces(keys); err != nil {
		return err
	}
	readyToCommit := false
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockConfigNamespacesTx(tx, keys); err != nil {
			return err
		}
		if err := write(tx); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		readyToCommit = true
		return nil
	})
	if err != nil && readyToCommit {
		return errors.Join(ErrConfigWriteUncertain, ctx.Err())
	}
	return err
}
func currentConfigNamespace(tx *gorm.DB, key configNamespaceKey) ([]*model.MonitorConfig, error) {
	// Preserve an indexable coarse equality before the byte-exact residual.
	// BINARY alone can scan-lock unrelated keys under RR despite their guards.
	query := tx.Model(&model.MonitorConfig{}).Where("config_type = ? AND instance_ip = ?", key.kind, key.raw)
	if tx.Dialector.Name() == "mysql" {
		query = query.Where("BINARY instance_ip = BINARY ?", key.raw)
	} else {
		query = query.Where("instance_ip COLLATE BINARY = ?", key.raw)
	}
	var rows []*model.MonitorConfig
	err := query.Clauses(clause.Locking{Strength: "UPDATE"}).Order("id ASC").Find(&rows).Error
	return rows, err
}
func currentConfigByID(tx *gorm.DB, id int) (*model.MonitorConfig, error) {
	var row model.MonitorConfig
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
func namespaceContains(rows []*model.MonitorConfig, id int) bool {
	for _, row := range rows {
		if row.ID == id {
			return true
		}
	}
	return false
}

func orderedConfigNamespaces(keys []configNamespaceKey) ([]configNamespaceKey, error) {
	unique := map[configNamespaceKey]bool{}
	ordered := make([]configNamespaceKey, 0, len(keys))
	for _, key := range keys {
		if !validConfigNamespace(key) {
			return nil, ErrInvalidConfigWrite
		}
		if !unique[key] {
			unique[key] = true
			ordered = append(ordered, key)
		}
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].kind != ordered[j].kind {
			return ordered[i].kind < ordered[j].kind
		}
		return ordered[i].raw < ordered[j].raw
	})
	return ordered, nil
}
func lockConfigNamespacesTx(tx *gorm.DB, keys []configNamespaceKey) error {
	ordered, err := orderedConfigNamespaces(keys)
	if err != nil {
		return err
	}
	ctx := tx.Statement.Context
	for _, key := range ordered {
		if err := ctx.Err(); err != nil {
			return err
		}
		guard := model.MonitorConfigNamespaceGuard{ConfigType: key.kind, InstanceKey: []byte(key.raw)}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "config_type"}, {Name: "instance_key"}}, DoUpdates: clause.Assignments(map[string]interface{}{"instance_key": gorm.Expr("instance_key")})}).Create(&guard).Error; err != nil {
			return err
		}
		var locked model.MonitorConfigNamespaceGuard
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("config_type = ? AND instance_key = ?", key.kind, []byte(key.raw)).First(&locked).Error; err != nil {
			return err
		}
		if locked.ConfigType != key.kind || string(locked.InstanceKey) != key.raw {
			return ErrConfigNamespaceConflict
		}
	}
	return nil
}

package config

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func generatedFixture(t *testing.T) (*monitorConfigDAO, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), NowFunc: func() time.Time { return time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC) }})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.AutoMigrate(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}); err != nil {
		t.Fatal(err)
	}
	return NewMonitorConfigDAO(zap.NewNop(), db).(*monitorConfigDAO), db
}
func generatedRow(t *testing.T, dao *monitorConfigDAO, name string, ips ...string) *model.MonitorConfig {
	t.Helper()
	row := &model.MonitorConfig{Name: name, PoolID: 7, InstanceIP: "127.0.0.1", ConfigType: model.ConfigTypeWebhookFile, ConfigContent: "before", ConfigHash: "before-hash", Status: 1, LastGeneratedTime: 1}
	if len(ips) > 0 {
		row.InstanceIP = ips[0]
	}
	if err := dao.CreateMonitorConfig(context.Background(), row); err != nil {
		t.Fatal(err)
	}
	return row
}
func TestGeneratedConfigConditionalUpdateContract(t *testing.T) {
	for _, mode := range []string{"normal", "same desired", "rename", "name conflict", "case conflict", "pool conflict", "type conflict", "IP conflict", "deleted conflict"} {
		t.Run(mode, func(t *testing.T) {
			dao, _ := generatedFixture(t)
			row := generatedRow(t, dao, "webhook-before")
			change := GeneratedConfigUpdate{Expected: GeneratedConfigOwnerOf(row), Desired: *row}
			change.Desired.ConfigContent = "after"
			change.Desired.ConfigHash = "after-hash"
			change.Desired.LastGeneratedTime = 2
			switch mode {
			case "rename":
				change.Desired.Name = "webhook-after"
			case "name conflict":
				row.Name = "custom"
			case "case conflict":
				row.Name = "WEBHOOK-BEFORE"
			case "pool conflict":
				row.PoolID = 99
			case "type conflict":
				row.ConfigType = model.ConfigTypeAlertRule
			case "IP conflict":
				row.InstanceIP = "127.0.0.9"
			case "deleted conflict":
				if err := dao.DeleteMonitorConfig(context.Background(), row.ID); err != nil {
					t.Fatal(err)
				}
			}
			conflict := strings.Contains(mode, "conflict")
			if conflict && mode != "deleted conflict" {
				if err := dao.UpdateMonitorConfig(context.Background(), row); err != nil {
					t.Fatal(err)
				}
			}
			err := dao.UpdateGeneratedMonitorConfigs(context.Background(), []GeneratedConfigUpdate{change})
			if conflict {
				if !errors.Is(err, ErrGeneratedConfigConflict) {
					t.Fatal("ownership conflict not classified")
				}
				if mode != "deleted conflict" {
					after, e := dao.GetMonitorConfigByID(context.Background(), row.ID)
					if e != nil || GeneratedConfigOwnerOf(after) != GeneratedConfigOwnerOf(row) || after.ConfigContent != "before" {
						t.Fatal("conflicting row was changed")
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if mode == "same desired" || mode == "rename" {
				if err := dao.UpdateGeneratedMonitorConfigs(context.Background(), []GeneratedConfigUpdate{change}); err != nil {
					t.Fatal("idempotent desired state or completed rename rejected")
				}
			}
			after, e := dao.GetMonitorConfigByID(context.Background(), row.ID)
			if e != nil || !generatedDesiredMatches(*after, change.Desired) {
				t.Fatal("desired artifact was not stored completely")
			}
		})
	}
}
func TestGeneratedConfigBatchConflictRollsBackPriorUpdate(t *testing.T) {
	dao, _ := generatedFixture(t)
	a := generatedRow(t, dao, "before-a")
	b := generatedRow(t, dao, "before-b", "127.0.0.2")
	x := GeneratedConfigUpdate{Expected: GeneratedConfigOwnerOf(a), Desired: *a}
	x.Desired.ConfigContent = "after-a"
	y := GeneratedConfigUpdate{Expected: GeneratedConfigOwnerOf(b), Desired: *b}
	y.Desired.ConfigContent = "after-b"
	b.Name = "operator-custom"
	if err := dao.UpdateMonitorConfig(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(dao.UpdateGeneratedMonitorConfigs(context.Background(), []GeneratedConfigUpdate{x, y}), ErrGeneratedConfigConflict) {
		t.Fatal("batch conflict not returned")
	}
	after, e := dao.GetMonitorConfigByID(context.Background(), a.ID)
	if e != nil || after.ConfigContent != "before" {
		t.Fatal("earlier batch write did not roll back")
	}
}
func TestGeneratedConfigDeleteOwnershipIsAtomic(t *testing.T) {
	for _, mode := range []string{"normal", "case conflict", "name conflict", "pool conflict", "type conflict", "IP conflict"} {
		t.Run(mode, func(t *testing.T) {
			dao, db := generatedFixture(t)
			row := generatedRow(t, dao, "owned-before")
			owner := GeneratedConfigOwnerOf(row)
			switch mode {
			case "case conflict":
				row.Name = "OWNED-BEFORE"
			case "name conflict":
				row.Name = "custom"
			case "pool conflict":
				row.PoolID = 99
			case "type conflict":
				row.ConfigType = 3
			case "IP conflict":
				row.InstanceIP = "127.0.0.9"
			}
			if mode != "normal" {
				if e := dao.UpdateMonitorConfig(context.Background(), row); e != nil {
					t.Fatal(e)
				}
			}
			err := dao.DeleteGeneratedMonitorConfig(context.Background(), owner)
			if mode != "normal" {
				if !errors.Is(err, ErrGeneratedConfigConflict) {
					t.Fatal("delete ownership drift accepted")
				}
				if _, e := dao.GetMonitorConfigByID(context.Background(), row.ID); e != nil {
					t.Fatal("unowned row was removed")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var tombstone model.MonitorConfig
			if e := db.Unscoped().First(&tombstone, row.ID).Error; e != nil || tombstone.DeletedAt == 0 {
				t.Fatal("conditional retire did not soft-delete")
			}
		})
	}
}
func TestGeneratedConfigMySQLGuardUsesBinaryParametersWithoutIO(t *testing.T) {
	db, e := gorm.Open(mysql.New(mysql.Config{DSN: "root@tcp(127.0.0.1:1)/unused", SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true, DryRun: true, SkipDefaultTransaction: true, Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal(e)
	}
	pool, e := db.DB()
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	owner := GeneratedConfigOwner{ID: 1, PoolID: 7, ConfigType: 5, Name: "case-sensitive-name", InstanceIP: "case-sensitive-ip"}
	stmt := generatedOwnerQuery(db.Model(&model.MonitorConfig{}), owner).Updates(map[string]any{"config_content": "after"}).Statement
	sql := stmt.SQL.String()
	if !strings.Contains(sql, "BINARY name = BINARY ?") || !strings.Contains(sql, "BINARY instance_ip = BINARY ?") || strings.Contains(sql, owner.Name) || strings.Contains(sql, owner.InstanceIP) {
		t.Fatal("MySQL guard was not byte-exact and parameterized")
	}
}
func TestGeneratedWriteCanceledAndInvalidOwnership(t *testing.T) {
	dao, _ := generatedFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	row := generatedRow(t, dao, "owned")
	change := GeneratedConfigUpdate{Expected: GeneratedConfigOwnerOf(row), Desired: *row}
	if !errors.Is(dao.UpdateGeneratedMonitorConfigs(ctx, []GeneratedConfigUpdate{change}), context.Canceled) {
		t.Fatal("cancellation was hidden")
	}
	if !errors.Is(dao.DeleteGeneratedMonitorConfig(context.Background(), GeneratedConfigOwner{}), ErrGeneratedConfigConflict) {
		t.Fatal("missing ownership accepted")
	}
}

func TestGeneratedConfigRetirementRetryContract(t *testing.T) {
	for _, mode := range []string{"same tombstone", "changed tombstone", "active replacement", "missing"} {
		t.Run(mode, func(t *testing.T) {
			dao, db := generatedFixture(t)
			row := generatedRow(t, dao, "owned")
			expected := GeneratedConfigOwnerOf(row)
			if err := dao.DeleteGeneratedMonitorConfig(context.Background(), expected); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "changed tombstone":
				if err := db.Unscoped().Model(&model.MonitorConfig{}).Where("id = ?", row.ID).Update("name", "foreign").Error; err != nil {
					t.Fatal(err)
				}
			case "active replacement":
				generatedRow(t, dao, "custom")
			case "missing":
				if err := db.Unscoped().Delete(&model.MonitorConfig{}, row.ID).Error; err != nil {
					t.Fatal(err)
				}
			}
			err := dao.DeleteGeneratedMonitorConfig(context.Background(), expected)
			if mode == "same tombstone" {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, ErrGeneratedConfigConflict) {
				t.Fatal("unsafe retirement retry accepted")
			}
			if mode == "active replacement" {
				var rows []model.MonitorConfig
				if err := db.Find(&rows).Error; err != nil || len(rows) != 1 || rows[0].Name != "custom" {
					t.Fatal("replacement changed")
				}
			}
		})
	}
}

func TestGeneratedConfigSQLiteNoCaseOwnerMustMatchBytes(t *testing.T) {
	dao, db := generatedFixture(t)
	if err := db.Migrator().DropTable(&model.MonitorConfig{}); err != nil {
		t.Fatal(err)
	}
	// Deliberately weaken the actual schema collation, as on existing MySQL CI columns.
	sql := `CREATE TABLE cl_monitor_configs (id integer primary key, created_at datetime, updated_at datetime, deleted_at integer NOT NULL DEFAULT 0, name text COLLATE NOCASE, pool_id integer, instance_ip text COLLATE NOCASE, config_type integer, config_content text, config_hash text, status integer, last_generated_time integer)`
	if err := db.Exec(sql).Error; err != nil {
		t.Fatal(err)
	}
	row := generatedRow(t, dao, "Owned")
	expected := GeneratedConfigOwnerOf(row)
	if err := db.Model(&model.MonitorConfig{}).Where("id = ?", row.ID).Update("name", "OWNED").Error; err != nil {
		t.Fatal(err)
	}
	next := *row
	next.ConfigContent = "after"
	if err := dao.UpdateGeneratedMonitorConfigs(context.Background(), []GeneratedConfigUpdate{{Expected: expected, Desired: next}}); !errors.Is(err, ErrGeneratedConfigConflict) {
		t.Fatal("NOCASE update bypassed byte ownership")
	}
	if err := dao.DeleteGeneratedMonitorConfig(context.Background(), expected); !errors.Is(err, ErrGeneratedConfigConflict) {
		t.Fatal("NOCASE delete bypassed byte ownership")
	}
}

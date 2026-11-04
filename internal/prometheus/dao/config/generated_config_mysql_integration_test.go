//go:build integration

package config

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	driver "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Opt-in only: the caller must provision a NEW EMPTY loopback schema and confirm
// its exact name separately. Never use the application's demo or runtime DSN.
// This fixture creates/drops only its own metadata and namespace-guard tables.
func TestGeneratedOwnerGuardNativeMySQL(t *testing.T) {
	path := os.Getenv("CLOUDOPS_OWNER_GUARD_MYSQL_DSN_FILE")
	if path == "" {
		t.Skip("private native MySQL fixture not configured")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 16384 {
		t.Fatal("private fixture DSN file must be a small owner-only regular file")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("private fixture DSN file unreadable")
	}
	cfg, err := driver.ParseDSN(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal("invalid private fixture DSN")
	}
	if cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:13306" || !regexp.MustCompile(`^cloudops_owner_cas_[a-z0-9_]+$`).MatchString(cfg.DBName) || os.Getenv("CLOUDOPS_OWNER_GUARD_MYSQL_DATABASE") != cfg.DBName {
		t.Fatal("fixture requires literal loopback TCP and separately confirmed private schema")
	}
	for _, foundRows := range []bool{false, true} {
		name := "changed rows"
		if foundRows {
			name = "found rows"
		}
		t.Run(name, func(t *testing.T) {
			private := *cfg
			private.ClientFoundRows = foundRows
			private.ParseTime = true
			db, err := gorm.Open(mysql.Open(private.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), NowFunc: func() time.Time { return time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC) }})
			if err != nil {
				t.Fatal("private fixture connection failed")
			}
			sqlDB, err := db.DB()
			if err != nil {
				t.Fatal("private fixture pool unavailable")
			}
			defer sqlDB.Close()
			var tables int64
			if err := db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ?", cfg.DBName, "cl_monitor_configs").Scan(&tables).Error; err != nil || tables != 0 {
				t.Fatal("fixture table must not already exist")
			}
			if err := db.AutoMigrate(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}, &model.MonitorConfigPoolControl{}, &model.MonitorConfigPoolOperation{}, &model.MonitorConfigArtifactEvent{}); err != nil {
				t.Fatal("fixture migration failed")
			}
			defer func() {
				if err := db.Migrator().DropTable(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}, &model.MonitorConfigPoolControl{}, &model.MonitorConfigPoolOperation{}, &model.MonitorConfigArtifactEvent{}); err != nil {
					t.Error("fixture table cleanup failed")
				}
			}()
			if err := db.Exec("ALTER TABLE cl_monitor_configs MODIFY name VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL, MODIFY instance_ip VARCHAR(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL").Error; err != nil {
				t.Fatal("fixture CI collation setup failed")
			}
			dao := NewMonitorConfigDAO(zap.NewNop(), db).(*monitorConfigDAO)
			ctx := context.Background()
			row := generatedRow(t, dao, "Owned")
			row.InstanceIP = "Host.Example"
			if dao.UpdateMonitorConfig(ctx, row) != nil {
				t.Fatal("native IP fixture setup failed")
			}
			expected := GeneratedConfigOwnerOf(row)
			// Verify this real connection's MySQL affected-row mode, then exercise the
			// DAO's zero-row path rather than assuming SQLite models MySQL semantics.
			control := db.Model(&model.MonitorConfig{}).Where("id = ?", row.ID).UpdateColumn("name", row.Name)
			wanted := int64(0)
			if foundRows {
				wanted = 1
			}
			if control.Error != nil || control.RowsAffected != wanted {
				t.Fatal("native affected-row mode not observed")
			}
			next := *row
			next.Name = "Renamed"
			next.ConfigContent = "after"
			next.ConfigHash = "after-hash"
			next.LastGeneratedTime = 2
			changes := []GeneratedConfigUpdate{{Expected: expected, Desired: next}}
			if dao.UpdateGeneratedMonitorConfigs(ctx, changes) != nil || dao.UpdateGeneratedMonitorConfigs(ctx, changes) != nil {
				t.Fatal("native update/rename retry failed")
			}
			renamed := GeneratedConfigOwnerOf(&next)
			if dao.UpdateGeneratedMonitorConfigs(ctx, []GeneratedConfigUpdate{{Expected: renamed, Desired: next}}) != nil {
				t.Fatal("native same-owner no-op failed")
			}

			for _, drift := range []struct{ column, value string }{{"name", "RENAMED"}, {"name", "Renamed "}, {"instance_ip", "HOST.EXAMPLE"}, {"instance_ip", "Host.Example "}} {
				if err := db.Model(&model.MonitorConfig{}).Where("id = ?", row.ID).UpdateColumn(drift.column, drift.value).Error; err != nil {
					t.Fatal("native owner drift setup failed")
				}
				if !errors.Is(dao.UpdateGeneratedMonitorConfigs(ctx, []GeneratedConfigUpdate{{Expected: renamed, Desired: next}}), ErrGeneratedConfigConflict) || !errors.Is(dao.DeleteGeneratedMonitorConfig(ctx, renamed), ErrGeneratedConfigConflict) {
					t.Fatal("native CI/padded column bypassed byte-exact owner guard")
				}
				if err := db.Model(&model.MonitorConfig{}).Where("id = ?", row.ID).Updates(map[string]any{"name": next.Name, "instance_ip": next.InstanceIP}).Error; err != nil {
					t.Fatal("native owner restore failed")
				}
			}

			if dao.DeleteGeneratedMonitorConfig(ctx, renamed) != nil || dao.DeleteGeneratedMonitorConfig(ctx, renamed) != nil {
				t.Fatal("native tombstone retry failed")
			}
			replacement := generatedRow(t, dao, "Replacement")
			replacement.InstanceIP = next.InstanceIP
			if dao.UpdateMonitorConfig(ctx, replacement) != nil {
				t.Fatal("native replacement setup failed")
			}
			if !errors.Is(dao.DeleteGeneratedMonitorConfig(ctx, renamed), ErrGeneratedConfigConflict) {
				t.Fatal("native tombstone retry ignored active replacement")
			}
		})
	}
}

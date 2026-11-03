//go:build integration

package alert_test

import (
	"context"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestAMPoolLifecycleNativeMySQL(t *testing.T) {
	path := os.Getenv("CLOUDOPS_AM_D1_MYSQL_DSN_FILE")
	if path == "" {
		t.Skip("private native MySQL fixture not configured")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 16384 {
		t.Fatal("fixture requires a small owner-only regular DSN file")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("fixture DSN unreadable")
	}
	cfg, err := driver.ParseDSN(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal("fixture DSN invalid")
	}
	if cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:13306" || !regexp.MustCompile(`^cloudops_am_d1_[a-z0-9_]+$`).MatchString(cfg.DBName) || cfg.DBName != os.Getenv("CLOUDOPS_AM_D1_MYSQL_DATABASE") {
		t.Fatal("fixture requires loopback and separately confirmed empty private schema")
	}
	cfg.ParseTime = true
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Discard, DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal("fixture connection failed")
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal("fixture pool unavailable")
	}
	defer conn.Close()
	conn.SetMaxOpenConns(6)
	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ?", cfg.DBName).Scan(&count).Error; err != nil || count != 0 {
		t.Fatal("fixture schema must be empty")
	}
	if err := db.Migrator().CreateTable(&model.MonitorAlertManagerPool{}, &model.MonitorSendGroup{}); err != nil {
		t.Fatal("fixture table creation failed")
	}
	links := []string{"cl_monitor_send_group_static_receive_users", "cl_monitor_send_group_first_upgrade_users", "cl_monitor_send_group_second_upgrade_users"}
	defer func() {
		for _, table := range links {
			if err := db.Migrator().DropTable(table); err != nil {
				t.Error("fixture link cleanup failed")
			}
		}
		if err := db.Migrator().DropTable(&model.MonitorSendGroup{}, &model.MonitorAlertManagerPool{}); err != nil {
			t.Error("fixture parent/child cleanup failed")
		}
	}()
	// Only the three existing association tables are needed for empty-list
	// replacements; no system-user or unrelated application tables are created.
	for _, table := range links {
		if err := db.Exec("CREATE TABLE " + table + " (monitor_send_group_id BIGINT NOT NULL, user_id BIGINT NOT NULL, PRIMARY KEY (monitor_send_group_id,user_id))").Error; err != nil {
			t.Fatal("fixture link creation failed")
		}
	}
	for _, operation := range []string{"create", "migrate"} {
		for _, winner := range []string{"write", "delete"} {
			t.Run(operation+"/"+winner+"-first", func(t *testing.T) {
				p, g := amDAOs(db)
				ctx := context.Background()
				target := &model.MonitorAlertManagerPool{Name: operation + winner}
				if err := db.Create(target).Error; err != nil {
					t.Fatal(err)
				}
				var row *model.MonitorSendGroup
				if operation == "migrate" {
					source := &model.MonitorAlertManagerPool{Name: "source" + winner}
					if err := db.Create(source).Error; err != nil {
						t.Fatal(err)
					}
					row = &model.MonitorSendGroup{Name: "migrating", PoolID: source.ID, Enable: 2}
					if err := g.CreateMonitorSendGroup(ctx, row); err != nil {
						t.Fatal(err)
					}
				}
				held, release := make(chan struct{}), make(chan struct{})
				var mu sync.Mutex
				armed := true
				callback := "fixture:am-parent-held"
				if err := db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
					if tx.Statement.Schema == nil || tx.Statement.Schema.Table != "cl_monitor_alert_manager_pools" || tx.Error != nil {
						return
					}
					mu.Lock()
					hold := armed
					if hold {
						armed = false
					}
					mu.Unlock()
					if hold {
						close(held)
						<-release
					}
				}); err != nil {
					t.Fatal(err)
				}
				defer db.Callback().Query().Remove(callback)
				write := func() error {
					if operation == "create" {
						return g.CreateMonitorSendGroup(ctx, &model.MonitorSendGroup{Name: "created", PoolID: target.ID, Enable: 2})
					}
					copy := *row
					copy.PoolID = target.ID
					return g.UpdateMonitorSendGroup(ctx, &copy)
				}
				remove := func() error { return p.DeleteMonitorAlertManagerPool(ctx, target.ID) }
				first, second := write, remove
				if winner == "delete" {
					first, second = remove, write
				}
				firstDone, secondDone := make(chan error, 1), make(chan error, 1)
				go func() { firstDone <- first() }()
				select {
				case <-held:
				case <-time.After(10 * time.Second):
					close(release)
					t.Fatal("parent lock checkpoint timed out")
				}
				go func() { secondDone <- second() }()
				close(release)
				if err := <-firstDone; err != nil {
					t.Fatalf("winning operation failed: %v", err)
				}
				if err := <-secondDone; err == nil {
					t.Fatal("competing operation bypassed parent lock/reference guard")
				}
			})
		}
	}
}

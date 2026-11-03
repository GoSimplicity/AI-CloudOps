//go:build integration

package scrape_test

import (
	"context"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	scrape "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	driver "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// This fixture accepts only a separately provisioned, EMPTY local schema. It
// never reads application configuration or connects to the demonstration DB.
func TestPoolLifecycleNativeMySQL(t *testing.T) {
	path := os.Getenv("CLOUDOPS_POOL_D1_MYSQL_DSN_FILE")
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
	if cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:13306" || !regexp.MustCompile(`^cloudops_pool_d1_[a-z0-9_]+$`).MatchString(cfg.DBName) || cfg.DBName != os.Getenv("CLOUDOPS_POOL_D1_MYSQL_DATABASE") {
		t.Fatal("fixture requires loopback and separately confirmed private schema")
	}
	cfg.ParseTime = true
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Discard})
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
	tables := []interface{}{&model.MonitorScrapePool{}, &model.MonitorScrapeJob{}, &model.MonitorAlertRule{}, &model.MonitorRecordRule{}}
	if err := db.AutoMigrate(tables...); err != nil {
		t.Fatal("fixture migration failed")
	}
	defer func() {
		if err := db.Migrator().DropTable(tables...); err != nil {
			t.Error("fixture cleanup failed")
		}
	}()
	for _, def := range definitions() {
		for _, winner := range []string{"create", "delete"} {
			t.Run(def.name+"/"+winner+"-first", func(t *testing.T) {
				pool := &model.MonitorScrapePool{Name: def.name + winner}
				if err := db.Create(pool).Error; err != nil {
					t.Fatal(err)
				}
				// Pause the winning transaction immediately after its real SELECT FOR
				// UPDATE. The competing transaction must contend on that same parent.
				held := make(chan struct{})
				release := make(chan struct{})
				var once sync.Once
				var mu sync.Mutex
				armed := true
				callback := "fixture:parent-held"
				if err := db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
					if tx.Statement.Schema == nil || tx.Statement.Schema.Table != "cl_monitor_scrape_pools" || tx.Error != nil {
						return
					}
					mu.Lock()
					hold := armed
					if hold {
						armed = false
					}
					mu.Unlock()
					if hold {
						once.Do(func() { close(held) })
						<-release
					}
				}); err != nil {
					t.Fatal(err)
				}
				defer db.Callback().Query().Remove(callback)
				create := func() error { return def.create(db, pool.ID, 2) }
				remove := func() error {
					return scrape.NewScrapePoolDAO(db, zap.NewNop(), nil).DeleteMonitorScrapePool(context.Background(), pool.ID)
				}
				first, second := create, remove
				if winner == "delete" {
					first, second = remove, create
				}
				firstDone := make(chan error, 1)
				secondDone := make(chan error, 1)
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
					t.Fatal("winning operation failed")
				}
				if err := <-secondDone; err == nil {
					t.Fatal("competing operation violated parent guard")
				}
			})
		}
	}
}

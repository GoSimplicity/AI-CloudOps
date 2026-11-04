//go:build integration

package config

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	driver "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func nativeNamespaceWriteRow(key string) *model.MonitorConfig {
	return &model.MonitorConfig{PoolID: 7, ConfigType: 4, InstanceIP: key, Name: "native fixture", ConfigContent: "fixture body", ConfigHash: "fixture hash", Status: 1, LastGeneratedTime: 1}
}

// Root must provision this new, initially empty schema/user. Never reuse an
// application/demo or prior-stage DSN. Both handles open independent SQL pools.
func TestNamespaceWritesNativeMySQL(t *testing.T) {
	path := os.Getenv("CLOUDOPS_NAMESPACE_WRITE_MYSQL_DSN_FILE")
	if path == "" {
		t.Skip("private namespace-write fixture not configured")
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode().Perm()&0077 != 0 || before.Size() > 16384 {
		t.Fatal("private fixture DSN must be an owner-only regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal("private DSN unavailable")
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) {
		file.Close()
		t.Fatal("private DSN changed")
	}
	raw, err := io.ReadAll(io.LimitReader(file, 16385))
	file.Close()
	if err != nil || len(raw) > 16384 {
		t.Fatal("private DSN unreadable")
	}
	cfg, err := driver.ParseDSN(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal("private DSN invalid")
	}
	if cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:13306" || !regexp.MustCompile(`^cloudops_namespace_write_[a-z0-9_]+$`).MatchString(cfg.DBName) || cfg.DBName != os.Getenv("CLOUDOPS_NAMESPACE_WRITE_MYSQL_DATABASE") {
		t.Fatal("fixture requires loopback and separately confirmed new private schema")
	}
	cfg.ParseTime = true
	open := func() *gorm.DB {
		t.Helper()
		db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Discard})
		if err != nil {
			t.Fatal("private fixture connection failed")
		}
		pool, err := db.DB()
		if err != nil {
			t.Fatal("private fixture pool unavailable")
		}
		pool.SetMaxOpenConns(1)
		t.Cleanup(func() { pool.Close() })
		return db
	}
	a, b := open(), open()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	var idA, idB int64
	if a.WithContext(ctx).Raw("SELECT CONNECTION_ID()").Scan(&idA).Error != nil || b.WithContext(ctx).Raw("SELECT CONNECTION_ID()").Scan(&idB).Error != nil || idA == idB {
		t.Fatal("fixture must use distinct real MySQL sessions")
	}
	var objects int64
	if a.WithContext(ctx).Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()").Scan(&objects).Error != nil || objects != 0 {
		t.Fatal("private schema must initially be empty")
	}
	if err := a.AutoMigrate(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}, &model.MonitorConfigPoolControl{}, &model.MonitorConfigPoolOperation{}, &model.MonitorConfigArtifactEvent{}); err != nil {
		t.Fatal("private schema migration failed")
	}
	defer func() {
		if err := a.Migrator().DropTable(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}, &model.MonitorConfigPoolControl{}, &model.MonitorConfigPoolOperation{}, &model.MonitorConfigArtifactEvent{}); err != nil {
			t.Error("private fixture cleanup failed")
		}
	}()
	da, db := NewMonitorConfigDAO(zap.NewNop(), a).(*monitorConfigDAO), NewMonitorConfigDAO(zap.NewNop(), b).(*monitorConfigDAO)

	t.Run("current locking read rejects foreign committed after RR snapshot", func(t *testing.T) {
		if a.Exec("SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ").Error != nil {
			t.Fatal("RR fixture unavailable")
		}
		held, resume := make(chan struct{}), make(chan struct{})
		var once sync.Once
		var snapshotCount int64
		var snapshotErr error
		name := "fixture:old-view"
		if err := a.Callback().Create().Before("gorm:create").Register(name, func(tx *gorm.DB) {
			if tx.Statement.Schema == nil || tx.Statement.Schema.Table != "cl_monitor_config_namespaces" {
				return
			}
			once.Do(func() {
				snapshotErr = tx.Session(&gorm.Session{NewDB: true}).Raw("SELECT COUNT(*) FROM cl_monitor_configs WHERE config_type=4 AND instance_ip='192.0.2.1' AND deleted_at=0").Scan(&snapshotCount).Error
				close(held)
				<-resume
			})
		}); err != nil {
			t.Fatal(err)
		}
		defer a.Callback().Create().Remove(name)
		defer func() {
			select {
			case <-resume:
			default:
				close(resume)
			}
		}()
		own := nativeNamespaceWriteRow("192.0.2.1")
		done := make(chan error, 1)
		go func() { done <- da.CreateMonitorConfig(ctx, own) }()
		select {
		case <-held:
		case <-ctx.Done():
			t.Fatal("snapshot barrier timed out")
		}
		if snapshotErr != nil || snapshotCount != 0 {
			t.Fatal("RR fixture did not establish empty consistent snapshot")
		}
		foreign := nativeNamespaceWriteRow("192.0.2.1")
		foreign.PoolID = 99
		foreign.Status = 2
		if err := db.CreateMonitorConfig(ctx, foreign); err != nil {
			t.Fatal("foreign fixture could not commit before guard")
		}
		close(resume)
		if err := <-done; !errors.Is(err, ErrConfigNamespaceConflict) || own.ID != 0 {
			t.Fatal("stale consistent view admitted a duplicate")
		}
	})

	t.Run("real lock wait fails safely then retry observes owner", func(t *testing.T) {
		held, resume := make(chan struct{}), make(chan struct{})
		var once sync.Once
		name := "fixture:hold-guard"
		if err := a.Callback().Create().After("gorm:create").Register(name, func(tx *gorm.DB) {
			if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "cl_monitor_config_namespaces" {
				once.Do(func() { close(held); <-resume })
			}
		}); err != nil {
			t.Fatal(err)
		}
		defer a.Callback().Create().Remove(name)
		defer func() {
			select {
			case <-resume:
			default:
				close(resume)
			}
		}()
		first := nativeNamespaceWriteRow("192.0.2.2")
		firstDone := make(chan error, 1)
		go func() { firstDone <- da.CreateMonitorConfig(ctx, first) }()
		select {
		case <-held:
		case <-ctx.Done():
			t.Fatal("lock barrier timed out")
		}
		if err := b.Exec("SET SESSION innodb_lock_wait_timeout=1").Error; err != nil {
			t.Fatal("lock timeout fixture unavailable")
		}
		actualTimeout := false
		capture := "fixture:capture-lock-timeout"
		if err := b.Callback().Create().After("gorm:create").Register(capture, func(tx *gorm.DB) {
			var sqlErr *driver.MySQLError
			if errors.As(tx.Error, &sqlErr) && sqlErr.Number == 1205 {
				actualTimeout = true
			}
		}); err != nil {
			t.Fatal(err)
		}
		second := nativeNamespaceWriteRow("192.0.2.2")
		err := db.CreateMonitorConfig(ctx, second)
		b.Callback().Create().Remove(capture)
		if !errors.Is(err, ErrConfigWriteRetry) || !actualTimeout || second.ID != 0 {
			t.Fatal("fixture did not exercise actual MySQL row-lock wait failure safely")
		}
		close(resume)
		if err := <-firstDone; err != nil {
			t.Fatal("first owner failed after releasing guard")
		}
		if err := db.CreateMonitorConfig(ctx, second); !errors.Is(err, ErrConfigNamespaceConflict) {
			t.Fatal("retry ignored committed namespace owner")
		}
	})

	t.Run("cancellation while waiting keeps input untouched", func(t *testing.T) {
		held, resume := make(chan struct{}), make(chan struct{})
		var once sync.Once
		name := "fixture:cancel-held-guard"
		if err := a.Callback().Create().After("gorm:create").Register(name, func(tx *gorm.DB) {
			if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "cl_monitor_config_namespaces" {
				once.Do(func() { close(held); <-resume })
			}
		}); err != nil {
			t.Fatal(err)
		}
		defer a.Callback().Create().Remove(name)
		defer func() {
			select {
			case <-resume:
			default:
				close(resume)
			}
		}()
		done := make(chan error, 1)
		go func() { done <- da.CreateMonitorConfig(ctx, nativeNamespaceWriteRow("192.0.2.3")) }()
		select {
		case <-held:
		case <-ctx.Done():
			t.Fatal("cancellation barrier timed out")
		}
		waiting, cancelWaiting := context.WithCancel(ctx)
		attempt := make(chan struct{}, 1)
		capture := "fixture:cancel-attempt"
		if err := b.Callback().Create().Before("gorm:create").Register(capture, func(tx *gorm.DB) {
			if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "cl_monitor_config_namespaces" {
				select {
				case attempt <- struct{}{}:
				default:
				}
			}
		}); err != nil {
			t.Fatal(err)
		}
		row := nativeNamespaceWriteRow("192.0.2.3")
		cancelled := make(chan error, 1)
		go func() { cancelled <- db.CreateMonitorConfig(waiting, row) }()
		select {
		case <-attempt:
		case <-ctx.Done():
			t.Fatal("waiting writer never attempted guard")
		}
		cancelWaiting()
		if err := <-cancelled; !errors.Is(err, context.Canceled) || row.ID != 0 || !row.CreatedAt.IsZero() {
			t.Fatal("cancelled guarded write lost cancellation or leaked success fields")
		}
		b.Callback().Create().Remove(capture)
		close(resume)
		if err := <-done; err != nil {
			t.Fatal("holder failed")
		}
	})

	t.Run("ordinary source drift never appends an unordered key lock", func(t *testing.T) {
		row := nativeNamespaceWriteRow("192.0.2.4")
		if err := da.CreateMonitorConfig(ctx, row); err != nil {
			t.Fatal(err)
		}
		held, resume := make(chan struct{}), make(chan struct{})
		var once sync.Once
		var locked []string
		name := "fixture:source-locator"
		if err := a.Callback().Create().Before("gorm:create").Register(name, func(tx *gorm.DB) {
			if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "cl_monitor_config_namespaces" {
				if guard, ok := tx.Statement.Dest.(*model.MonitorConfigNamespaceGuard); ok {
					locked = append(locked, string(guard.InstanceKey))
				}
				once.Do(func() { close(held); <-resume })
			}
		}); err != nil {
			t.Fatal(err)
		}
		defer a.Callback().Create().Remove(name)
		defer func() {
			select {
			case <-resume:
			default:
				close(resume)
			}
		}()
		patch := &model.MonitorConfig{Model: model.Model{ID: row.ID}, Name: "late ordinary edit"}
		done := make(chan error, 1)
		go func() { done <- da.UpdateMonitorConfig(ctx, patch) }()
		select {
		case <-held:
		case <-ctx.Done():
			t.Fatal("source barrier timed out")
		}
		moved := *row
		moved.InstanceIP = "192.0.2.5"
		if err := db.UpdateMonitorConfig(ctx, &moved); err != nil {
			t.Fatal("source migration could not commit")
		}
		close(resume)
		if err := <-done; !errors.Is(err, ErrConfigNamespaceConflict) {
			t.Fatal("stale ordinary source locator was accepted")
		}
		if len(locked) != 1 || locked[0] != "192.0.2.4" {
			t.Fatal("writer appended the new key after holding an obsolete guard")
		}
	})

	t.Run("101st real SQL PK failure rolls back all rows and input IDs", func(t *testing.T) {
		occupied := nativeNamespaceWriteRow("192.0.2.100")
		occupied.ID = 900000
		if err := da.CreateMonitorConfig(ctx, occupied); err != nil {
			t.Fatal(err)
		}
		rows := make([]*model.MonitorConfig, 101)
		for i := range rows {
			rows[i] = nativeNamespaceWriteRow(fmt.Sprintf("batch-key-%03d", i))
			rows[i].Name = "batch"
		}
		rows[100].ID = occupied.ID
		if err := da.BatchCreateMonitorConfigs(ctx, rows); err == nil {
			t.Fatal("real late primary-key failure was accepted")
		}
		for i, row := range rows {
			want := 0
			if i == 100 {
				want = occupied.ID
			}
			if row.ID != want || !row.CreatedAt.IsZero() || !row.UpdatedAt.IsZero() {
				t.Fatal("rolled-back IDs/timestamps leaked to caller")
			}
		}
		var count int64
		if err := a.Model(&model.MonitorConfig{}).Where("name = ?", "batch").Count(&count).Error; err != nil || count != 0 {
			t.Fatal("earlier batch chunk was committed")
		}
	})

	t.Run("legacy duplicate inactive peer blocks generated update and first retire", func(t *testing.T) {
		own := nativeNamespaceWriteRow("192.0.2.6")
		if err := da.CreateMonitorConfig(ctx, own); err != nil {
			t.Fatal(err)
		}
		legacy := *own
		legacy.Model = model.Model{}
		legacy.Name = "legacy foreign"
		legacy.PoolID = 99
		legacy.Status = 2
		// Explicit legacy corruption seed bypasses normal admission; do not relax DAO.
		if err := a.Create(&legacy).Error; err != nil {
			t.Fatal("legacy duplicate fixture failed")
		}
		desired := *own
		desired.ConfigContent = "must not apply"
		if err := da.UpdateGeneratedMonitorConfigs(ctx, []GeneratedConfigUpdate{{Expected: GeneratedConfigOwnerOf(own), Desired: desired}}); !errors.Is(err, ErrGeneratedConfigConflict) {
			t.Fatal("generated update ignored inactive namespace peer")
		}
		if err := da.DeleteGeneratedMonitorConfig(ctx, GeneratedConfigOwnerOf(own)); !errors.Is(err, ErrGeneratedConfigConflict) {
			t.Fatal("first retirement ignored inactive namespace peer")
		}
		if err := da.DeleteMonitorConfig(ctx, legacy.ID); err != nil {
			t.Fatal("explicit legacy duplicate repair was blocked")
		}
		if err := da.DeleteGeneratedMonitorConfig(ctx, GeneratedConfigOwnerOf(own)); err != nil {
			t.Fatal("repaired namespace could not retire")
		}
	})
	t.Run("unrelated namespace updates do not scan-lock each other", func(t *testing.T) {
		for _, handle := range []*gorm.DB{a, b} {
			if handle.Exec("SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ").Error != nil || handle.Exec("SET SESSION innodb_lock_wait_timeout=1").Error != nil {
				t.Fatal("independent update session settings failed")
			}
		}

		left, right := nativeNamespaceWriteRow("203.0.113.101"), nativeNamespaceWriteRow("203.0.113.102")
		if err := da.CreateMonitorConfig(ctx, left); err != nil {
			t.Fatal(err)
		}
		if err := db.CreateMonitorConfig(ctx, right); err != nil {
			t.Fatal(err)
		}
		both := make(chan struct{})
		var started atomic.Int32
		var onceA, onceB sync.Once
		var mu sync.Mutex
		var engineCodes []uint16
		var querySQL string
		var queryArgs []interface{}
		callback := func(expected int, once *sync.Once) func(*gorm.DB) {
			return func(tx *gorm.DB) {
				if tx.Statement.Schema == nil || tx.Statement.Schema.Table != "cl_monitor_configs" || !strings.Contains(tx.Statement.SQL.String(), "FOR UPDATE") {
					return
				}
				var sqlErr *driver.MySQLError
				if errors.As(tx.Error, &sqlErr) {
					mu.Lock()
					engineCodes = append(engineCodes, sqlErr.Number)
					mu.Unlock()
				}
				if row, ok := tx.Statement.Dest.(*model.MonitorConfig); ok && row.ID == expected && tx.Error == nil {
					once.Do(func() {
						if started.Add(1) == 2 {
							close(both)
						}
						select {
						case <-both:
						case <-ctx.Done():
							tx.AddError(ctx.Err())
						}
					})
				} else if _, ok := tx.Statement.Dest.(*[]*model.MonitorConfig); ok {
					mu.Lock()
					querySQL = tx.Statement.SQL.String()
					queryArgs = append([]interface{}(nil), tx.Statement.Vars...)
					mu.Unlock()
				}
			}
		}
		name := "fixture:independent-row-locks"
		if err := a.Callback().Query().After("gorm:query").Register(name, callback(left.ID, &onceA)); err != nil {
			t.Fatal(err)
		}
		if err := b.Callback().Query().After("gorm:query").Register(name, callback(right.ID, &onceB)); err != nil {
			t.Fatal(err)
		}
		defer a.Callback().Query().Remove(name)
		defer b.Callback().Query().Remove(name)
		first, second := make(chan error, 1), make(chan error, 1)
		go func() {
			first <- da.UpdateMonitorConfig(ctx, &model.MonitorConfig{Model: model.Model{ID: left.ID}, Name: "independent A"})
		}()
		go func() {
			second <- db.UpdateMonitorConfig(ctx, &model.MonitorConfig{Model: model.Model{ID: right.ID}, Name: "independent B"})
		}()
		e1, e2 := <-first, <-second
		if querySQL != "" {
			var plans []struct {
				Key   sql.NullString
				Type  string
				Rows  int64
				Extra sql.NullString
			}
			if err := a.WithContext(ctx).Raw("EXPLAIN "+querySQL, queryArgs...).Scan(&plans).Error; err != nil {
				t.Fatal("actual namespace SQL EXPLAIN failed")
			}
			for _, plan := range plans {
				t.Logf("namespace current-read plan: key=%s type=%s estimated_rows=%d", plan.Key.String, plan.Type, plan.Rows)
			}
		}
		if e1 != nil || e2 != nil {
			t.Fatalf("independent-key updates failed: first=%v second=%v engine_codes=%v", e1, e2, engineCodes)
		}
	})

}

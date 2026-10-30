//go:build integration

package dbschema_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

func TestClockInitializationMySQL(t *testing.T) {
	// fixture creates and verifies its own randomly labelled MySQL 8.4.11
	// container; it accepts no external DSN and removes its volume on cleanup.
	db, address := fixture(t)
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := dbschema.InitializeFreshClock(ctx, db, "UTC"); !errors.Is(err, schema.ErrPreflight) {
		t.Fatal("missing manifest history was accepted", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()").Scan(&count); err != nil || count != 0 {
		t.Fatal("clock initialization created schema")
	}
	manifest, err := dbschema.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Migrate(ctx, db, manifest); err != nil {
		t.Fatal(err)
	}
	if err := dbschema.VerifyClock(ctx, db, "UTC"); !errors.Is(err, dbschema.ErrClockMissing) {
		t.Fatal("missing marker not rejected", err)
	}

	t.Run("SELECT-only role cannot initialize", func(t *testing.T) {
		if _, err := db.ExecContext(ctx, "CREATE USER 'clock_reader'@'%' IDENTIFIED BY 'fixture-only'"); err != nil {
			t.Fatal("cannot create owned fixture role")
		}
		if _, err := db.ExecContext(ctx, "GRANT SELECT ON baseline_source.* TO 'clock_reader'@'%'"); err != nil {
			t.Fatal("cannot grant owned fixture role")
		}
		reader, err := sql.Open("mysql", "clock_reader:fixture-only@tcp("+address+")/baseline_source?parseTime=true&timeout=3s&readTimeout=10s&writeTimeout=10s")
		if err != nil {
			t.Fatal("cannot open owned read-only fixture")
		}
		defer reader.Close()
		err = dbschema.InitializeFreshClock(ctx, reader, "UTC")
		if !errors.Is(err, dbschema.ErrClockDatabase) {
			t.Fatal("read-only role unexpectedly initialized marker", err)
		}
		if strings.Contains(err.Error(), "fixture-only") || strings.Contains(err.Error(), "clock_reader") {
			t.Fatal("database details leaked")
		}
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cl_workorder_notification_clock").Scan(&count); err != nil || count != 0 {
			t.Fatal("read-only initialization changed marker")
		}
		if err := dbschema.InitializeFreshClock(ctx, db, "UTC"); err != nil {
			t.Fatal(err)
		}
		if err := dbschema.VerifyClock(ctx, reader, "UTC"); err != nil {
			t.Fatal("read-only verification failed", err)
		}
	})
	t.Run("exact marker and idempotence", func(t *testing.T) {
		if err := dbschema.InitializeFreshClock(ctx, db, "UTC"); err != nil {
			t.Fatal(err)
		}
		if err := dbschema.VerifyClock(ctx, db, "Etc/UTC"); !errors.Is(err, dbschema.ErrClockMismatch) {
			t.Fatal("zone alias silently accepted", err)
		}
		if err := dbschema.InitializeFreshClock(ctx, db, "Asia/Shanghai"); !errors.Is(err, dbschema.ErrClockMismatch) {
			t.Fatal("different zone overwrote marker", err)
		}
		if err := dbschema.VerifyClock(ctx, db, "UTC"); err != nil {
			t.Fatal(err)
		}
	})

	db.SetMaxOpenConns(6)
	for _, test := range []struct {
		name      string
		zones     [2]string
		different bool
	}{
		{"same zone concurrent", [2]string{"UTC", "UTC"}, false},
		{"different zones concurrent", [2]string{"UTC", "Asia/Shanghai"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := db.ExecContext(ctx, "DELETE FROM cl_workorder_notification_clock"); err != nil {
				t.Fatal(err)
			}
			start := make(chan struct{})
			results := make([]error, 2)
			var wg sync.WaitGroup
			for i, zone := range test.zones {
				wg.Add(1)
				go func() { defer wg.Done(); <-start; results[i] = dbschema.InitializeFreshClock(ctx, db, zone) }()
			}
			close(start)
			wg.Wait()
			successes := 0
			for _, err := range results {
				if err == nil {
					successes++
					continue
				}
				if !errors.Is(err, schema.ErrLocked) && !errors.Is(err, dbschema.ErrClockMismatch) {
					t.Fatal("unexpected concurrent initialization failure", err)
				}
				if !test.different && errors.Is(err, dbschema.ErrClockMismatch) {
					t.Fatal("identical zones conflicted")
				}
			}
			if successes < 1 || (test.different && successes != 1) {
				t.Fatalf("unsafe success count %d", successes)
			}
			var stored string
			if err := db.QueryRowContext(ctx, "SELECT time_zone FROM cl_workorder_notification_clock WHERE id=1").Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cl_workorder_notification_clock").Scan(&count); err != nil || count != 1 {
				t.Fatal("concurrent initialization created extra markers")
			}
			// GET_LOCK is fail-fast. A caller may explicitly retry a busy operation once
			// the owner finishes; identical zones must then be a read-only no-op.
			if err := dbschema.InitializeFreshClock(ctx, db, stored); err != nil {
				t.Fatal("explicit same-zone retry was not idempotent", err)
			}
			other := "UTC"
			if stored == "UTC" {
				other = "Asia/Shanghai"
			}
			if err := dbschema.InitializeFreshClock(ctx, db, other); !errors.Is(err, dbschema.ErrClockMismatch) {
				t.Fatal("explicit conflicting retry accepted", err)
			}
		})
	}

	for _, test := range []struct{ name, seed, cleanup, preserved string }{
		{"event history", "INSERT INTO cl_workorder_notification_event (instance_id,event_type,snapshot,sender_id,status,next_attempt_at) VALUES (1,'fixture','{}',0,'pending','2020-01-02 03:04:05')", "DELETE FROM cl_workorder_notification_event", "SELECT COUNT(*) FROM cl_workorder_notification_event WHERE next_attempt_at='2020-01-02 03:04:05'"},
		{"scheduled configuration", "INSERT INTO cl_workorder_notification (name,event_types,channels,recipient_types,message_template,operator_id,scheduled_time) VALUES ('clock-fixture','[]','[]','[]','fixture',0,'2020-01-02 03:04:05')", "DELETE FROM cl_workorder_notification", "SELECT COUNT(*) FROM cl_workorder_notification WHERE scheduled_time='2020-01-02 03:04:05'"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := db.ExecContext(ctx, "DELETE FROM cl_workorder_notification_clock"); err != nil {
				t.Fatal(err)
			}
			if _, err := db.ExecContext(ctx, test.seed); err != nil {
				t.Fatal("cannot seed fake time history")
			}
			if err := dbschema.InitializeFreshClock(ctx, db, "UTC"); !errors.Is(err, dbschema.ErrClockHistory) {
				t.Fatal("history was implicitly assigned a zone", err)
			}
			if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cl_workorder_notification_clock").Scan(&count); err != nil || count != 0 {
				t.Fatal("history rejection wrote marker")
			}
			if err := db.QueryRowContext(ctx, test.preserved).Scan(&count); err != nil || count != 1 {
				t.Fatal("history rejection changed stored time values")
			}
			if _, err := db.ExecContext(ctx, test.cleanup); err != nil {
				t.Fatal(err)
			}
		})
	}

	t.Run("dirty schema rejected before marker write", func(t *testing.T) {
		if _, err := db.ExecContext(ctx, "UPDATE cloudops_schema_history SET state='failed' WHERE version=1"); err != nil {
			t.Fatal(err)
		}
		if err := dbschema.InitializeFreshClock(ctx, db, "UTC"); !errors.Is(err, schema.ErrDirty) {
			t.Fatal("dirty manifest accepted", err)
		}
		if _, err := db.ExecContext(ctx, "UPDATE cloudops_schema_history SET state='applied' WHERE version=1"); err != nil {
			t.Fatal(err)
		}
	})
	for _, table := range []string{"cl_workorder_notification_event", "cl_workorder_notification", "cl_workorder_notification_clock"} {
		t.Run("missing table "+table, func(t *testing.T) {
			// Rename only the owned fixture table, preserving constraints/data. No
			// missing table may be interpreted as empty legacy history.
			hidden := "hidden_clock_fixture_table"
			if _, err := db.ExecContext(ctx, "RENAME TABLE `"+table+"` TO `"+hidden+"`"); err != nil {
				t.Fatal("cannot rename owned fixture table")
			}
			defer func() {
				if _, err := db.ExecContext(ctx, "RENAME TABLE `"+hidden+"` TO `"+table+"`"); err != nil {
					t.Error("cannot restore owned fixture table")
				}
			}()
			if err := dbschema.InitializeFreshClock(ctx, db, "UTC"); !errors.Is(err, dbschema.ErrClockDatabase) {
				t.Fatal("missing table accepted", err)
			}
			if table == "cl_workorder_notification_clock" {
				if err := dbschema.VerifyClock(ctx, db, "UTC"); !errors.Is(err, dbschema.ErrClockDatabase) {
					t.Fatal("verification accepted missing clock table", err)
				}
			}
		})
	}
}

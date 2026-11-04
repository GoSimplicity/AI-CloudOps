//go:build integration

package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

// The hook pauses after the real read-only CLI preflight has completed and
// released its connection. Another real pool then changes the database before
// the CLI can acquire the migration lock. No sleeps or production hooks are used.
type namespacePreflightBarrier struct {
	schemaDatabase
	reached chan<- int64
	resume  <-chan struct{}
}

func (b namespacePreflightBarrier) pause(ctx context.Context, err error) error {
	if err != nil {
		return err
	}
	actual, ok := b.schemaDatabase.(sqlDatabase)
	if !ok {
		return schema.ErrPreflight
	}
	var id int64
	if err := actual.db.QueryRowContext(ctx, "SELECT CONNECTION_ID()").Scan(&id); err != nil {
		return schema.ErrPreflight
	}
	select {
	case b.reached <- id:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-b.resume:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (b namespacePreflightBarrier) PreflightFresh(ctx context.Context) error {
	return b.pause(ctx, b.schemaDatabase.PreflightFresh(ctx))
}
func (b namespacePreflightBarrier) PreflightPending(ctx context.Context, m []schema.Migration) error {
	return b.pause(ctx, b.schemaDatabase.PreflightPending(ctx, m))
}

// Opt-in: Root provisions one NEW EMPTY loopback schema with a schema-scoped
// user. No CREATE DATABASE, application DSN, demonstration database or host is used.
func TestNamespaceSchemaNativeMySQL(t *testing.T) {
	path := os.Getenv("CLOUDOPS_NAMESPACE_MYSQL_DSN_FILE")
	if path == "" {
		t.Skip("private namespace schema fixture not configured")
	}
	raw, err := readCredentials(path, func(string) string { return "" })
	if err != nil {
		t.Fatal("private fixture DSN file invalid")
	}
	cfg, err := databaseConfig(raw, 30*time.Second)
	if err != nil || cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:13306" || !regexp.MustCompile(`^cloudops_namespace_[a-z0-9_]+$`).MatchString(cfg.DBName) || cfg.DBName != os.Getenv("CLOUDOPS_NAMESPACE_MYSQL_DATABASE") {
		t.Fatal("fixture requires exact loopback and separately confirmed private schema")
	}
	cfg.ParseTime = true
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal("private fixture pool failed")
	}
	defer db.Close()
	db.SetMaxOpenConns(2)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	current, err := dbschema.CurrentManifest()
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := dbschema.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{schemaHistoryTable: true, configNamespaceTable: true}
	for _, name := range currentBusinessTables() {
		allowed[name] = true
	}
	inventory := func(t *testing.T) []string {
		t.Helper()
		rows, e := db.QueryContext(ctx, "SELECT table_name, table_type FROM information_schema.tables WHERE table_schema=DATABASE() ORDER BY table_name")
		if e != nil {
			t.Fatal("fixture inventory failed")
		}
		defer rows.Close()
		var names []string
		for rows.Next() {
			var name, kind string
			if rows.Scan(&name, &kind) != nil || kind != "BASE TABLE" || !allowed[name] {
				t.Fatal("unknown fixture object; refusing any cleanup")
			}
			names = append(names, name)
		}
		if rows.Err() != nil {
			t.Fatal("fixture inventory incomplete")
		}
		return names
	}
	if len(inventory(t)) != 0 {
		t.Fatal("fixture schema must initially be empty")
	}
	clean := func(t *testing.T, keepHistory bool) {
		t.Helper()
		names := inventory(t)
		conn, e := db.Conn(ctx)
		if e != nil {
			t.Fatal("cleanup connection unavailable")
		}
		defer conn.Close()
		if _, e := conn.ExecContext(ctx, "SET SESSION FOREIGN_KEY_CHECKS=0"); e != nil {
			t.Fatal("fixture cleanup guard failed")
		}
		defer conn.ExecContext(ctx, "SET SESSION FOREIGN_KEY_CHECKS=1")
		sort.Sort(sort.Reverse(sort.StringSlice(names)))
		for _, name := range names {
			if keepHistory && name == schemaHistoryTable {
				continue
			}
			if _, e := conn.ExecContext(ctx, "DROP TABLE `"+name+"`"); e != nil {
				t.Fatal("known fixture table cleanup failed")
			}
		}
	}
	// This fixture alone created every allowed object after the initial-empty gate.
	defer clean(t, false)
	invoke := func(t *testing.T, want int, args ...string) {
		t.Helper()
		var out, diagnostic strings.Builder
		code := run(ctx, args, &out, &diagnostic, dependencies{manifest: dbschema.CurrentManifest, getenv: func(string) string { return cfg.FormatDSN() }, open: openDatabase})
		if code != want {
			t.Fatalf("owned CLI fixture failed command=%s code=%d expected=%d", args[0], code, want)
		}
	}
	verifyPrefix := func(t *testing.T, want error) {
		t.Helper()
		conn, e := db.Conn(ctx)
		if e != nil {
			t.Fatal("prefix connection unavailable")
		}
		defer conn.Close()
		if err := schema.VerifyAppliedPrefixConnection(ctx, conn, current); !errors.Is(err, want) {
			t.Fatal("unexpected applied prefix classification")
		}
	}
	t.Run("fresh and empty ledger", func(t *testing.T) {
		verifyPrefix(t, schema.ErrPreflight)
		invoke(t, 1, "migrate", "--confirm-pending-migrations", "--confirm-no-writers")
		invoke(t, 0, "migrate", "--confirm-new-database")
		if schema.Verify(ctx, db, current) != nil {
			t.Fatal("fresh current history not complete")
		}
		invoke(t, 0, "init-clock", "--confirm-no-writers", "--notification-timezone", "UTC")
		invoke(t, 0, "verify-clock", "--notification-timezone", "UTC")
		key := strings.Repeat("🌍", 45)
		for _, rawKey := range []string{"2001:db8::1", "2001:DB8::1", "2001:db8::1 ", " 2001:db8::1", key} {
			if _, e := db.ExecContext(ctx, "INSERT INTO cl_monitor_config_namespaces(config_type,instance_key) VALUES(1,?)", []byte(rawKey)); e != nil {
				t.Fatal("byte-exact or 45-character UTF-8 key rejected")
			}
		}
		if _, e := db.ExecContext(ctx, "INSERT INTO cl_monitor_config_namespaces(config_type,instance_key) VALUES(1,?)", []byte(key)); e == nil {
			t.Fatal("same raw namespace duplicate accepted")
		}
		if _, e := db.ExecContext(ctx, "INSERT INTO cl_monitor_config_namespaces(config_type,instance_key) VALUES(2,?)", []byte(key)); e != nil {
			t.Fatal("cross-type raw key incorrectly merged")
		}
		if _, e := db.ExecContext(ctx, "DELETE FROM cloudops_schema_history"); e != nil {
			t.Fatal("empty ledger fixture failed")
		}
		clean(t, true)
		verifyPrefix(t, schema.ErrPending)
		invoke(t, 1, "migrate", "--confirm-pending-migrations", "--confirm-no-writers")
		invoke(t, 0, "migrate", "--confirm-new-database")
		if schema.Verify(ctx, db, current) != nil {
			t.Fatal("empty ledger could not apply current manifest")
		}
	})
	clean(t, false)
	t.Run("applied v1 appends current pending versions", func(t *testing.T) {
		if schema.Migrate(ctx, db, legacy) != nil {
			t.Fatal("v1 fixture failed")
		}
		verifyPrefix(t, nil)
		invoke(t, 1, "init-clock", "--confirm-no-writers", "--notification-timezone", "UTC")
		if !errors.Is(dbschema.InitializeFreshClock(ctx, db, "UTC"), schema.ErrPending) {
			t.Fatal("v1 prefix authorized current clock initialization")
		}
		var markers int
		if db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cl_workorder_notification_clock").Scan(&markers) != nil || markers != 0 {
			t.Fatal("pending v2 wrote clock marker")
		}
		for _, name := range []string{"legacy custom A", "legacy custom B"} {
			if _, e := db.ExecContext(ctx, "INSERT INTO cl_monitor_configs(name,pool_id,instance_ip,config_type,config_content,config_hash,status,last_generated_time,deleted_at) VALUES(?,7,'192.0.2.1',4,'preserve','hash',2,1,0)", name); e != nil {
				t.Fatal("historical duplicate fixture failed")
			}
		}
		invoke(t, 1, "verify")
		invoke(t, 1, "migrate", "--confirm-new-database")
		invoke(t, 0, "migrate", "--confirm-pending-migrations", "--confirm-no-writers")
		invoke(t, 0, "verify")
		invoke(t, 0, "init-clock", "--confirm-no-writers", "--notification-timezone", "UTC")
		invoke(t, 0, "verify-clock", "--notification-timezone", "UTC")
		var count int
		var checksum string
		if db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cl_monitor_configs WHERE pool_id=7 AND status=2 AND config_content='preserve'").Scan(&count) != nil || count != 2 {
			t.Fatal("v2 altered historical duplicate rows")
		}
		if db.QueryRowContext(ctx, "SELECT checksum FROM cloudops_schema_history WHERE version=1 AND state='applied'").Scan(&checksum) != nil || checksum != legacy[0].Checksum {
			t.Fatal("v1 provenance changed")
		}
		invoke(t, 0, "migrate", "--confirm-new-database")
	})
	clean(t, false)
	t.Run("adopt remains v1 then explicit v2", func(t *testing.T) {
		conn, e := db.Conn(ctx)
		if e != nil {
			t.Fatal("legacy fixture connection failed")
		}
		e = legacy[0].Apply(ctx, conn)
		conn.Close()
		if e != nil {
			t.Fatal("legacy v1 fixture failed")
		}
		note := filepath.Join(t.TempDir(), "note")
		if os.WriteFile(note, []byte("owned fixture schema inspected"), 0600) != nil {
			t.Fatal("note fixture failed")
		}
		invoke(t, 0, "adopt", "--confirm-legacy-schema", "--confirm-no-writers", "--adoption-note-file", note)
		if !errors.Is(schema.Verify(ctx, db, current), schema.ErrPending) {
			t.Fatal("adoption silently claimed v2")
		}
		invoke(t, 0, "migrate", "--confirm-pending-migrations", "--confirm-no-writers")
		var origin string
		if db.QueryRowContext(ctx, "SELECT origin FROM cloudops_schema_history WHERE version=1").Scan(&origin) != nil || origin != "adopted" {
			t.Fatal("v2 changed baseline adoption provenance")
		}
	})
	clean(t, false)
	for _, race := range []string{"fresh becomes v1", "pending becomes empty", "pending becomes missing"} {
		t.Run(race+" after CLI preflight", func(t *testing.T) {
			if race != "fresh becomes v1" {
				if schema.Migrate(ctx, db, legacy) != nil {
					t.Fatal("v1 race fixture failed")
				}
			}
			reached, resume := make(chan int64, 1), make(chan struct{})
			done := make(chan int, 1)
			var out, diagnostic strings.Builder
			args := []string{"migrate", "--confirm-new-database"}
			if race != "fresh becomes v1" {
				args = []string{"migrate", "--confirm-pending-migrations", "--confirm-no-writers"}
			}
			released, completed := false, false
			release := func() {
				if !released {
					close(resume)
					released = true
				}
			}
			defer func() {
				release()
				if !completed {
					select {
					case <-done:
					case <-ctx.Done():
					}
				}
			}()
			go func() {
				done <- run(ctx, args, &out, &diagnostic, dependencies{manifest: dbschema.CurrentManifest, getenv: func(string) string { return cfg.FormatDSN() }, open: func(dsn string, deadline time.Duration) (schemaDatabase, error) {
					actual, err := openDatabase(dsn, deadline)
					if err != nil {
						return nil, err
					}
					return namespacePreflightBarrier{schemaDatabase: actual, reached: reached, resume: resume}, nil
				}})
			}()
			var cliID int64
			select {
			case cliID = <-reached:
			case code := <-done:
				completed = true
				t.Fatalf("CLI did not reach successful preflight: code=%d", code)
			case <-ctx.Done():
				t.Fatal("preflight barrier timed out")
			}
			var competingID int64
			if db.QueryRowContext(ctx, "SELECT CONNECTION_ID()").Scan(&competingID) != nil || competingID == cliID {
				t.Fatal("race fixture did not use distinct real sessions")
			}
			switch race {
			case "fresh becomes v1":
				if schema.Migrate(ctx, db, legacy) != nil {
					t.Fatal("competing v1 migration failed")
				}
			case "pending becomes empty":
				if _, err := db.ExecContext(ctx, "DELETE FROM cloudops_schema_history"); err != nil {
					t.Fatal("competing ledger clear failed")
				}
			case "pending becomes missing":
				if _, err := db.ExecContext(ctx, "DROP TABLE cloudops_schema_history"); err != nil {
					t.Fatal("competing ledger removal failed")
				}
			}
			release()
			var code int
			select {
			case code = <-done:
				completed = true
			case <-ctx.Done():
				t.Fatal("confirmed migration did not return")
			}
			if code != 1 || !strings.Contains(diagnostic.String(), "migration confirmation does not match the database state under the migration lock") {
				t.Fatal("stale CLI preflight authorized migration or failed outside locked mode check")
			}
			var guards int
			if db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='cl_monitor_config_namespaces'").Scan(&guards) != nil || guards != 0 {
				t.Fatal("stale mode appended v2")
			}
			if race == "pending becomes missing" {
				var ledgers int
				if db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='cloudops_schema_history'").Scan(&ledgers) != nil || ledgers != 0 {
					t.Fatal("pending recreated missing ledger")
				}
			} else {
				var records int
				wanted := 0
				if race == "fresh becomes v1" {
					wanted = 1
				}
				if db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cloudops_schema_history").Scan(&records) != nil || records != wanted {
					t.Fatal("stale mode wrote ledger")
				}
			}
		})
		clean(t, false)
	}
	for _, state := range []string{"failed", "drift"} {
		t.Run(state+" tail refuses writes", func(t *testing.T) {
			if schema.Migrate(ctx, db, legacy) != nil {
				t.Fatal("v1 fixture failed")
			}
			checksum := current[1].Checksum
			recordState := "failed"
			want := schema.ErrDirty
			if state == "drift" {
				checksum = strings.Repeat("0", 64)
				recordState = "applied"
				want = schema.ErrDrift
			}
			if _, e := db.ExecContext(ctx, "INSERT INTO cloudops_schema_history(version,name,checksum,state) VALUES(2,?,?,?)", current[1].Name, checksum, recordState); e != nil {
				t.Fatal("dirty ledger fixture failed")
			}
			verifyPrefix(t, want)
			invoke(t, 1, "migrate", "--confirm-pending-migrations", "--confirm-no-writers")
			var count int
			if db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='cl_monitor_config_namespaces'").Scan(&count) != nil || count != 0 {
				t.Fatal("dirty/drift ledger executed v2")
			}
		})
		clean(t, false)
	}
}

//go:build integration

package schema

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

func TestMySQLAdoptionLeavesLegacyDataAndPreservesAudit(t *testing.T) {
	admin, address := ownedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	newLegacy := func(t *testing.T, name string) *sql.DB {
		t.Helper()
		if _, err := admin.ExecContext(ctx, "CREATE DATABASE `"+name+"`"); err != nil {
			t.Fatal("owned adoption fixture database setup failed")
		}
		config := mysql.NewConfig()
		config.User, config.Net, config.Addr, config.DBName = "root", "tcp", address, name
		config.ParseTime, config.Timeout, config.ReadTimeout, config.WriteTimeout = true, 3*time.Second, 10*time.Second, 10*time.Second
		config.Logger = &mysql.NopLogger{}
		connector, err := mysql.NewConnector(config)
		if err != nil {
			t.Fatal("owned adoption fixture connector failed")
		}
		db := sql.OpenDB(connector)
		db.SetMaxOpenConns(1)
		t.Cleanup(func() { _ = db.Close() })
		for _, statement := range []string{
			"CREATE TABLE legacy_item (id BIGINT PRIMARY KEY, value VARCHAR(100) NOT NULL, archived TINYINT NOT NULL DEFAULT 0)",
			"INSERT INTO legacy_item VALUES (1,'preserved live value',0),(2,'preserved archived value',1)",
		} {
			if _, err := db.ExecContext(ctx, statement); err != nil {
				t.Fatal("owned legacy data setup failed")
			}
		}
		return db
	}
	manifest := []Migration{SQLMigration(1, "reviewed legacy baseline", "CREATE TABLE legacy_item (id BIGINT PRIMARY KEY, value VARCHAR(100) NOT NULL, archived TINYINT NOT NULL DEFAULT 0)")}
	manifest[0].Apply = func(context.Context, *sql.Conn) error { t.Fatal("adoption executed migration Apply"); return nil }
	assertPreserved := func(t *testing.T, db *sql.DB) {
		t.Helper()
		var values string
		if err := db.QueryRowContext(ctx, "SELECT GROUP_CONCAT(CONCAT(id,':',value,':',archived) ORDER BY id SEPARATOR '|') FROM legacy_item").Scan(&values); err != nil || values != "1:preserved live value:0|2:preserved archived value:1" {
			t.Fatal("legacy rows were modified")
		}
	}
	ledgerExists := func(t *testing.T, db *sql.DB) bool {
		t.Helper()
		var exists bool
		if err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='cloudops_schema_history')").Scan(&exists); err != nil {
			t.Fatal("ledger existence query failed")
		}
		return exists
	}

	t.Run("adopt and repeat", func(t *testing.T) {
		db := newLegacy(t, "adopt_success")
		var beforeTable, beforeDDL string
		if err := db.QueryRowContext(ctx, "SHOW CREATE TABLE legacy_item").Scan(&beforeTable, &beforeDDL); err != nil {
			t.Fatal("legacy DDL snapshot failed")
		}
		validationCalls := 0
		var actor string
		validate := func(ctx context.Context, conn *sql.Conn) error {
			validationCalls++
			var connection, lockOwner int64
			lockName := "cloudops.schema." + Checksum([]byte("adopt_success"))[:40]
			if err := conn.QueryRowContext(ctx, "SELECT CONNECTION_ID(), IS_USED_LOCK(?), CURRENT_USER()", lockName).Scan(&connection, &lockOwner, &actor); err != nil || connection != lockOwner {
				t.Fatal("validator does not own the shared migration lock")
			}
			var tables, rows int
			if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()").Scan(&tables); err != nil || tables != 1 {
				return errors.New("fixture shape mismatch")
			}
			if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM legacy_item").Scan(&rows); err != nil || rows != 2 {
				return errors.New("fixture row mismatch")
			}
			return nil
		}
		note := "已核对旧表和历史修复；保留原值 ' ; DROP TABLE legacy_item; --\nprivate-note-canary"
		if err := AdoptBaseline(ctx, db, manifest, validate, note); err != nil {
			t.Fatal("trusted adoption failed")
		}
		assertPreserved(t, db)
		var afterTable, afterDDL string
		if err := db.QueryRowContext(ctx, "SHOW CREATE TABLE legacy_item").Scan(&afterTable, &afterDDL); err != nil || beforeTable != afterTable || beforeDDL != afterDDL {
			t.Fatal("adoption changed business DDL")
		}
		var origin, storedNote, storedActor, state string
		var timesMatch bool
		if err := db.QueryRowContext(ctx, "SELECT origin,adoption_note,adopted_by,state,started_at=finished_at FROM cloudops_schema_history WHERE version=1").Scan(&origin, &storedNote, &storedActor, &state, &timesMatch); err != nil || origin != "adopted" || storedNote != note || storedActor != actor || state != "applied" || !timesMatch {
			t.Fatal("adoption evidence was not recorded atomically")
		}
		if err := Verify(ctx, db, manifest); err != nil {
			t.Fatal("adopted schema history cannot verify")
		}
		if err := AdoptBaseline(ctx, db, manifest, func(context.Context, *sql.Conn) error { t.Fatal("repeat called legacy validator"); return nil }, "different note must not overwrite evidence"); err != nil {
			t.Fatal("repeat adoption failed read-only verification")
		}
		if err := db.QueryRowContext(ctx, "SELECT adoption_note FROM cloudops_schema_history WHERE version=1").Scan(&storedNote); err != nil || storedNote != note || validationCalls != 1 {
			t.Fatal("repeat adoption changed evidence")
		}
		// New audit defaults remain compatible with the ordinary migration
		// INSERT field list when a later forward version is executed.
		next := append(append([]Migration{}, manifest...), SQLMigration(2, "forward fixture", "CREATE TABLE next_item (id BIGINT PRIMARY KEY)"))
		if err := Migrate(ctx, db, next); err != nil {
			t.Fatal("audit variant broke normal forward migration")
		}
		var nextOrigin string
		var nextNote, nextActor sql.NullString
		if err := db.QueryRowContext(ctx, "SELECT origin,adoption_note,adopted_by FROM cloudops_schema_history WHERE version=2").Scan(&nextOrigin, &nextNote, &nextActor); err != nil || nextOrigin != "migrated" || nextNote.Valid || nextActor.Valid {
			t.Fatal("forward migration audit defaults are incompatible")
		}
		assertPreserved(t, db)
	})

	for _, state := range []string{"empty", "started", "failed", "drift"} {
		t.Run("existing ledger "+state, func(t *testing.T) {
			db := newLegacy(t, "adopt_"+state)
			if _, err := db.ExecContext(ctx, ledgerDDL); err != nil {
				t.Fatal("existing ledger setup failed")
			}
			want := ErrDirty
			if state == "empty" {
				want = ErrPending
			} else {
				checksum, storedState := manifest[0].Checksum, state
				if state == "drift" {
					checksum, storedState, want = Checksum([]byte("changed")), "applied", ErrDrift
				}
				if _, err := db.ExecContext(ctx, "INSERT INTO cloudops_schema_history (version,name,checksum,state) VALUES (1,?,?,?)", manifest[0].Name, checksum, storedState); err != nil {
					t.Fatal("existing history setup failed")
				}
			}
			err := AdoptBaseline(ctx, db, manifest, func(context.Context, *sql.Conn) error { t.Fatal("existing ledger called validator"); return nil }, "must not alter existing ledger")
			if !errors.Is(err, want) {
				t.Fatal("existing ledger did not fail closed")
			}
			var auditColumns int
			if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='cloudops_schema_history' AND column_name IN ('origin','adoption_note','adopted_by')").Scan(&auditColumns); err != nil || auditColumns != 0 {
				t.Fatal("existing ledger was altered to add adoption fields")
			}
			assertPreserved(t, db)
		})
	}
	t.Run("fresh applied ledger remains unchanged", func(t *testing.T) {
		db := newLegacy(t, "adopt_fresh")
		if _, err := db.ExecContext(ctx, ledgerDDL); err != nil {
			t.Fatal("fresh history setup failed")
		}
		if _, err := db.ExecContext(ctx, "INSERT INTO cloudops_schema_history (version,name,checksum,state) VALUES (1,?,?,'applied')", manifest[0].Name, manifest[0].Checksum); err != nil {
			t.Fatal("fresh history setup failed")
		}
		if err := AdoptBaseline(ctx, db, manifest, func(context.Context, *sql.Conn) error { t.Fatal("fresh applied ledger called validator"); return nil }, "unused note"); err != nil {
			t.Fatal("fresh ledger did not verify")
		}
		var columns int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='cloudops_schema_history'").Scan(&columns); err != nil || columns != 6 {
			t.Fatal("fresh ledger columns changed")
		}
		assertPreserved(t, db)
	})
	t.Run("validator failure has no ledger", func(t *testing.T) {
		db := newLegacy(t, "adopt_invalid")
		err := AdoptBaseline(ctx, db, manifest, func(context.Context, *sql.Conn) error { return errors.New("private-validator-canary") }, "private-note-canary")
		if !errors.Is(err, ErrPreflight) || strings.Contains(err.Error(), "private-") || ledgerExists(t, db) {
			t.Fatal("failed validator leaked evidence or created a ledger")
		}
		assertPreserved(t, db)
	})
	t.Run("validator panic releases physical lock", func(t *testing.T) {
		db := newLegacy(t, "adopt_panic")
		var before int64
		err := AdoptBaseline(ctx, db, manifest, func(ctx context.Context, conn *sql.Conn) error {
			if err := conn.QueryRowContext(ctx, "SELECT CONNECTION_ID()").Scan(&before); err != nil {
				t.Fatal("panic fixture identity read failed")
			}
			panic("private-panic-canary")
		}, "private-note-canary")
		if !errors.Is(err, ErrUncertain) || strings.Contains(err.Error(), "private-") || ledgerExists(t, db) {
			t.Fatal("panic was not sanitized or created history")
		}
		if err := WithMigrationLock(ctx, db, func(conn *sql.Conn) error {
			var after int64
			if err := conn.QueryRowContext(ctx, "SELECT CONNECTION_ID()").Scan(&after); err != nil || after == before {
				t.Fatal("panicked physical connection was reused")
			}
			return nil
		}); err != nil {
			t.Fatal("lock was not released after validator panic")
		}
		assertPreserved(t, db)
	})
}

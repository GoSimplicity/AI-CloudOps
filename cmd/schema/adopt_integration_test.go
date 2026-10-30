//go:build integration

package main

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

func TestMySQLCLILegacyInspectionAndAuditedAdoption(t *testing.T) {
	db, dsn := cliMySQLFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	manifest, err := dbschema.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.WithMigrationLock(ctx, db, func(conn *sql.Conn) error { return manifest[0].Apply(ctx, conn) }); err != nil {
		t.Fatal("owned legacy structure setup failed")
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO cl_system_departments (id,name,parent_id) VALUES (901,'preserved legacy fixture',0)"); err != nil {
		t.Fatal("legacy row fixture failed")
	}
	note := filepath.Join(t.TempDir(), "adoption-note")
	const auditNote = "fixture operator reviewed exact schema after private backup"
	if err := os.WriteFile(note, []byte(auditNote), 0600); err != nil {
		t.Fatal(err)
	}
	invoke := func(want int, args ...string) {
		t.Helper()
		var out, diagnostic bytes.Buffer
		code := run(ctx, args, &out, &diagnostic, dependencies{manifest: dbschema.Manifest, getenv: func(string) string { return dsn }, open: openDatabase})
		if code != want {
			t.Fatalf("unexpected CLI status %d expected %d; safe diagnostic: %s", code, want, diagnostic.String())
		}
		if strings.Contains(out.String()+diagnostic.String(), auditNote) || strings.Contains(out.String()+diagnostic.String(), dsn) {
			t.Fatal("CLI exposed private input")
		}
	}
	assertNoLedger := func() {
		t.Helper()
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='cloudops_schema_history'").Scan(&count); err != nil || count != 0 {
			t.Fatal("failed legacy validation created a ledger")
		}
	}
	invoke(0, "preflight-legacy")
	assertNoLedger()
	if _, err := db.ExecContext(ctx, "ALTER TABLE cl_system_departments ADD COLUMN unexpected_fixture INT"); err != nil {
		t.Fatal("drift fixture failed")
	}
	invoke(1, "preflight-legacy")
	adoptArgs := []string{"adopt", "--confirm-legacy-schema", "--confirm-no-writers", "--adoption-note-file", note}
	invoke(1, adoptArgs...)
	assertNoLedger()
	if _, err := db.ExecContext(ctx, "ALTER TABLE cl_system_departments DROP COLUMN unexpected_fixture"); err != nil {
		t.Fatal("drift fixture restoration failed")
	}
	invoke(0, adoptArgs...)
	invoke(0, "verify")
	invoke(0, adoptArgs...)
	var origin, storedNote, actor string
	if err := db.QueryRowContext(ctx, "SELECT origin,adoption_note,adopted_by FROM cloudops_schema_history WHERE version=1").Scan(&origin, &storedNote, &actor); err != nil || origin != "adopted" || storedNote != auditNote || actor == "" {
		t.Fatal("adoption audit missing")
	}
	var name string
	if err := db.QueryRowContext(ctx, "SELECT name FROM cl_system_departments WHERE id=901").Scan(&name); err != nil || name != "preserved legacy fixture" {
		t.Fatal("adoption changed legacy business row")
	}
	invoke(1, "verify-clock", "--notification-timezone", "UTC")
	invoke(0, "init-clock", "--confirm-no-writers", "--notification-timezone", "UTC")
	invoke(0, "verify-clock", "--notification-timezone", "UTC")
}

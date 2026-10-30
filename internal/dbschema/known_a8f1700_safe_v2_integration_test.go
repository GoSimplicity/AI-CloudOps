//go:build integration

package dbschema_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
	"github.com/go-sql-driver/mysql"
)

func reconnectKnownRole(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	parsed, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("owned role DSN invalid")
	}
	connector, err := mysql.NewConnector(parsed)
	if err != nil {
		t.Fatal("owned role connector invalid")
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func runSafeKnownCLI(t *testing.T, ctx context.Context, dsn, backup string) int {
	t.Helper()
	directory := t.TempDir()
	note := filepath.Join(directory, "note")
	proof := filepath.Join(directory, "proof")
	if err := os.WriteFile(note, []byte("owned fixture backup and no-writers reviewed"), 0600); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"version": 1, "source_revision": dbschema.KnownA8Source, "backup_sha256": backup, "backup_ready": true})
	if err := os.WriteFile(proof, body, 0600); err != nil {
		t.Fatal(err)
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", "./cmd/schema", "upgrade-a8f1700", "--execute", "--confirm-no-writers", "--upgrade-note-file", note, "--backup-proof-file", proof, "--notification-timezone", "UTC", "--attest-historical-time")
	command.Dir = repo
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "CLOUDOPS_SCHEMA_DSN=") {
			command.Env = append(command.Env, entry)
		}
	}
	command.Env = append(command.Env, "CLOUDOPS_SCHEMA_DSN="+dsn)
	if _, err := command.CombinedOutput(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		t.Fatal("owned guarded CLI failed without an exit status")
	}
	return 0
}

func TestKnownA8SafeV2RejectsHiddenAndVisibleTriggersBeforeLedger(t *testing.T) {
	admin, address := knownUpgradeMySQL(t)
	source := knownSourceFixture(t, admin)
	seedKnownRows(t, admin)
	before := knownDataDigest(t, admin, source)
	backup := knownPrivateBackup(t, admin, source)
	_, dsn := minimalKnownMigrator(t, admin, address)
	execKnownFixture(t, admin, "REVOKE TRIGGER ON knownupgrade.* FROM 'known_migration'@'%'")
	execKnownFixture(t, admin, "REVOKE TRIGGER ON knownupgrade.* FROM 'known_migrator'@'%'")
	execKnownFixture(t, admin, "CREATE TRIGGER extra_notification_update BEFORE UPDATE ON cl_workorder_notification FOR EACH ROW SET NEW.message_template='changed-by-unknown-trigger'")
	referencesOnly := reconnectKnownRole(t, dsn)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	var message string
	var initialDefault int
	if err := admin.QueryRow("SELECT message_template,is_default FROM cl_workorder_notification WHERE id=61").Scan(&message, &initialDefault); err != nil || initialDefault != 0 {
		t.Fatal("fixture source row was not initialized")
	}
	var visibleTriggers int
	if err := referencesOnly.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=DATABASE()").Scan(&visibleTriggers); err != nil || visibleTriggers != 0 {
		t.Fatal("REFERENCES-only fixture unexpectedly exposed triggers")
	}
	// Demonstrate the actual minimal-role risk without upgrading or retaining any
	// mutation: the frozen repair fires the root-defined trigger despite an empty
	// visible inventory. Roll back this one DML transaction immediately.
	tx, err := referencesOnly.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal("owned MRE transaction failed")
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "UPDATE cl_workorder_notification SET is_default=2 WHERE is_default=0"); err != nil {
		t.Fatal("owned frozen repair MRE failed")
	}
	var changed string
	if err := tx.QueryRowContext(ctx, "SELECT message_template FROM cl_workorder_notification WHERE id=61").Scan(&changed); err != nil || changed != "changed-by-unknown-trigger" {
		t.Fatal("unknown trigger did not demonstrate nonrepair field mutation")
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal("owned MRE rollback failed")
	}
	unsafe, err := dbschema.KnownA8Upgrade()
	if err != nil {
		t.Fatal(err)
	}
	conn, err := referencesOnly.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := unsafe.ValidateSource(ctx, conn, schema.UpgradeEvidence{NotificationZone: "UTC", HistoricalTimeAttested: true}); err != nil {
		t.Fatal("old table-only source gate did not reproduce the missing-trigger check")
	}
	conn.Close()
	safe, err := dbschema.KnownA8SafeUpgradeV2()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := dbschema.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	evidence := schema.UpgradeEvidence{Note: "owned fixture proof", BackupSHA256: backup, NotificationZone: "UTC", HistoricalTimeAttested: true}
	if err := schema.UpgradeBaseline(ctx, referencesOnly, manifest, safe, evidence); !errors.Is(err, schema.ErrPreflight) {
		t.Fatal("unproven trigger visibility was accepted")
	}
	if runSafeKnownCLI(t, ctx, dsn, backup) == 0 {
		t.Fatal("CLI bypassed direct TRIGGER privilege proof")
	}
	assertUntouched := func() {
		t.Helper()
		var ledger int
		if err := admin.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='cloudops_schema_history'").Scan(&ledger); err != nil || ledger != 0 {
			t.Fatal("rejected trigger source created a ledger")
		}
		var actual string
		var value int
		if err := admin.QueryRow("SELECT message_template,is_default FROM cl_workorder_notification WHERE id=61").Scan(&actual, &value); err != nil || actual != message || value != initialDefault {
			t.Fatal("rejected trigger source changed old fields")
		}
		if knownDataDigest(t, admin, source) != before {
			t.Fatal("rejected source changed historical data")
		}
	}
	assertUntouched()
	execKnownFixture(t, admin, "GRANT TRIGGER ON knownupgrade.* TO 'known_migrator'@'%'")
	visible := reconnectKnownRole(t, dsn)
	if err := visible.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=DATABASE()").Scan(&visibleTriggers); err != nil || visibleTriggers != 1 {
		t.Fatal("direct TRIGGER metadata grant did not expose the fixture trigger")
	}
	if err := schema.UpgradeBaseline(ctx, visible, manifest, safe, evidence); !errors.Is(err, schema.ErrPreflight) {
		t.Fatal("visible unknown trigger was accepted")
	}
	if runSafeKnownCLI(t, ctx, dsn, backup) == 0 {
		t.Fatal("CLI default path did not reject the visible trigger")
	}
	assertUntouched()
	execKnownFixture(t, admin, "DROP TRIGGER extra_notification_update")
	if runSafeKnownCLI(t, ctx, dsn, backup) != 0 {
		t.Fatal("normal no-trigger source failed guarded CLI upgrade")
	}
	if knownDataDigest(t, admin, source) != before {
		t.Fatal("guarded 15-statement upgrade changed old columns outside repairs")
	}
	var implementation string
	if err := admin.QueryRow("SELECT upgrade_checksum FROM cloudops_schema_history WHERE version=1 AND state='applied'").Scan(&implementation); err != nil || implementation != safe.ImplementationChecksum {
		t.Fatal("guarded CLI did not record v2 provenance")
	}
	if err := schema.Verify(ctx, visible, manifest); err != nil {
		t.Fatal("normal guarded target failed manifest verification")
	}
}

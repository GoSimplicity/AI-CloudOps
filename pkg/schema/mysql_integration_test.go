//go:build integration

package schema

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// ownedMySQL never uses an existing service or arbitrary DSN. The fixture starts
// a disposable, labelled local container and removes its anonymous data volume.
func ownedMySQL(t *testing.T) (*sql.DB, string) {
	t.Helper()
	run := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("docker", args...).CombinedOutput()
		if err != nil {
			t.Fatal("owned MySQL fixture command failed; output withheld")
		}
		return strings.TrimSpace(string(out))
	}
	if endpoint := run("context", "inspect", "--format", "{{.Endpoints.docker.Host}}"); !strings.HasPrefix(endpoint, "unix://") {
		t.Fatal("fixture requires a local Unix-socket Docker context")
	}
	var token [12]byte
	if _, err := rand.Read(token[:]); err != nil {
		t.Fatal(err)
	}
	name := "cloudops-schema-test-" + hex.EncodeToString(token[:])
	label := "cloudops.schema.fixture=" + name
	run("run", "--detach", "--name", name, "--label", label, "-e", "MYSQL_ALLOW_EMPTY_PASSWORD=yes", "-e", "MYSQL_ROOT_HOST=%", "-e", "MYSQL_DATABASE=schema_fixture", "-p", "127.0.0.1::3306", "mysql:8.4.11")
	t.Cleanup(func() {
		if run("inspect", "--format", `{{index .Config.Labels "cloudops.schema.fixture"}}`, name) != name {
			t.Error("refusing to remove container with mismatched owner label")
			return
		}
		run("rm", "--force", "--volumes", name)
	})
	address := run("port", name, "3306/tcp")
	if !strings.HasPrefix(address, "127.0.0.1:") {
		t.Fatal("fixture port is not loopback")
	}
	db, err := sql.Open("mysql", "root@tcp("+address+")/schema_fixture?parseTime=true&timeout=2s&readTimeout=5s&writeTimeout=5s")
	if err != nil {
		t.Fatal("fixture database open failed")
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	for {
		if db.PingContext(ctx) == nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("owned MySQL did not become ready")
		case <-ticker.C:
		}
	}
	return db, address
}

func TestMySQLMigrationLifecycle(t *testing.T) {
	db, address := ownedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	manifest := []Migration{SQLMigration(1, "create fixture", "CREATE TABLE fixture_item (id BIGINT PRIMARY KEY, value VARCHAR(40))"), SQLMigration(2, "seed fixture", "INSERT INTO fixture_item VALUES (1, 'preserved')")}
	if err := Verify(ctx, db, manifest); !errors.Is(err, ErrPreflight) {
		t.Fatalf("missing ledger must fail closed: %v", err)
	}
	if err := Migrate(ctx, db, manifest); err != nil {
		t.Fatal(err)
	}
	if err := Verify(ctx, db, manifest); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db, manifest); err != nil {
		t.Fatal("repeated migration must preserve data", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM fixture_item").Scan(&count); err != nil || count != 1 {
		t.Fatal("migration repeated seed")
	}
	edited := append([]Migration(nil), manifest...)
	edited[0].Checksum = Checksum([]byte("edited"))
	if err := Migrate(ctx, db, edited); !errors.Is(err, ErrDrift) {
		t.Fatalf("edited source not rejected: %v", err)
	}
	pending := append(append([]Migration(nil), manifest...), SQLMigration(3, "next", "ALTER TABLE fixture_item ADD COLUMN extra INT"))
	if err := Verify(ctx, db, pending); !errors.Is(err, ErrPending) {
		t.Fatalf("pending migration not rejected: %v", err)
	}
	// A held callback establishes a deterministic concurrency barrier. The second
	// migrator must fail immediately before executing any of its callbacks.
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	original := pending[2].Apply
	var calls atomic.Int32
	pending[2].Apply = func(ctx context.Context, conn *sql.Conn) error {
		calls.Add(1)
		close(entered)
		select {
		case <-release:
			return original(ctx, conn)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	go func() { done <- Migrate(ctx, db, pending) }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("first migrator did not reach barrier")
	}
	if err := Verify(ctx, db, pending); !errors.Is(err, ErrDirty) {
		t.Errorf("in-flight migration did not block verify: %v", err)
	}
	if err := Migrate(ctx, db, pending); !errors.Is(err, ErrLocked) {
		t.Errorf("concurrent migration not locked: %v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("migration callback ran more than once")
	}
	// Runtime credentials can verify with SELECT only. They cannot migrate.
	if _, err := db.ExecContext(ctx, "CREATE USER 'schema_runtime'@'%' IDENTIFIED BY 'fixture-only'"); err != nil {
		t.Fatal("fixture runtime role creation failed")
	}
	if _, err := db.ExecContext(ctx, "GRANT SELECT ON schema_fixture.cloudops_schema_history TO 'schema_runtime'@'%'"); err != nil {
		t.Fatal("fixture runtime role grant failed")
	}
	runtimeDB, err := sql.Open("mysql", "schema_runtime:fixture-only@tcp("+address+")/schema_fixture?timeout=2s&readTimeout=5s&writeTimeout=5s")
	if err != nil {
		t.Fatal("fixture runtime pool failed")
	}
	defer runtimeDB.Close()
	if err := Verify(ctx, runtimeDB, pending); err != nil {
		t.Fatal("SELECT-only runtime cannot verify", err)
	}
	if err := Migrate(ctx, runtimeDB, pending); err == nil {
		t.Fatal("runtime credentials unexpectedly migrated")
	}
	// MySQL's successful ALTER survives a later failing statement. The ledger
	// must remain dirty and prevent rerunning the partially committed migration.
	failure := append(append([]Migration(nil), pending...), SQLMigration(4, "partial ddl", "ALTER TABLE fixture_item ADD COLUMN partial INT", "INSERT INTO missing_table VALUES (1)"))
	if err := Migrate(ctx, db, failure); !errors.Is(err, ErrDirty) {
		t.Fatalf("partial DDL failure not dirty: %v", err)
	}
	var state string
	if err := db.QueryRowContext(ctx, "SELECT state FROM cloudops_schema_history WHERE version=4").Scan(&state); err != nil || state != "failed" {
		t.Fatal("failed ledger record missing")
	}
	if _, err := db.ExecContext(ctx, "SELECT partial FROM fixture_item"); err != nil {
		t.Fatal("test must demonstrate implicitly committed DDL")
	}
	if err := Migrate(ctx, db, failure); !errors.Is(err, ErrDirty) {
		t.Fatal("failed version was retried", err)
	}
	if err := Verify(ctx, db, failure); !errors.Is(err, ErrDirty) {
		t.Fatal("dirty state did not block startup", err)
	}
	if err := Verify(ctx, db, manifest); !errors.Is(err, ErrDrift) {
		t.Fatal("older binary accepted future history", err)
	}
}

func TestMySQLInterruptedMigration(t *testing.T) {
	db, _ := ownedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	migration := SQLMigration(1, "interrupted DDL", "CREATE TABLE interruption_fixture (id BIGINT)")
	original := migration.Apply
	migration.Apply = func(ctx context.Context, conn *sql.Conn) error {
		if err := original(ctx, conn); err != nil {
			return err
		}
		var connection uint64
		if err := conn.QueryRowContext(ctx, "SELECT CONNECTION_ID()").Scan(&connection); err != nil {
			return err
		}
		// Only kill the session obtained from this fixture's pinned migration
		// connection, never a service or a process discovered on the host.
		if _, err := db.ExecContext(ctx, fmt.Sprintf("KILL CONNECTION %d", connection)); err != nil {
			return err
		}
		_, err := conn.ExecContext(ctx, "SELECT 1")
		if err == nil {
			t.Error("killed connection unexpectedly still usable")
		}
		return err
	}
	manifest := []Migration{migration}
	if err := Migrate(ctx, db, manifest); !errors.Is(err, ErrUncertain) {
		t.Fatalf("lost DDL connection not uncertain: %v", err)
	}
	var state string
	if err := db.QueryRowContext(ctx, "SELECT state FROM cloudops_schema_history WHERE version=1").Scan(&state); err != nil || state != "started" {
		t.Fatal("interrupted migration intent was not preserved")
	}
	if _, err := db.ExecContext(ctx, "SELECT id FROM interruption_fixture"); err != nil {
		t.Fatal("implicitly committed table was not retained")
	}
	if err := Migrate(ctx, db, manifest); !errors.Is(err, ErrDirty) {
		t.Fatalf("interrupted version was not blocked: %v", err)
	}
	if err := Verify(ctx, db, manifest); !errors.Is(err, ErrDirty) {
		t.Fatal("interrupted version passed startup verification", err)
	}
}

func TestMySQLPanicAfterDDLLeavesDirtyHistory(t *testing.T) {
	db, _ := ownedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	migration := SQLMigration(1, "panic after DDL", "CREATE TABLE panic_fixture (id BIGINT)")
	original := migration.Apply
	migration.Apply = func(ctx context.Context, conn *sql.Conn) error {
		if err := original(ctx, conn); err != nil {
			return err
		}
		panic("private-ddl-panic-canary")
	}
	err := Migrate(ctx, db, []Migration{migration})
	if !errors.Is(err, ErrDirty) || strings.Contains(err.Error(), "private-ddl-panic-canary") {
		t.Fatal("panic did not become sanitized dirty outcome")
	}
	var state string
	if err := db.QueryRowContext(ctx, "SELECT state FROM cloudops_schema_history WHERE version=1").Scan(&state); err != nil || state != "failed" {
		t.Fatal("panic intent not durably marked failed")
	}
	if _, err := db.ExecContext(ctx, "SELECT id FROM panic_fixture"); err != nil {
		t.Fatal("test must demonstrate implicit DDL commit")
	}
	if err := Migrate(ctx, db, []Migration{migration}); !errors.Is(err, ErrDirty) {
		t.Fatal("panicked version was replayed")
	}
}

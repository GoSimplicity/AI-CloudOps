//go:build integration

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
)

// The CLI integration only talks to this labelled disposable MySQL. It cannot
// consume an operator DSN or connect to an existing database.
func cliMySQLFixture(t *testing.T) (*sql.DB, string) {
	t.Helper()
	run := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("docker", args...).CombinedOutput()
		if err != nil {
			t.Fatal("owned CLI fixture command failed; output withheld")
		}
		return strings.TrimSpace(string(out))
	}
	if endpoint := run("context", "inspect", "--format", "{{.Endpoints.docker.Host}}"); !strings.HasPrefix(endpoint, "unix://") {
		t.Fatal("CLI fixture requires local Unix-socket Docker")
	}
	var token [12]byte
	if _, err := rand.Read(token[:]); err != nil {
		t.Fatal(err)
	}
	name := "cloudops-schema-cli-" + hex.EncodeToString(token[:])
	run("run", "--detach", "--pull=never", "--name", name, "--label", "cloudops.schema.cli="+name, "-e", "MYSQL_ALLOW_EMPTY_PASSWORD=yes", "-e", "MYSQL_ROOT_HOST=%", "-e", "MYSQL_DATABASE=cli_fixture", "-p", "127.0.0.1::3306", "mysql:8.4.11")
	t.Cleanup(func() {
		if run("inspect", "--format", `{{index .Config.Labels "cloudops.schema.cli"}}`, name) != name {
			t.Error("ownership label differs; refusing cleanup")
			return
		}
		run("rm", "--force", "--volumes", name)
	})
	address := run("port", name, "3306/tcp")
	if !strings.HasPrefix(address, "127.0.0.1:") {
		t.Fatal("fixture port must be loopback")
	}
	dsn := "root@tcp(" + address + ")/cli_fixture?parseTime=true"
	db, err := sql.Open("mysql", dsn+"&timeout=2s&readTimeout=5s&writeTimeout=5s")
	if err != nil {
		t.Fatal("fixture pool failed")
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		// The image also runs a transient init server. Do not mistake it for the
		// final PID-1 mysqld process before testing socket and TCP readiness.
		comm, _ := exec.Command("docker", "exec", name, "cat", "/proc/1/comm").Output()
		if strings.TrimSpace(string(comm)) == "mysqld" && db.PingContext(ctx) == nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("owned CLI MySQL startup timed out")
		case <-ticker.C:
		}
	}
	return db, dsn
}

func TestMySQLCLIFreshBaselineAndLegacyRefusal(t *testing.T) {
	db, dsn := cliMySQLFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	invoke := func(want int, args ...string) string {
		t.Helper()
		var out, diagnostic bytes.Buffer
		status := run(ctx, args, &out, &diagnostic, dependencies{
			manifest: dbschema.Manifest,
			getenv: func(key string) string {
				if key != credentialEnvironment {
					t.Fatal("unexpected credential source")
				}
				return dsn
			},
			open: openDatabase,
		})
		if status != want {
			t.Fatalf("CLI status %d, expected %d; diagnostics: %s", status, want, diagnostic.String())
		}
		return out.String() + diagnostic.String()
	}
	countTables := func(want int) {
		t.Helper()
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()").Scan(&count); err != nil {
			t.Fatal("fixture count failed")
		}
		if count != want {
			t.Fatalf("unexpected table count: got %d want %d", count, want)
		}
	}
	invoke(0)
	invoke(1, "verify")
	invoke(2, "migrate")
	countTables(0)
	invoke(0, "migrate", "--confirm-new-database")
	countTables(59) // 58 business tables plus the version ledger.
	invoke(0, "verify")
	invoke(0, "migrate", "--confirm-new-database")
	var versions int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cloudops_schema_history WHERE state='applied'").Scan(&versions); err != nil || versions != 1 {
		t.Fatal("baseline must be applied exactly once")
	}

	// Runtime verification can succeed with SELECT on the ledger only. It never
	// asks for DDL, data repairs, lock acquisition or history mutation privileges.
	if _, err := db.ExecContext(ctx, "CREATE USER 'cli_reader'@'%' IDENTIFIED BY 'isolated-reader-only'"); err != nil {
		t.Fatal("fixture reader creation failed")
	}
	if _, err := db.ExecContext(ctx, "GRANT SELECT ON cli_fixture.cloudops_schema_history TO 'cli_reader'@'%'"); err != nil {
		t.Fatal("fixture reader grant failed")
	}
	rootDSN := dsn
	dsn = strings.Replace(dsn, "root@", "cli_reader:isolated-reader-only@", 1)
	invoke(0, "verify")
	dsn = rootDSN
	if _, err := db.ExecContext(ctx, "UPDATE cloudops_schema_history SET state='failed' WHERE version=1"); err != nil {
		t.Fatal("dirty fixture setup failed")
	}
	if text := invoke(1, "migrate", "--confirm-new-database"); !strings.Contains(text, "dirty") {
		t.Fatal("dirty history was not diagnosed")
	}
	var state string
	if err := db.QueryRowContext(ctx, "SELECT state FROM cloudops_schema_history WHERE version=1").Scan(&state); err != nil || state != "failed" {
		t.Fatal("CLI repaired dirty history without reconciliation")
	}

	if _, err := db.ExecContext(ctx, "CREATE DATABASE cli_legacy"); err != nil {
		t.Fatal("legacy fixture setup failed")
	}
	if _, err := db.ExecContext(ctx, "CREATE TABLE cli_legacy.sentinel (id BIGINT PRIMARY KEY)"); err != nil {
		t.Fatal("legacy fixture setup failed")
	}
	dsn = strings.Replace(rootDSN, "/cli_fixture?", "/cli_legacy?", 1)
	if text := invoke(1, "migrate", "--confirm-new-database"); !strings.Contains(text, "legacy adoption") {
		t.Fatal("legacy database was not diagnosed")
	}
	var legacyTables int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='cli_legacy'").Scan(&legacyTables); err != nil || legacyTables != 1 {
		t.Fatal("refusing legacy migration still changed schema")
	}
	t.Log("fresh CLI baseline, read-only verification, dirty refusal and unchanged legacy schema verified")
}

func TestReadCredentialsRequiresPrivateRegularFile(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "private-dsn")
	if err := os.WriteFile(good, []byte("fixture-user:fixture-pass@tcp(127.0.0.1:9)/fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readCredentials(good, func(string) string { t.Fatal("explicit file fell back to environment"); return "" })
	if err != nil || !strings.HasSuffix(got, "/fixture") {
		t.Fatal("private file rejected")
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(good, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readCredentials(link, nil); err == nil {
		t.Fatal("symlink accepted")
	}
	if _, err := readCredentials(dir, nil); err == nil {
		t.Fatal("directory accepted")
	}
	if err := os.Chmod(good, 0640); err != nil {
		t.Fatal(err)
	}
	if _, err := readCredentials(good, nil); err == nil {
		t.Fatal("group-readable credential accepted")
	}
	if err := os.Chmod(good, 0600); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"", "one\ntwo", "one\x00two", strings.Repeat("x", 64*1024+1)} {
		if err := os.WriteFile(good, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := readCredentials(good, nil); err == nil {
			t.Fatal("invalid credential accepted")
		}
	}
}

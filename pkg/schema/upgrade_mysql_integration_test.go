//go:build integration

package schema

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

// This fixture pins the local socket on every invocation. It refuses remote or
// context/TLS overrides rather than letting a safe context inspect be followed
// by a run command redirected through ambient DOCKER_HOST.
func ownedUpgradeMySQL(t *testing.T) (*sql.DB, string) {
	t.Helper()
	for _, key := range []string{"DOCKER_HOST", "DOCKER_CONTEXT", "DOCKER_TLS_VERIFY", "DOCKER_CERT_PATH"} {
		if os.Getenv(key) != "" {
			t.Fatal("upgrade fixture refuses Docker endpoint overrides")
		}
	}
	environment := []string{}
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "DOCKER_") {
			environment = append(environment, entry)
		}
	}
	run := func(args ...string) (string, error) {
		command := exec.Command("docker", append([]string{"--host", "unix:///var/run/docker.sock"}, args...)...)
		command.Env = environment
		output, err := command.CombinedOutput()
		return strings.TrimSpace(string(output)), err
	}
	var token [12]byte
	if _, err := rand.Read(token[:]); err != nil {
		t.Fatal(err)
	}
	name := "cloudops-upgrade-" + hex.EncodeToString(token[:])
	if _, err := run("run", "--detach", "--name", name, "--label", "cloudops.schema.upgrade.fixture="+name, "-e", "MYSQL_ALLOW_EMPTY_PASSWORD=yes", "-e", "MYSQL_ROOT_HOST=%", "-e", "MYSQL_DATABASE=upgrade_admin", "-p", "127.0.0.1::3306", "mysql:8.4.11"); err != nil {
		t.Fatal("owned upgrade fixture start failed; output withheld")
	}
	t.Cleanup(func() {
		owner, err := run("inspect", "--format", `{{index .Config.Labels "cloudops.schema.upgrade.fixture"}}`, name)
		if err != nil || owner != name {
			t.Error("refusing fixture cleanup: ownership differs")
			return
		}
		if _, err := run("rm", "--force", "--volumes", name); err != nil {
			t.Error("owned upgrade fixture cleanup failed")
		}
	})
	owner, err := run("inspect", "--format", `{{index .Config.Labels "cloudops.schema.upgrade.fixture"}}`, name)
	if err != nil || owner != name {
		t.Fatal("upgrade fixture ownership differs")
	}
	address, err := run("port", name, "3306/tcp")
	if err != nil || !strings.HasPrefix(address, "127.0.0.1:") {
		t.Fatal("upgrade fixture must publish only loopback")
	}
	admin := upgradeFixtureConnection(t, address, "upgrade_admin", "root", "")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := run("exec", name, "mysqladmin", "ping", "--host=127.0.0.1", "--silent"); err == nil && admin.PingContext(ctx) == nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("owned upgrade MySQL startup timed out")
		case <-ticker.C:
		}
	}
	return admin, address
}
func upgradeFixtureConnection(t *testing.T, address, database, user, password string) *sql.DB {
	t.Helper()
	if !strings.HasPrefix(address, "127.0.0.1:") {
		t.Fatal("nonlocal upgrade fixture address")
	}
	config := mysql.NewConfig()
	config.User, config.Passwd, config.Net, config.Addr, config.DBName = user, password, "tcp", address, database
	config.ParseTime = true
	config.Timeout = 3 * time.Second
	config.ReadTimeout = 10 * time.Second
	config.WriteTimeout = 10 * time.Second
	config.Logger = &mysql.NopLogger{}
	connector, err := mysql.NewConnector(config)
	if err != nil {
		t.Fatal("upgrade fixture connector failed")
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestMySQLUpgradeStateMachine(t *testing.T) {
	admin, address := ownedUpgradeMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	newSource := func(t *testing.T, name string) *sql.DB {
		t.Helper()
		if _, err := admin.ExecContext(ctx, "CREATE DATABASE `"+name+"`"); err != nil {
			t.Fatal("owned source database setup failed")
		}
		db := upgradeFixtureConnection(t, address, name, "root", "")
		for _, statement := range []string{"CREATE TABLE upgrade_fixture_item(id BIGINT PRIMARY KEY,value VARCHAR(100) NOT NULL)", "INSERT INTO upgrade_fixture_item VALUES(1,'preserved')"} {
			if _, err := db.ExecContext(ctx, statement); err != nil {
				t.Fatal("owned source setup failed")
			}
		}
		return db
	}
	manifest := validManifest()
	manifest[0].Apply = func(context.Context, *sql.Conn) error { t.Fatal("upgrade called fresh baseline Apply"); return nil }
	evidence := upgradeFixtureEvidence()
	evidence.NotificationZone = "Asia/Shanghai"
	planFor := func(t *testing.T) BaselineUpgrade {
		t.Helper()
		plan := upgradeFixturePlan()
		plan.ValidateSource = func(ctx context.Context, conn *sql.Conn, _ UpgradeEvidence) error {
			var database string
			var connectionID int64
			if err := conn.QueryRowContext(ctx, "SELECT LOWER(DATABASE()), CONNECTION_ID()").Scan(&database, &connectionID); err != nil {
				return err
			}
			lockName := "cloudops.schema." + Checksum([]byte(database))[:40]
			var owner sql.NullInt64
			if err := conn.QueryRowContext(ctx, "SELECT IS_USED_LOCK(?)", lockName).Scan(&owner); err != nil || !owner.Valid || owner.Int64 != connectionID {
				return errors.New("private-lock-canary")
			}
			var rows int
			if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM upgrade_fixture_item WHERE value='preserved'").Scan(&rows); err != nil || rows != 1 {
				return errors.New("private-source-canary")
			}
			return nil
		}
		plan.Apply = func(ctx context.Context, conn *sql.Conn, e UpgradeEvidence) error {
			for _, statement := range []string{"ALTER TABLE upgrade_fixture_item ADD COLUMN upgraded INT NOT NULL DEFAULT 1", "CREATE TABLE upgrade_fixture_clock(id INT PRIMARY KEY,time_zone VARCHAR(100) NOT NULL)"} {
				if _, err := conn.ExecContext(ctx, statement); err != nil {
					return err
				}
			}
			_, err := conn.ExecContext(ctx, "INSERT INTO upgrade_fixture_clock VALUES(1,?)", e.NotificationZone)
			return err
		}
		plan.ValidateTarget = func(ctx context.Context, conn *sql.Conn, e UpgradeEvidence) error {
			var state, origin, source, sourceHash, implementationHash, actor, currentActor, note, backup, zone, checksum string
			var attested bool
			if err := conn.QueryRowContext(ctx, `SELECT state,origin,source_revision,source_checksum,upgrade_checksum,upgrade_actor,CURRENT_USER(),upgrade_note,backup_sha256,notification_zone,historical_time_attested,checksum FROM cloudops_schema_history WHERE version=1`).Scan(&state, &origin, &source, &sourceHash, &implementationHash, &actor, &currentActor, &note, &backup, &zone, &attested, &checksum); err != nil {
				return err
			}
			if state != "started" || origin != "upgraded" || source != plan.Source || sourceHash != plan.SourceChecksum || implementationHash != plan.ImplementationChecksum || actor != currentActor || note != e.Note || backup != e.BackupSHA256 || zone != e.NotificationZone || attested != e.HistoricalTimeAttested || checksum != manifest[0].Checksum {
				return errors.New("private-target-canary")
			}
			var saved string
			if err := conn.QueryRowContext(ctx, "SELECT time_zone FROM upgrade_fixture_clock WHERE id=1").Scan(&saved); err != nil || saved != e.NotificationZone {
				return errors.New("private-zone-canary")
			}
			return nil
		}
		return plan
	}
	stateOf := func(t *testing.T, db *sql.DB) string {
		t.Helper()
		var state string
		if err := db.QueryRowContext(ctx, "SELECT state FROM cloudops_schema_history WHERE version=1").Scan(&state); err != nil {
			t.Fatal("owned upgrade history query failed")
		}
		return state
	}
	t.Run("upgrade repeat and future migration", func(t *testing.T) {
		db := newSource(t, "upgrade_success")
		plan := planFor(t)
		if err := UpgradeBaseline(ctx, db, manifest, plan, evidence); err != nil {
			t.Fatal(err)
		}
		if err := Verify(ctx, db, manifest); err != nil {
			t.Fatal(err)
		}
		changed := evidence
		changed.Note = "different review note"
		never := func(context.Context, *sql.Conn, UpgradeEvidence) error {
			t.Fatal("repeat reran upgrade callback")
			return nil
		}
		plan.ValidateSource, plan.Apply, plan.ValidateTarget = never, never, never
		if err := UpgradeBaseline(ctx, db, manifest, plan, changed); err != nil {
			t.Fatal(err)
		}
		var note string
		if err := db.QueryRowContext(ctx, "SELECT upgrade_note FROM cloudops_schema_history WHERE version=1").Scan(&note); err != nil || note != evidence.Note {
			t.Fatal("repeat changed original upgrade evidence")
		}
		future := append(append([]Migration(nil), manifest...), SQLMigration(2, "next", "ALTER TABLE upgrade_fixture_item ADD COLUMN future INT NULL"))
		if err := Migrate(ctx, db, future); err != nil {
			t.Fatal("derived upgrade ledger blocked a future regular migration", err)
		}
		var origin string
		var absent bool
		if err := db.QueryRowContext(ctx, "SELECT origin,source_revision IS NULL AND upgrade_checksum IS NULL AND upgrade_note IS NULL FROM cloudops_schema_history WHERE version=2").Scan(&origin, &absent); err != nil || origin != "migrated" || !absent {
			t.Fatal("future migration forged upgrade origin/evidence")
		}
	})
	for _, test := range []struct {
		name, phase string
		panicValue  bool
	}{{"source_error", "source", false}, {"source_panic", "source", true}, {"partial_apply", "apply", false}, {"target_error", "target", false}, {"target_panic", "target", true}, {"apply_panic_dml_rollback", "apply", true}} {
		t.Run(test.name, func(t *testing.T) {
			db := newSource(t, "upgrade_"+test.name)
			plan := planFor(t)
			failing := func(context.Context, *sql.Conn, UpgradeEvidence) error {
				if test.panicValue {
					panic("private-panic-canary")
				}
				return errors.New("private-callback-canary")
			}
			switch test.phase {
			case "source":
				plan.ValidateSource = failing
			case "target":
				plan.ValidateTarget = failing
			case "apply":
				plan.Apply = func(ctx context.Context, conn *sql.Conn, e UpgradeEvidence) error {
					if _, err := conn.ExecContext(ctx, "ALTER TABLE upgrade_fixture_item ADD COLUMN committed_before_failure INT"); err != nil {
						return err
					}
					if test.panicValue {
						tx, err := conn.BeginTx(ctx, nil)
						if err != nil {
							return err
						}
						defer tx.Rollback()
						if _, err := tx.ExecContext(ctx, "UPDATE upgrade_fixture_item SET value='must-rollback'"); err != nil {
							return err
						}
						panic("private-panic-canary")
					}
					return errors.New("private-callback-canary")
				}
			}
			err := UpgradeBaseline(ctx, db, manifest, plan, evidence)
			expected := ErrDirty
			if test.phase == "source" {
				expected = ErrPreflight
			}
			if !errors.Is(err, expected) || strings.Contains(err.Error(), "private") {
				t.Fatal("upgrade callback failure was not sanitized and blocked")
			}
			if test.phase == "source" {
				var exists bool
				if err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='cloudops_schema_history')").Scan(&exists); err != nil || exists {
					t.Fatal("source failure created a ledger")
				}
				return
			}
			if stateOf(t, db) != "failed" {
				t.Fatal("failed upgrade was not durably blocked")
			}
			if test.phase == "apply" {
				if _, err := db.ExecContext(ctx, "SELECT committed_before_failure FROM upgrade_fixture_item"); err != nil {
					t.Fatal("test did not prove partially committed DDL")
				}
				var value string
				if err := db.QueryRowContext(ctx, "SELECT value FROM upgrade_fixture_item WHERE id=1").Scan(&value); err != nil || value != "preserved" {
					t.Fatal("deferred DML rollback did not preserve original row")
				}
			}
			plan = planFor(t)
			if err := UpgradeBaseline(ctx, db, manifest, plan, evidence); !errors.Is(err, ErrDirty) {
				t.Fatal("failed upgrade retried", err)
			}
		})
	}
	t.Run("concurrent upgrade is locked", func(t *testing.T) {
		db := newSource(t, "upgrade_concurrent")
		db.SetMaxOpenConns(4)
		plan := planFor(t)
		original := plan.Apply
		entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
		plan.Apply = func(ctx context.Context, conn *sql.Conn, e UpgradeEvidence) error {
			close(entered)
			select {
			case <-release:
				return original(ctx, conn, e)
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		go func() { done <- UpgradeBaseline(ctx, db, manifest, plan, evidence) }()
		select {
		case <-entered:
		case <-ctx.Done():
			t.Fatal("upgrade did not reach lock barrier")
		}
		second := UpgradeBaseline(ctx, db, manifest, plan, evidence)
		close(release)
		if !errors.Is(second, ErrLocked) {
			t.Fatal("concurrent upgrade did not fail before applying", second)
		}
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	})
	t.Run("read-only role cannot upgrade but can verify applied", func(t *testing.T) {
		db := newSource(t, "upgrade_readonly")
		for _, statement := range []string{"CREATE USER 'upgrade_reader'@'%' IDENTIFIED BY 'fixture-only'", "GRANT SELECT ON upgrade_readonly.* TO 'upgrade_reader'@'%'"} {
			if _, err := admin.ExecContext(ctx, statement); err != nil {
				t.Fatal("owned read-only role setup failed")
			}
		}
		reader := upgradeFixtureConnection(t, address, "upgrade_readonly", "upgrade_reader", "fixture-only")
		plan := planFor(t)
		if err := UpgradeBaseline(ctx, reader, manifest, plan, evidence); !errors.Is(err, ErrUncertain) {
			t.Fatal("read-only role unexpectedly upgraded", err)
		}
		if err := UpgradeBaseline(ctx, db, manifest, plan, evidence); err != nil {
			t.Fatal(err)
		}
		if err := UpgradeBaseline(ctx, reader, manifest, plan, evidence); err != nil {
			t.Fatal("read-only complete verification failed", err)
		}
	})
}

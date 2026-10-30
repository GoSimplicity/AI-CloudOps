//go:build integration

package dbschema_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
	"github.com/GoSimplicity/AI-CloudOps/pkg/di"
	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// The schema generator has no DSN input. It can only read its own disposable
// MySQL container, which is identified by a random ownership label before use.
func fixture(t *testing.T) (*sql.DB, string) {
	t.Helper()
	run := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("docker", args...).CombinedOutput()
		if err != nil {
			t.Fatal("owned baseline fixture command failed; output withheld")
		}
		return strings.TrimSpace(string(out))
	}
	if endpoint := run("context", "inspect", "--format", "{{.Endpoints.docker.Host}}"); !strings.HasPrefix(endpoint, "unix://") {
		t.Fatal("baseline fixture requires local Docker")
	}
	var token [12]byte
	if _, err := rand.Read(token[:]); err != nil {
		t.Fatal(err)
	}
	name := "cloudops-baseline-" + hex.EncodeToString(token[:])
	run("run", "--detach", "--name", name, "--label", "cloudops.baseline.fixture="+name, "-e", "MYSQL_ALLOW_EMPTY_PASSWORD=yes", "-e", "MYSQL_ROOT_HOST=%", "-e", "MYSQL_DATABASE=baseline_source", "-p", "127.0.0.1::3306", "mysql:8.4.11")
	t.Cleanup(func() {
		if run("inspect", "--format", `{{index .Config.Labels "cloudops.baseline.fixture"}}`, name) != name {
			t.Error("refusing cleanup: ownership label differs")
			return
		}
		run("rm", "--force", "--volumes", name)
	})
	if run("inspect", "--format", `{{index .Config.Labels "cloudops.baseline.fixture"}}`, name) != name {
		t.Fatal("fixture ownership label differs")
	}
	address := run("port", name, "3306/tcp")
	if !strings.HasPrefix(address, "127.0.0.1:") {
		t.Fatal("fixture must bind loopback")
	}
	db, err := sql.Open("mysql", dsn(address, "baseline_source"))
	if err != nil {
		t.Fatal("fixture pool failed")
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	for {
		if exec.Command("docker", "exec", name, "mysqladmin", "ping", "--host=127.0.0.1", "--silent").Run() == nil && db.PingContext(ctx) == nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("fixture startup timed out")
		case <-ticker.C:
		}
	}
	return db, address
}

func dsn(address, database string) string {
	return "root@tcp(" + address + ")/" + database + "?parseTime=true&timeout=3s&readTimeout=15s&writeTimeout=15s"
}

type repairCapture struct {
	logger.Interface
	statements []string
}

func (c *repairCapture) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	statement, _ := fc()
	if strings.HasPrefix(statement, "UPDATE ") {
		c.statements = append(c.statements, statement)
	}
}

func initializeCurrent(t *testing.T, db *sql.DB) []string {
	t.Helper()
	capture := &repairCapture{Interface: logger.Discard}
	orm, err := gorm.Open(mysql.New(mysql.Config{Conn: db}), &gorm.Config{Logger: capture})
	if err != nil {
		t.Fatal("source ORM initialization failed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := di.InitTables(orm.WithContext(ctx)); err != nil {
		t.Fatal("current registered schema initialization failed")
	}
	if len(capture.statements) != 2 {
		t.Fatalf("expected both legacy data repairs, captured %d", len(capture.statements))
	}
	return capture.statements
}

func tableDDL(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name <> 'cloudops_schema_history' ORDER BY table_name")
	if err != nil {
		t.Fatal("cannot read owned fixture tables")
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	result := make(map[string]string, len(names))
	for _, name := range names {
		var returned, ddl string
		if err := db.QueryRowContext(ctx, "SHOW CREATE TABLE `"+strings.ReplaceAll(name, "`", "``")+"`").Scan(&returned, &ddl); err != nil {
			t.Fatal("cannot read owned fixture DDL")
		}
		result[name] = normalizeDDL(ddl)
	}
	return result
}

func snapshot(t *testing.T, db *sql.DB, repairs []string) []string {
	t.Helper()
	definitions := tableDDL(t, db)
	// Table order comes from SHOW's lexicographically sorted inventory, never a
	// copied model list. All foreign keys are added after all tables exist.
	rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema=DATABASE() ORDER BY table_name")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	var creates, foreignKeys []string
	for _, name := range names {
		lines := strings.Split(definitions[name], "\n")
		var kept []string
		for _, line := range lines {
			if strings.Contains(line, "CONSTRAINT ") && strings.Contains(line, " FOREIGN KEY ") {
				foreignKeys = append(foreignKeys, "ALTER TABLE `"+strings.ReplaceAll(name, "`", "``")+"` ADD "+strings.TrimSuffix(strings.TrimSpace(line), ","))
				continue
			}
			kept = append(kept, line)
		}
		for i := len(kept) - 2; i >= 0; i-- {
			if strings.TrimSpace(kept[i]) != "" {
				kept[i] = strings.TrimSuffix(kept[i], ",")
				break
			}
		}
		creates = append(creates, strings.Join(kept, "\n"))
	}
	statements := append(creates, foreignKeys...)
	return append(statements, repairs...)
}

// Generation is deliberately explicit and is for creating version 1 once. Once
// released, never regenerate version 1 to accommodate changed live models: add a
// new immutable version and let TestCurrentSchemaMatchesManifest detect drift.
func TestGenerateBaselineSnapshot(t *testing.T) {
	if os.Getenv("CLOUDOPS_GENERATE_BASELINE_V1") != "1" {
		t.Skip("explicit initial baseline generation only")
	}
	previous, err := os.ReadFile("baseline_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var existing []string
	if json.Unmarshal(previous, &existing) != nil || len(existing) != 0 {
		t.Fatal("refusing to overwrite immutable baseline; add a new migration version")
	}
	db, _ := fixture(t)
	repairs := initializeCurrent(t, db)
	statements := snapshot(t, db, repairs)
	encoded, err := json.MarshalIndent(statements, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("baseline_v1.json", append(encoded, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("generated %d statements from %d actual MySQL tables including implicit join tables", len(statements), len(tableDDL(t, db)))
}

func TestCurrentSchemaMatchesManifest(t *testing.T) {
	source, address := fixture(t)
	initializeCurrent(t, source)
	manifest, err := dbschema.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if _, err := source.ExecContext(ctx, "CREATE DATABASE baseline_target CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal("target fixture database creation failed")
	}
	target, err := sql.Open("mysql", dsn(address, "baseline_target"))
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	if err := schema.Migrate(ctx, target, manifest); err != nil {
		t.Fatal(err)
	}
	if err := schema.Verify(ctx, target, manifest); err != nil {
		t.Fatal(err)
	}
	actual, want := tableDDL(t, target), tableDDL(t, source)
	if !reflect.DeepEqual(actual, want) {
		for name, expected := range want {
			if actual[name] != expected {
				t.Errorf("table %s differs in columns, indexes or foreign keys; add a new migration instead of editing baseline", name)
			}
		}
		if len(actual) != len(want) {
			t.Errorf("table inventory differs: snapshot=%d current=%d", len(actual), len(want))
		}
	}
	if err := schema.Migrate(ctx, target, manifest); err != nil {
		t.Fatal("applied baseline was not idempotent", err)
	}
	verifyFrozenRepairs(t, source, target)
	t.Logf("compared %d complete SHOW CREATE definitions including implicit join tables", len(want))
}

func TestFreshPreflightRejectsLegacy(t *testing.T) {
	db, _ := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := dbschema.PreflightFresh(ctx, conn); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, "CREATE TABLE unrelated_legacy (id BIGINT)"); err != nil {
		t.Fatal(err)
	}
	if err := dbschema.PreflightFresh(ctx, conn); !errors.Is(err, dbschema.ErrLegacy) {
		t.Fatal("legacy table was not rejected", err)
	}
	var count int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()").Scan(&count); err != nil || count != 1 {
		t.Fatal("read-only preflight modified fixture")
	}
	manifest, err := dbschema.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Migrate(ctx, db, manifest); !errors.Is(err, schema.ErrDirty) {
		t.Fatal("manifest did not block legacy database", err)
	}
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM unrelated_legacy").Scan(&count); err != nil || count != 0 {
		t.Fatal("legacy table was changed")
	}
}

// Seed only fake values into both owned databases. Compare the frozen repair SQL
// against the actual legacy initializer, including deleted rows and timestamps.
func verifyFrozenRepairs(t *testing.T, source, target *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	seeds := []string{
		"INSERT INTO cl_tree_cloud_account (id,name,access_key,secret_key,create_user_name) VALUES (1,'fixture','','','fixture')",
		"INSERT INTO cl_tree_cloud_account_region (id,cloud_account_id,region) VALUES (1,1,'fixture-region')",
		"INSERT INTO cl_tree_cloud_resource (id,name,cloud_account_id,cloud_account_region_id) VALUES (1,'fixture-resource',1,1)",
		"INSERT INTO cl_tree_cloud_resource_change_log (resource_id,field_name,old_value,new_value,change_time,updated_at,deleted_at) VALUES (1,' KEY ','fake-old','fake-new','2020-01-02 03:04:05','2020-01-02 03:04:05',1),(1,'password','fake-old','fake-new','2020-01-02 03:04:05','2020-01-02 03:04:05',0),(1,'environment','before','after','2020-01-02 03:04:05','2020-01-02 03:04:05',0)",
		"INSERT INTO cl_workorder_notification (name,event_types,channels,recipient_types,message_template,operator_id,is_default,updated_at,deleted_at) VALUES ('zero','[]','[]','[]','fixture',1,0,'2020-01-02 03:04:05',1),('yes','[]','[]','[]','fixture',1,1,'2020-01-02 03:04:05',0),('no','[]','[]','[]','fixture',1,2,'2020-01-02 03:04:05',0)",
	}
	for _, db := range []*sql.DB{source, target} {
		for _, statement := range seeds {
			if _, err := db.ExecContext(ctx, statement); err != nil {
				t.Fatal("fake repair fixture insert failed")
			}
		}
	}
	currentRepairs := initializeCurrent(t, source)
	raw, err := os.ReadFile("baseline_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot []string
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatal(err)
	}
	frozenRepairs := snapshot[len(snapshot)-2:]
	if !reflect.DeepEqual(currentRepairs, frozenRepairs) {
		t.Fatal("current data-repair implementation differs from frozen baseline; add a new migration version")
	}
	for _, statement := range frozenRepairs {
		if _, err := target.ExecContext(ctx, statement); err != nil {
			t.Fatal("frozen data repair failed")
		}
	}
	for _, db := range []*sql.DB{source, target} {
		var redacted, ordinary, normalized, unchangedYes, preservedTimes int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cl_tree_cloud_resource_change_log WHERE field_name IN (' KEY ','password') AND old_value='[REDACTED]' AND new_value='[REDACTED]'").Scan(&redacted); err != nil || redacted != 2 {
			t.Fatal("credential audit repair differs, including soft-deleted audit")
		}
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cl_tree_cloud_resource_change_log WHERE field_name='environment' AND old_value='before' AND new_value='after'").Scan(&ordinary); err != nil || ordinary != 1 {
			t.Fatal("ordinary audit was changed")
		}
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cl_workorder_notification WHERE name IN ('zero','no') AND is_default=2").Scan(&normalized); err != nil || normalized != 2 {
			t.Fatal("invalid default repair differs")
		}
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cl_workorder_notification WHERE name='yes' AND is_default=1").Scan(&unchangedYes); err != nil || unchangedYes != 1 {
			t.Fatal("valid default was changed")
		}
		if err := db.QueryRowContext(ctx, "SELECT (SELECT COUNT(*) FROM cl_tree_cloud_resource_change_log WHERE updated_at='2020-01-02 03:04:05' AND change_time='2020-01-02 03:04:05')+(SELECT COUNT(*) FROM cl_workorder_notification WHERE updated_at='2020-01-02 03:04:05')").Scan(&preservedTimes); err != nil || preservedTimes != 6 {
			t.Fatal("data repair changed historical timestamps")
		}
	}
}

func normalizeDDL(ddl string) string {
	lines := strings.Split(ddl, "\n")
	var columns, keys []string
	for _, line := range lines[1 : len(lines)-1] {
		line = strings.TrimSuffix(line, ",")
		if strings.HasPrefix(strings.TrimSpace(line), "`") {
			columns = append(columns, line)
		} else {
			keys = append(keys, line)
		}
	}
	sort.Strings(keys)
	all := append(columns, keys...)
	return lines[0] + "\n" + strings.Join(all, ",\n") + "\n" + lines[len(lines)-1]
}

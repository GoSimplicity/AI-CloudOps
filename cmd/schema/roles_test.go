package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
)

func TestRolePlanHasOnlyReviewedTableGrants(t *testing.T) {
	plan, err := BuildRolePlan("cloudopsTest", RoleAccount{Name: "runtime_role", Host: "localhost"}, RoleAccount{Name: "migrator_role", Host: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Runtime) != 59 || len(plan.Migrator) != 60 {
		t.Fatalf("unexpected grant count: %d / %d", len(plan.Runtime), len(plan.Migrator))
	}
	for _, statement := range append(append([]string{}, plan.Runtime...), plan.Migrator...) {
		if !strings.HasPrefix(statement, "GRANT ") || (strings.Contains(statement, "*") && !strings.HasPrefix(statement, "GRANT REFERENCES, TRIGGER ON `cloudopsTest`.* TO ")) || strings.Contains(statement, "PASSWORD") || strings.Contains(statement, "IDENTIFIED") || strings.Contains(statement, "GRANT OPTION") || strings.Contains(statement, "DROP") || !strings.HasSuffix(statement, ";") {
			t.Fatal("plan contains a broad or unsupported statement")
		}
	}
	for _, statement := range plan.Runtime {
		protected := strings.Contains(statement, "`.`"+schemaHistoryTable+"`") || strings.Contains(statement, "`.`"+notificationClockTable+"`")
		wantPrefix := "GRANT SELECT, INSERT, UPDATE, DELETE ON "
		if protected {
			wantPrefix = "GRANT SELECT ON "
		}
		if !strings.HasPrefix(statement, wantPrefix) || !strings.HasSuffix(statement, "TO 'runtime_role'@'localhost';") {
			t.Fatal("runtime received incorrect privileges")
		}
	}
	for _, statement := range plan.Migrator {
		if !strings.HasSuffix(statement, "TO 'migrator_role'@'127.0.0.1';") {
			t.Fatal("wrong migration role")
		}
		if strings.Contains(strings.Split(statement, " ON ")[0], "UPDATE") && !strings.Contains(statement, "`.`"+schemaHistoryTable+"`") && !strings.Contains(statement, "`.`"+notificationClockTable+"`") && !strings.Contains(statement, "`.`cl_tree_cloud_resource_change_log`") && !strings.Contains(statement, "`.`cl_workorder_notification`") {
			t.Fatal("unexpected business repair privilege")
		}
	}
}

func TestRolePlanIdentifiersFailClosedWithoutEcho(t *testing.T) {
	goodRuntime, goodMigrator := RoleAccount{Name: "runtime", Host: "localhost"}, RoleAccount{Name: "migrator", Host: "%"}
	tests := []struct {
		name, database    string
		runtime, migrator RoleAccount
	}{
		{"empty database", "", goodRuntime, goodMigrator},
		{"database injection", "db`;GRANT ALL ON *.* TO x", goodRuntime, goodMigrator},
		{"wildcard database", "cloud%ops", goodRuntime, goodMigrator},
		{"underscore database", "cloud_ops", goodRuntime, goodMigrator},
		{"oversized database", strings.Repeat("d", 65), goodRuntime, goodMigrator},
		{"system database", "mysql", goodRuntime, goodMigrator},
		{"case system database", "MySQL", goodRuntime, goodMigrator},
		{"system metadata", "information_schema", goodRuntime, goodMigrator},
		{"system performance", "performance_schema", goodRuntime, goodMigrator},
		{"system views", "sys", goodRuntime, goodMigrator},
		{"missing name", "cloudops", RoleAccount{Host: "localhost"}, goodMigrator},
		{"missing host", "cloudops", RoleAccount{Name: "runtime"}, goodMigrator},
		{"account injection", "cloudops", RoleAccount{Name: "runtime'", Host: "localhost"}, goodMigrator},
		{"host injection", "cloudops", RoleAccount{Name: "runtime", Host: "local'host"}, goodMigrator},
		{"oversized role", "cloudops", RoleAccount{Name: strings.Repeat("x", 33), Host: "localhost"}, goodMigrator},
		{"identical roles", "cloudops", goodRuntime, goodRuntime},
		{"host case same role", "cloudops", goodRuntime, RoleAccount{Name: "runtime", Host: "LOCALHOST"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := BuildRolePlan(tc.database, tc.runtime, tc.migrator)
			if !errors.Is(err, errRolePlan) || len(plan.Runtime)+len(plan.Migrator) != 0 {
				t.Fatal("invalid identifier produced a plan")
			}
			if err.Error() != errRolePlan.Error() {
				t.Fatal("validation error echoed input")
			}
		})
	}
}

func TestRolePlansDoNotShareMutableState(t *testing.T) {
	roles := []RoleAccount{{Name: "runtime", Host: "%"}, {Name: "migrator", Host: "%"}}
	plan, err := BuildRolePlan("cloudops", roles[0], roles[1])
	if err != nil {
		t.Fatal(err)
	}
	want := plan.Runtime[0]
	plan.Runtime[0] = "mutated"
	second, err := BuildRolePlan("cloudops", roles[0], roles[1])
	if err != nil {
		t.Fatal(err)
	}
	if second.Runtime[0] != want {
		t.Fatal("caller mutation escaped into later plan")
	}
}

// Execute the embedded immutable migration against an in-memory SQL recorder.
// This proves exact inventory without real DB/filesystem I/O or production SQL
// parsing. The fake only supports the two frozen preflight SELECT statements.
func TestRolePlanInventoryExactlyMatchesFrozenBaseline(t *testing.T) {
	capture := &roleSQLCapture{}
	db := sql.OpenDB(roleCaptureConnector{capture})
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	manifest, err := dbschema.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	if err := manifest[0].Apply(context.Background(), conn); err != nil {
		t.Fatal("frozen baseline capture failed")
	}
	create := regexp.MustCompile("^CREATE TABLE `([a-z0-9_]+)` ")
	repair := regexp.MustCompile("^UPDATE `([a-z0-9_]+)` ")
	var tables, updates []string
	for _, statement := range capture.statements {
		if strings.HasPrefix(statement, "CREATE TABLE ") {
			match := create.FindStringSubmatch(statement)
			if len(match) != 2 {
				t.Fatal("unrecognized frozen CREATE TABLE statement")
			}
			tables = append(tables, match[1])
		}
		if strings.HasPrefix(statement, "UPDATE ") {
			match := repair.FindStringSubmatch(statement)
			if len(match) != 2 {
				t.Fatal("unrecognized frozen repair statement")
			}
			updates = append(updates, match[1])
		}
	}
	want := append([]string{}, baselineBusinessTables[:]...)
	sort.Strings(tables)
	sort.Strings(want)
	sort.Strings(updates)
	if len(tables) != 58 || !reflect.DeepEqual(tables, want) {
		t.Fatalf("fixed grant inventory differs from baseline: got %v", tables)
	}
	if !reflect.DeepEqual(updates, []string{"cl_tree_cloud_resource_change_log", "cl_workorder_notification"}) {
		t.Fatal("repair UPDATE grant inventory differs from baseline")
	}
}

type roleSQLCapture struct{ statements []string }
type roleCaptureConnector struct{ capture *roleSQLCapture }

func (c roleCaptureConnector) Connect(context.Context) (driver.Conn, error) {
	return roleCaptureConnection{c.capture}, nil
}
func (c roleCaptureConnector) Driver() driver.Driver { return roleCaptureDriver{} }

type roleCaptureDriver struct{}

func (roleCaptureDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("unexpected non-connector open")
}

type roleCaptureConnection struct{ capture *roleSQLCapture }

func (roleCaptureConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (roleCaptureConnection) Close() error { return nil }
func (roleCaptureConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("unexpected transaction")
}
func (c roleCaptureConnection) ExecContext(_ context.Context, statement string, _ []driver.NamedValue) (driver.Result, error) {
	c.capture.statements = append(c.capture.statements, statement)
	return driver.RowsAffected(0), nil
}
func (roleCaptureConnection) QueryContext(_ context.Context, statement string, _ []driver.NamedValue) (driver.Rows, error) {
	switch statement {
	case "SELECT DATABASE()":
		return &roleCaptureRows{value: "fixture"}, nil
	case "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name <> 'cloudops_schema_history'":
		return &roleCaptureRows{value: int64(0)}, nil
	default:
		return nil, fmt.Errorf("unexpected preflight SELECT")
	}
}

type roleCaptureRows struct {
	value driver.Value
	read  bool
}

func (*roleCaptureRows) Columns() []string { return []string{"fixture_value"} }
func (*roleCaptureRows) Close() error      { return nil }
func (r *roleCaptureRows) Next(values []driver.Value) error {
	if r.read {
		return io.EOF
	}
	r.read = true
	values[0] = r.value
	return nil
}

func TestTriggerMetadataPrivilegeIsMigrationOnly(t *testing.T) {
	plan, err := BuildRolePlan("cloudopsTest", RoleAccount{Name: "runtime", Host: "%"}, RoleAccount{Name: "migration", Host: "%"})
	if err != nil {
		t.Fatal(err)
	}
	want := "GRANT REFERENCES, TRIGGER ON `cloudopsTest`.* TO 'migration'@'%';"
	if plan.Migrator[0] != want {
		t.Fatal("missing direct schema-wide trigger metadata grant")
	}
	for _, statement := range plan.Runtime {
		if strings.Contains(statement, "TRIGGER") {
			t.Fatal("runtime received trigger administration authority")
		}
	}
}

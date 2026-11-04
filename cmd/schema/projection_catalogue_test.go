package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
)

func TestCurrentRoleCatalogueExactlyMatchesFullMigrationTables(t *testing.T) {
	capture := &roleSQLCapture{}
	db := sql.OpenDB(widthCatalogueConnector{capture})
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	manifest, err := dbschema.CurrentManifest()
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range manifest {
		if migration.Apply(context.Background(), conn) != nil {
			t.Fatal("offline current DDL capture failed")
		}
	}
	create := regexp.MustCompile("^CREATE TABLE `([a-z0-9_]+)` ")
	var tables []string
	for _, statement := range capture.statements {
		if strings.HasPrefix(statement, "CREATE TABLE ") {
			match := create.FindStringSubmatch(statement)
			if len(match) != 2 {
				t.Fatal("current DDL table not recognized")
			}
			tables = append(tables, match[1])
		}
	}
	want := currentBusinessTables()
	sort.Strings(want)
	sort.Strings(tables)
	if !reflect.DeepEqual(want, tables) {
		t.Fatal("current role catalogue diverges from migration")
	}
	mutated := currentBusinessTables()
	mutated[0] = "changed"
	if currentBusinessTables()[0] == "changed" {
		t.Fatal("current catalogue exposes shared mutation")
	}
}
func TestProjectionRolesPermitRetainedDMLButNeverDelete(t *testing.T) {
	plan, err := BuildRolePlan("cloudopsTest", RoleAccount{Name: "runtime", Host: "localhost"}, RoleAccount{Name: "migrator", Host: "localhost"})
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{configPoolControlTable, configPoolOperationTable, configArtifactEventTable} {
		runtime, migrator := false, false
		for _, statement := range plan.Runtime {
			if strings.Contains(statement, "`.`"+table+"`") {
				runtime = true
				if !strings.HasPrefix(statement, "GRANT SELECT, INSERT, UPDATE ON ") {
					t.Fatal("runtime retained-history permissions exceeded")
				}
			}
		}
		for _, statement := range plan.Migrator {
			if strings.Contains(statement, "`.`"+table+"`") {
				migrator = true
				if !strings.HasPrefix(statement, "GRANT SELECT, CREATE, ALTER, INDEX ON ") {
					t.Fatal("migration role received journal repair powers")
				}
			}
		}
		if !runtime || !migrator {
			t.Fatal("projection role inventory incomplete")
		}
	}
}

// The DDL recorder has no physical information_schema. Supply the released
// source attributes for the forward-only width ALTER preflight; real attribute
// and index preservation is exercised by the native width fixture.
type widthCatalogueConnector struct{ capture *roleSQLCapture }

func (c widthCatalogueConnector) Connect(context.Context) (driver.Conn, error) {
	return widthCatalogueConnection{roleCaptureConnection{c.capture}}, nil
}
func (c widthCatalogueConnector) Driver() driver.Driver { return roleCaptureDriver{} }

type widthCatalogueConnection struct{ roleCaptureConnection }

func (c widthCatalogueConnection) QueryContext(ctx context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(q, "collation_character_set_applicability") {
		return &widthCatalogueRows{values: []driver.Value{"utf8mb4_0900_ai_ci", "utf8mb4"}}, nil
	}
	if !strings.Contains(q, "information_schema.columns") {
		return c.roleCaptureConnection.QueryContext(ctx, q, args)
	}
	if len(args) != 2 {
		return nil, errors.New("unexpected column query")
	}
	for _, v := range []struct {
		table, name, kind string
		size              int64
		comment           string
	}{
		{"cl_monitor_configs", "name", "varchar", 100, "配置名称"}, {"cl_monitor_configs", "instance_ip", "varchar", 45, "实例IP地址"},
		{configNamespaceTable, "instance_key", "varbinary", 180, ""}, {configArtifactEventTable, "instance_key", "varbinary", 180, ""},
	} {
		if args[0].Value == v.table && args[1].Value == v.name {
			var charset, collation driver.Value
			if v.kind == "varchar" {
				charset = "utf8mb4"
				collation = "utf8mb4_0900_ai_ci"
			}
			return &widthCatalogueRows{values: []driver.Value{v.kind, v.size, "NO", nil, v.comment, charset, collation, "", ""}}, nil
		}
	}
	return nil, errors.New("unknown column")
}

type widthCatalogueRows struct {
	values []driver.Value
	read   bool
}

func (r *widthCatalogueRows) Columns() []string { return make([]string, len(r.values)) }
func (r *widthCatalogueRows) Close() error      { return nil }
func (r *widthCatalogueRows) Next(v []driver.Value) error {
	if r.read {
		return io.EOF
	}
	r.read = true
	copy(v, r.values)
	return nil
}

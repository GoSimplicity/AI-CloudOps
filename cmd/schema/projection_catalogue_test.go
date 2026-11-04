package main

import (
	"context"
	"database/sql"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
)

func TestCurrentRoleCatalogueExactlyMatchesFullMigrationTables(t *testing.T) {
	capture := &roleSQLCapture{}
	db := sql.OpenDB(roleCaptureConnector{capture})
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

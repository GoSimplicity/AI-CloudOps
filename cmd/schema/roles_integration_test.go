//go:build integration

package main

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
	"github.com/GoSimplicity/AI-CloudOps/pkg/di"
	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

func TestMySQLRolePlanEnforcesRuntimeAndMigratorBoundaries(t *testing.T) {
	admin, adminDSN := cliMySQLFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	adminSQL := func(query string) {
		t.Helper()
		if _, err := admin.ExecContext(ctx, query); err != nil {
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) {
				t.Fatalf("owned role fixture administrative SQL denied with MySQL error number %d; details withheld", mysqlErr.Number)
			}
			t.Fatal("owned role fixture administrative SQL failed; details withheld")
		}
	}
	adminSQL("CREATE DATABASE cloudopsroles")
	adminSQL("CREATE DATABASE cloudopsrolesOther")
	adminSQL("CREATE ROLE 'runtime_role'@'%', 'migrator_role'@'%'")
	adminSQL("CREATE USER 'runtime_fixture'@'%' IDENTIFIED BY 'isolated-runtime-only'")
	adminSQL("CREATE USER 'migrator_fixture'@'%' IDENTIFIED BY 'isolated-migrator-only'")
	adminSQL("GRANT 'runtime_role'@'%' TO 'runtime_fixture'@'%'")
	adminSQL("GRANT 'migrator_role'@'%' TO 'migrator_fixture'@'%'")
	adminSQL("SET DEFAULT ROLE 'runtime_role'@'%' TO 'runtime_fixture'@'%'")
	adminSQL("SET DEFAULT ROLE 'migrator_role'@'%' TO 'migrator_fixture'@'%'")
	plan, err := BuildRolePlan("cloudopsroles", RoleAccount{Name: "runtime_role", Host: "%"}, RoleAccount{Name: "migrator_role", Host: "%"})
	if err != nil {
		t.Fatal(err)
	}
	// Grants are valid before their target tables exist. This allows the fresh
	// baseline to run with the dedicated role rather than an administrator.
	for _, statement := range plan.Migrator {
		adminSQL(strings.TrimSuffix(statement, ";"))
	}
	openRole := func(user, password string) *sql.DB {
		t.Helper()
		config, err := mysql.ParseDSN(adminDSN)
		if err != nil {
			t.Fatal("fixture DSN parse failed")
		}
		config.DBName, config.User, config.Passwd = "cloudopsroles", user, password
		config.Timeout, config.ReadTimeout, config.WriteTimeout = 3*time.Second, 15*time.Second, 15*time.Second
		config.Logger = &mysql.NopLogger{}
		connector, err := mysql.NewConnector(config)
		if err != nil {
			t.Fatal("fixture role connector failed")
		}
		db := sql.OpenDB(connector)
		db.SetMaxOpenConns(1)
		t.Cleanup(func() { _ = db.Close() })
		if err := db.PingContext(ctx); err != nil {
			t.Fatal("fixture role connection failed")
		}
		return db
	}
	migratorDB := openRole("migrator_fixture", "isolated-migrator-only")
	denied := func(db *sql.DB, query string) {
		t.Helper()
		_, err := db.ExecContext(ctx, query)
		var mysqlErr *mysql.MySQLError
		if !errors.As(err, &mysqlErr) {
			t.Fatal("operation was not denied by a MySQL privilege error")
		}
		switch mysqlErr.Number {
		case 1044, 1142, 1143, 1227:
		default:
			t.Fatal("denial did not originate from privilege enforcement")
		}
	}
	metadata := func(db *sql.DB) []string {
		t.Helper()
		rows, err := db.QueryContext(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema=DATABASE() ORDER BY table_name")
		if err != nil {
			t.Fatal("role metadata query failed")
		}
		defer rows.Close()
		var names []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				t.Fatal("role metadata scan failed")
			}
			names = append(names, name)
		}
		if rows.Err() != nil {
			t.Fatal("role metadata read failed")
		}
		return names
	}
	// Metadata visibility must include tables absent from the fixed grant list.
	// Otherwise an old database with hidden tables could be mistaken for empty.
	adminSQL("CREATE TABLE cloudopsroles.unknown_legacy (id BIGINT PRIMARY KEY)")
	if !reflect.DeepEqual(metadata(migratorDB), []string{"unknown_legacy"}) {
		t.Fatal("schema REFERENCES failed to expose unknown legacy metadata")
	}
	conn, err := migratorDB.Conn(ctx)
	if err != nil {
		t.Fatal("role preflight connection failed")
	}
	err = dbschema.PreflightFresh(ctx, conn)
	_ = conn.Close()
	if !errors.Is(err, dbschema.ErrLegacy) {
		t.Fatal("hidden legacy table bypassed the fresh preflight")
	}
	denied(migratorDB, "SELECT * FROM unknown_legacy")
	adminSQL("DROP TABLE cloudopsroles.unknown_legacy")

	manifest, err := dbschema.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Migrate(ctx, migratorDB, manifest); err != nil {
		t.Fatal("fresh baseline failed under the generated migration role")
	}
	// Runtime table-only DML grants require the actual tables to exist. They
	// are intentionally applied after baseline rather than giving runtime CREATE.
	for _, statement := range plan.Runtime {
		adminSQL(strings.TrimSuffix(statement, ";"))
	}
	runtimeDB := openRole("runtime_fixture", "isolated-runtime-only")
	if err := schema.Verify(ctx, runtimeDB, manifest); err != nil {
		t.Fatal("runtime SELECT-only ledger verification failed")
	}
	runtimeConfig, err := mysql.ParseDSN(adminDSN)
	if err != nil {
		t.Fatal("runtime fixture config failed")
	}
	runtimeConfig.DBName, runtimeConfig.User, runtimeConfig.Passwd = "cloudopsroles", "runtime_fixture", "isolated-runtime-only"
	runtimeDSN := runtimeConfig.FormatDSN()
	separator := "?"
	if strings.Contains(runtimeDSN, "?") {
		separator = "&"
	}
	runtimeDSN += separator + "loc=UTC"
	appConfig := &config.Config{MySQL: config.MySQLConfig{Addr: runtimeDSN}}
	if opened, err := di.InitDB(appConfig, zap.NewNop()); err == nil {
		pool, _ := opened.DB()
		_ = pool.Close()
		t.Fatal("runtime startup silently initialized a missing clock")
	}
	if err := dbschema.InitializeFreshClock(ctx, migratorDB, "UTC"); err != nil {
		t.Fatal("clock initialization failed under migration role")
	}
	if err := dbschema.VerifyClock(ctx, runtimeDB, "UTC"); err != nil {
		t.Fatal("clock verification failed under runtime role")
	}
	// Exercise the actual application startup with DDL and ledger/clock writes
	// forbidden, not merely the two lower-level verification helpers.
	for range 2 {
		opened, err := di.InitDB(appConfig, zap.NewNop())
		if err != nil {
			t.Fatal("actual application startup failed under runtime role")
		}
		pool, err := opened.DB()
		if err != nil {
			t.Fatal("runtime pool unavailable")
		}
		_ = pool.Close()
	}
	wantTables := append([]string{}, baselineBusinessTables[:]...)
	wantTables = append(wantTables, schemaHistoryTable)
	sort.Strings(wantTables)
	for _, db := range []*sql.DB{runtimeDB, migratorDB} {
		if !reflect.DeepEqual(metadata(db), wantTables) {
			t.Fatal("role cannot enumerate all 58 business tables and ledger")
		}
	}

	// A real business CRUD cycle is allowed, including deleting the created row.
	if _, err := runtimeDB.ExecContext(ctx, "INSERT INTO cl_system_departments (id,name,parent_id) VALUES (901,'fixture department',0)"); err != nil {
		t.Fatal("runtime business INSERT denied")
	}
	var name string
	if err := runtimeDB.QueryRowContext(ctx, "SELECT name FROM cl_system_departments WHERE id=901").Scan(&name); err != nil || name != "fixture department" {
		t.Fatal("runtime business SELECT failed")
	}
	if _, err := runtimeDB.ExecContext(ctx, "UPDATE cl_system_departments SET name='updated fixture' WHERE id=901"); err != nil {
		t.Fatal("runtime business UPDATE denied")
	}
	if _, err := runtimeDB.ExecContext(ctx, "DELETE FROM cl_system_departments WHERE id=901"); err != nil {
		t.Fatal("runtime business DELETE denied")
	}
	for _, statement := range []string{
		"INSERT INTO cloudops_schema_history (version,name,checksum,state) VALUES (2,'forbidden',REPEAT('a',64),'started')",
		"UPDATE cloudops_schema_history SET state='failed' WHERE version=1",
		"DELETE FROM cloudops_schema_history WHERE version=1",
		"INSERT INTO cl_workorder_notification_clock (id,time_zone) VALUES (2,'UTC')",
		"UPDATE cl_workorder_notification_clock SET time_zone='Asia/Shanghai' WHERE id=1",
		"DELETE FROM cl_workorder_notification_clock WHERE id=1",
		"ALTER TABLE cl_system_departments ADD COLUMN forbidden INT",
		"CREATE TABLE forbidden (id BIGINT PRIMARY KEY)",
		"DROP TABLE cl_system_departments",
		"SELECT User FROM mysql.user",
	} {
		denied(runtimeDB, statement)
	}
	for _, statement := range []string{
		"UPDATE cl_system_departments SET name='forbidden' WHERE id=901",
		"INSERT INTO cl_system_departments (id,name) VALUES (902,'forbidden')",
		"DELETE FROM cl_workorder_notification_clock WHERE id=1",

		"DROP TABLE cl_system_departments",
		"CREATE TABLE forbidden (id BIGINT PRIMARY KEY)",
		"CREATE TABLE cloudopsrolesOther.forbidden (id BIGINT PRIMARY KEY)",
		"SELECT User FROM mysql.user",
	} {
		denied(migratorDB, statement)
	}
	for _, statement := range []string{
		"UPDATE cl_tree_cloud_resource_change_log SET old_value='[REDACTED]' WHERE id=-1",
		"UPDATE cl_workorder_notification SET is_default=2 WHERE id=-1",
		"UPDATE cl_workorder_notification_clock SET time_zone='UTC' WHERE id=1",
		"INSERT INTO cloudops_schema_history (version,name,checksum,state) VALUES (2,'fixture',REPEAT('a',64),'started')",
		"UPDATE cloudops_schema_history SET state='failed' WHERE version=2",
		"DELETE FROM cloudops_schema_history WHERE version=2",
	} {
		if _, err := migratorDB.ExecContext(ctx, statement); err != nil {
			t.Fatal("required migration data privilege missing")
		}
	}
	if err := schema.Verify(ctx, runtimeDB, manifest); err != nil {
		t.Fatal("runtime denial probes changed ledger")
	}
	if err := dbschema.VerifyClock(ctx, runtimeDB, "UTC"); err != nil {
		t.Fatal("runtime or migrator denial probes changed clock")
	}
}

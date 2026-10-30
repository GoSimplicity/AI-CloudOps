//go:build integration

package dbschema_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
)

func legacyFixtureRows(t *testing.T, db *sql.DB) string {
	t.Helper()
	definitions := tableDDL(t, db)
	var names []string
	for name := range definitions {
		names = append(names, name)
	}
	sort.Strings(names)
	digest := sha256.New()
	for _, name := range names {
		rows, err := db.Query("SELECT * FROM `" + name + "`")
		if err != nil {
			t.Fatal("cannot fingerprint owned fixture rows")
		}
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			t.Fatal(err)
		}
		var encodedRows []string
		for rows.Next() {
			values := make([]any, len(columns))
			destinations := make([]any, len(columns))
			for i := range values {
				destinations[i] = &values[i]
			}
			if err := rows.Scan(destinations...); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			encoded, err := json.Marshal(values)
			if err != nil {
				rows.Close()
				t.Fatal(err)
			}
			encodedRows = append(encodedRows, string(encoded))
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		rows.Close()
		sort.Strings(encodedRows)
		encoded, err := json.Marshal([]any{name, columns, encodedRows})
		if err != nil {
			t.Fatal(err)
		}
		_, _ = digest.Write(encoded)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func legacySnapshotStatement(t *testing.T, prefix string) string {
	t.Helper()
	raw, err := os.ReadFile("baseline_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var statements []string
	if err := json.Unmarshal(raw, &statements); err != nil {
		t.Fatal(err)
	}
	for _, statement := range statements {
		if strings.HasPrefix(statement, prefix) {
			return statement
		}
	}
	t.Fatal("required frozen statement missing")
	return ""
}

func TestLegacyCurrentMySQL(t *testing.T) {
	db, address := fixture(t)
	initializeCurrent(t, db)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	exec := func(statement string) {
		t.Helper()
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal("owned legacy fixture mutation failed; details withheld")
		}
	}
	// Only fake fixture values. No real credential or application service is used.
	for _, statement := range []string{
		"INSERT INTO cl_tree_cloud_account (id,name,access_key,secret_key,create_user_name) VALUES (1,'fixture','','','fixture')",
		"INSERT INTO cl_tree_cloud_account_region (id,cloud_account_id,region) VALUES (1,1,'fixture-region')",
		"INSERT INTO cl_tree_cloud_resource (id,name,cloud_account_id,cloud_account_region_id) VALUES (1,'fixture-resource',1,1)",
		"INSERT INTO cl_tree_cloud_resource_change_log (id,resource_id,field_name,old_value,new_value,updated_at,deleted_at) VALUES (1,1,' KEY ','[REDACTED]','[REDACTED]','2020-01-02 03:04:05',1)",
		"INSERT INTO cl_workorder_notification (id,name,event_types,channels,recipient_types,message_template,operator_id,is_default,scheduled_time,updated_at,deleted_at) VALUES (1,'legacy-fixture','[]','[]','[]','fixture',0,2,'2020-01-02 03:04:05','2020-01-02 03:04:05',1)",
		"ALTER TABLE cl_workorder_notification AUTO_INCREMENT=4000",
	} {
		exec(statement)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	before := legacyFixtureRows(t, db)
	if err := dbschema.ValidateLegacyCurrent(ctx, conn); err != nil {
		t.Fatal("current InitTables schema must pass without claiming a version", err)
	}
	if after := legacyFixtureRows(t, db); after != before {
		t.Fatal("read-only validator changed business rows or timestamps")
	}
	assertNoLedger := func() {
		t.Helper()
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='cloudops_schema_history'").Scan(&count); err != nil || count != 0 {
			t.Fatal("legacy preflight created version history")
		}
	}
	assertNoLedger()

	t.Run("read-only complete metadata role", func(t *testing.T) {
		exec("CREATE USER 'legacy_reader'@'%' IDENTIFIED BY 'fixture-only'")
		exec("GRANT SELECT ON baseline_source.* TO 'legacy_reader'@'%'")
		reader, err := sql.Open("mysql", "legacy_reader:fixture-only@tcp("+address+")/baseline_source?parseTime=true&timeout=3s&readTimeout=10s&writeTimeout=10s")
		if err != nil {
			t.Fatal("owned read-only fixture connection failed")
		}
		defer reader.Close()
		readConn, err := reader.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer readConn.Close()
		if err := dbschema.ValidateLegacyCurrent(ctx, readConn); err != nil {
			t.Fatal("read-only role must validate complete visible metadata", err)
		}
		if legacyFixtureRows(t, db) != before {
			t.Fatal("read-only validation changed business data")
		}
		assertNoLedger()
	})
	t.Run("REFERENCES metadata visibility includes unknown views", func(t *testing.T) {
		exec("CREATE USER 'legacy_reference'@'%' IDENTIFIED BY 'fixture-only'")
		exec("GRANT REFERENCES ON baseline_source.* TO 'legacy_reference'@'%'")
		exec("GRANT SELECT ON baseline_source.cl_workorder_notification TO 'legacy_reference'@'%'")
		exec("GRANT SELECT ON baseline_source.cl_tree_cloud_resource_change_log TO 'legacy_reference'@'%'")
		reader, err := sql.Open("mysql", "legacy_reference:fixture-only@tcp("+address+")/baseline_source?parseTime=true&timeout=3s&readTimeout=10s&writeTimeout=10s")
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		readConn, err := reader.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer readConn.Close()
		if err := dbschema.ValidateLegacyCurrent(ctx, readConn); err != nil {
			t.Fatal("schema REFERENCES metadata role must validate", err)
		}
		exec("CREATE VIEW legacy_extra_fixture AS SELECT 1 AS id")
		defer exec("DROP VIEW legacy_extra_fixture")
		if err := dbschema.ValidateLegacyCurrent(ctx, readConn); !errors.Is(err, dbschema.ErrLegacyStructure) {
			t.Fatal("unknown view hidden from REFERENCES role", err)
		}
	})
	t.Run("inherited role requires direct metadata grant", func(t *testing.T) {
		exec("CREATE ROLE 'legacy_inherited_role'@'%'")
		exec("GRANT REFERENCES ON baseline_source.* TO 'legacy_inherited_role'@'%'")
		exec("GRANT SELECT ON baseline_source.cl_workorder_notification TO 'legacy_inherited_role'@'%'")
		exec("GRANT SELECT ON baseline_source.cl_tree_cloud_resource_change_log TO 'legacy_inherited_role'@'%'")
		exec("CREATE USER 'legacy_inherited_user'@'%' IDENTIFIED BY 'fixture-only'")
		exec("GRANT 'legacy_inherited_role'@'%' TO 'legacy_inherited_user'@'%'")
		exec("SET DEFAULT ROLE 'legacy_inherited_role'@'%' TO 'legacy_inherited_user'@'%'")
		reader, err := sql.Open("mysql", "legacy_inherited_user:fixture-only@tcp("+address+")/baseline_source?parseTime=true&timeout=3s&readTimeout=10s&writeTimeout=10s")
		if err != nil {
			t.Fatal("owned role fixture connection failed")
		}
		defer reader.Close()
		roleConn, err := reader.Conn(ctx)
		if err != nil {
			t.Fatal("owned role fixture connection failed")
		}
		defer roleConn.Close()
		var enabled int
		if err := roleConn.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.ENABLED_ROLES WHERE ROLE_NAME='legacy_inherited_role' AND ROLE_HOST='%'").Scan(&enabled); err != nil || enabled != 1 {
			t.Fatal("fixture role was not enabled")
		}
		// ENABLED_ROLES proves activation, not the scope of a role's privileges.
		// MySQL does not expose this role's schema grant to the unprivileged
		// account through SCHEMA_PRIVILEGES, so the verifier stays conservative.
		if err := dbschema.ValidateLegacyCurrent(ctx, roleConn); !errors.Is(err, dbschema.ErrLegacyVisibility) {
			t.Fatal("role activation alone was accepted as complete metadata proof", err)
		}
		if _, err := roleConn.ExecContext(ctx, "SET ROLE NONE"); err != nil {
			t.Fatal("cannot deactivate owned fixture role")
		}
		if err := dbschema.ValidateLegacyCurrent(ctx, roleConn); !errors.Is(err, dbschema.ErrLegacyVisibility) {
			t.Fatal("inactive inherited role was accepted", err)
		}
		if _, err := roleConn.ExecContext(ctx, "SET ROLE 'legacy_inherited_role'@'%'"); err != nil {
			t.Fatal("cannot activate owned fixture role")
		}
		exec("GRANT REFERENCES ON baseline_source.* TO 'legacy_inherited_user'@'%'")
		if err := dbschema.ValidateLegacyCurrent(ctx, roleConn); err != nil {
			t.Fatal("explicit account metadata grant with active data role must pass", err)
		}
		exec("CREATE VIEW legacy_extra_fixture AS SELECT 1 AS id")
		defer exec("DROP VIEW legacy_extra_fixture")
		if err := dbschema.ValidateLegacyCurrent(ctx, roleConn); !errors.Is(err, dbschema.ErrLegacyStructure) {
			t.Fatal("unknown view was missed by inherited role plus direct metadata grant", err)
		}
	})
	t.Run("partial metadata visibility rejected", func(t *testing.T) {
		exec("CREATE USER 'legacy_partial'@'%' IDENTIFIED BY 'fixture-only'")
		exec("GRANT SELECT ON baseline_source.cl_workorder_notification TO 'legacy_partial'@'%'")
		reader, err := sql.Open("mysql", "legacy_partial:fixture-only@tcp("+address+")/baseline_source?parseTime=true&timeout=3s&readTimeout=10s&writeTimeout=10s")
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		readConn, err := reader.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer readConn.Close()
		if err := dbschema.ValidateLegacyCurrent(ctx, readConn); !errors.Is(err, dbschema.ErrLegacyVisibility) {
			t.Fatal("partial metadata must not be treated as complete inventory", err)
		}
	})

	clockDDL := legacySnapshotStatement(t, "CREATE TABLE `cl_workorder_notification_clock`")
	cronDDL := legacySnapshotStatement(t, "CREATE TABLE `cl_cron_jobs`")
	var nameDefinition, indexDefinition string
	for _, line := range strings.Split(cronDDL, "\n") {
		content := strings.TrimSuffix(strings.TrimSpace(line), ",")
		if strings.HasPrefix(content, "`name` ") {
			nameDefinition = content
		}
		if strings.HasPrefix(content, "UNIQUE KEY `idx_cl_cron_jobs_name` ") {
			indexDefinition = content
		}
	}
	if nameDefinition == "" || indexDefinition == "" {
		t.Fatal("immutable cron column/index definition missing")
	}
	foreignSQL := legacySnapshotStatement(t, "ALTER TABLE `cl_cron_jobs` ADD CONSTRAINT `fk_cl_cron_jobs_ssh_resource`")
	tests := []struct {
		name            string
		change, restore []string
	}{
		{"missing table", []string{"DROP TABLE cl_workorder_notification_clock"}, []string{clockDDL}},
		{"extra table", []string{"CREATE TABLE legacy_extra_fixture(id BIGINT)"}, []string{"DROP TABLE legacy_extra_fixture"}},
		{"extra view", []string{"CREATE VIEW legacy_extra_fixture AS SELECT 1 AS id"}, []string{"DROP VIEW legacy_extra_fixture"}},
		{"view replaces expected table", []string{"DROP TABLE cl_workorder_notification_clock", "CREATE VIEW cl_workorder_notification_clock AS SELECT 1 AS id"}, []string{"DROP VIEW cl_workorder_notification_clock", clockDDL}},
		{"column definition", []string{"ALTER TABLE cl_cron_jobs MODIFY COLUMN " + strings.Replace(nameDefinition, "varchar(100)", "varchar(101)", 1)}, []string{"ALTER TABLE cl_cron_jobs MODIFY COLUMN " + nameDefinition}},
		{"column order", []string{"ALTER TABLE cl_cron_jobs MODIFY COLUMN " + nameDefinition + " FIRST"}, []string{"ALTER TABLE cl_cron_jobs MODIFY COLUMN " + nameDefinition + " AFTER lease_expires_at"}},
		{"changed index", []string{"ALTER TABLE cl_cron_jobs DROP INDEX idx_cl_cron_jobs_name", "ALTER TABLE cl_cron_jobs ADD UNIQUE KEY idx_cl_cron_jobs_name (name DESC)"}, []string{"ALTER TABLE cl_cron_jobs DROP INDEX idx_cl_cron_jobs_name", "ALTER TABLE cl_cron_jobs ADD " + indexDefinition}},
		{"changed foreign key", []string{"ALTER TABLE cl_cron_jobs DROP FOREIGN KEY fk_cl_cron_jobs_ssh_resource", foreignSQL + " ON DELETE CASCADE"}, []string{"ALTER TABLE cl_cron_jobs DROP FOREIGN KEY fk_cl_cron_jobs_ssh_resource", foreignSQL}},
		{"changed engine", []string{"ALTER TABLE cl_workorder_notification_clock ENGINE=MyISAM"}, []string{"ALTER TABLE cl_workorder_notification_clock ENGINE=InnoDB"}},
		{"changed collation", []string{"ALTER TABLE cl_workorder_notification_clock CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"}, []string{"ALTER TABLE cl_workorder_notification_clock CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"}},
		{"existing ledger", []string{"CREATE TABLE cloudops_schema_history(id BIGINT)"}, []string{"DROP TABLE cloudops_schema_history"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, statement := range test.change {
				exec(statement)
			}
			defer func() {
				for _, statement := range test.restore {
					exec(statement)
				}
			}()
			if err := dbschema.ValidateLegacyCurrent(ctx, conn); !errors.Is(err, dbschema.ErrLegacyStructure) {
				t.Fatal("structural drift was not rejected", err)
			}
		})
		if err := dbschema.ValidateLegacyCurrent(ctx, conn); err != nil {
			t.Fatal("restored fixture no longer matches immutable schema", err)
		}
	}
	t.Run("soft-deleted default repair missing", func(t *testing.T) {
		exec("UPDATE cl_workorder_notification SET is_default=0 WHERE id=1")
		defer exec("UPDATE cl_workorder_notification SET is_default=2 WHERE id=1")
		if err := dbschema.ValidateLegacyCurrent(ctx, conn); !errors.Is(err, dbschema.ErrLegacyData) {
			t.Fatal("soft-deleted invalid default was not rejected", err)
		}
	})
	for _, test := range []struct {
		name, old, new string
		valid          bool
	}{
		{"redacted", "'[REDACTED]'", "'[REDACTED]'", true},
		{"both NULL", "NULL", "NULL", true},
		{"NULL old redacted new", "NULL", "'[REDACTED]'", true},
		{"redacted old NULL new", "'[REDACTED]'", "NULL", true},
		{"NULL old unrepaired new", "NULL", "'fake-private-marker'", false},
		{"unrepaired old NULL new", "'fake-private-marker'", "NULL", false},
		{"soft-deleted audit unrepaired", "'fake-private-marker'", "'fake-private-marker'", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			exec("UPDATE cl_tree_cloud_resource_change_log SET old_value=" + test.old + ",new_value=" + test.new + " WHERE id=1")
			defer exec("UPDATE cl_tree_cloud_resource_change_log SET old_value='[REDACTED]',new_value='[REDACTED]' WHERE id=1")
			err := dbschema.ValidateLegacyCurrent(ctx, conn)
			if test.valid && err != nil {
				t.Fatal("snapshot NULL semantics changed", err)
			}
			if !test.valid && !errors.Is(err, dbschema.ErrLegacyData) {
				t.Fatal("unrepaired sensitive audit was not rejected", err)
			}
			if err != nil && strings.Contains(err.Error(), "fake-private-marker") {
				t.Fatal("legacy error leaked data")
			}
		})
	}
	if legacyFixtureRows(t, db) != before {
		t.Fatal("validation or fixture restoration changed original business data")
	}
	assertNoLedger()
}

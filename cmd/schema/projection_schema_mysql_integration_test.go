//go:build integration

package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

// Only Root's NEW EMPTY, schema-scoped fixture is accepted. This creates/drops
// known test tables only, never databases/users or application/preview resources.
func TestProjectionSchemaNativeMySQL(t *testing.T) {
	path := os.Getenv("CLOUDOPS_PROJECTION_SCHEMA_MYSQL_DSN_FILE")
	if path == "" {
		t.Skip("private projection schema fixture not configured")
	}
	raw, err := readCredentials(path, func(string) string { return "" })
	if err != nil {
		t.Fatal("private projection schema DSN file invalid")
	}
	cfg, err := databaseConfig(raw, 30*time.Second)
	if err != nil || cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:13306" || !regexp.MustCompile(`^cloudops_projection_schema_[a-z0-9_]+$`).MatchString(cfg.DBName) || cfg.DBName != os.Getenv("CLOUDOPS_PROJECTION_SCHEMA_MYSQL_DATABASE") {
		t.Fatal("projection fixture requires exact loopback and confirmed private schema")
	}
	cfg.ParseTime = true
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal("projection schema fixture unavailable")
	}
	defer db.Close()
	db.SetMaxOpenConns(2)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	current, err := dbschema.CurrentManifest()
	if err != nil || len(current) < 3 {
		t.Fatal("current projection manifest unavailable")
	}
	allowed := map[string]bool{schemaHistoryTable: true}
	for _, table := range currentBusinessTables() {
		allowed[table] = true
	}
	inventory := func(t *testing.T) []string {
		t.Helper()
		rows, e := db.QueryContext(ctx, "SELECT table_name,table_type FROM information_schema.tables WHERE table_schema=DATABASE() ORDER BY table_name")
		if e != nil {
			t.Fatal("fixture metadata unavailable")
		}
		defer rows.Close()
		var names []string
		for rows.Next() {
			var name, kind string
			if rows.Scan(&name, &kind) != nil || kind != "BASE TABLE" || !allowed[name] {
				t.Fatal("unexpected object; refusing fixture cleanup")
			}
			names = append(names, name)
		}
		if rows.Err() != nil {
			t.Fatal("fixture metadata incomplete")
		}
		return names
	}
	if len(inventory(t)) != 0 {
		t.Fatal("projection schema fixture must initially be empty")
	}
	clean := func(t *testing.T) {
		t.Helper()
		names := inventory(t)
		conn, e := db.Conn(ctx)
		if e != nil {
			t.Fatal("cleanup connection unavailable")
		}
		defer conn.Close()
		if _, e := conn.ExecContext(ctx, "SET SESSION FOREIGN_KEY_CHECKS=0"); e != nil {
			t.Fatal("cleanup guard failed")
		}
		defer conn.ExecContext(ctx, "SET SESSION FOREIGN_KEY_CHECKS=1")
		sort.Sort(sort.Reverse(sort.StringSlice(names)))
		for _, name := range names {
			if _, e := conn.ExecContext(ctx, "DROP TABLE `"+name+"`"); e != nil {
				t.Fatal("known projection fixture cleanup failed")
			}
		}
	}
	defer clean(t)
	projectionTables := []string{configPoolControlTable, configPoolOperationTable, configArtifactEventTable}
	t.Run("fresh schema and storage constraints", func(t *testing.T) {
		if err := schema.MigrateFresh(ctx, db, current); err != nil {
			t.Fatal("fresh projection migration failed")
		}
		if schema.Verify(ctx, db, current) != nil {
			t.Fatal("fresh current ledger invalid")
		}
		if dbschema.InitializeFreshClock(ctx, db, "UTC") != nil || dbschema.VerifyClock(ctx, db, "UTC") != nil {
			t.Fatal("current clock chain rejected current target")
		}
		for _, column := range []string{"family", "pool_id"} {
			var extra string
			if db.QueryRowContext(ctx, "SELECT extra FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?", configPoolControlTable, column).Scan(&extra) != nil || strings.Contains(extra, "auto_increment") {
				t.Fatal("permanent control identity became auto increment")
			}
		}
		var first uint64
		result, e := db.ExecContext(ctx, "INSERT INTO cl_monitor_config_pool_operations(family,pool_id,generation,nonce,phase,source_hash,manifest) VALUES(1,7,1,?,'projected',?,'[]')", bytes.Repeat([]byte{1}, 16), bytes.Repeat([]byte{2}, 32))
		if e != nil {
			t.Fatal("committed empty operation storage failed")
		}
		id, e := result.LastInsertId()
		if e != nil || id <= 0 {
			t.Fatal("operation identity not assigned")
		}
		first = uint64(id)
		result, e = db.ExecContext(ctx, "INSERT INTO cl_monitor_config_pool_operations(family,pool_id,generation,nonce,phase,recovery_of_operation_id) VALUES(1,7,2,?,'failed',?)", bytes.Repeat([]byte{3}, 16), first)
		if e != nil {
			t.Fatal("failed reservation history storage failed")
		}
		second, e := result.LastInsertId()
		if e != nil || second <= id {
			t.Fatal("operation history did not append")
		}
		if _, e := db.ExecContext(ctx, "INSERT INTO cl_monitor_config_pool_controls(family,pool_id,reserved_generation,current_attempt_operation_id,last_committed_operation_id,last_projected_operation_id) VALUES(1,7,2,?,?,?)", second, first, first); e != nil {
			t.Fatal("control receipt pointers storage failed")
		}
		var reservedNull, emptyArray int
		if db.QueryRowContext(ctx, "SELECT manifest IS NULL FROM cl_monitor_config_pool_operations WHERE operation_id=?", second).Scan(&reservedNull) != nil || reservedNull != 1 || db.QueryRowContext(ctx, "SELECT JSON_TYPE(manifest)='ARRAY' AND JSON_LENGTH(manifest)=0 FROM cl_monitor_config_pool_operations WHERE operation_id=?", first).Scan(&emptyArray) != nil || emptyArray != 1 {
			t.Fatal("NULL and empty committed manifest were conflated")
		}
		var committed, projected uint64
		if db.QueryRowContext(ctx, "SELECT last_committed_operation_id,last_projected_operation_id FROM cl_monitor_config_pool_controls WHERE family=1 AND pool_id=7").Scan(&committed, &projected) != nil || committed != first || projected != first {
			t.Fatal("new failed attempt overwrote last-good pointers")
		}
		if _, e := db.ExecContext(ctx, "INSERT INTO cl_monitor_config_pool_operations(family,pool_id,generation,nonce,phase) VALUES(1,7,2,?,'reserved')", bytes.Repeat([]byte{4}, 16)); e == nil {
			t.Fatal("duplicate generation accepted")
		}
		if _, e := db.ExecContext(ctx, "INSERT INTO cl_monitor_config_pool_operations(family,pool_id,generation,nonce,phase) VALUES(1,7,3,?,'reserved')", bytes.Repeat([]byte{3}, 16)); e == nil {
			t.Fatal("duplicate nonce accepted")
		}
		if _, e := db.ExecContext(ctx, "INSERT INTO cl_monitor_config_pool_operations(family,pool_id,generation,nonce,phase,manifest) VALUES(1,7,3,?,'committed','{broken')", bytes.Repeat([]byte{5}, 16)); e == nil {
			t.Fatal("invalid JSON accepted")
		}
		keys := []string{"2001:db8::1", "2001:DB8::1", "192.0.2.1 ", " 192.0.2.1", "pool:7:ips", strings.Repeat("🌍", 45)}
		for _, key := range keys {
			if _, e := db.ExecContext(ctx, "INSERT INTO cl_monitor_config_artifact_events(config_type,instance_key,action,projection_policy,delivery_state) VALUES(4,?,'invalidate','not_applicable','pending')", []byte(key)); e != nil {
				t.Fatal("journal rejected raw byte key or full 45-character UTF-8 key")
			}
		}
		for _, key := range keys {
			var count int
			if db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cl_monitor_config_artifact_events WHERE config_type=4 AND BINARY instance_key=BINARY ?", []byte(key)).Scan(&count) != nil || count != 1 {
				t.Fatal("journal byte keys were aliased")
			}
		}
		if _, e := db.ExecContext(ctx, "INSERT INTO cl_monitor_config_artifact_events(config_type,instance_key,action,projection_policy,delivery_state) VALUES(4,?,'invalidate','not_applicable','pending')", []byte(strings.Repeat("🌍", 129))); e == nil {
			t.Fatal("binary schema capacity silently truncated key")
		}
		for _, table := range projectionTables {
			var unexpected int
			if db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name IN ('deleted_at','config_content','yaml_body','credential_body')", table).Scan(&unexpected) != nil || unexpected != 0 {
				t.Fatal("projection schema contains body or soft-deletion fields")
			}
		}
	})
	clean(t)
	t.Run("v2 appends only v3 without data repair", func(t *testing.T) {
		if schema.Migrate(ctx, db, current[:2]) != nil {
			t.Fatal("released v2 fixture failed")
		}
		if _, e := db.ExecContext(ctx, "INSERT INTO cl_monitor_configs(name,pool_id,instance_ip,config_type,config_content,config_hash,status,last_generated_time,deleted_at) VALUES('legacy A',7,'192.0.2.1',4,'preserve','hash',2,1,0),('legacy B',99,'192.0.2.1',4,'preserve','hash',2,1,0)"); e != nil {
			t.Fatal("legacy duplicate fixture failed")
		}
		key := []byte(strings.Repeat("🌍", 45))
		if _, e := db.ExecContext(ctx, "INSERT INTO cl_monitor_config_namespaces(config_type,instance_key) VALUES(4,?)", key); e != nil {
			t.Fatal("released guard fixture failed")
		}
		if err := schema.Verify(ctx, db, current[:3]); !errors.Is(err, schema.ErrPending) {
			t.Fatal("v2 not recognized as pending current")
		}
		if schema.MigratePending(ctx, db, current[:3]) != nil || schema.Verify(ctx, db, current[:3]) != nil {
			t.Fatal("explicit v2 to v3 append failed")
		}
		var count int
		if db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cl_monitor_configs WHERE status=2 AND config_content='preserve'").Scan(&count) != nil || count != 2 {
			t.Fatal("v3 changed historical config rows")
		}
		var rawKey []byte
		if db.QueryRowContext(ctx, "SELECT instance_key FROM cl_monitor_config_namespaces WHERE config_type=4").Scan(&rawKey) != nil || !bytes.Equal(rawKey, key) {
			t.Fatal("v3 changed released namespace guard")
		}
		for _, table := range projectionTables {
			if db.QueryRowContext(ctx, "SELECT COUNT(*) FROM `"+table+"`").Scan(&count) != nil || count != 0 {
				t.Fatal("schema migration invented journal authority")
			}
		}
		for _, version := range current[:2] {
			var checksum string
			if db.QueryRowContext(ctx, "SELECT checksum FROM cloudops_schema_history WHERE version=? AND state='applied'", version.Version).Scan(&checksum) != nil || checksum != version.Checksum {
				t.Fatal("released migration provenance changed")
			}
		}
	})
	clean(t)
	for _, failure := range []string{"dirty", "drift"} {
		t.Run(failure+" v3 rejects before DDL", func(t *testing.T) {
			if schema.Migrate(ctx, db, current[:2]) != nil {
				t.Fatal("v2 fixture failed")
			}
			last := current[2]
			state, checksum, want := "failed", last.Checksum, schema.ErrDirty
			if failure == "drift" {
				state, checksum, want = "applied", strings.Repeat("0", 64), schema.ErrDrift
			}
			if _, e := db.ExecContext(ctx, "INSERT INTO cloudops_schema_history(version,name,checksum,state) VALUES(?,?,?,?)", last.Version, last.Name, checksum, state); e != nil {
				t.Fatal("dirty/drift fixture failed")
			}
			if err := schema.MigratePending(ctx, db, current[:3]); !errors.Is(err, want) {
				t.Fatal("dirty/drift current schema was accepted")
			}
			var count int
			for _, table := range projectionTables {
				if db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?", table).Scan(&count) != nil || count != 0 {
					t.Fatal("dirty/drift ledger executed projection DDL")
				}
			}
		})
		clean(t)
	}
}

//go:build integration

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

// Opt-in only: Root provisions one NEW EMPTY, schema-scoped loopback fixture.
// No application credentials, existing schema, CREATE DATABASE or remote host.
func TestConfigWidthsSchemaNativeMySQL(t *testing.T) {
	path := os.Getenv("CLOUDOPS_CONFIG_WIDTH_MYSQL_DSN_FILE")
	if path == "" {
		t.Skip("private width schema fixture not configured")
	}
	raw, err := readCredentials(path, func(string) string { return "" })
	if err != nil {
		t.Fatal("private width DSN file invalid")
	}
	cfg, err := databaseConfig(raw, 30*time.Second)
	if err != nil || cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:13306" || !regexp.MustCompile(`^cloudops_config_width_[a-z0-9_]+$`).MatchString(cfg.DBName) || cfg.DBName != os.Getenv("CLOUDOPS_CONFIG_WIDTH_MYSQL_DATABASE") {
		t.Fatal("width fixture requires exact loopback and confirmed new schema")
	}
	cfg.ParseTime = true
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	cfg.Params["charset"] = "utf8mb4"
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal("width fixture unavailable")
	}
	defer db.Close()
	db.SetMaxOpenConns(2)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	current, err := dbschema.CurrentManifest()
	if err != nil || len(current) != 4 {
		t.Fatal("current width target invalid")
	}
	allowed := map[string]bool{schemaHistoryTable: true}
	for _, name := range currentBusinessTables() {
		allowed[name] = true
	}
	inventory := func(t *testing.T) []string {
		t.Helper()
		rows, err := db.QueryContext(ctx, "SELECT table_name,table_type FROM information_schema.tables WHERE table_schema=DATABASE() ORDER BY table_name")
		if err != nil {
			t.Fatal("fixture inventory unavailable")
		}
		defer rows.Close()
		var names []string
		for rows.Next() {
			var name, kind string
			if rows.Scan(&name, &kind) != nil || kind != "BASE TABLE" || !allowed[name] {
				t.Fatal("unknown object; refusing fixture cleanup")
			}
			names = append(names, name)
		}
		if rows.Err() != nil {
			t.Fatal("fixture inventory incomplete")
		}
		return names
	}
	if len(inventory(t)) != 0 {
		t.Fatal("width fixture must initially be empty")
	}
	clean := func(t *testing.T) {
		t.Helper()
		names := inventory(t)
		conn, err := db.Conn(ctx)
		if err != nil {
			t.Fatal("cleanup connection unavailable")
		}
		defer conn.Close()
		if _, err := conn.ExecContext(ctx, "SET SESSION FOREIGN_KEY_CHECKS=0"); err != nil {
			t.Fatal("cleanup guard failed")
		}
		defer conn.ExecContext(ctx, "SET SESSION FOREIGN_KEY_CHECKS=1")
		sort.Sort(sort.Reverse(sort.StringSlice(names)))
		for _, name := range names {
			if _, err := conn.ExecContext(ctx, "DROP TABLE `"+name+"`"); err != nil {
				t.Fatal("owned fixture cleanup failed")
			}
		}
	}
	defer clean(t)
	var mode string
	if db.QueryRowContext(ctx, "SELECT @@SESSION.sql_mode").Scan(&mode) != nil || (!strings.Contains(mode, "STRICT_TRANS_TABLES") && !strings.Contains(mode, "STRICT_ALL_TABLES")) {
		t.Fatal("native capacity validation requires strict MySQL")
	}
	exec := func(t *testing.T, q string, args ...any) sql.Result {
		t.Helper()
		result, err := db.ExecContext(ctx, q, args...)
		if err != nil {
			t.Fatal("private width fixture SQL failed; details withheld")
		}
		return result
	}
	insertConfig := `INSERT INTO cl_monitor_configs(name,pool_id,instance_ip,config_type,config_content,config_hash,status,last_generated_time,deleted_at) VALUES(?,7,?,5,'synthetic body','synthetic hash',2,17,0)`
	widths := func(t *testing.T, wantName, wantKey, wantBytes int) {
		t.Helper()
		for _, tc := range []struct {
			table, col string
			want       int
		}{{"cl_monitor_configs", "name", wantName}, {"cl_monitor_configs", "instance_ip", wantKey}, {configNamespaceTable, "instance_key", wantBytes}, {configArtifactEventTable, "instance_key", wantBytes}} {
			var actual int
			if db.QueryRowContext(ctx, "SELECT character_maximum_length FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?", tc.table, tc.col).Scan(&actual) != nil || actual != tc.want {
				t.Fatal("native column width differs")
			}
		}
	}
	capacity := func(t *testing.T) {
		t.Helper()
		for _, tc := range []struct{ name, key string }{
			{strings.Repeat("🌍", model.MonitorConfigNameMaxRunes), strings.Repeat("🌍", model.MonitorConfigKeyMaxRunes)},
			{"webhook_" + strings.Repeat("p", 50) + "_1_ffff:ffff:ffff:ffff:ffff:ffff:255.255.255.255", "ffff:ffff:ffff:ffff:ffff:ffff:255.255.255.255_1"},
			{"webhook_" + strings.Repeat("p", 50) + "_9223372036854775807_ffff:ffff:ffff:ffff:ffff:ffff:255.255.255.255", "ffff:ffff:ffff:ffff:ffff:ffff:255.255.255.255_9223372036854775807"},
		} {
			result := exec(t, insertConfig, tc.name, tc.key)
			id, err := result.LastInsertId()
			if err != nil {
				t.Fatal("config ID unavailable")
			}
			var name, key string
			if db.QueryRowContext(ctx, "SELECT name,instance_ip FROM cl_monitor_configs WHERE id=?", id).Scan(&name, &key) != nil || name != tc.name || key != tc.key {
				t.Fatal("config bytes changed at capacity")
			}
			exec(t, "INSERT INTO cl_monitor_config_namespaces(config_type,instance_key) VALUES(5,?)", []byte(tc.key))
			exec(t, "INSERT INTO cl_monitor_config_artifact_events(config_type,instance_key,action,projection_policy,delivery_state) VALUES(5,?,'invalidate','not_applicable','pending')", []byte(tc.key))
		}
		for _, tc := range []struct{ name, key string }{{strings.Repeat("🌍", model.MonitorConfigNameMaxRunes+1), "over-name"}, {"over-key", strings.Repeat("🌍", model.MonitorConfigKeyMaxRunes+1)}} {
			if _, err := db.ExecContext(ctx, insertConfig, tc.name, tc.key); err == nil {
				t.Fatal("MySQL silently accepted or truncated oversized text")
			}
		}
		over := []byte(strings.Repeat("🌍", model.MonitorConfigKeyMaxRunes+1))
		for _, q := range []string{"INSERT INTO cl_monitor_config_namespaces(config_type,instance_key) VALUES(5,?)", "INSERT INTO cl_monitor_config_artifact_events(config_type,instance_key,action,projection_policy,delivery_state) VALUES(5,?,'invalidate','not_applicable','pending')"} {
			if _, err := db.ExecContext(ctx, q, over); err == nil {
				t.Fatal("MySQL silently accepted or truncated oversized binary key")
			}
		}
	}
	t.Run("fresh current migration and clock", func(t *testing.T) {
		if schema.MigrateFresh(ctx, db, current) != nil || schema.Verify(ctx, db, current) != nil || dbschema.InitializeFreshClock(ctx, db, "UTC") != nil || dbschema.VerifyClock(ctx, db, "UTC") != nil {
			t.Fatal("fresh current clock chain failed")
		}
		widths(t, 255, 128, 512)
		capacity(t)
	})
	clean(t)
	t.Run("v3 upgrade preserves all rows and indexes", func(t *testing.T) {
		if schema.MigrateFresh(ctx, db, current[:3]) != nil {
			t.Fatal("frozen v3 source failed")
		}
		widths(t, 100, 45, 180)
		// These are the concrete native red cases under the released source widths.
		ip := "ffff:ffff:ffff:ffff:ffff:ffff:255.255.255.255"
		if _, err := db.ExecContext(ctx, insertConfig, "old-key-red", ip+"_1"); err == nil {
			t.Fatal("released 45-character schema unexpectedly accepts webhook key")
		}
		if _, err := db.ExecContext(ctx, insertConfig, "webhook_"+strings.Repeat("p", 50)+"_1_"+ip, "old-name-red"); err == nil {
			t.Fatal("released 100-character schema unexpectedly accepts generated name")
		}
		result := exec(t, insertConfig, strings.Repeat("🌍", 100), strings.Repeat("🌍", 45))
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatal("legacy row ID unavailable")
		}
		// Historical duplicates and inactive data are deliberately retained.
		exec(t, insertConfig, "historical duplicate", strings.Repeat("🌍", 45))
		rawKey := []byte(strings.Repeat("🌍", 45))
		exec(t, "INSERT INTO cl_monitor_config_namespaces(config_type,instance_key) VALUES(5,?)", rawKey)
		result = exec(t, "INSERT INTO cl_monitor_config_pool_operations(family,pool_id,generation,nonce,phase,source_hash,manifest) VALUES(4,7,1,?,'committed',?,'[]')", bytes.Repeat([]byte{1}, 16), bytes.Repeat([]byte{2}, 32))
		op, err := result.LastInsertId()
		if err != nil {
			t.Fatal("operation ID unavailable")
		}
		result = exec(t, "INSERT INTO cl_monitor_config_pool_operations(family,pool_id,generation,nonce,phase,recovery_of_operation_id) VALUES(4,7,2,?,'reserved',?)", bytes.Repeat([]byte{3}, 16), op)
		attempt, err := result.LastInsertId()
		if err != nil {
			t.Fatal("attempt ID unavailable")
		}
		exec(t, "INSERT INTO cl_monitor_config_pool_controls(family,pool_id,reserved_generation,current_attempt_operation_id,last_committed_operation_id) VALUES(4,7,2,?,?)", attempt, op)
		result = exec(t, "INSERT INTO cl_monitor_config_artifact_events(operation_id,config_type,instance_key,action,projection_policy,config_row_id,new_owner_digest,body_digest,binding,delivery_state) VALUES(?,5,?,'set','generated',?,?,?,'{\"synthetic\":true}','applied')", op, rawKey, id, bytes.Repeat([]byte{4}, 32), bytes.Repeat([]byte{5}, 32))
		event, err := result.LastInsertId()
		if err != nil {
			t.Fatal("event ID unavailable")
		}
		exec(t, "INSERT INTO cl_monitor_config_artifact_events(config_type,instance_key,action,projection_policy,binding_event_id,delivery_state) VALUES(5,?,'invalidate','lineage_only',?,'pending')", rawKey, event)
		queries := []string{"SELECT * FROM cl_monitor_configs ORDER BY id", "SELECT * FROM cl_monitor_config_namespaces ORDER BY config_type,instance_key", "SELECT * FROM cl_monitor_config_pool_controls ORDER BY family,pool_id", "SELECT * FROM cl_monitor_config_pool_operations ORDER BY operation_id", "SELECT * FROM cl_monitor_config_artifact_events ORDER BY event_id", "SELECT table_name,index_name,non_unique,seq_in_index,column_name,sub_part,collation,index_type,is_visible,expression FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name IN ('cl_monitor_configs','cl_monitor_config_namespaces','cl_monitor_config_artifact_events') ORDER BY table_name,index_name,seq_in_index", "SELECT version,name,checksum,state FROM cloudops_schema_history ORDER BY version"}
		before := make([][][]string, len(queries))
		for i, q := range queries {
			before[i] = widthNativeRows(t, ctx, db, q)
		}
		if !errors.Is(schema.Verify(ctx, db, current), schema.ErrPending) || schema.MigratePending(ctx, db, current) != nil || schema.Verify(ctx, db, current) != nil {
			t.Fatal("explicit v3 to v4 append failed")
		}
		for i, q := range queries {
			after := widthNativeRows(t, ctx, db, q)
			if i == len(queries)-1 {
				after = after[:len(before[i])]
			}
			if !reflect.DeepEqual(before[i], after) {
				t.Fatal("width migration changed existing data, lineage, index or released provenance")
			}
		}
		widths(t, 255, 128, 512)
		capacity(t)
	})
	clean(t)
	for _, failure := range []string{"dirty", "drift", "source collation drift", "source table default drift"} {
		t.Run(failure+" rejects without ALTER", func(t *testing.T) {
			if schema.MigrateFresh(ctx, db, current[:3]) != nil {
				t.Fatal("v3 source failed")
			}
			want := schema.ErrDirty
			switch failure {
			case "dirty":
				exec(t, "INSERT INTO cloudops_schema_history(version,name,checksum,state) VALUES(4,?,?,'failed')", current[3].Name, current[3].Checksum)
			case "drift":
				want = schema.ErrDrift
				exec(t, "INSERT INTO cloudops_schema_history(version,name,checksum,state) VALUES(4,?,?,'applied')", current[3].Name, strings.Repeat("0", 64))
			case "source table default drift":
				want = schema.ErrDirty
				exec(t, "ALTER TABLE cl_monitor_configs DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
			default:
				// The migration engine durably marks a failed Apply as dirty;
				// the pure Apply test checks the underlying ErrPreflight.
				want = schema.ErrDirty
				exec(t, "ALTER TABLE cl_monitor_configs MODIFY COLUMN instance_ip varchar(45) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '实例IP地址'")
			}
			if err := schema.MigratePending(ctx, db, current); !errors.Is(err, want) {
				t.Fatal("unsupported source/ledger was accepted")
			}
			widths(t, 100, 45, 180)
		})
		clean(t)
	}
}

func widthNativeRows(t *testing.T, ctx context.Context, db *sql.DB, q string) [][]string {
	t.Helper()
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		t.Fatal("private snapshot query failed")
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		t.Fatal("snapshot columns unavailable")
	}
	out := [][]string{cols}
	for rows.Next() {
		values := make([]any, len(cols))
		targets := make([]any, len(cols))
		for i := range values {
			targets[i] = &values[i]
		}
		if rows.Scan(targets...) != nil {
			t.Fatal("snapshot row unreadable")
		}
		row := make([]string, len(cols))
		for i, v := range values {
			switch x := v.(type) {
			case nil:
				row[i] = "NULL"
			case []byte:
				row[i] = "bytes:" + hex.EncodeToString(x)
			default:
				row[i] = fmt.Sprintf("%T:%v", x, x)
			}
		}
		out = append(out, row)
	}
	if rows.Err() != nil {
		t.Fatal("snapshot rows incomplete")
	}
	return out
}

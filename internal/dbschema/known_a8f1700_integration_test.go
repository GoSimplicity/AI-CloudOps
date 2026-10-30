//go:build integration

package dbschema_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
	"github.com/go-sql-driver/mysql"
)

type knownSnapshotFixture struct {
	Tables map[string]string `json:"tables"`
}

func knownUpgradeMySQL(t *testing.T) (*sql.DB, string) {
	t.Helper()
	for _, key := range []string{"DOCKER_HOST", "DOCKER_CONTEXT"} {
		if os.Getenv(key) != "" {
			t.Fatal("known upgrade fixture refuses Docker endpoint overrides")
		}
	}
	var environment []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if key != "DOCKER_HOST" && key != "DOCKER_CONTEXT" && key != "DOCKER_TLS" && key != "DOCKER_TLS_VERIFY" && key != "DOCKER_CERT_PATH" {
			environment = append(environment, entry)
		}
	}
	command := func(arguments ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "docker", arguments...)
		cmd.Env = environment
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal("owned known-source Docker operation failed; details withheld")
		}
		return strings.TrimSpace(string(out))
	}
	endpoint := command("context", "inspect", "--format", "{{.Endpoints.docker.Host}}")
	if !strings.HasPrefix(endpoint, "unix:///") {
		t.Fatal("known-source fixture requires local Unix Docker")
	}
	docker := func(arguments ...string) string {
		t.Helper()
		return command(append([]string{"--host", endpoint}, arguments...)...)
	}
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	name := "cloudops-known-upgrade-" + hex.EncodeToString(random[:])
	docker("run", "-d", "--name", name, "--pull=never", "--label", "cloudops.known.upgrade="+name, "-e", "MYSQL_ALLOW_EMPTY_PASSWORD=yes", "-e", "MYSQL_ROOT_HOST=%", "-e", "MYSQL_DATABASE=knownupgrade", "-p", "127.0.0.1::3306", "mysql:8.4.11")
	t.Cleanup(func() {
		if docker("inspect", "--format", `{{index .Config.Labels "cloudops.known.upgrade"}}`, name) != name {
			t.Error("fixture owner differs; refusing removal")
			return
		}
		docker("rm", "-fv", name)
	})
	address := docker("port", name, "3306/tcp")
	if !strings.HasPrefix(address, "127.0.0.1:") {
		t.Fatal("known-source fixture must bind loopback")
	}
	config := mysql.NewConfig()
	config.User = "root"
	config.Net = "tcp"
	config.Addr = address
	config.DBName = "knownupgrade"
	config.ParseTime = true
	config.Timeout = 2 * time.Second
	config.ReadTimeout = 10 * time.Second
	config.WriteTimeout = 10 * time.Second
	config.Logger = &mysql.NopLogger{}
	connector, err := mysql.NewConnector(config)
	if err != nil {
		t.Fatal("fixture connector failed")
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		if db.PingContext(ctx) == nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("owned known-source MySQL readiness timeout")
		case <-ticker.C:
		}
	}
	return db, address
}

func knownSourceFixture(t *testing.T, db *sql.DB) knownSnapshotFixture {
	t.Helper()
	raw, err := os.ReadFile("known_a8f1700.json")
	if err != nil {
		t.Fatal(err)
	}
	var source knownSnapshotFixture
	if json.Unmarshal(raw, &source) != nil || len(source.Tables) != 49 {
		t.Fatal("invalid frozen old fixture")
	}
	var names []string
	for name := range source.Tables {
		names = append(names, name)
	}
	sort.Strings(names)
	var constraints []string
	for _, name := range names {
		lines := strings.Split(source.Tables[name], "\n")
		kept := []string{lines[0]}
		for _, line := range lines[1 : len(lines)-1] {
			if strings.HasPrefix(strings.TrimSpace(line), "CONSTRAINT ") {
				constraints = append(constraints, "ALTER TABLE `"+name+"` ADD "+strings.TrimSuffix(strings.TrimSpace(line), ","))
			} else {
				kept = append(kept, line)
			}
		}
		kept[len(kept)-1] = strings.TrimSuffix(kept[len(kept)-1], ",")
		kept = append(kept, lines[len(lines)-1])
		execKnownFixture(t, db, strings.Join(kept, "\n"))
	}
	for _, statement := range constraints {
		execKnownFixture(t, db, statement)
	}
	return source
}

func execKnownFixture(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			identifier := regexp.MustCompile("(?i)(?:INSERT INTO|UPDATE|ALTER TABLE|CREATE TABLE) `?([A-Za-z0-9_]+)").FindStringSubmatch(query)
			if len(identifier) == 2 {
				t.Fatalf("owned fixture SQL failed: table=%s number=%d; values withheld", identifier[1], mysqlErr.Number)
			}
		}
		t.Fatal("owned known-source fixture SQL failed; details withheld")
	}
}

func seedKnownRows(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, query := range []string{
		"INSERT INTO cl_system_departments (id,name,parent_id,created_at,deleted_at) VALUES (11,'fixture department',0,'2020-01-02 03:04:05',0)",
		"INSERT INTO cl_system_users (id,department_id,username,password,created_at,deleted_at) VALUES (12,11,'fixture-user','fixture-hash-not-auth','2020-01-02 03:04:05',0)",
		"INSERT INTO cl_tree_cloud_account (id,name,provider,access_key,secret_key,status,create_user_id,create_user_name,created_at,deleted_at) VALUES (21,'fixture-account',1,'ENC-FIXTURE-KEY+/==','ENC-FIXTURE-SECRET+/==',2,12,'fixture owner','2020-01-02 03:04:05',0)",
		"INSERT INTO cl_tree_cloud_account_region (id,cloud_account_id,region,region_name,status,created_at,deleted_at) VALUES (31,21,'fixture-region','fixture region',2,'2020-01-02 03:04:05',0)",
		"INSERT INTO cl_tree_cloud_resource (id,name,cloud_account_id,cloud_account_region_id,created_at,deleted_at) VALUES (41,'fixture resource',21,31,'2020-01-02 03:04:05',0)",
		"INSERT INTO cl_tree_cloud_resource_sync_history (id,cloud_account_id,sync_mode,sync_status,created_at,deleted_at) VALUES (42,21,'incremental','success','2020-01-02 03:04:05',0)",
		"INSERT INTO cl_tree_cloud_resource_change_log (id,resource_id,field_name,old_value,new_value,created_at,deleted_at) VALUES (51,41,' password ','fixture-old','fixture-new','2020-01-02 03:04:05',123),(52,41,'key',NULL,NULL,'2020-01-02 03:04:05',0),(53,41,'name','before','after','2020-01-02 03:04:05',0)",
		"INSERT INTO cl_workorder_notification (id,name,event_types,channels,recipient_types,message_template,operator_id,is_default,status,created_at,deleted_at) VALUES (61,'fixture notification','[\"instance_created\"]','[\"email\"]','[\"custom\"]','fixture content',12,0,2,'2020-01-02 03:04:05',0)",
		"INSERT INTO cl_workorder_notification (id,name,event_types,channels,recipient_types,message_template,operator_id,is_default,status,trigger_type,scheduled_time,created_at,deleted_at) VALUES (62,'fixture legacy advanced','[\"instance_created\"]','[\"email\"]','[\"custom\"]','fixture old advanced',12,1,2,'scheduled','2030-01-02 03:04:05','2020-01-02 03:04:05',123)",
		"INSERT INTO cl_workorder_notification_queue (id,notification_id,event_type,channel,recipient_type,recipient_id,recipient_addr,content,status,scheduled_at,created_at,deleted_at) VALUES (71,61,'instance_created','email','custom','fixture-recipient','fixture@example.test','queue must not replay',2,'2030-01-02 03:04:05','2020-01-02 03:04:05',0)",
		"INSERT INTO cl_cron_jobs (id,name,schedule,status,created_at,deleted_at) VALUES (81,'disabled fixture job','0 * * * *',2,'2020-01-02 03:04:05',0)",
		"INSERT INTO cl_workorder_form_design (id,name,`schema`,operator_id,operator_name,created_at,deleted_at) VALUES (91,'fixture form','{}',12,'fixture owner','2020-01-02 03:04:05',0)",
		"INSERT INTO cl_workorder_process (id,name,form_design_id,definition,operator_id,operator_name,created_at,deleted_at) VALUES (92,'fixture process',91,'{}',12,'fixture owner','2020-01-02 03:04:05',0)",
		"INSERT INTO cl_workorder_instance (id,title,serial_number,process_id,operator_id,operator_name,created_at,deleted_at) VALUES (93,'fixture workorder','fixture-93',92,12,'fixture owner','2020-01-02 03:04:05',0)",
		"INSERT INTO cl_workorder_instance_timeline (id,instance_id,action,operator_id,created_at,deleted_at) VALUES (94,93,'created',12,'2020-01-02 03:04:05',0)",
	} {
		execKnownFixture(t, db, query)
	}
}

func knownDataDigest(t *testing.T, db *sql.DB, source knownSnapshotFixture) string {
	t.Helper()
	column := regexp.MustCompile("^  `([A-Za-z0-9_]+)` ")
	result := map[string][]string{}
	for name, ddl := range source.Tables {
		var columns []string
		for _, line := range strings.Split(ddl, "\n") {
			match := column.FindStringSubmatch(line)
			if match == nil {
				continue
			}
			field := match[1]
			if name == "cl_workorder_notification" && field == "is_default" {
				continue
			}
			if name == "cl_tree_cloud_resource_change_log" && (field == "old_value" || field == "new_value") {
				continue
			}
			columns = append(columns, "`"+field+"`")
		}
		rows, err := db.Query("SELECT " + strings.Join(columns, ",") + " FROM `" + name + "`")
		if err != nil {
			t.Fatal("fixture data digest query failed")
		}
		var values []string
		for rows.Next() {
			data := make([]any, len(columns))
			args := make([]any, len(columns))
			for i := range data {
				args[i] = &data[i]
			}
			if err := rows.Scan(args...); err != nil {
				rows.Close()
				t.Fatal("fixture digest scan failed")
			}
			encoded, err := json.Marshal(data)
			if err != nil {
				rows.Close()
				t.Fatal(err)
			}
			values = append(values, string(encoded))
		}
		if rows.Err() != nil {
			rows.Close()
			t.Fatal("fixture digest rows failed")
		}
		rows.Close()
		sort.Strings(values)
		result[name] = values
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func minimalKnownMigrator(t *testing.T, db *sql.DB, address string) (*sql.DB, string) {
	t.Helper()
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "run", "./cmd/schema", "roles", "--database", "knownupgrade", "--runtime-role", "known_runtime", "--runtime-role-host", "%", "--migrator-role", "known_migration", "--migrator-role-host", "%", "--section", "migrator", "--format", "json")
	command.Dir = repo
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "CLOUDOPS_SCHEMA_DSN=") {
			command.Env = append(command.Env, entry)
		}
	}
	raw, err := command.Output()
	if err != nil {
		t.Fatal("offline production role plan failed")
	}
	var plan struct {
		Statements []string `json:"statements"`
	}
	if json.Unmarshal(raw, &plan) != nil || len(plan.Statements) == 0 {
		t.Fatal("invalid role plan")
	}
	execKnownFixture(t, db, "CREATE ROLE 'known_migration'@'%'")
	execKnownFixture(t, db, "CREATE USER 'known_migrator'@'%' IDENTIFIED BY 'fixture-only-migration'")
	execKnownFixture(t, db, "GRANT 'known_migration'@'%' TO 'known_migrator'@'%'")
	execKnownFixture(t, db, "SET DEFAULT ROLE 'known_migration'@'%' TO 'known_migrator'@'%'")
	for _, statement := range plan.Statements {
		execKnownFixture(t, db, statement)
	}
	// Strict metadata proof must be direct for CURRENT_USER, not role inference.
	execKnownFixture(t, db, "GRANT REFERENCES ON knownupgrade.* TO 'known_migrator'@'%'")
	config := mysql.NewConfig()
	config.User = "known_migrator"
	config.Passwd = "fixture-only-migration"
	config.Net = "tcp"
	config.Addr = address
	config.DBName = "knownupgrade"
	config.ParseTime = true
	config.Timeout = 3 * time.Second
	config.ReadTimeout = 15 * time.Second
	config.WriteTimeout = 15 * time.Second
	config.Logger = &mysql.NopLogger{}
	connector, err := mysql.NewConnector(config)
	if err != nil {
		t.Fatal(err)
	}
	role := sql.OpenDB(connector)
	role.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = role.Close() })
	return role, config.FormatDSN()
}

// The private fixture artifact is a complete logical schema+row snapshot of
// this owned DB only. It is not evidence that Redis/AIOps or a full stack restore
// has been tested. No old application, webhook poller or worker is ever started.
func knownPrivateBackup(t *testing.T, db *sql.DB, source knownSnapshotFixture) string {
	t.Helper()
	payload := struct {
		Source string                          `json:"source_revision"`
		Tables map[string]string               `json:"tables"`
		Rows   map[string][]map[string]*string `json:"rows"`
	}{dbschema.KnownA8Source, source.Tables, map[string][]map[string]*string{}}
	for name := range source.Tables {
		rows, err := db.Query("SELECT * FROM `" + name + "`")
		if err != nil {
			t.Fatal("owned fixture backup query failed")
		}
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			t.Fatal(err)
		}
		var encodedRows []map[string]*string
		for rows.Next() {
			values := make([]any, len(columns))
			targets := make([]any, len(columns))
			for i := range values {
				targets[i] = &values[i]
			}
			if err := rows.Scan(targets...); err != nil {
				rows.Close()
				t.Fatal("owned backup row read failed")
			}
			record := map[string]*string{}
			for i, value := range values {
				if value == nil {
					record[columns[i]] = nil
					continue
				}
				var raw []byte
				switch typed := value.(type) {
				case []byte:
					raw = typed
				case time.Time:
					raw = []byte(typed.Format("2006-01-02 15:04:05.999999999"))
				default:
					raw = []byte(fmt.Sprint(typed))
				}
				text := hex.EncodeToString(raw)
				record[columns[i]] = &text
			}
			encodedRows = append(encodedRows, record)
		}
		if rows.Err() != nil {
			rows.Close()
			t.Fatal("owned backup row iteration failed")
		}
		rows.Close()
		sort.Slice(encodedRows, func(i, j int) bool {
			a, _ := json.Marshal(encodedRows[i])
			b, _ := json.Marshal(encodedRows[j])
			return string(a) < string(b)
		})
		payload.Rows[name] = encodedRows
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "owned-old-logical-backup.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func TestKnownA8RealUpgradePreservesOriginalDataUsingMinimalRole(t *testing.T) {
	admin, address := knownUpgradeMySQL(t)
	source := knownSourceFixture(t, admin)
	seedKnownRows(t, admin)
	before := knownDataDigest(t, admin, source)
	backupSHA := knownPrivateBackup(t, admin, source)
	plan, err := dbschema.KnownA8Upgrade()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := dbschema.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	evidence := schema.UpgradeEvidence{Note: "owned fixture backup and timezone explicitly reviewed", BackupSHA256: backupSHA, NotificationZone: "UTC", HistoricalTimeAttested: true}
	role, dsn := minimalKnownMigrator(t, admin, address)
	cliEvidence := t.TempDir()
	noteFile, proofFile := filepath.Join(cliEvidence, "note"), filepath.Join(cliEvidence, "proof")
	if err := os.WriteFile(noteFile, []byte(evidence.Note), 0600); err != nil {
		t.Fatal(err)
	}
	proof, err := json.Marshal(map[string]any{"version": 1, "source_revision": dbschema.KnownA8Source, "backup_sha256": backupSHA, "backup_ready": true})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(proofFile, proof, 0600); err != nil {
		t.Fatal(err)
	}
	runCLI := func(zone string) int {
		t.Helper()
		repo, err := filepath.Abs("../..")
		if err != nil {
			t.Fatal(err)
		}
		command := exec.CommandContext(ctx, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", "./cmd/schema", "upgrade-a8f1700", "--execute", "--confirm-no-writers", "--upgrade-note-file", noteFile, "--backup-proof-file", proofFile, "--notification-timezone", zone, "--attest-historical-time")
		command.Dir = repo
		for _, entry := range os.Environ() {
			if !strings.HasPrefix(entry, "CLOUDOPS_SCHEMA_DSN=") {
				command.Env = append(command.Env, entry)
			}
		}
		command.Env = append(command.Env, "CLOUDOPS_SCHEMA_DSN="+dsn)
		if _, err := command.CombinedOutput(); err != nil {
			var exit *exec.ExitError
			if errors.As(err, &exit) {
				return exit.ExitCode()
			}
			t.Fatal("owned CLI process failed without an exit status")
		}
		return 0
	}
	if runCLI("UTC") != 0 {
		t.Fatal("actual explicit known-source CLI upgrade failed; private output withheld")
	}
	if runCLI("UTC") != 0 {
		t.Fatal("actual CLI repeat did not verify read-only")
	}
	if runCLI("Asia/Shanghai") == 0 {
		t.Fatal("existing-applied CLI accepted a different clock marker")
	}
	if after := knownDataDigest(t, admin, source); after != before {
		t.Fatal("old columns changed beyond the two approved repair values")
	}
	if err := schema.Verify(ctx, role, manifest); err != nil {
		t.Fatal("upgraded history failed verification")
	}
	if err := dbschema.VerifyClock(ctx, role, "UTC"); err != nil {
		t.Fatal("explicit zone marker failed verification")
	}
	var kind string
	var version uint64
	var expires sql.NullTime
	var token, provider sql.NullString
	if err := admin.QueryRow("SELECT credential_type,credential_version,credential_expires_at,session_token,provider_credentials FROM cl_tree_cloud_account WHERE id=21").Scan(&kind, &version, &expires, &token, &provider); err != nil || kind != "long_term" || version != 0 || expires.Valid || token.Valid || provider.Valid {
		t.Fatal("old credential tuple or new defaults invalid")
	}
	var defaultNo, defaultYes, triggerVersion int
	if err := admin.QueryRow("SELECT is_default,trigger_version FROM cl_workorder_notification WHERE id=61").Scan(&defaultNo, &triggerVersion); err != nil || defaultNo != 2 || triggerVersion != 0 {
		t.Fatal("notification repair/default changed intent")
	}
	if err := admin.QueryRow("SELECT is_default,trigger_version FROM cl_workorder_notification WHERE id=62").Scan(&defaultYes, &triggerVersion); err != nil || defaultYes != 1 || triggerVersion != 0 {
		t.Fatal("historical advanced trigger was automatically activated")
	}
	var oldValue, newValue sql.NullString
	if err := admin.QueryRow("SELECT old_value,new_value FROM cl_tree_cloud_resource_change_log WHERE id=51").Scan(&oldValue, &newValue); err != nil || oldValue.String != "[REDACTED]" || newValue.String != "[REDACTED]" {
		t.Fatal("soft-deleted credential audit was not repaired")
	}
	if err := admin.QueryRow("SELECT old_value,new_value FROM cl_tree_cloud_resource_change_log WHERE id=52").Scan(&oldValue, &newValue); err != nil || oldValue.Valid || newValue.Valid {
		t.Fatal("NULL repair semantics changed")
	}
	if err := admin.QueryRow("SELECT old_value,new_value FROM cl_tree_cloud_resource_change_log WHERE id=53").Scan(&oldValue, &newValue); err != nil || oldValue.String != "before" || newValue.String != "after" {
		t.Fatal("noncredential audit was altered")
	}
	var origin, sourceRevision, impl, backup string
	if err := admin.QueryRow("SELECT origin,source_revision,upgrade_checksum,backup_sha256 FROM cloudops_schema_history WHERE version=1").Scan(&origin, &sourceRevision, &impl, &backup); err != nil || origin != "upgraded" || sourceRevision != dbschema.KnownA8Source || impl != plan.ImplementationChecksum || backup != backupSHA {
		t.Fatal("upgrade provenance incomplete")
	}
	repeat := plan
	repeat.Apply = func(context.Context, *sql.Conn, schema.UpgradeEvidence) error {
		t.Fatal("repeat executed SQL")
		return nil
	}
	if err := schema.UpgradeBaseline(ctx, role, manifest, repeat, evidence); err != nil {
		t.Fatal("repeat did not verify read-only")
	}
}

func TestKnownA8SourceGuardsBeforeLedger(t *testing.T) {
	db, _ := knownUpgradeMySQL(t)
	knownSourceFixture(t, db)
	seedKnownRows(t, db)
	plan, err := dbschema.KnownA8Upgrade()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := dbschema.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	evidence := schema.UpgradeEvidence{Note: "fixture reviewed", BackupSHA256: strings.Repeat("a", 64), NotificationZone: "UTC"}
	if err := schema.UpgradeBaseline(ctx, db, manifest, plan, evidence); !errors.Is(err, schema.ErrPreflight) {
		t.Fatal("historical time without independent attestation was accepted")
	}
	var ledger int
	if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='cloudops_schema_history'").Scan(&ledger); err != nil || ledger != 0 {
		t.Fatal("source rejection wrote a ledger")
	}
	evidence.HistoricalTimeAttested = true
	for _, drift := range []struct{ create, remove string }{
		{"CREATE TABLE known_extra_fixture (id BIGINT PRIMARY KEY)", "DROP TABLE known_extra_fixture"},
		{"ALTER TABLE cl_tree_cloud_account ADD INDEX known_extra_index (name)", "ALTER TABLE cl_tree_cloud_account DROP INDEX known_extra_index"},
	} {
		execKnownFixture(t, db, drift.create)
		if err := schema.UpgradeBaseline(ctx, db, manifest, plan, evidence); !errors.Is(err, schema.ErrPreflight) {
			t.Fatal("source schema drift reached upgrade writes")
		}
		execKnownFixture(t, db, drift.remove)
	}
	for _, unsupported := range []int{4, 5, 6} {
		execKnownFixture(t, db, "UPDATE cl_tree_cloud_account SET provider=?,status=1 WHERE id=21", unsupported)
		if err := schema.UpgradeBaseline(ctx, db, manifest, plan, evidence); !errors.Is(err, schema.ErrPreflight) {
			t.Fatal("enabled unsupported provider was not rejected")
		}
	}
	execKnownFixture(t, db, "UPDATE cl_tree_cloud_account SET provider=4,status=1,deleted_at=123 WHERE id=21")
	if err := schema.UpgradeBaseline(ctx, db, manifest, plan, evidence); !errors.Is(err, schema.ErrPreflight) {
		t.Fatal("soft-deleted but still enabled unsupported provider was accepted")
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='cloudops_schema_history'").Scan(&ledger); err != nil || ledger != 0 {
		t.Fatal("guard rejections wrote an upgrade ledger")
	}
	execKnownFixture(t, db, "UPDATE cl_tree_cloud_account SET provider=4,status=2 WHERE id=21")
	if err := schema.UpgradeBaseline(ctx, db, manifest, plan, evidence); err != nil {
		t.Fatalf("reviewed disabled provider could not upgrade: %v", err)
	}
	var status int
	if err := db.QueryRow("SELECT status FROM cl_tree_cloud_account WHERE id=21").Scan(&status); err != nil || status != 2 {
		t.Fatal("upgrade reactivated a disabled legacy provider")
	}
}

func TestKnownA8FailedDDLBlocksAutomaticRetry(t *testing.T) {
	db, _ := knownUpgradeMySQL(t)
	knownSourceFixture(t, db)
	plan, err := dbschema.KnownA8Upgrade()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := dbschema.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	evidence := schema.UpgradeEvidence{Note: "fixture backup attested", BackupSHA256: strings.Repeat("a", 64), NotificationZone: "UTC"}
	calls := 0
	original := plan.Apply
	plan.Apply = func(ctx context.Context, conn *sql.Conn, evidence schema.UpgradeEvidence) error {
		calls++
		if _, err := conn.ExecContext(ctx, "CREATE TABLE partial_known_fixture (id BIGINT PRIMARY KEY)"); err != nil {
			return err
		}
		return errors.New("private-DDL-failure-canary")
	}
	if err := schema.UpgradeBaseline(ctx, db, manifest, plan, evidence); !errors.Is(err, schema.ErrDirty) || strings.Contains(err.Error(), "private-") {
		t.Fatal("partial DDL did not preserve a sanitized failed blocker")
	}
	plan.Apply = original
	if err := schema.UpgradeBaseline(ctx, db, manifest, plan, evidence); !errors.Is(err, schema.ErrDirty) {
		t.Fatal("failed known upgrade was automatically retried")
	}
	if calls != 1 {
		t.Fatal("partial DDL executed twice")
	}
	var state string
	if err := db.QueryRow("SELECT state FROM cloudops_schema_history WHERE version=1").Scan(&state); err != nil || state != "failed" {
		t.Fatal("failed evidence was rewritten")
	}
}

func TestKnownA8ConcurrentUpgradeUsesOneDatabaseLock(t *testing.T) {
	db, address := knownUpgradeMySQL(t)
	knownSourceFixture(t, db)
	plan, err := dbschema.KnownA8Upgrade()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := dbschema.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	config := mysql.NewConfig()
	config.User = "root"
	config.Net = "tcp"
	config.Addr = address
	config.DBName = "knownupgrade"
	config.Timeout = 3 * time.Second
	config.Logger = &mysql.NopLogger{}
	connector, err := mysql.NewConnector(config)
	if err != nil {
		t.Fatal(err)
	}
	other := sql.OpenDB(connector)
	defer other.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	evidence := schema.UpgradeEvidence{Note: "concurrent owned fixture reviewed", BackupSHA256: strings.Repeat("a", 64), NotificationZone: "UTC"}
	entered, release := make(chan struct{}), make(chan struct{})
	original := plan.ValidateSource
	plan.ValidateSource = func(ctx context.Context, conn *sql.Conn, evidence schema.UpgradeEvidence) error {
		if err := original(ctx, conn, evidence); err != nil {
			return err
		}
		close(entered)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	done := make(chan error, 1)
	go func() { done <- schema.UpgradeBaseline(ctx, db, manifest, plan, evidence) }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("known source did not reach deterministic preflight barrier")
	}
	if err := schema.UpgradeBaseline(ctx, other, manifest, plan, evidence); !errors.Is(err, schema.ErrLocked) {
		close(release)
		t.Fatal("concurrent known upgrade was not locked out")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal("single winning known-source upgrade failed")
	}
}

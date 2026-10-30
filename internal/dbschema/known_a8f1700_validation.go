package dbschema

import (
	"context"
	"database/sql"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

var knownA8Digest = regexp.MustCompile("^[0-9a-f]{64}$")

// All callbacks are SELECT/SHOW only. Their caller holds the shared schema lock
// and has stopped every old writer, worker and migration job before execution.
func validateKnownA8Catalogue(ctx context.Context, conn *sql.Conn, expected map[string]legacyDefinition, allowOwnLedger bool) error {
	if conn == nil {
		return ErrKnownA8Read
	}
	var database sql.NullString
	var account string
	if err := conn.QueryRowContext(ctx, "SELECT DATABASE(), CURRENT_USER()").Scan(&database, &account); err != nil || !database.Valid || database.String == "" {
		return ErrKnownA8Read
	}
	separator := strings.LastIndex(account, "@")
	if separator < 1 {
		return ErrKnownA8Read
	}
	grantee := "'" + account[:separator] + "'@'" + account[separator+1:] + "'"
	var visible bool
	if err := conn.QueryRowContext(ctx, `SELECT EXISTS(
 SELECT 1 FROM information_schema.SCHEMA_PRIVILEGES WHERE TABLE_SCHEMA=DATABASE() AND GRANTEE=? AND PRIVILEGE_TYPE IN ('SELECT','REFERENCES')
 UNION ALL SELECT 1 FROM information_schema.USER_PRIVILEGES WHERE GRANTEE=? AND PRIVILEGE_TYPE IN ('SELECT','REFERENCES')
 )`, grantee, grantee).Scan(&visible); err != nil {
		return ErrKnownA8Read
	}
	if !visible {
		return ErrKnownA8Schema
	}
	rows, err := conn.QueryContext(ctx, "SELECT TABLE_NAME,TABLE_TYPE FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() ORDER BY TABLE_NAME LIMIT ?", len(expected)+2)
	if err != nil {
		return ErrKnownA8Read
	}
	var names []string
	ledger := false
	valid := true
	for rows.Next() {
		var name, kind string
		if err := rows.Scan(&name, &kind); err != nil {
			rows.Close()
			return ErrKnownA8Read
		}
		if kind != "BASE TABLE" {
			valid = false
		}
		if name == "cloudops_schema_history" && allowOwnLedger {
			ledger = true
			continue
		}
		if _, exists := expected[name]; !exists {
			valid = false
		}
		names = append(names, name)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return ErrKnownA8Read
	}
	if !valid || len(names) != len(expected) || (allowOwnLedger && !ledger) {
		return ErrKnownA8Schema
	}
	for _, name := range names {
		var returned, ddl string
		if err := conn.QueryRowContext(ctx, "SHOW CREATE TABLE `"+name+"`").Scan(&returned, &ddl); err != nil {
			return ErrKnownA8Read
		}
		parsed, actual, err := parseLegacyCreate(ddl)
		if err != nil || name != returned || parsed != name || actual.canonical() != expected[name].canonical() {
			return ErrKnownA8Schema
		}
	}
	return nil
}

func validateKnownA8Source(ctx context.Context, conn *sql.Conn, expected map[string]legacyDefinition, evidence schema.UpgradeEvidence) error {
	zone := evidence.NotificationZone
	if zone == "" || strings.TrimSpace(zone) != zone || strings.EqualFold(zone, "Local") || len(zone) > 100 {
		return ErrClockInvalidZone
	}
	if _, err := time.LoadLocation(zone); err != nil {
		return ErrClockInvalidZone
	}
	if err := validateKnownA8Catalogue(ctx, conn, expected, false); err != nil {
		return err
	}
	var enabledUnsupported bool
	if err := conn.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM cl_tree_cloud_account WHERE provider IN (4,5,6) AND status=1 LIMIT 1)").Scan(&enabledUnsupported); err != nil {
		return ErrKnownA8Read
	}
	if enabledUnsupported {
		return ErrKnownA8Provider
	}
	if evidence.HistoricalTimeAttested {
		return nil
	}
	// Every historical DATETIME interpretation needs separate attestation, not
	// just scheduled notifications. No historical values are ever converted.
	names := make([]string, 0, len(expected))
	for name := range expected {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		var conditions []string
		for _, column := range expected[name].columns {
			parts := strings.SplitN(column, " ", 3)
			if len(parts) >= 2 && strings.HasPrefix(strings.ToLower(parts[1]), "datetime") {
				conditions = append(conditions, parts[0]+" IS NOT NULL")
			}
		}
		if len(conditions) == 0 {
			continue
		}
		var historical bool
		query := "SELECT EXISTS(SELECT 1 FROM `" + name + "` WHERE " + strings.Join(conditions, " OR ") + " LIMIT 1)"
		if err := conn.QueryRowContext(ctx, query).Scan(&historical); err != nil {
			return ErrKnownA8Read
		}
		if historical {
			return ErrKnownA8Time
		}
	}
	return nil
}

func validateKnownA8Target(ctx context.Context, conn *sql.Conn, target schema.Migration, plan schema.BaselineUpgrade, evidence schema.UpgradeEvidence) error {
	expected, err := expectedLegacyDefinitions(baselineV1SQL)
	if err != nil {
		return err
	}
	if err := validateKnownA8Catalogue(ctx, conn, expected, true); err != nil {
		return err
	}
	// Ignore only this operation's started ledger after checking complete audit
	// provenance; never blanket-ignore an arbitrary history table or fake applied.
	var matching, historyRows uint64
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM cloudops_schema_history
 WHERE version=1 AND name=? AND checksum=? AND state='started' AND origin='upgraded'
 AND source_revision=? AND source_checksum=? AND upgrade_checksum=? AND upgrade_actor=CURRENT_USER()
 AND upgrade_note=? AND backup_sha256=? AND notification_zone=? AND historical_time_attested=?`,
		target.Name, target.Checksum, plan.Source, plan.SourceChecksum, plan.ImplementationChecksum,
		evidence.Note, evidence.BackupSHA256, evidence.NotificationZone, evidence.HistoricalTimeAttested).Scan(&matching); err != nil {
		return ErrKnownA8Read
	}
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM cloudops_schema_history").Scan(&historyRows); err != nil || matching != 1 || historyRows != 1 {
		return ErrKnownA8Target
	}
	var invalidDefaults, unrepairedAudit uint64
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM cl_workorder_notification WHERE is_default=0").Scan(&invalidDefaults); err != nil {
		return ErrKnownA8Read
	}
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM cl_tree_cloud_resource_change_log
 WHERE LOWER(TRIM(field_name)) IN ('key','password') AND (old_value <> '[REDACTED]' OR new_value <> '[REDACTED]')`).Scan(&unrepairedAudit); err != nil {
		return ErrKnownA8Read
	}
	var zone string
	if err := conn.QueryRowContext(ctx, "SELECT time_zone FROM cl_workorder_notification_clock WHERE id=1").Scan(&zone); err != nil || zone != evidence.NotificationZone || invalidDefaults != 0 || unrepairedAudit != 0 {
		return ErrKnownA8Target
	}
	return nil
}

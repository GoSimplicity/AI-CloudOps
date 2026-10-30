package schema

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrUpgrade = errors.New("invalid baseline upgrade request")

type UpgradeEvidence struct {
	Note                   string
	BackupSHA256           string
	NotificationZone       string
	HistoricalTimeAttested bool
}

type BaselineUpgrade struct {
	Source                 string
	SourceChecksum         string
	ImplementationChecksum string
	ValidateSource         func(context.Context, *sql.Conn, UpgradeEvidence) error
	Apply                  func(context.Context, *sql.Conn, UpgradeEvidence) error
	ValidateTarget         func(context.Context, *sql.Conn, UpgradeEvidence) error
}

func upgradeHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validateUpgrade(plan BaselineUpgrade, evidence UpgradeEvidence) error {
	if !upgradeHex(plan.Source, 40) || plan.Source != strings.ToLower(plan.Source) || !upgradeHex(plan.SourceChecksum, 64) || !upgradeHex(plan.ImplementationChecksum, 64) || plan.ValidateSource == nil || plan.Apply == nil || plan.ValidateTarget == nil {
		return ErrUpgrade
	}
	if !utf8.ValidString(evidence.Note) || strings.TrimSpace(evidence.Note) == "" || len(evidence.Note) > 1000 || !upgradeHex(evidence.BackupSHA256, 64) {
		return ErrUpgrade
	}
	zone := evidence.NotificationZone
	if zone == "" || len(zone) > 100 || strings.TrimSpace(zone) != zone || strings.EqualFold(zone, "Local") {
		return ErrUpgrade
	}
	if _, err := time.LoadLocation(zone); err != nil {
		return ErrUpgrade
	}
	return nil
}

// UpgradeBaseline runs a trusted, frozen upgrade to the sole version-1 target.
// The plan provider must bind its actual source, implementation, and SQL to the
// supplied checksums. This engine validates their form, not their provenance.
// Stop all writers first: advisory locks only coordinate participating tools.
//
// Source/target validators must be SELECT-only. Apply must use only conn, never
// call Migration.Apply, and finish/rollback its own DML transactions with defer
// even on panic. MySQL DDL can commit implicitly and is never rolled back here.
// Any non-applied or empty existing ledger blocks a new attempt. An already
// applied target is verified without changing its original origin or evidence.
func UpgradeBaseline(ctx context.Context, db *sql.DB, migrations []Migration, plan BaselineUpgrade, evidence UpgradeEvidence) error {
	if len(migrations) != 1 || Validate(migrations) != nil {
		return ErrManifest
	}
	if err := validateUpgrade(plan, evidence); err != nil {
		return err
	}
	ddl, err := upgradeLedgerDDL()
	if err != nil {
		return err
	}
	return WithMigrationLock(ctx, db, func(conn *sql.Conn) error {
		var exists bool
		if err := conn.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'cloudops_schema_history')").Scan(&exists); err != nil {
			return ErrPreflight
		}
		if exists {
			return VerifyConnection(ctx, conn, migrations)
		}
		if err := runUpgradeCallback(ctx, conn, evidence, plan.ValidateSource); err != nil {
			return ErrPreflight
		}
		if _, err := conn.ExecContext(ctx, ddl); err != nil {
			return ErrUncertain
		}
		target := migrations[0]
		inserted, err := conn.ExecContext(ctx, `INSERT INTO cloudops_schema_history
 (version,name,checksum,state,origin,source_revision,source_checksum,upgrade_checksum,upgrade_actor,upgrade_note,backup_sha256,notification_zone,historical_time_attested)
 VALUES (?,?,?,'started','upgraded',?,?,?,CURRENT_USER(),?,?,?,?)`, target.Version, target.Name, target.Checksum, plan.Source, plan.SourceChecksum, plan.ImplementationChecksum, evidence.Note, evidence.BackupSHA256, evidence.NotificationZone, evidence.HistoricalTimeAttested)
		if err != nil || !upgradeOneRow(inserted) {
			return ErrUncertain
		}
		if err := runUpgradeCallback(ctx, conn, evidence, plan.Apply); err != nil {
			return markUpgradeFailed(conn, target.Version)
		}
		// Target validation deliberately sees the durable started record and all
		// upgrade audit fields, never a prematurely claimed applied version.
		if err := runUpgradeCallback(ctx, conn, evidence, plan.ValidateTarget); err != nil {
			return markUpgradeFailed(conn, target.Version)
		}
		applied, err := conn.ExecContext(ctx, "UPDATE cloudops_schema_history SET state = 'applied', finished_at = CURRENT_TIMESTAMP(6) WHERE version = ? AND state = 'started'", target.Version)
		if err != nil || !upgradeOneRow(applied) {
			return ErrUncertain
		}
		return nil
	})
}

func upgradeOneRow(result sql.Result) bool {
	if result == nil {
		return false
	}
	count, err := result.RowsAffected()
	return err == nil && count == 1
}

func runUpgradeCallback(ctx context.Context, conn *sql.Conn, evidence UpgradeEvidence, callback func(context.Context, *sql.Conn, UpgradeEvidence) error) (result error) {
	defer func() {
		if recover() != nil {
			result = ErrDirty
		}
	}()
	// The callback's private error or panic payload never crosses the engine API.
	if err := callback(ctx, conn, evidence); err != nil {
		return ErrDirty
	}
	return nil
}

func markUpgradeFailed(conn *sql.Conn, version uint64) error {
	cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	failed, err := conn.ExecContext(cleanup, "UPDATE cloudops_schema_history SET state = 'failed', finished_at = CURRENT_TIMESTAMP(6) WHERE version = ? AND state = 'started'", version)
	if err != nil || !upgradeOneRow(failed) {
		return ErrUncertain
	}
	return ErrDirty
}

// Derive the upgrade audit variant from exactly the canonical body. No old
// ledger or adoption DDL is modified, and no IF NOT EXISTS masks an intervening
// administrator-created object after source validation.
func upgradeLedgerDDL() (string, error) {
	const prefix = "CREATE TABLE IF NOT EXISTS cloudops_schema_history ("
	const suffix = "\n) ENGINE=InnoDB"
	if !strings.HasPrefix(ledgerDDL, prefix) || !strings.HasSuffix(ledgerDDL, suffix) {
		return "", ErrManifest
	}
	body := strings.TrimSuffix(strings.TrimPrefix(ledgerDDL, prefix), suffix)
	return "CREATE TABLE cloudops_schema_history (" + body + `,
 origin VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT 'migrated',
 source_revision CHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
 source_checksum CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
 upgrade_checksum CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
 upgrade_actor VARCHAR(288) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NULL,
 upgrade_note VARCHAR(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NULL,
 backup_sha256 CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
 notification_zone VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NULL,
 historical_time_attested TINYINT(1) NULL
) ENGINE=InnoDB`, nil
}

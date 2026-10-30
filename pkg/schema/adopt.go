package schema

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"
)

var ErrAdoption = errors.New("invalid baseline adoption request")

// AdoptBaseline records an already-existing version-1 schema after a trusted,
// complete read-only validator proves its structure and repair invariants. It
// never calls Migration.Apply and never writes application tables. Stop all
// application writers and migration jobs before use: advisory locks cannot stop
// unrelated administrator DDL or clients that do not participate in this lock.
//
// The validator must use only conn, have complete schema metadata visibility,
// perform SELECT only, and leave no transaction or session changes behind. Its
// errors and panic values are never returned to the caller. A successful repeat
// verifies existing history without rerunning validation or changing its audit.
func AdoptBaseline(ctx context.Context, db *sql.DB, migrations []Migration, validate func(context.Context, *sql.Conn) error, note string) error {
	if len(migrations) != 1 || Validate(migrations) != nil {
		return ErrManifest
	}
	if validate == nil || !utf8.ValidString(note) || strings.TrimSpace(note) == "" || len(note) > 1000 {
		return ErrAdoption
	}
	ddl, err := adoptionLedgerDDL()
	if err != nil {
		return err
	}
	return WithMigrationLock(ctx, db, func(conn *sql.Conn) error {
		var exists bool
		if err := conn.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'cloudops_schema_history')").Scan(&exists); err != nil {
			return ErrPreflight
		}
		if exists {
			// An empty ledger can be evidence of a failed adoption. Never turn
			// it into a new attempt, and never add audit columns to an old ledger.
			return VerifyConnection(ctx, conn, migrations)
		}
		if err := validate(ctx, conn); err != nil {
			return ErrPreflight
		}
		if _, err := conn.ExecContext(ctx, ddl); err != nil {
			return ErrUncertain
		}
		baseline := migrations[0]
		result, err := conn.ExecContext(ctx, `INSERT INTO cloudops_schema_history
 (version, name, checksum, state, started_at, finished_at, origin, adoption_note, adopted_by)
 VALUES (?, ?, ?, 'applied', CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), 'adopted', ?, CURRENT_USER())`, baseline.Version, baseline.Name, baseline.Checksum, note)
		if err != nil {
			return ErrUncertain
		}
		count, err := result.RowsAffected()
		if err != nil || count != 1 {
			return ErrUncertain
		}
		return nil
	})
}

// Derive the audit variant from the engine's canonical ledger rather than
// duplicating its columns. The regular DDL cannot be used directly: it has no
// adoption evidence columns and IF NOT EXISTS would conceal an unrelated table
// created between validation and DDL. Unexpected canonical changes fail closed.
func adoptionLedgerDDL() (string, error) {
	const prefix = "CREATE TABLE IF NOT EXISTS cloudops_schema_history ("
	const suffix = "\n) ENGINE=InnoDB"
	if !strings.HasPrefix(ledgerDDL, prefix) || !strings.HasSuffix(ledgerDDL, suffix) {
		return "", ErrManifest
	}
	body := strings.TrimSuffix(strings.TrimPrefix(ledgerDDL, prefix), suffix)
	return "CREATE TABLE cloudops_schema_history (" + body + `,
 origin VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT 'migrated',
 adoption_note VARCHAR(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NULL,
 adopted_by VARCHAR(288) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NULL
) ENGINE=InnoDB`, nil
}

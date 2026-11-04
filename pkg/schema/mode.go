package schema

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrMigrationMode = errors.New("migration confirmation does not match locked database state")

// MigrateFresh authorizes only an empty database, including an empty existing
// ledger. A fully applied current manifest is an idempotent read-only outcome.
// The mode is checked on the lock-owning connection before any DDL/ledger write.
func MigrateFresh(ctx context.Context, db *sql.DB, migrations []Migration) error {
	return migrateConfirmed(ctx, db, migrations, false)
}

// MigratePending authorizes only a clean nonempty applied prefix, never a fresh
// database. Callers must stop all writers. This does not change released Migrate.
func MigratePending(ctx context.Context, db *sql.DB, migrations []Migration) error {
	return migrateConfirmed(ctx, db, migrations, true)
}
func migrateConfirmed(ctx context.Context, db *sql.DB, migrations []Migration, pending bool) error {
	if err := Validate(migrations); err != nil {
		return err
	}
	return WithMigrationLock(ctx, db, func(conn *sql.Conn) error {
		var exists bool
		if err := conn.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'cloudops_schema_history')").Scan(&exists); err != nil {
			return ErrPreflight
		}
		var records []Record
		if exists {
			var err error
			records, err = history(ctx, conn, len(migrations)+1)
			if err != nil {
				return fmt.Errorf("%w: schema history unavailable", ErrPreflight)
			}
			if err := inspect(migrations, records); err != nil {
				return err
			}
			if len(records) == len(migrations) {
				return nil
			}
		}
		if pending {
			if len(records) == 0 {
				return ErrMigrationMode
			}
		} else {
			if len(records) != 0 {
				return ErrMigrationMode
			}
			var business int64
			if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name <> 'cloudops_schema_history'").Scan(&business); err != nil {
				return ErrPreflight
			}
			if business != 0 {
				return ErrMigrationMode
			}
			if !exists {
				if _, err := conn.ExecContext(ctx, ledgerDDL); err != nil {
					return fmt.Errorf("%w: cannot initialize schema history", ErrUncertain)
				}
			}
		}
		return applyPendingMigrations(ctx, conn, migrations, records)
	})
}

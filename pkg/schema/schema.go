// Package schema runs explicit, forward-only MySQL migrations. MySQL DDL can
// commit implicitly: a failed migration is never automatically retried or rolled
// back. Operators must inspect the database and repair forward before proceeding.
package schema

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

var (
	ErrManifest  = errors.New("invalid migration manifest")
	ErrDrift     = errors.New("schema history differs from migration manifest")
	ErrPending   = errors.New("schema migrations are pending")
	ErrDirty     = errors.New("schema migration requires operator reconciliation")
	ErrLocked    = errors.New("schema migration lock unavailable")
	ErrPreflight = errors.New("schema preflight failed")
	ErrUncertain = errors.New("schema operation outcome uncertain; inspect database before retry")
)

// Migration.Checksum identifies the complete immutable implementation, including
// data repairs. Apply must use only conn, must not close it, and must not open a
// transaction: a MySQL transaction cannot provide DDL rollback guarantees.
// Checksum must change if implementation or its referenced model definitions change.
type Migration struct {
	Version  uint64
	Name     string
	Checksum string
	Apply    func(context.Context, *sql.Conn) error
}

func Checksum(source []byte) string {
	sum := sha256.Sum256(source)
	return hex.EncodeToString(sum[:])
}

// SQLMigration binds checksum and execution to the same ordered SQL source.
// Each element is one statement; multiStatements need not be enabled in the DSN.
func SQLMigration(version uint64, name string, statements ...string) Migration {
	if len(statements) == 0 {
		return Migration{Version: version, Name: name}
	}
	copied := append([]string(nil), statements...)
	source := ""
	for _, statement := range copied {
		source += fmt.Sprintf("%d:%s", len(statement), statement)
	}
	return Migration{Version: version, Name: name, Checksum: Checksum([]byte(source)), Apply: func(ctx context.Context, conn *sql.Conn) error {
		for _, statement := range copied {
			if _, err := conn.ExecContext(ctx, statement); err != nil {
				return err
			}
		}
		return nil
	}}
}

type Record struct {
	Version  uint64
	Name     string
	Checksum string
	State    string
}

// Validate checks manifest structure without contacting a database. Versions are
// contiguous from 1; removing, reordering, or editing released versions is invalid.
func Validate(migrations []Migration) error {
	if len(migrations) == 0 {
		return ErrManifest
	}
	for i, migration := range migrations {
		decoded, err := hex.DecodeString(migration.Checksum)
		if migration.Version != uint64(i+1) || migration.Name == "" || len(migration.Name) > 190 || len(migration.Checksum) != 64 || err != nil || len(decoded) != 32 || migration.Apply == nil {
			return fmt.Errorf("%w at manifest index %d", ErrManifest, i)
		}
	}
	return nil
}

func inspect(migrations []Migration, records []Record) error {
	if len(records) > len(migrations) {
		return ErrDrift
	}
	for i, record := range records {
		expected := migrations[i]
		if record.Version != expected.Version || record.Name != expected.Name || record.Checksum != expected.Checksum {
			return fmt.Errorf("%w at version %d", ErrDrift, record.Version)
		}
		if record.State != "applied" {
			return fmt.Errorf("%w at version %d", ErrDirty, record.Version)
		}
	}
	return nil
}

// Verify performs SELECTs only. Runtime credentials need only SELECT on the
// ledger; missing history, pending versions and dirty records block startup.
// This verifies version history, not arbitrary out-of-band table/column changes.
func Verify(ctx context.Context, db *sql.DB, migrations []Migration) error {
	if err := Validate(migrations); err != nil {
		return err
	}
	if db == nil {
		return ErrPreflight
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("%w: connection unavailable", ErrPreflight)
	}
	defer conn.Close()
	return VerifyConnection(ctx, conn, migrations)
}

// VerifyConnection performs the same bounded, read-only history verification on
// an already-owned connection. Callers can verify while holding the migration
// lock without acquiring a second pool connection.
func VerifyConnection(ctx context.Context, conn *sql.Conn, migrations []Migration) error {
	if err := Validate(migrations); err != nil {
		return err
	}
	if conn == nil {
		return ErrPreflight
	}
	records, err := history(ctx, conn, len(migrations)+1)
	if err != nil {
		return fmt.Errorf("%w: schema history unavailable", ErrPreflight)
	}
	if err := inspect(migrations, records); err != nil {
		return err
	}
	if len(records) != len(migrations) {
		return ErrPending
	}
	return nil
}

// Migrate is exclusively for the separately provisioned migration role. It holds
// one database-scoped advisory lock on the same physical connection used for all
// history and migration operations. No transaction is used for DDL.
// A non-applied version blocks all future runs, including after a process crash.
func Migrate(ctx context.Context, db *sql.DB, migrations []Migration) (result error) {
	if err := Validate(migrations); err != nil {
		return err
	}
	return WithMigrationLock(ctx, db, func(conn *sql.Conn) error {
		return migrateConnection(ctx, conn, migrations)
	})
}

// WithMigrationLock serializes explicit schema and initialization operations on
// one primary-only MySQL connection. The callback must use only this connection,
// finish or roll back every transaction, and must not close the connection.
// This function does not initialize history or execute DDL itself. Lost lock
// acknowledgements discard the physical session; callback panics are sanitized.
func WithMigrationLock(ctx context.Context, db *sql.DB, run func(*sql.Conn) error) (result error) {
	if run == nil {
		return ErrPreflight
	}
	if db == nil {
		return ErrPreflight
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("%w: connection unavailable", ErrPreflight)
	}
	defer conn.Close()
	var database string
	var autocommit, readOnly int
	if err := conn.QueryRowContext(ctx, "SELECT LOWER(DATABASE()), @@session.autocommit, @@global.read_only").Scan(&database, &autocommit, &readOnly); err != nil || database == "" || autocommit != 1 || readOnly != 0 {
		return ErrPreflight
	}
	// A stable hash fits MySQL's 64-character lock name limit and prevents two
	// applications using this ledger in the same database from racing each other.
	// Normalize database casing so case-insensitive MySQL installations share a lock.
	// Advisory locks are local to one MySQL server; use a primary-only DSN, never
	// a multi-writer router, for migration coordination.
	lockName := "cloudops.schema." + Checksum([]byte(database))[:40]
	var acquired sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", lockName).Scan(&acquired); err != nil || !acquired.Valid {
		// A lost acknowledgement may hide a successfully acquired lock. Close
		// the physical connection rather than returning it to the pool.
		_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		return errors.Join(ErrLocked, ErrUncertain)
	}
	if acquired.Int64 != 1 {
		return ErrLocked
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var released sql.NullInt64
		err := conn.QueryRowContext(cleanup, "SELECT RELEASE_LOCK(?)", lockName).Scan(&released)
		if err != nil || !released.Valid || released.Int64 != 1 {
			// A pooled session with a possibly-held lock must never be reused.
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
			result = errors.Join(result, ErrUncertain)
		}
	}()
	defer func() {
		if recover() != nil {
			// The callback may have left an open transaction. Discard the
			// physical session so neither that transaction nor its lock is reused.
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
			result = errors.Join(result, ErrUncertain)
		}
	}()
	return run(conn)
}

func migrateConnection(ctx context.Context, conn *sql.Conn, migrations []Migration) error {
	if _, err := conn.ExecContext(ctx, ledgerDDL); err != nil {
		return fmt.Errorf("%w: cannot initialize schema history", ErrUncertain)
	}
	records, err := history(ctx, conn, len(migrations)+1)
	if err != nil {
		return fmt.Errorf("%w: cannot read schema history", ErrPreflight)
	}
	if err := inspect(migrations, records); err != nil {
		return err
	}
	for _, migration := range migrations[len(records):] {
		// Record intent before DDL. If this insert's acknowledgement is lost, no DDL
		// is attempted; any persisted started record still requires reconciliation.
		if _, err := conn.ExecContext(ctx, "INSERT INTO cloudops_schema_history (version, name, checksum, state) VALUES (?, ?, ?, 'started')", migration.Version, migration.Name, migration.Checksum); err != nil {
			return fmt.Errorf("%w at version %d", ErrUncertain, migration.Version)
		}
		if err := applyMigration(ctx, conn, migration.Apply); err != nil {
			// Use a fresh bounded context so a cancelled migration can still be marked.
			// Failed means the callback returned an error, not that DDL was rolled back.
			cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			marked, markErr := conn.ExecContext(cleanup, "UPDATE cloudops_schema_history SET state = 'failed', finished_at = CURRENT_TIMESTAMP(6) WHERE version = ? AND state = 'started'", migration.Version)
			cancel()
			var count int64
			if markErr == nil {
				count, markErr = marked.RowsAffected()
			}
			if markErr != nil || count != 1 {
				return fmt.Errorf("%w at version %d", ErrUncertain, migration.Version)
			}
			return fmt.Errorf("%w at version %d (DDL may be partially committed)", ErrDirty, migration.Version)
		}
		updated, err := conn.ExecContext(ctx, "UPDATE cloudops_schema_history SET state = 'applied', finished_at = CURRENT_TIMESTAMP(6) WHERE version = ? AND state = 'started'", migration.Version)
		if err != nil {
			return fmt.Errorf("%w at version %d", ErrUncertain, migration.Version)
		}
		count, err := updated.RowsAffected()
		if err != nil || count != 1 {
			return fmt.Errorf("%w at version %d", ErrUncertain, migration.Version)
		}
	}
	return nil
}

// Callback panics are dirty outcomes too. Do not include panic values in errors:
// migration callbacks may have SQL or credentials in their local error objects.
func applyMigration(ctx context.Context, conn *sql.Conn, apply func(context.Context, *sql.Conn) error) (result error) {
	defer func() {
		if recover() != nil {
			result = ErrDirty
		}
	}()
	return apply(ctx, conn)
}

func history(ctx context.Context, conn *sql.Conn, limit int) ([]Record, error) {
	rows, err := conn.QueryContext(ctx, "SELECT version, name, checksum, state FROM cloudops_schema_history ORDER BY version LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []Record
	for rows.Next() {
		var record Record
		if err := rows.Scan(&record.Version, &record.Name, &record.Checksum, &record.State); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

const ledgerDDL = `CREATE TABLE IF NOT EXISTS cloudops_schema_history (
 version BIGINT UNSIGNED NOT NULL PRIMARY KEY,
 name VARCHAR(190) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
 checksum CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 state VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 started_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 finished_at TIMESTAMP(6) NULL
) ENGINE=InnoDB`

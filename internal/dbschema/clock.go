package dbschema

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

var (
	ErrClockInvalidZone = errors.New("notification time zone must be explicit UTC or a valid IANA zone, excluding Local")
	ErrClockMissing     = errors.New("notification time zone marker is missing; explicit initialization is required")
	ErrClockMismatch    = errors.New("notification time zone marker differs from configured zone")
	ErrClockHistory     = errors.New("historical notification time data exists; explicit reconciliation is required")
	ErrClockDatabase    = errors.New("notification time zone operation failed; database details withheld")
)

// ValidateClockZone validates an explicitly chosen DATETIME interpretation. It
// never derives a zone from the machine or converts any stored time values.
func ValidateClockZone(zone string) error { return validateClockZone(zone, time.LoadLocation) }

func validateClockZone(zone string, load func(string) (*time.Location, error)) error {
	if zone == "" || len(zone) > 100 || strings.TrimSpace(zone) != zone || strings.EqualFold(zone, "Local") {
		return ErrClockInvalidZone
	}
	if _, err := load(zone); err != nil {
		return ErrClockInvalidZone
	}
	return nil
}

// VerifyClock performs SELECT only. Missing markers and tables fail closed; the
// runtime role must not initialize a marker or silently accept another zone.
func VerifyClock(ctx context.Context, db *sql.DB, zone string) error {
	if err := ValidateClockZone(zone); err != nil {
		return err
	}
	if db == nil {
		return ErrClockDatabase
	}
	var stored string
	err := db.QueryRowContext(ctx, "SELECT time_zone FROM cl_workorder_notification_clock WHERE id = 1").Scan(&stored)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrClockMissing
	}
	if err != nil {
		return ErrClockDatabase
	}
	if stored != zone {
		return ErrClockMismatch
	}
	return nil
}

// InitializeFreshClock is an explicit maintenance operation, never a runtime
// startup hook. Stop all business writers before calling it: advisory locks
// coordinate migration tools, not arbitrary application DML. Existing identical
// markers are verified without rewriting them. A missing marker can be inserted
// only when no notification events or scheduled notification values exist.
// No table is created and no historical DATETIME is converted.
func InitializeFreshClock(ctx context.Context, db *sql.DB, zone string) error {
	if err := ValidateClockZone(zone); err != nil {
		return err
	}
	if db == nil {
		return ErrClockDatabase
	}
	manifest, err := Manifest()
	if err != nil {
		return err
	}
	return schema.WithMigrationLock(ctx, db, func(conn *sql.Conn) error {
		if err := schema.VerifyConnection(ctx, conn, manifest); err != nil {
			return err
		}
		return initializeClockTransaction(ctx, conn, zone)
	})
}

func initializeClockTransaction(ctx context.Context, conn *sql.Conn, zone string) (result error) {
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return ErrClockDatabase
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			// Do not return a physical session with an uncertain open transaction to
			// the pool. The shared lock helper also treats lost lock release as unsafe.
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
			result = errors.Join(result, schema.ErrUncertain)
		}
	}()
	var stored string
	err = tx.QueryRowContext(ctx, "SELECT time_zone FROM cl_workorder_notification_clock WHERE id = 1 FOR UPDATE").Scan(&stored)
	missing := errors.Is(err, sql.ErrNoRows)
	if err != nil && !missing {
		return ErrClockDatabase
	}
	// Query both tables even when a marker already exists: a dropped table is a
	// failure rather than proof that there is no history to reconcile.
	var events, scheduled bool
	if err := tx.QueryRowContext(ctx, `SELECT
 EXISTS(SELECT 1 FROM cl_workorder_notification_event LIMIT 1),
 EXISTS(SELECT 1 FROM cl_workorder_notification WHERE scheduled_time IS NOT NULL LIMIT 1)`).Scan(&events, &scheduled); err != nil {
		return ErrClockDatabase
	}
	if !missing {
		if stored != zone {
			return ErrClockMismatch
		}
		if err := tx.Commit(); err != nil {
			return errors.Join(ErrClockDatabase, schema.ErrUncertain)
		}
		return nil
	}
	if events || scheduled {
		return ErrClockHistory
	}
	if _, err := tx.ExecContext(ctx, "INSERT IGNORE INTO cl_workorder_notification_clock (id, time_zone) VALUES (1, ?)", zone); err != nil {
		return ErrClockDatabase
	}
	if err := tx.QueryRowContext(ctx, "SELECT time_zone FROM cl_workorder_notification_clock WHERE id = 1 FOR UPDATE").Scan(&stored); err != nil {
		return ErrClockDatabase
	}
	if stored != zone {
		return ErrClockMismatch
	}
	if err := tx.Commit(); err != nil {
		return errors.Join(ErrClockDatabase, schema.ErrUncertain)
	}
	return nil
}

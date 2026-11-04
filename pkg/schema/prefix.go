package schema

import (
	"context"
	"database/sql"
	"fmt"
)

// VerifyAppliedPrefixConnection establishes that at least one complete released
// version is applied and all present history is an exact, clean manifest prefix.
// It permits pending later versions solely for explicit migration preflight; it
// must never replace VerifyConnection at runtime. Empty history is not a baseline.
func VerifyAppliedPrefixConnection(ctx context.Context, conn *sql.Conn, migrations []Migration) error {
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
	if len(records) == 0 {
		return ErrPending
	}
	return nil
}

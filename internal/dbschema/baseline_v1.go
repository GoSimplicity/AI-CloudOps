package dbschema

// This file and baseline_v1.json are immutable version-1 release inputs. The
// checksum binds the exact executable wrapper source and ordered SQL. Future
// migrations must use separate files instead of editing this implementation.
import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

//go:embed baseline_v1.json
var baselineV1SQL []byte

//go:embed baseline_v1.go
var baselineV1Implementation []byte

var ErrLegacy = errors.New("unversioned database is not empty; explicit legacy adoption is required")

func baselineV1Migration(snapshot, implementation []byte) (schema.Migration, error) {
	var statements []string
	if err := json.Unmarshal(snapshot, &statements); err != nil || len(statements) == 0 || len(implementation) == 0 {
		return schema.Migration{}, fmt.Errorf("%w: embedded baseline invalid", schema.ErrManifest)
	}
	for _, statement := range statements {
		if strings.TrimSpace(statement) == "" {
			return schema.Migration{}, schema.ErrManifest
		}
	}
	migration := schema.SQLMigration(1, "initial_mysql_schema", statements...)
	sqlChecksum := migration.Checksum
	migration.Checksum = schema.Checksum([]byte(fmt.Sprintf("%d:%s%d:%s", len(implementation), implementation, len(sqlChecksum), sqlChecksum)))
	applySQL := migration.Apply
	migration.Apply = func(ctx context.Context, conn *sql.Conn) error {
		// Recheck on the lock-owning physical connection. An earlier CLI preflight
		// cannot replace this guard because another process may create tables later.
		if err := PreflightFresh(ctx, conn); err != nil {
			return err
		}
		return applySQL(ctx, conn)
	}
	return migration, nil
}

// PreflightFresh performs SELECT only and rejects every existing business table
// or view, including tables unknown to this release. The engine's own history
// table is the sole exception. Call this only when version 1 is pending; an
// already migrated database is checked with schema.Verify instead. This baseline
// does not adopt or upgrade any pre-existing, unversioned production database.
func PreflightFresh(ctx context.Context, conn *sql.Conn) error {
	if conn == nil {
		return schema.ErrPreflight
	}
	var database sql.NullString
	if err := conn.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&database); err != nil || !database.Valid || database.String == "" {
		return schema.ErrPreflight
	}
	var count int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name <> 'cloudops_schema_history'").Scan(&count); err != nil {
		return schema.ErrPreflight
	}
	if count != 0 {
		return ErrLegacy
	}
	return nil
}

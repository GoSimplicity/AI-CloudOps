package dbschema

// Version 2 adds an explicit no-trigger gate around the immutable a8f1700
// implementation. Keep both the released implementation and its audit hash;
// do not edit its source/SQL snapshots to hide this safety correction.
import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

//go:embed known_a8f1700_safe_v2.go
var knownA8SafeV2Implementation []byte

var (
	ErrKnownA8TriggerVisibility = errors.New("known-source upgrade requires direct schema/global TRIGGER metadata visibility")
	ErrKnownA8Triggers          = errors.New("known-source upgrade refuses unrecognized database triggers")
	ErrKnownA8TriggerRead       = errors.New("known-source trigger inspection failed; database details withheld")
)

// KnownA8SafeUpgradeV2 keeps the original 15 SQL statements unchanged, and binds
// its guard implementation plus the complete prior implementation into a new
// provenance checksum. The CLI uses this plan for every new upgrade attempt.
func KnownA8SafeUpgradeV2() (schema.BaselineUpgrade, error) {
	legacy, err := KnownA8Upgrade()
	if err != nil {
		return schema.BaselineUpgrade{}, err
	}
	manifest, err := Manifest()
	if err != nil || len(manifest) != 1 {
		return schema.BaselineUpgrade{}, ErrKnownA8Snapshot
	}
	plan := legacy
	material := fmt.Sprintf("known-a8-trigger-safe-v2:%d:%s%d:%s", len(legacy.ImplementationChecksum), legacy.ImplementationChecksum, len(knownA8SafeV2Implementation), knownA8SafeV2Implementation)
	plan.ImplementationChecksum = schema.Checksum([]byte(material))
	plan.ValidateSource = func(ctx context.Context, conn *sql.Conn, evidence schema.UpgradeEvidence) error {
		if err := validateKnownA8NoTriggersV2(ctx, conn); err != nil {
			return err
		}
		return legacy.ValidateSource(ctx, conn, evidence)
	}
	plan.ValidateTarget = func(ctx context.Context, conn *sql.Conn, evidence schema.UpgradeEvidence) error {
		if err := validateKnownA8NoTriggersV2(ctx, conn); err != nil {
			return err
		}
		// The original closure captured the old checksum. Reuse the frozen strict
		// validator with this plan's new audit checksum rather than mismatching it.
		return validateKnownA8Target(ctx, conn, manifest[0], plan, evidence)
	}
	return plan, nil
}

func validateKnownA8NoTriggersV2(ctx context.Context, conn *sql.Conn) error {
	if conn == nil {
		return ErrKnownA8TriggerRead
	}
	var database sql.NullString
	var account string
	if err := conn.QueryRowContext(ctx, "SELECT DATABASE(), CURRENT_USER()").Scan(&database, &account); err != nil || !database.Valid || database.String == "" {
		return ErrKnownA8TriggerRead
	}
	separator := strings.LastIndex(account, "@")
	if separator < 1 {
		return ErrKnownA8TriggerVisibility
	}
	grantee := "'" + account[:separator] + "'@'" + account[separator+1:] + "'"
	var visible bool
	// REFERENCES exposes table metadata but not trigger definitions. A zero-row
	// trigger inventory is meaningful only with an independently proven direct
	// schema/global TRIGGER grant for the authenticated account, not a role name.
	if err := conn.QueryRowContext(ctx, `SELECT EXISTS(
 SELECT 1 FROM information_schema.SCHEMA_PRIVILEGES WHERE TABLE_SCHEMA=DATABASE() AND GRANTEE=? AND PRIVILEGE_TYPE='TRIGGER'
 UNION ALL SELECT 1 FROM information_schema.USER_PRIVILEGES WHERE GRANTEE=? AND PRIVILEGE_TYPE='TRIGGER'
 )`, grantee, grantee).Scan(&visible); err != nil {
		return ErrKnownA8TriggerRead
	}
	if !visible {
		return ErrKnownA8TriggerVisibility
	}
	var triggers bool
	if err := conn.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=DATABASE() LIMIT 1)").Scan(&triggers); err != nil {
		return ErrKnownA8TriggerRead
	}
	if triggers {
		return ErrKnownA8Triggers
	}
	return nil
}

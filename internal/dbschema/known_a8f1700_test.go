package dbschema

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

func TestKnownA8FrozenInventoryAndSQLAreAdditive(t *testing.T) {
	source, err := knownA8Definitions()
	if err != nil || len(source) != 49 {
		t.Fatal("invalid known source inventory")
	}
	var statements []string
	if err := json.Unmarshal(knownA8SQLJSON, &statements); err != nil || len(statements) != 15 {
		t.Fatal("invalid immutable upgrade SQL")
	}
	counts := map[string]int{}
	for _, statement := range statements {
		upper := strings.ToUpper(statement)
		if strings.Contains(upper, "IF NOT EXISTS") || strings.Contains(upper, "FOREIGN_KEY_CHECKS") || strings.Contains(upper, " DROP ") || strings.Contains(upper, " MODIFY ") || strings.Contains(upper, " CHANGE ") {
			t.Fatal("upgrade contains non-reviewed or destructive operations")
		}
		switch {
		case strings.HasPrefix(statement, "CREATE TABLE "):
			name, _, err := parseLegacyCreate(statement)
			if err != nil {
				t.Fatal("new table is not a strict frozen definition")
			}
			if _, exists := source[name]; exists {
				t.Fatal("upgrade tries to recreate an existing source table")
			}
			counts["create"]++
		case strings.HasPrefix(statement, "ALTER TABLE "):
			counts["alter"]++
			counts["columns"] += strings.Count(statement, "ADD COLUMN ")
			counts["after"] += strings.Count(statement, " AFTER ")
			counts["indexes"] += strings.Count(statement, "ADD KEY ")
		case statement == legacyAuditRepair || statement == legacyDefaultRepair:
			counts["repair"]++
		default:
			t.Fatal("unknown upgrade statement")
		}
	}
	if counts["create"] != 9 || counts["alter"] != 4 || counts["columns"] != 14 || counts["after"] != 14 || counts["indexes"] != 2 || counts["repair"] != 2 {
		t.Fatalf("upgrade statement inventory changed: %v", counts)
	}
}

func TestKnownA8PlanDoesNotShareCallerMutableState(t *testing.T) {
	first, err := KnownA8Upgrade()
	if err != nil {
		t.Fatal(err)
	}
	original := first.ImplementationChecksum
	first.Source = "changed"
	first.Apply = func(context.Context, *sql.Conn, schema.UpgradeEvidence) error { return nil }
	next, err := KnownA8Upgrade()
	if err != nil || next.Source != KnownA8Source || next.ImplementationChecksum != original {
		t.Fatal("caller mutation affected subsequent frozen plan")
	}
}

func TestKnownA8ReleasedChecksums(t *testing.T) {
	plan, err := KnownA8Upgrade()
	if err != nil {
		t.Fatal(err)
	}
	if plan.SourceChecksum != "06ba5fbf1ea6132368446ab98e19226838f86b1721b517f6367a702b846ee1dc" || plan.ImplementationChecksum != "44e466f5ff9dce7e1f304107d2e7d150810d61b9c39e957a8a67f1901d901e67" {
		t.Fatal("immutable known-source inputs changed; release a new reviewed upgrade implementation")
	}
	manifest, err := Manifest()
	if err != nil || manifest[0].Checksum != "92d4474d148d35643dbf8ed4044e6734e2d9268c36b6738bcb26ef220afc0616" {
		t.Fatal("published target baseline changed")
	}
}

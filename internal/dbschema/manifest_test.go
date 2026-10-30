package dbschema

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

func TestManifest(t *testing.T) {
	tests := []struct {
		name                     string
		snapshot, implementation []byte
		want                     error
	}{
		{"released baseline", baselineV1SQL, baselineV1Implementation, nil},
		{"invalid JSON", []byte("not-json"), baselineV1Implementation, schema.ErrManifest},
		{"empty snapshot", []byte("[]"), baselineV1Implementation, schema.ErrManifest},
		{"null snapshot", []byte("null"), baselineV1Implementation, schema.ErrManifest},
		{"empty statement", []byte(`[" "]`), baselineV1Implementation, schema.ErrManifest},
		{"missing implementation", baselineV1SQL, nil, schema.ErrManifest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			migration, err := baselineV1Migration(test.snapshot, test.implementation)
			if !errors.Is(err, test.want) {
				t.Fatalf("got %v want %v", err, test.want)
			}
			if err == nil {
				if err := schema.Validate([]schema.Migration{migration}); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestBaselineChecksumBindsSQLAndImplementation(t *testing.T) {
	original, err := baselineV1Migration(baselineV1SQL, baselineV1Implementation)
	if err != nil {
		t.Fatal(err)
	}
	var statements []string
	if err := json.Unmarshal(baselineV1SQL, &statements); err != nil {
		t.Fatal(err)
	}
	statements[len(statements)-1] += " /* changed SQL */"
	changedSQL, err := json.Marshal(statements)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name                     string
		snapshot, implementation []byte
	}{
		{"SQL changed", changedSQL, baselineV1Implementation},
		{"implementation changed", baselineV1SQL, append(append([]byte(nil), baselineV1Implementation...), []byte("\n// changed implementation\n")...)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed, err := baselineV1Migration(test.snapshot, test.implementation)
			if err != nil {
				t.Fatal(err)
			}
			if changed.Checksum == original.Checksum {
				t.Fatal("changed release input did not change checksum")
			}
		})
	}
}

func TestBaselineStatementsAreForwardOnly(t *testing.T) {
	var statements []string
	if err := json.Unmarshal(baselineV1SQL, &statements); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, statement := range statements {
		if strings.Contains(strings.ToUpper(statement), "IF NOT EXISTS") || strings.Contains(strings.ToUpper(statement), "FOREIGN_KEY_CHECKS") {
			t.Fatal("baseline must neither adopt old tables nor disable foreign key checks")
		}
		switch {
		case strings.HasPrefix(statement, "CREATE TABLE "):
			counts["tables"]++
			if strings.Contains(statement, " FOREIGN KEY ") {
				t.Fatal("foreign key must be added after every table exists")
			}
		case strings.HasPrefix(statement, "ALTER TABLE "):
			counts["foreign keys"]++
		case strings.HasPrefix(statement, "UPDATE "):
			counts["repairs"]++
		default:
			t.Fatal("unexpected baseline statement type")
		}
	}
	if counts["tables"] != 58 || counts["foreign keys"] != 34 || counts["repairs"] != 2 {
		t.Fatalf("frozen baseline inventory differs: %v", counts)
	}
}

func TestManifestDoesNotShareMutableSlice(t *testing.T) {
	first, err := Manifest()
	if err != nil {
		t.Fatal(err)
	}
	checksum := first[0].Checksum
	first[0].Checksum = "changed"
	first[0].Apply = nil
	next, err := Manifest()
	if err != nil {
		t.Fatal(err)
	}
	if next[0].Checksum != checksum || next[0].Apply == nil {
		t.Fatal("manifest mutation escaped to another caller")
	}
}

// A released baseline is append-only: changing either input requires a new
// migration version, never refreshing this expected checksum.
func TestReleasedBaselineChecksum(t *testing.T) {
	manifest, err := Manifest()
	if err != nil {
		t.Fatal(err)
	}
	const expected = "92d4474d148d35643dbf8ed4044e6734e2d9268c36b6738bcb26ef220afc0616"
	if manifest[0].Checksum != expected {
		t.Fatal("immutable version 1 changed; add a new migration instead")
	}
}

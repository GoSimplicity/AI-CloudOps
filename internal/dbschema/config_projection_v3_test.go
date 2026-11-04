package dbschema

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

func TestProjectionV3AppendsWithoutChangingReleasedInputs(t *testing.T) {
	current, err := CurrentManifest()
	if err != nil || len(current) < 3 || schema.Validate(current) != nil {
		t.Fatal("current projection target invalid")
	}
	if current[0].Checksum != "92d4474d148d35643dbf8ed4044e6734e2d9268c36b6738bcb26ef220afc0616" || current[1].Checksum != "0264ef8ecc9f31ae99067abce868cdebb42348fbcac21019596b476bf70ec9d2" || current[2].Version != 3 {
		t.Fatal("released target changed")
	}
	legacy, err := Manifest()
	if err != nil || len(legacy) != 1 {
		t.Fatal("legacy upgrade target widened")
	}
	if _, err := KnownA8SafeUpgradeV2(); err != nil {
		t.Fatal("frozen upgrade provider no longer resolves")
	}
	current[2].Apply = nil
	next, err := CurrentManifest()
	if err != nil || next[2].Apply == nil {
		t.Fatal("current manifest shares mutation")
	}
}
func TestProjectionV3BindsOnlyItsOwnFrozenSQLAndWrapper(t *testing.T) {
	original, err := configProjectionV3Migration(configProjectionV3SQL, configProjectionV3Implementation)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := configProjectionV3Migration(configProjectionV3SQL, append(append([]byte(nil), configProjectionV3Implementation...), []byte("\n// changed\n")...))
	if err != nil || changed.Checksum == original.Checksum {
		t.Fatal("v3 wrapper not bound")
	}
	altered := strings.Replace(string(configProjectionV3SQL), "varbinary(180)", "varbinary(181)", 1)
	changed, err = configProjectionV3Migration([]byte(altered), configProjectionV3Implementation)
	if err != nil || changed.Checksum == original.Checksum {
		t.Fatal("v3 SQL not bound")
	}
	for _, raw := range [][]byte{nil, []byte(`[]`), []byte(`["DROP TABLE x"]`)} {
		if _, err := configProjectionV3Migration(raw, configProjectionV3Implementation); !errors.Is(err, schema.ErrManifest) {
			t.Fatal("malformed v3 accepted")
		}
	}
	var statements []string
	if json.Unmarshal(configProjectionV3SQL, &statements) != nil || len(statements) != 3 {
		t.Fatal("v3 inventory invalid")
	}
	for _, sql := range statements {
		if !strings.HasPrefix(sql, "CREATE TABLE ") || strings.Contains(sql, "IF NOT EXISTS") || strings.Contains(sql, "deleted_at") || strings.Contains(sql, "config_content") {
			t.Fatal("v3 performs implicit adoption/repair or copies bodies")
		}
	}
}

func TestReleasedProjectionV3Checksum(t *testing.T) {
	current, err := CurrentManifest()
	if err != nil {
		t.Fatal(err)
	}
	const expected = "9d45e4d010c9b7abec274c5d856f3f7cafa92cd428a13e5345a4a70f1be82443"
	if current[2].Checksum != expected {
		t.Fatal("immutable v3 changed; append a new migration instead")
	}
}

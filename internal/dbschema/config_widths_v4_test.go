package dbschema

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

func TestConfigWidthsV4AppendsOnlyLosslessColumnWidening(t *testing.T) {
	current, err := CurrentManifest()
	if err != nil || len(current) != 4 || schema.Validate(current) != nil {
		t.Fatal("current width manifest invalid")
	}
	for i, want := range []string{"92d4474d148d35643dbf8ed4044e6734e2d9268c36b6738bcb26ef220afc0616", "0264ef8ecc9f31ae99067abce868cdebb42348fbcac21019596b476bf70ec9d2", "9d45e4d010c9b7abec274c5d856f3f7cafa92cd428a13e5345a4a70f1be82443"} {
		if current[i].Checksum != want {
			t.Fatal("released migration input changed")
		}
	}
	var statements []string
	if json.Unmarshal(configWidthsV4SQL, &statements) != nil || len(statements) != 3 {
		t.Fatal("width inventory invalid")
	}
	for _, statement := range statements {
		if !strings.HasPrefix(statement, "ALTER TABLE ") || strings.Contains(statement, "DROP ") || strings.Contains(statement, "UPDATE ") || strings.Contains(statement, "ADD ") {
			t.Fatal("width migration does more than widen columns")
		}
	}
	if !strings.Contains(statements[0], "varchar(255)") || !strings.Contains(statements[0], "varchar(128)") || strings.Count(statements[0], "NOT NULL COMMENT") != 2 || strings.Contains(statements[0], "CHARACTER SET") || strings.Contains(statements[0], "COLLATE") || !strings.Contains(statements[1], "varbinary(512)") || !strings.Contains(statements[2], "varbinary(512)") {
		t.Fatal("lossless width attributes missing")
	}
}
func TestConfigWidthsV4BindsOnlyFrozenWrapperAndSQL(t *testing.T) {
	original, err := configWidthsV4Migration(configWidthsV4SQL, configWidthsV4Implementation)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name                string
		sql, implementation []byte
	}{
		{"SQL", []byte(strings.Replace(string(configWidthsV4SQL), "varchar(255)", "varchar(256)", 1)), configWidthsV4Implementation},
		{"wrapper", configWidthsV4SQL, append(append([]byte(nil), configWidthsV4Implementation...), []byte("\n// change\n")...)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed, err := configWidthsV4Migration(tc.sql, tc.implementation)
			if err != nil || changed.Checksum == original.Checksum {
				t.Fatal("release input not bound")
			}
		})
	}
	for _, raw := range [][]byte{nil, []byte(`[]`), []byte(`["DROP TABLE x"]`)} {
		if _, err := configWidthsV4Migration(raw, configWidthsV4Implementation); !errors.Is(err, schema.ErrManifest) {
			t.Fatal("invalid width manifest accepted")
		}
	}
}

func TestReleasedConfigWidthsV4Checksum(t *testing.T) {
	current, err := CurrentManifest()
	if err != nil {
		t.Fatal(err)
	}
	if current[3].Checksum != "897054d3d184ce9b562daade71bd568739407fc4f738751ccb431a080295f2c3" {
		t.Fatal("immutable V4 changed; append a migration instead")
	}
}

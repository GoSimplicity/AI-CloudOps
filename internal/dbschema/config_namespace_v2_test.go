package dbschema

import (
	"errors"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

func TestCurrentManifestAppendsGuardWithoutChangingReleasedTargets(t *testing.T) {
	legacy, err := Manifest()
	if err != nil || len(legacy) != 1 {
		t.Fatal("released v1 target changed")
	}
	current, err := CurrentManifest()
	if err != nil || len(current) < 2 || schema.Validate(current) != nil {
		t.Fatal("current target invalid")
	}
	if current[0].Checksum != legacy[0].Checksum || current[0].Checksum != "92d4474d148d35643dbf8ed4044e6734e2d9268c36b6738bcb26ef220afc0616" || current[1].Version != 2 {
		t.Fatal("released baseline changed")
	}
	if _, err := KnownA8Upgrade(); err != nil {
		t.Fatal("frozen original upgrade no longer resolves v1")
	}
	if _, err := KnownA8SafeUpgradeV2(); err != nil {
		t.Fatal("frozen safe upgrade no longer resolves v1")
	}
	current[1].Checksum = "changed"
	fresh, err := CurrentManifest()
	if err != nil || fresh[1].Checksum == "changed" {
		t.Fatal("current manifest shares caller mutation")
	}
}
func TestNamespaceV2ChecksumBindsOnlyNewReleaseInputs(t *testing.T) {
	original, err := configNamespaceV2Migration(configNamespaceV2SQL, configNamespaceV2Implementation)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := configNamespaceV2Migration(configNamespaceV2SQL, append(append([]byte(nil), configNamespaceV2Implementation...), []byte("\n// changed\n")...))
	if err != nil || changed.Checksum == original.Checksum {
		t.Fatal("v2 wrapper not bound")
	}
	changedSQL := strings.Replace(string(configNamespaceV2SQL), "varbinary(180)", "varbinary(181)", 1)
	changed, err = configNamespaceV2Migration([]byte(changedSQL), configNamespaceV2Implementation)
	if err != nil || changed.Checksum == original.Checksum {
		t.Fatal("v2 SQL not bound")
	}
	for _, snapshot := range [][]byte{nil, []byte(`[]`), []byte(`["DROP TABLE x"]`)} {
		if _, err := configNamespaceV2Migration(snapshot, configNamespaceV2Implementation); !errors.Is(err, schema.ErrManifest) {
			t.Fatal("invalid v2 inputs accepted")
		}
	}
}

func TestReleasedNamespaceV2Checksum(t *testing.T) {
	current, err := CurrentManifest()
	if err != nil {
		t.Fatal(err)
	}
	const expected = "0264ef8ecc9f31ae99067abce868cdebb42348fbcac21019596b476bf70ec9d2"
	if current[1].Checksum != expected {
		t.Fatal("immutable v2 changed; append a new migration instead")
	}
}

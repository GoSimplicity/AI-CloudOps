package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

func TestKnownUpgradeDefaultsToOfflinePlan(t *testing.T) {
	var out, diagnostic bytes.Buffer
	status := run(context.Background(), []string{knownUpgradeCommand}, &out, &diagnostic, dependencies{
		manifest: dbschema.Manifest,
		getenv:   func(string) string { t.Fatal("offline plan read credentials"); return "" },
		open: func(string, time.Duration) (schemaDatabase, error) {
			t.Fatal("offline plan opened database")
			return nil, nil
		},
	})
	if status != 0 || !strings.Contains(out.String(), dbschema.KnownA8Source) || !strings.Contains(out.String(), "15 frozen") || !strings.Contains(out.String(), "implementation_checksum=") {
		t.Fatal("offline known-source plan was not returned")
	}
}

func TestKnownUpgradeRejectsIncompleteOrMisplacedExecutionOptions(t *testing.T) {
	cases := [][]string{
		{knownUpgradeCommand, "--execute"},
		{knownUpgradeCommand, "--dsn-file", privateCanary},
		{knownUpgradeCommand, "--upgrade-note-file", privateCanary},
		{knownUpgradeCommand, "--confirm-no-writers"},
		{knownUpgradeCommand, "--attest-historical-time"},
		{knownUpgradeCommand, "--execute", "--confirm-no-writers", "--upgrade-note-file", "note", "--backup-proof-file", "proof", "--notification-timezone", "Local"},
		{"verify", "--execute"},
		{"adopt", "--upgrade-note-file", privateCanary},
	}
	for _, args := range cases {
		var out, diagnostic bytes.Buffer
		status := run(context.Background(), args, &out, &diagnostic, dependencies{
			manifest: func() ([]schema.Migration, error) { t.Fatal("invalid options loaded manifest"); return nil, nil },
			getenv:   func(string) string { t.Fatal("invalid options read credentials"); return "" },
			open: func(string, time.Duration) (schemaDatabase, error) {
				t.Fatal("invalid options opened DB")
				return nil, nil
			},
		})
		if status != 2 || strings.Contains(diagnostic.String()+out.String(), privateCanary) {
			t.Fatal("invalid upgrade options were accepted or echoed")
		}
	}
}

func upgradeFiles(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	note, proof := filepath.Join(dir, "note"), filepath.Join(dir, "proof")
	if err := os.WriteFile(note, []byte("fixture backup and source inspected"), 0600); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(backupAttestation{1, dbschema.KnownA8Source, strings.Repeat("b", 64), true})
	if err := os.WriteFile(proof, payload, 0600); err != nil {
		t.Fatal(err)
	}
	return note, proof
}

func TestKnownUpgradeRunsOnlyExplicitMethodThenVerifiesSchemaAndClock(t *testing.T) {
	note, proof := upgradeFiles(t)
	args := []string{knownUpgradeCommand, "--execute", "--confirm-no-writers", "--upgrade-note-file", note, "--backup-proof-file", proof, "--notification-timezone", "UTC", "--attest-historical-time"}
	for _, fail := range []bool{false, true} {
		db := &fakeDatabase{}
		if fail {
			db.upgradeError = errors.New(privateCanary)
		}
		var out, diagnostic bytes.Buffer
		status := run(context.Background(), args, &out, &diagnostic, dependencies{
			manifest: dbschema.Manifest,
			getenv:   func(string) string { return "fixture-dsn" },
			open:     func(string, time.Duration) (schemaDatabase, error) { return db, nil },
		})
		want, calls := 0, []string{"upgrade", "verify", "verify-clock", "close"}
		if fail {
			want, calls = 1, []string{"upgrade", "close"}
		}
		if status != want || !reflect.DeepEqual(db.actions, calls) || strings.Contains(diagnostic.String()+out.String(), privateCanary) {
			t.Fatalf("unexpected execution status/actions: %d %v", status, db.actions)
		}
		if db.upgradeEvidence.NotificationZone != "UTC" || !db.upgradeEvidence.HistoricalTimeAttested || db.upgradeEvidence.BackupSHA256 != strings.Repeat("b", 64) {
			t.Fatal("explicit execution evidence was not propagated")
		}
	}
}

func TestKnownUpgradeRejectsInvalidProofBeforeOpeningDatabase(t *testing.T) {
	note, proof := upgradeFiles(t)
	if err := os.WriteFile(proof, []byte(`{"version":1,"source_revision":"wrong","backup_sha256":"private-proof-canary","backup_ready":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	status := run(context.Background(), []string{knownUpgradeCommand, "--execute", "--confirm-no-writers", "--upgrade-note-file", note, "--backup-proof-file", proof, "--notification-timezone", "UTC"}, &out, &diagnostic, dependencies{
		manifest: dbschema.Manifest,
		getenv:   func(string) string { t.Fatal("invalid proof read DSN"); return "" },
		open: func(string, time.Duration) (schemaDatabase, error) {
			t.Fatal("invalid proof opened DB")
			return nil, nil
		},
	})
	if status != 2 || strings.Contains(diagnostic.String(), "private-proof-canary") {
		t.Fatal("invalid proof accepted or leaked")
	}
}

func TestExistingAppliedUpgradeStillChecksRequestedClock(t *testing.T) {
	note, proof := upgradeFiles(t)
	db := &fakeDatabase{clockError: dbschema.ErrClockMismatch}
	var out, diagnostic bytes.Buffer
	status := run(context.Background(), []string{knownUpgradeCommand, "--execute", "--confirm-no-writers", "--upgrade-note-file", note, "--backup-proof-file", proof, "--notification-timezone", "Asia/Shanghai"}, &out, &diagnostic, dependencies{manifest: dbschema.Manifest, getenv: func(string) string { return "fixture" }, open: func(string, time.Duration) (schemaDatabase, error) { return db, nil }})
	if status != 1 || !reflect.DeepEqual(db.actions, []string{"upgrade", "verify", "verify-clock", "close"}) || strings.Contains(out.String(), "verified") {
		t.Fatal("existing applied path failed to reject a timezone mismatch")
	}
}

func TestUpgradeProofRequiresPrivateCurrentOwnerFileAndExactAttestation(t *testing.T) {
	for _, kind := range []string{"group-readable", "symlink", "unknown-field", "not-ready", "trailing-object"} {
		t.Run(kind, func(t *testing.T) {
			note, proof := upgradeFiles(t)
			switch kind {
			case "group-readable":
				if err := os.Chmod(proof, 0644); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				link := filepath.Join(t.TempDir(), "linked")
				if err := os.Symlink(proof, link); err != nil {
					t.Fatal(err)
				}
				proof = link
			case "unknown-field":
				raw, _ := json.Marshal(map[string]any{"version": 1, "source_revision": dbschema.KnownA8Source, "backup_sha256": strings.Repeat("b", 64), "backup_ready": true, "untrusted": "private-extra-canary"})
				if err := os.WriteFile(proof, raw, 0600); err != nil {
					t.Fatal(err)
				}
			case "not-ready":
				raw, _ := json.Marshal(backupAttestation{1, dbschema.KnownA8Source, strings.Repeat("b", 64), false})
				if err := os.WriteFile(proof, raw, 0600); err != nil {
					t.Fatal(err)
				}
			case "trailing-object":
				raw, err := os.ReadFile(proof)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(proof, append(raw, []byte(" {}")...), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := readUpgradeEvidence(options{upgradeNoteFile: note, backupProofFile: proof, zone: "UTC"}); !errors.Is(err, errUpgradeEvidence) || strings.Contains(err.Error(), "private-extra-canary") {
				t.Fatal("invalid private attestation was accepted or leaked")
			}
		})
	}
}

func TestBackupProofRejectsDuplicateOrAmbiguousJSONFieldsBeforeDatabase(t *testing.T) {
	base := `"version":1,"source_revision":"` + dbschema.KnownA8Source + `","backup_sha256":"` + strings.Repeat("b", 64) + `","backup_ready":true`
	for _, suffix := range []string{`,"version":1`, `,"backup_ready":true`, `,"backup_sha256":"private-duplicate-canary"`, `,"source_revision":"` + dbschema.KnownA8Source + `"`, `,"backup\u005fready":true`, `,"ver\u0073ion":1`} {
		t.Run(suffix, func(t *testing.T) {
			note, proof := upgradeFiles(t)
			if err := os.WriteFile(proof, []byte(`{`+base+suffix+`}`), 0600); err != nil {
				t.Fatal(err)
			}
			var out, diagnostic bytes.Buffer
			status := run(context.Background(), []string{knownUpgradeCommand, "--execute", "--confirm-no-writers", "--upgrade-note-file", note, "--backup-proof-file", proof, "--notification-timezone", "UTC"}, &out, &diagnostic, dependencies{manifest: dbschema.Manifest, getenv: func(string) string { t.Fatal("duplicate proof read credentials"); return "" }, open: func(string, time.Duration) (schemaDatabase, error) {
				t.Fatal("duplicate proof opened database")
				return nil, nil
			}})
			if status != 2 || strings.Contains(diagnostic.String(), "private-duplicate-canary") {
				t.Fatal("duplicate audit JSON was accepted or echoed")
			}
		})
	}
	if uniqueBackupAttestationObject(`{` + strings.Replace(base, `"version"`, `"Version"`, 1) + `}`) {
		t.Fatal("case-folded unexpected field was accepted")
	}
}

func TestKnownUpgradePlanAndOfflineOutputSelectSafeV2(t *testing.T) {
	selected, err := knownUpgradePlan()
	if err != nil {
		t.Fatal(err)
	}
	guarded, err := dbschema.KnownA8SafeUpgradeV2()
	if err != nil {
		t.Fatal(err)
	}
	old, err := dbschema.KnownA8Upgrade()
	if err != nil {
		t.Fatal(err)
	}
	if selected.ImplementationChecksum != guarded.ImplementationChecksum || selected.ImplementationChecksum == old.ImplementationChecksum {
		t.Fatal("CLI selected the unguarded historical plan")
	}
	var out, diagnostic bytes.Buffer
	status := run(context.Background(), []string{knownUpgradeCommand}, &out, &diagnostic, dependencies{manifest: dbschema.Manifest, getenv: func(string) string { t.Fatal("offline guard plan read credentials"); return "" }, open: func(string, time.Duration) (schemaDatabase, error) {
		t.Fatal("offline guard plan connected database")
		return nil, nil
	}})
	if status != 0 || !strings.Contains(out.String(), "implementation_checksum="+guarded.ImplementationChecksum) {
		t.Fatal("offline plan and execution selector differ")
	}
}

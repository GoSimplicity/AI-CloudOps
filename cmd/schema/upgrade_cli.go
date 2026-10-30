package main

import (
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/GoSimplicity/AI-CloudOps/internal/dbschema"
	"github.com/GoSimplicity/AI-CloudOps/pkg/schema"
)

const knownUpgradeCommand = "upgrade-a8f1700"

var errUpgradeEvidence = errors.New("upgrade requires valid private note and backup-attestation files")
var backupSHA256Pattern = regexp.MustCompile("^[0-9a-f]{64}$")

type backupAttestation struct {
	Version        int    `json:"version"`
	SourceRevision string `json:"source_revision"`
	BackupSHA256   string `json:"backup_sha256"`
	BackupReady    bool   `json:"backup_ready"`
}

// JSON field names are decoded before comparison, so escaped-equivalent keys
// cannot silently replace an earlier attestation. Only this version's exact
// four field names are admitted; encoding/json's case-folded aliases are not.
func uniqueBackupAttestationObject(proof string) bool {
	decoder := json.NewDecoder(strings.NewReader(proof))
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return false
	}
	allowed := map[string]bool{"version": true, "source_revision": true, "backup_sha256": true, "backup_ready": true}
	seen := map[string]bool{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return false
		}
		name, ok := token.(string)
		if !ok || !allowed[name] || seen[name] {
			return false
		}
		seen[name] = true
		var value json.RawMessage
		if decoder.Decode(&value) != nil {
			return false
		}
	}
	last, err := decoder.Token()
	if err != nil || last != json.Delim('}') || len(seen) != len(allowed) {
		return false
	}
	var extra any
	return decoder.Decode(&extra) == io.EOF
}

func readUpgradeEvidence(config options) (schema.UpgradeEvidence, error) {
	note, err := readCredentials(config.upgradeNoteFile, func(string) string { return "" })
	if err != nil || len(note) > 1000 || !utf8.ValidString(note) {
		return schema.UpgradeEvidence{}, errUpgradeEvidence
	}
	proof, err := readCredentials(config.backupProofFile, func(string) string { return "" })
	if err != nil {
		return schema.UpgradeEvidence{}, errUpgradeEvidence
	}
	if !uniqueBackupAttestationObject(proof) {
		return schema.UpgradeEvidence{}, errUpgradeEvidence
	}
	var attestation backupAttestation
	decoder := json.NewDecoder(strings.NewReader(proof))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&attestation) != nil || attestation.Version != 1 || attestation.SourceRevision != dbschema.KnownA8Source || !attestation.BackupReady || !backupSHA256Pattern.MatchString(attestation.BackupSHA256) {
		return schema.UpgradeEvidence{}, errUpgradeEvidence
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return schema.UpgradeEvidence{}, errUpgradeEvidence
	}
	return schema.UpgradeEvidence{
		Note: note, BackupSHA256: attestation.BackupSHA256, NotificationZone: config.zone,
		HistoricalTimeAttested: config.attestHistoricalTime,
	}, nil
}

package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func validPublicationScope(family int8, poolID int) bool {
	return family >= 1 && family <= 5 && poolID > 0
}
func familyAllowsKind(family, kind int8) bool {
	switch family {
	case 1:
		return kind == 1
	case 2:
		return kind == 2 || kind == 5
	case 3:
		return kind == 3
	case 4:
		return kind == 4
	case 5:
		return kind == 5
	}
	return false
}
func bindingNamespace(binding ProjectionBinding) (configNamespaceKey, error) {
	if binding.Version != 1 || binding.Role != "artifact" || net.ParseIP(binding.InstanceIP) == nil {
		return configNamespaceKey{}, ErrJournalInvalid
	}
	raw := binding.InstanceIP
	if binding.ConfigType == model.ConfigTypeWebhookFile {
		if binding.GroupID == nil || *binding.GroupID <= 0 {
			return configNamespaceKey{}, ErrJournalInvalid
		}
		raw = fmt.Sprintf("%s_%d", binding.InstanceIP, *binding.GroupID)
	} else if binding.GroupID != nil {
		return configNamespaceKey{}, ErrJournalInvalid
	}
	key := configNamespaceKey{binding.ConfigType, raw}
	if !validConfigNamespace(key) {
		return configNamespaceKey{}, ErrJournalInvalid
	}
	return key, nil
}
func cloneBinding(value *ProjectionBinding) *ProjectionBinding {
	if value == nil {
		return nil
	}
	copy := *value
	if value.GroupID != nil {
		x := *value.GroupID
		copy.GroupID = &x
	}
	return &copy
}

// Decode stored MySQL JSON semantically: MySQL may reorder fields/whitespace.
// Unknown fields, trailing values and free physical keys are never accepted.
func ParseProjectionBinding(raw string) (ProjectionBinding, error) {
	var binding ProjectionBinding
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&binding) != nil {
		return binding, ErrJournalInvalid
	}
	if decoder.Decode(new(interface{})) != io.EOF {
		return binding, ErrJournalInvalid
	}
	if _, err := bindingNamespace(binding); err != nil {
		return binding, err
	}
	return binding, nil
}
func canonicalBinding(value *ProjectionBinding) (*string, error) {
	if value == nil {
		return nil, ErrJournalInvalid
	}
	if _, err := bindingNamespace(*value); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, ErrJournalInvalid
	}
	s := string(raw)
	return &s, nil
}
func generatedRowNameValid(row *model.MonitorConfig) bool {
	if row == nil {
		return false
	}
	switch row.ConfigType {
	case 1:
		return row.Name == fmt.Sprintf("prometheus_scrape_pool_%d_%s.yaml", row.PoolID, row.InstanceIP)
	case 2:
		return row.Name == fmt.Sprintf("alertmanager_pool_%d_%s.yaml", row.PoolID, row.InstanceIP)
	case 3:
		return row.Name == fmt.Sprintf("prometheus_alert_rule_%d_%s.yaml", row.PoolID, row.InstanceIP)
	case 4:
		return row.Name == fmt.Sprintf("prometheus_record_rule_%d_%s.yaml", row.PoolID, row.InstanceIP)
	case 5:
		split := strings.LastIndex(row.InstanceIP, "_")
		if split <= 0 {
			return false
		}
		suffix := "_" + row.InstanceIP[split+1:] + "_" + row.InstanceIP[:split]
		return strings.HasPrefix(row.Name, "webhook_") && strings.HasSuffix(row.Name, suffix) && len(row.Name) > len("webhook_")+len(suffix)
	}
	return false
}
func ownerDigest(row *model.MonitorConfig) []byte {
	if row == nil {
		return nil
	}
	raw, _ := json.Marshal(GeneratedConfigOwnerOf(row))
	digest := sha256.Sum256(raw)
	return append([]byte(nil), digest[:]...)
}
func bodyDigest(row *model.MonitorConfig) []byte {
	if row == nil {
		return nil
	}
	digest := sha256.Sum256([]byte(row.ConfigContent))
	return append([]byte(nil), digest[:]...)
}
func digestEqual(a, b []byte) bool     { return bytes.Equal(a, b) }
func nonzeroNonce(nonce [16]byte) bool { return nonce != [16]byte{} }
func cloneUint64(value *uint64) *uint64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
func sameOptionalID(a, b *uint64) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

type poolManifestEntry struct {
	SourceID   uint64 `json:"source_id"`
	EventID    uint64 `json:"event_id"`
	PlanDigest string `json:"plan_digest"`
}

func decodeLegacyPoolManifest(raw *string) ([]poolManifestEntry, error) {
	if raw == nil {
		return nil, ErrJournalConflict
	}
	var entries []poolManifestEntry
	decoder := json.NewDecoder(strings.NewReader(*raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&entries) != nil || entries == nil || decoder.Decode(new(interface{})) != io.EOF {
		return nil, ErrJournalConflict
	}
	seenSources, seenEvents := map[uint64]bool{}, map[uint64]bool{}
	for _, entry := range entries {
		decoded, err := hex.DecodeString(entry.PlanDigest)
		if entry.SourceID == 0 || entry.EventID == 0 || seenSources[entry.SourceID] || seenEvents[entry.EventID] || err != nil || len(decoded) != 32 {
			return nil, ErrJournalConflict
		}
		seenSources[entry.SourceID] = true
		seenEvents[entry.EventID] = true
	}
	return entries, nil
}

func decodePoolManifest(raw *string) ([]poolManifestEntry, error) {
	value, err := decodeProjectionManifest(raw)
	return value.Artifacts, err
}

// ProjectionBindingForNamespace derives typed logical components only. It
// cannot accept a physical key, and group IDs remain strictly positive.
func ProjectionBindingForNamespace(kind int8, key string) (ProjectionBinding, error) {
	binding := ProjectionBinding{Version: 1, Role: "artifact", ConfigType: kind, InstanceIP: key}
	if kind == 5 {
		split := strings.LastIndex(key, "_")
		if split <= 0 {
			return ProjectionBinding{}, ErrJournalInvalid
		}
		group, err := strconv.Atoi(key[split+1:])
		if err != nil || group <= 0 || strconv.Itoa(group) != key[split+1:] {
			return ProjectionBinding{}, ErrJournalInvalid
		}
		binding.InstanceIP = key[:split]
		binding.GroupID = &group
	}
	actual, err := bindingNamespace(binding)
	if err != nil || actual.raw != key {
		return ProjectionBinding{}, ErrJournalInvalid
	}
	return binding, nil
}
func GeneratedProjectionOwnerValid(owner GeneratedConfigOwner) bool {
	return validGeneratedOwner(owner) && generatedRowNameValid(scrapeRuleOwnerModel(owner))
}

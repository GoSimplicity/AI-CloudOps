package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type projectionManifestEnvelope struct {
	Version    uint8                     `json:"version"`
	Artifacts  []poolManifestEntry       `json:"artifacts"`
	Controls   []IssuedProjectionControl `json:"controls"`
	PlanDigest string                    `json:"plan_digest"`
}
type preparedProjectionPlan struct {
	request  GeneratedProjectionPlanRequest
	controls []ProjectionControl
	keys     []configNamespaceKey
	digest   [32]byte
}

func prepareProjectionPlan(req GeneratedProjectionPlanRequest) (*preparedProjectionPlan, error) {
	prepared, _, artifactDigest, err := prepareGeneratedPlan(req.GeneratedPlanRequest)
	if err != nil {
		return nil, err
	}
	controls := cloneProjectionControls(req.Controls)
	sort.Slice(controls, func(i, j int) bool { return controls[i].ControlID < controls[j].ControlID })
	seenID := map[uint64]bool{}
	seenKind := map[ProjectionControlKind]bool{}
	nonAckIDs := []uint64{}
	artifactIDs := []uint64{}
	renderIPs := map[string]bool{}
	sources := map[uint64]GeneratedPlanItem{}
	family := req.Reservation.Family
	mainKind := family
	if family == 5 {
		mainKind = 0
	}
	for _, entry := range prepared {
		item := entry.item
		sources[item.SourceID] = item
		artifactIDs = append(artifactIDs, item.SourceID)
		if item.Action == model.ConfigProjectionRender && item.Binding != nil && item.Binding.ConfigType == mainKind {
			renderIPs[item.Binding.InstanceIP] = true
		}
	}
	artifactIDs, _ = sortedUniqueIDs(artifactIDs)
	plan := &preparedProjectionPlan{request: req, controls: controls}
	for i := range controls {
		control := &controls[i]
		if control.ControlID == 0 || seenID[control.ControlID] || seenKind[control.Kind] || !allowedControlFamily(family, control.Kind) {
			return nil, ErrJournalInvalid
		}
		seenID[control.ControlID] = true
		seenKind[control.Kind] = true
		payloads := 0
		if control.Members != nil {
			payloads++
		}
		if control.Intents != nil {
			payloads++
		}
		if control.Ack != nil {
			payloads++
		}
		if payloads != 1 {
			return nil, ErrJournalInvalid
		}
		switch control.Kind {
		case ProjectionMembersFull:
			if control.Members == nil {
				return nil, ErrJournalInvalid
			}
			set := map[string]bool{}
			for _, ip := range control.Members.DesiredIPs {
				if net.ParseIP(ip) == nil || !validConfigNamespace(configNamespaceKey{mainKind, ip}) {
					return nil, ErrJournalInvalid
				}
				set[ip] = true
			}
			if !reflect.DeepEqual(set, renderIPs) {
				return nil, ErrJournalInvalid
			}
			ips := make([]string, 0, len(set))
			for ip := range set {
				ips = append(ips, ip)
			}
			sort.Strings(ips)
			control.Members.DesiredIPs = ips
			nonAckIDs = append(nonAckIDs, control.ControlID)
		case ProjectionWebhookIntents:
			if control.Intents == nil || !control.Intents.RequireEmpty {
				return nil, ErrJournalInvalid
			}
			if control.Intents.Add == nil {
				control.Intents.Add = []IntentOwnerChange{}
			}
			if control.Intents.Remove == nil {
				control.Intents.Remove = []IntentOwnerChange{}
			}
			for _, entries := range [][]IntentOwnerChange{control.Intents.Add, control.Intents.Remove} {
				seenOwners := map[GeneratedConfigOwner]bool{}
				for _, entry := range entries {
					owner := entry.Owner
					if !validGeneratedOwner(owner) || owner.ConfigType != 5 || owner.PoolID != req.Reservation.PoolID ||
						!validConfigNamespace(configNamespaceKey{5, owner.InstanceIP}) || !utf8.ValidString(owner.Name) ||
						utf8.RuneCountInString(owner.Name) > model.MonitorConfigNameMaxRunes || seenOwners[owner] {
						return nil, ErrJournalInvalid
					}
					seenOwners[owner] = true
					if (entry.Proof.SourceID == 0) == (entry.Proof.EventID == 0) {
						return nil, ErrJournalInvalid
					}
					if entry.Proof.SourceID != 0 {
						item, ok := sources[entry.Proof.SourceID]
						if !ok {
							return nil, ErrJournalInvalid
						}
						key := configNamespaceKey{}
						if item.Desired != nil {
							key = configKey(item.Desired)
						} else {
							key = configNamespaceKey{item.Expected.ConfigType, item.Expected.InstanceIP}
						}
						if key != (configNamespaceKey{5, owner.InstanceIP}) {
							return nil, ErrJournalInvalid
						}
					}
					row := &model.MonitorConfig{Model: model.Model{ID: owner.ID}, PoolID: owner.PoolID, ConfigType: 5, InstanceIP: owner.InstanceIP, Name: owner.Name}
					if !generatedRowNameValid(row) {
						return nil, ErrJournalInvalid
					}
					plan.keys = append(plan.keys, configNamespaceKey{5, owner.InstanceIP})
				}
			}
			nonAckIDs = append(nonAckIDs, control.ControlID)
		case ProjectionACKFull, ProjectionACKWebhookSubset:
			if control.Ack == nil {
				return nil, ErrJournalInvalid
			}
			if family == 5 {
				if control.Ack.LegacySHA != nil {
					return nil, ErrJournalInvalid
				}
			} else if control.Ack.LegacySHA == nil || *control.Ack.LegacySHA != req.SourceHash {
				return nil, ErrJournalInvalid
			}
			ids, ok := sortedUniqueIDs(control.Ack.ArtifactSourceIDs)
			if !ok || !reflect.DeepEqual(ids, artifactIDs) {
				return nil, ErrJournalInvalid
			}
			control.Ack.ArtifactSourceIDs = ids
		default:
			return nil, ErrJournalInvalid
		}
	}
	if family == 5 {
		if len(controls) != 1 || !seenKind[ProjectionACKWebhookSubset] {
			return nil, ErrJournalInvalid
		}
	} else {
		want := 2
		if family == 2 {
			want = 3
		}
		if len(controls) != want || !seenKind[ProjectionMembersFull] || !seenKind[ProjectionACKFull] || family == 2 && !seenKind[ProjectionWebhookIntents] {
			return nil, ErrJournalInvalid
		}
	}
	nonAckIDs, _ = sortedUniqueIDs(nonAckIDs)
	for i := range controls {
		if controls[i].Ack != nil {
			ids, ok := sortedUniqueIDs(controls[i].Ack.RequiredControlIDs)
			if !ok || !reflect.DeepEqual(ids, nonAckIDs) {
				return nil, ErrJournalInvalid
			}
			controls[i].Ack.RequiredControlIDs = ids
		}
	}
	plan.controls = controls
	raw, err := json.Marshal(struct {
		ArtifactDigest string              `json:"artifact_digest"`
		Controls       []ProjectionControl `json:"controls"`
	}{artifactDigest, controls})
	if err != nil {
		return nil, ErrJournalInvalid
	}
	plan.digest = sha256.Sum256(raw)
	return plan, nil
}

func (d *monitorConfigDAO) resolveProjectionControls(tx *gorm.DB, prepared *preparedProjectionPlan, entries []poolManifestEntry, events map[uint64]model.MonitorConfigArtifactEvent) ([]IssuedProjectionControl, error) {
	vector := make([]ProjectionVectorEntry, 0, len(entries))
	for _, entry := range entries {
		vector = append(vector, ProjectionVectorEntry{entry.SourceID, entry.EventID})
	}
	issued := make([]IssuedProjectionControl, 0, len(prepared.controls))
	digests := []ProjectionControlDigest{}
	for _, control := range prepared.controls {
		if control.Ack != nil {
			continue
		}
		item := IssuedProjectionControl{ControlID: control.ControlID, Kind: control.Kind}
		if control.Members != nil {
			copy := *control.Members
			copy.DesiredIPs = append([]string{}, copy.DesiredIPs...)
			item.Members = &copy
		}
		if control.Intents != nil {
			result := &IssuedWebhookIntentsControl{Add: []IssuedIntentOwnerChange{}, Remove: []IssuedIntentOwnerChange{}, RequireEmpty: true}
			for which, changes := range [][]IntentOwnerChange{control.Intents.Add, control.Intents.Remove} {
				for _, change := range changes {
					event := model.MonitorConfigArtifactEvent{}
					if change.Proof.SourceID != 0 {
						var ok bool
						event, ok = events[change.Proof.SourceID]
						if !ok {
							return nil, ErrJournalConflict
						}
					} else {
						if err := tx.Where("event_id = ?", change.Proof.EventID).First(&event).Error; err != nil {
							return nil, err
						}
						if event.OperationID != nil {
							var op model.MonitorConfigPoolOperation
							if err := tx.Where("operation_id = ?", *event.OperationID).First(&op).Error; err != nil {
								return nil, err
							}
							if op.PoolID != prepared.request.Reservation.PoolID || (op.Family != 2 && op.Family != 5) {
								return nil, ErrJournalConflict
							}
							receipt, err := d.loadReceipt(tx, op)
							if err != nil {
								return nil, err
							}
							found := false
							for _, artifact := range receipt.Artifacts {
								if artifact.EventID == event.EventID {
									found = true
								}
							}
							if !found {
								return nil, ErrJournalConflict
							}
						}
					}
					if event.ConfigType != 5 || string(event.InstanceKey) != change.Owner.InstanceIP {
						return nil, ErrJournalConflict
					}
					var binding *ProjectionBinding
					if change.Proof.SourceID != 0 && event.ProjectionPolicy == model.ConfigProjectionGenerated && event.Binding != nil {
						value, err := ParseProjectionBinding(*event.Binding)
						if err != nil {
							return nil, err
						}
						binding = &value
					} else {
						root, err := trustedBindingRoot(tx, &event, configNamespaceKey{5, change.Owner.InstanceIP})
						if err != nil {
							return nil, err
						}
						if root == nil || root.Binding == nil {
							return nil, ErrJournalConflict
						}
						if root.OperationID == nil {
							return nil, ErrJournalConflict
						}
						var rootOp model.MonitorConfigPoolOperation
						if err := tx.Where("operation_id = ?", *root.OperationID).First(&rootOp).Error; err != nil {
							return nil, err
						}
						if rootOp.PoolID != prepared.request.Reservation.PoolID || (rootOp.Family != 2 && rootOp.Family != 5) {
							return nil, ErrJournalConflict
						}
						value, err := ParseProjectionBinding(*root.Binding)
						if err != nil {
							return nil, err
						}
						binding = &value
					}
					key, err := bindingNamespace(*binding)
					if err != nil || key != (configNamespaceKey{5, change.Owner.InstanceIP}) {
						return nil, ErrJournalConflict
					}
					namespaceKey := configNamespaceKey{5, change.Owner.InstanceIP}
					latest, err := latestArtifactEvent(tx, namespaceKey)
					if err != nil {
						return nil, err
					}
					if latest == nil {
						return nil, ErrJournalConflict
					}
					latestFromCurrent := latest.OperationID != nil && *latest.OperationID == prepared.request.Reservation.OperationID && latest.ProjectionPolicy == model.ConfigProjectionGenerated
					if !latestFromCurrent {
						latestRoot, err := trustedBindingRoot(tx, latest, namespaceKey)
						if err != nil {
							return nil, err
						}
						if latestRoot == nil || latestRoot.OperationID == nil {
							return nil, ErrJournalConflict
						}
						var latestOp model.MonitorConfigPoolOperation
						if err := tx.Where("operation_id = ?", *latestRoot.OperationID).First(&latestOp).Error; err != nil {
							return nil, err
						}
						if latestOp.PoolID != prepared.request.Reservation.PoolID || (latestOp.Family != 2 && latestOp.Family != 5) {
							return nil, ErrJournalConflict
						}
					}
					currentRows, err := currentConfigNamespace(tx, namespaceKey)
					if err != nil {
						return nil, err
					}
					if len(currentRows) > 1 {
						return nil, ErrJournalConflict
					}
					if len(currentRows) == 1 && (currentRows[0].PoolID != prepared.request.Reservation.PoolID ||
						!generatedRowNameValid(currentRows[0]) || !digestEqual(ownerDigest(currentRows[0]), latest.NewOwnerDigest)) {
						return nil, ErrJournalConflict
					}
					if len(currentRows) == 0 && len(latest.NewOwnerDigest) != 0 {
						return nil, ErrJournalConflict
					}
					var actual model.MonitorConfig
					if err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).First(&actual, change.Owner.ID).Error; err != nil {
						return nil, err
					}
					attestedOwner := &model.MonitorConfig{
						Model: model.Model{ID: change.Owner.ID}, PoolID: change.Owner.PoolID,
						ConfigType: change.Owner.ConfigType, InstanceIP: change.Owner.InstanceIP, Name: change.Owner.Name,
					}
					if event.ConfigRowID == nil || *event.ConfigRowID != change.Owner.ID ||
						(!digestEqual(ownerDigest(attestedOwner), event.OldOwnerDigest) && !digestEqual(ownerDigest(attestedOwner), event.NewOwnerDigest)) {
						return nil, ErrJournalConflict
					}
					actualOwner := GeneratedConfigOwnerOf(&actual)
					if actualOwner != change.Owner {
						// Same-row canonical webhook rename is a locator alias, never new raw
						// authority. The current event must attest the actual immutable owner.
						candidate := &model.MonitorConfig{
							Model: model.Model{ID: change.Owner.ID}, PoolID: change.Owner.PoolID,
							ConfigType: 5, InstanceIP: change.Owner.InstanceIP, Name: change.Owner.Name,
						}
						if actual.ID != change.Owner.ID || actual.PoolID != change.Owner.PoolID ||
							actual.ConfigType != 5 || actual.InstanceIP != change.Owner.InstanceIP ||
							!generatedRowNameValid(&actual) || !generatedRowNameValid(candidate) ||
							!digestEqual(ownerDigest(&actual), event.OldOwnerDigest) && !digestEqual(ownerDigest(&actual), event.NewOwnerDigest) {
							return nil, ErrJournalConflict
						}
					}
					value := IssuedIntentOwnerChange{Owner: change.Owner, ProofEventID: event.EventID, ProofSourceID: change.Proof.SourceID}
					if which == 0 {
						result.Add = append(result.Add, value)
					} else {
						result.Remove = append(result.Remove, value)
					}
				}
			}
			item.Intents = result
		}
		item.Digest = issuedControlDigest(item)
		issued = append(issued, item)
		digests = append(digests, ProjectionControlDigest{item.ControlID, item.Digest})
	}
	for _, control := range prepared.controls {
		if control.Ack != nil {
			ack := &IssuedProjectionAckControl{ArtifactVector: vector, RequiredControls: digests}
			if control.Ack.LegacySHA != nil {
				hash := *control.Ack.LegacySHA
				ack.LegacySHA = &hash
			}
			item := IssuedProjectionControl{ControlID: control.ControlID, Kind: control.Kind, Ack: ack}
			item.Digest = issuedControlDigest(item)
			issued = append(issued, item)
		}
	}
	sort.Slice(issued, func(i, j int) bool { return issued[i].ControlID < issued[j].ControlID })
	return issued, nil
}

func decodeProjectionManifest(raw *string) (projectionManifestEnvelope, error) {
	if raw == nil {
		return projectionManifestEnvelope{}, ErrJournalConflict
	}
	text := strings.TrimSpace(*raw)
	if strings.HasPrefix(text, "[") {
		entries, err := decodeLegacyPoolManifest(raw)
		if err != nil {
			return projectionManifestEnvelope{}, err
		}
		return projectionManifestEnvelope{Version: 1, Artifacts: entries}, nil
	}
	var envelope projectionManifestEnvelope
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&envelope) != nil || decoder.Decode(new(interface{})) != io.EOF || envelope.Version != 2 || envelope.Artifacts == nil || envelope.Controls == nil {
		return envelope, ErrJournalConflict
	}
	digest, err := hex.DecodeString(envelope.PlanDigest)
	if err != nil || len(digest) != 32 {
		return envelope, ErrJournalConflict
	}
	encoded, _ := json.Marshal(envelope.Artifacts)
	artifactText := string(encoded)
	if _, err := decodeLegacyPoolManifest(&artifactText); err != nil {
		return envelope, err
	}
	for _, entry := range envelope.Artifacts {
		if entry.PlanDigest != envelope.PlanDigest {
			return envelope, ErrJournalConflict
		}
	}
	if err := validateIssuedControlShape(envelope.Controls, envelope.Artifacts); err != nil {
		return envelope, err
	}
	return envelope, nil
}
func validateIssuedControlShape(controls []IssuedProjectionControl, artifacts []poolManifestEntry) error {
	seen := map[uint64]bool{}
	digests := []ProjectionControlDigest{}
	var ack *IssuedProjectionAckControl
	vector := make([]ProjectionVectorEntry, 0, len(artifacts))
	for _, entry := range artifacts {
		vector = append(vector, ProjectionVectorEntry{entry.SourceID, entry.EventID})
	}
	for _, control := range controls {
		if control.ControlID == 0 || seen[control.ControlID] || control.Digest != issuedControlDigest(control) {
			return ErrJournalConflict
		}
		seen[control.ControlID] = true
		n := 0
		if control.Members != nil {
			n++
		}
		if control.Intents != nil {
			n++
		}
		if control.Ack != nil {
			n++
		}
		if n != 1 {
			return ErrJournalConflict
		}
		switch control.Kind {
		case ProjectionMembersFull:
			if control.Members == nil {
				return ErrJournalConflict
			}
		case ProjectionWebhookIntents:
			if control.Intents == nil || !control.Intents.RequireEmpty {
				return ErrJournalConflict
			}
			for _, changes := range [][]IssuedIntentOwnerChange{control.Intents.Add, control.Intents.Remove} {
				owners := map[int]bool{}
				for _, change := range changes {
					if !validGeneratedOwner(change.Owner) || change.Owner.ConfigType != 5 || change.ProofEventID == 0 || owners[change.Owner.ID] {
						return ErrJournalConflict
					}
					owners[change.Owner.ID] = true
					if change.ProofSourceID != 0 {
						found := false
						for _, entry := range artifacts {
							if entry.SourceID == change.ProofSourceID && entry.EventID == change.ProofEventID {
								found = true
								break
							}
						}
						if !found {
							return ErrJournalConflict
						}
					}
				}
			}
		case ProjectionACKFull, ProjectionACKWebhookSubset:
			if control.Ack == nil || ack != nil {
				return ErrJournalConflict
			}
			ack = control.Ack
		default:
			return ErrJournalConflict
		}
		if control.Ack == nil {
			digests = append(digests, ProjectionControlDigest{control.ControlID, control.Digest})
		}
	}
	if ack == nil || !reflect.DeepEqual(ack.ArtifactVector, vector) || !reflect.DeepEqual(ack.RequiredControls, digests) {
		return ErrJournalConflict
	}
	return nil
}

func (d *monitorConfigDAO) CommitGeneratedProjectionPlan(ctx context.Context, req GeneratedProjectionPlanRequest) (ProjectionReceipt, error) {
	prepared, err := prepareProjectionPlan(req)
	if err != nil {
		return ProjectionReceipt{}, err
	}
	receipt, err := d.commitGeneratedPlan(ctx, req.GeneratedPlanRequest, prepared)
	if err != nil {
		return ProjectionReceipt{}, err
	}
	return d.loadProjectionReceiptByRef(ctx, receipt.Reservation)
}
func (d *monitorConfigDAO) loadProjectionReceipt(tx *gorm.DB, op model.MonitorConfigPoolOperation) (ProjectionReceipt, error) {
	base, err := d.loadReceipt(tx, op)
	if err != nil {
		return ProjectionReceipt{}, err
	}
	envelope, err := decodeProjectionManifest(op.Manifest)
	if err != nil {
		return ProjectionReceipt{}, err
	}
	result := ProjectionReceipt{PoolReceipt: base, ManifestVersion: envelope.Version, CompleteControls: envelope.Version == 2}
	if envelope.Version == 2 {
		decoded, _ := hex.DecodeString(envelope.PlanDigest)
		copy(result.PlanDigest[:], decoded)
		result.Controls = envelope.Controls
		for _, control := range result.Controls {
			if !allowedControlFamily(op.Family, control.Kind) {
				return ProjectionReceipt{}, ErrJournalConflict
			}
			if control.Ack != nil {
				if op.Family == 5 {
					if control.Kind != ProjectionACKWebhookSubset || control.Ack.LegacySHA != nil {
						return ProjectionReceipt{}, ErrJournalConflict
					}
				} else if control.Kind != ProjectionACKFull || control.Ack.LegacySHA == nil || !sameHash(*control.Ack.LegacySHA, op.SourceHash) {
					return ProjectionReceipt{}, ErrJournalConflict
				}
			}
		}
	}
	if envelope.Version == 2 {
		members, intents, ack := 0, 0, 0
		desiredIPs := map[string]bool{}
		for _, artifact := range result.Artifacts {
			if artifact.Action == model.ConfigProjectionRender && artifact.ConfigType == op.Family {
				if artifact.Binding == nil {
					return ProjectionReceipt{}, ErrJournalConflict
				}
				desiredIPs[artifact.Binding.InstanceIP] = true
			}
		}
		for _, control := range result.Controls {
			if control.Members != nil {
				members++
				if control.Members.DesiredIPs == nil {
					return ProjectionReceipt{}, ErrJournalConflict
				}
				seen := map[string]bool{}
				for _, ip := range control.Members.DesiredIPs {
					if !desiredIPs[ip] || seen[ip] {
						return ProjectionReceipt{}, ErrJournalConflict
					}
					seen[ip] = true
				}
				if len(seen) != len(desiredIPs) {
					return ProjectionReceipt{}, ErrJournalConflict
				}
			}
			if control.Intents != nil {
				intents++
				if !control.Intents.RequireEmpty || control.Intents.Add == nil || control.Intents.Remove == nil {
					return ProjectionReceipt{}, ErrJournalConflict
				}
			}
			if control.Ack != nil {
				ack++
			}
		}
		if op.Family == 5 {
			if members != 0 || intents != 0 || ack != 1 || len(result.Controls) != 1 {
				return ProjectionReceipt{}, ErrJournalConflict
			}
		} else {
			expectedIntents := 0
			if op.Family == 2 {
				expectedIntents = 1
			}
			if members != 1 || intents != expectedIntents || ack != 1 || len(result.Controls) != 2+expectedIntents {
				return ProjectionReceipt{}, ErrJournalConflict
			}
		}
	}
	return result, nil
}
func (d *monitorConfigDAO) loadProjectionReceiptByRef(ctx context.Context, ref PoolReservation) (ProjectionReceipt, error) {
	var op model.MonitorConfigPoolOperation
	if err := d.db.WithContext(ctx).Where("operation_id = ?", ref.OperationID).First(&op).Error; err != nil {
		return ProjectionReceipt{}, journalError(ctx, err)
	}
	if !operationMatches(op, ref) {
		return ProjectionReceipt{}, ErrJournalConflict
	}
	receipt, err := d.loadProjectionReceipt(d.db.WithContext(ctx), op)
	if err != nil {
		return ProjectionReceipt{}, journalError(ctx, err)
	}
	return receipt, nil
}
func (d *monitorConfigDAO) LoadProjectionByNonce(ctx context.Context, nonce [16]byte) (ProjectionOperationSnapshot, error) {
	snapshot, err := d.LoadOperationByNonce(ctx, nonce)
	if err != nil {
		return ProjectionOperationSnapshot{}, err
	}
	result := ProjectionOperationSnapshot{PoolOperationSnapshot: snapshot}
	if snapshot.Receipt != nil {
		receipt, err := d.loadProjectionReceiptByRef(ctx, snapshot.Reservation)
		if err != nil {
			return ProjectionOperationSnapshot{}, err
		}
		result.Projection = &receipt
	}
	return result, nil
}
func (d *monitorConfigDAO) LoadLastCommittedProjection(ctx context.Context, family int8, poolID int) (*ProjectionReceipt, error) {
	base, err := d.LoadLastCommitted(ctx, family, poolID)
	if err != nil || base == nil {
		return nil, err
	}
	receipt, err := d.loadProjectionReceiptByRef(ctx, base.Reservation)
	if err != nil {
		return nil, err
	}
	return &receipt, nil
}
func (d *monitorConfigDAO) LoadLastProjectedProjection(ctx context.Context, family int8, poolID int) (*ProjectionReceipt, error) {
	base, err := d.LoadLastProjected(ctx, family, poolID)
	if err != nil || base == nil {
		return nil, err
	}
	receipt, err := d.loadProjectionReceiptByRef(ctx, base.Reservation)
	if err != nil {
		return nil, err
	}
	return &receipt, nil
}

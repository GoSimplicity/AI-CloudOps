package config

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"reflect"
	"sort"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ProjectionControlKind string

const (
	ProjectionMembersFull      ProjectionControlKind = "members_full"
	ProjectionWebhookIntents   ProjectionControlKind = "webhook_intents"
	ProjectionACKFull          ProjectionControlKind = "ack_full"
	ProjectionACKWebhookSubset ProjectionControlKind = "ack_webhook_subset"
)

var ErrProjectionControlsRequired = ErrJournalConflict

type ProjectionControl struct {
	ControlID uint64                 `json:"control_id"`
	Kind      ProjectionControlKind  `json:"kind"`
	Members   *FullMembersControl    `json:"members,omitempty"`
	Intents   *WebhookIntentsControl `json:"intents,omitempty"`
	Ack       *ProjectionAckControl  `json:"ack,omitempty"`
}
type FullMembersControl struct {
	DesiredIPs []string `json:"desired_ips"`
}
type ProjectionProofRef struct {
	SourceID uint64 `json:"source_id,omitempty"`
	EventID  uint64 `json:"event_id,omitempty"`
}
type IntentOwnerChange struct {
	Owner             GeneratedConfigOwner  `json:"owner"`
	Proof             ProjectionProofRef    `json:"proof"`
	AliasCurrentOwner *GeneratedConfigOwner `json:"alias_current_owner,omitempty"`
}
type WebhookIntentsControl struct {
	Add          []IntentOwnerChange `json:"add"`
	Remove       []IntentOwnerChange `json:"remove"`
	RequireEmpty bool                `json:"require_empty"`
}
type ProjectionAckControl struct {
	LegacySHA          *[32]byte `json:"legacy_sha,omitempty"`
	ArtifactSourceIDs  []uint64  `json:"artifact_source_ids"`
	RequiredControlIDs []uint64  `json:"required_control_ids"`
}
type ProjectionVectorEntry struct {
	SourceID uint64 `json:"source_id"`
	EventID  uint64 `json:"event_id"`
}
type ProjectionControlDigest struct {
	ControlID uint64   `json:"control_id"`
	Digest    [32]byte `json:"digest"`
}
type IssuedIntentOwnerChange struct {
	Owner             GeneratedConfigOwner  `json:"owner"`
	ProofEventID      uint64                `json:"proof_event_id"`
	ProofSourceID     uint64                `json:"proof_source_id,omitempty"`
	AliasCurrentOwner *GeneratedConfigOwner `json:"alias_current_owner,omitempty"`
}
type IssuedWebhookIntentsControl struct {
	Add          []IssuedIntentOwnerChange `json:"add"`
	Remove       []IssuedIntentOwnerChange `json:"remove"`
	RequireEmpty bool                      `json:"require_empty"`
}
type IssuedProjectionAckControl struct {
	LegacySHA        *[32]byte                 `json:"legacy_sha,omitempty"`
	ArtifactVector   []ProjectionVectorEntry   `json:"artifact_vector"`
	RequiredControls []ProjectionControlDigest `json:"required_controls"`
}
type IssuedProjectionControl struct {
	ControlID uint64                       `json:"control_id"`
	Kind      ProjectionControlKind        `json:"kind"`
	Members   *FullMembersControl          `json:"members,omitempty"`
	Intents   *IssuedWebhookIntentsControl `json:"intents,omitempty"`
	Ack       *IssuedProjectionAckControl  `json:"ack,omitempty"`
	Digest    [32]byte                     `json:"digest"`
}
type GeneratedProjectionPlanRequest struct {
	GeneratedPlanRequest
	Controls []ProjectionControl
}
type ProjectionReceipt struct {
	PoolReceipt
	ManifestVersion  uint8
	CompleteControls bool
	PlanDigest       [32]byte
	Controls         []IssuedProjectionControl
	AMControls       *AMCommittedVector `json:",omitempty"`
}
type ProjectionOperationSnapshot struct {
	PoolOperationSnapshot
	Projection *ProjectionReceipt
}
type ProjectionAckProof struct {
	Reservation      PoolReservation
	SourceHash       [32]byte
	PlanDigest       [32]byte
	ArtifactVector   []ProjectionVectorEntry
	RequiredControls []ProjectionControlDigest
}

type ProjectionPlanJournalDAO interface {
	ProjectionJournalDAO
	CommitGeneratedProjectionPlan(context.Context, GeneratedProjectionPlanRequest) (ProjectionReceipt, error)
	LoadProjectionByNonce(context.Context, [16]byte) (ProjectionOperationSnapshot, error)
	LoadLastCommittedProjection(context.Context, int8, int) (*ProjectionReceipt, error)
	LoadLastProjectedProjection(context.Context, int8, int) (*ProjectionReceipt, error)
	MarkProjectionCompleteIfCurrent(context.Context, ProjectionReceipt, ProjectionAckProof) (bool, error)
	WithPoolProjectionLease(context.Context, ProjectionReceipt, func(ProjectionLease) error) error
	WithArtifactProjectionLease(context.Context, ArtifactReceipt, func(ProjectionLease) error) error
}

func NewProjectionPlanJournalDAO(logger *zap.Logger, db *gorm.DB) ProjectionPlanJournalDAO {
	return &monitorConfigDAO{db: db, l: logger}
}

func cloneProjectionControls(input []ProjectionControl) []ProjectionControl {
	out := make([]ProjectionControl, len(input))
	for i, value := range input {
		out[i] = value
		if value.Members != nil {
			copy := *value.Members
			copy.DesiredIPs = append([]string(nil), value.Members.DesiredIPs...)
			out[i].Members = &copy
		}
		if value.Intents != nil {
			copy := *value.Intents
			copy.Add = append([]IntentOwnerChange(nil), value.Intents.Add...)
			copy.Remove = append([]IntentOwnerChange(nil), value.Intents.Remove...)
			for j := range copy.Add {
				if copy.Add[j].AliasCurrentOwner != nil {
					owner := *copy.Add[j].AliasCurrentOwner
					copy.Add[j].AliasCurrentOwner = &owner
				}
			}
			for j := range copy.Remove {
				if copy.Remove[j].AliasCurrentOwner != nil {
					owner := *copy.Remove[j].AliasCurrentOwner
					copy.Remove[j].AliasCurrentOwner = &owner
				}
			}

			out[i].Intents = &copy
		}
		if value.Ack != nil {
			copy := *value.Ack
			if value.Ack.LegacySHA != nil {
				hash := *value.Ack.LegacySHA
				copy.LegacySHA = &hash
			}
			copy.ArtifactSourceIDs = append([]uint64(nil), value.Ack.ArtifactSourceIDs...)
			copy.RequiredControlIDs = append([]uint64(nil), value.Ack.RequiredControlIDs...)
			out[i].Ack = &copy
		}
	}
	return out
}
func cloneProjectionReceipt(input ProjectionReceipt) ProjectionReceipt {
	raw, _ := json.Marshal(input)
	var out ProjectionReceipt
	_ = json.Unmarshal(raw, &out)
	return out
}
func issuedControlDigest(value IssuedProjectionControl) [32]byte {
	value.Digest = [32]byte{}
	raw, _ := json.Marshal(value)
	return sha256.Sum256(raw)
}
func sortedUniqueIDs(ids []uint64) ([]uint64, bool) {
	out := append([]uint64(nil), ids...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	for i, id := range out {
		if id == 0 || i > 0 && id == out[i-1] {
			return nil, false
		}
	}
	if out == nil {
		out = []uint64{}
	}
	return out, true
}
func equalProjectionPayload(a, b ProjectionReceipt) bool {
	return a.Reservation == b.Reservation && a.SourceHash == b.SourceHash &&
		a.ManifestVersion == b.ManifestVersion && a.CompleteControls == b.CompleteControls &&
		a.PlanDigest == b.PlanDigest && reflect.DeepEqual(a.Artifacts, b.Artifacts) &&
		reflect.DeepEqual(a.Controls, b.Controls) && reflect.DeepEqual(a.AMControls, b.AMControls)
}
func projectionAckProof(receipt ProjectionReceipt) (ProjectionAckProof, error) {
	for _, control := range receipt.Controls {
		if control.Ack != nil {
			return ProjectionAckProof{
				Reservation:      receipt.Reservation,
				SourceHash:       receipt.SourceHash,
				PlanDigest:       receipt.PlanDigest,
				ArtifactVector:   append([]ProjectionVectorEntry(nil), control.Ack.ArtifactVector...),
				RequiredControls: append([]ProjectionControlDigest(nil), control.Ack.RequiredControls...),
			}, nil
		}
	}
	return ProjectionAckProof{}, ErrJournalConflict
}
func allowedControlFamily(family int8, kind ProjectionControlKind) bool {
	switch kind {
	case ProjectionMembersFull:
		return family >= 1 && family <= 4
	case ProjectionWebhookIntents:
		return family == model.ConfigPublicationFamilyAlertManager
	case ProjectionACKFull:
		return family >= 1 && family <= 4
	case ProjectionACKWebhookSubset:
		return family == model.ConfigPublicationFamilyAlertManagerWebhook
	}
	return false
}

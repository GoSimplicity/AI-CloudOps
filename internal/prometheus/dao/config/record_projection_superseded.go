package config

import (
	"context"
)

const (
	RecordSupersededAcceptedArtifact = "accepted_artifact"
	RecordSupersededLegacyOwner      = "legacy_owner"
)

type RecordSupersededArtifact struct {
	Origin                 string
	RetiredDeletedAt       uint64
	AcceptedOperationID    uint64
	AcceptedEventID        uint64
	AcceptedAction         string
	RetiredOwner           GeneratedConfigOwner
	ReplacementOperationID uint64
	ReplacementEventID     uint64
	ReplacementOwner       GeneratedConfigOwner
	ReplacementBodyDigest  [32]byte
	ReplacementHash        string
	ReplacementStatus      int8
}

func (d *monitorConfigDAO) GetSupersededRecordArtifacts(ctx context.Context, previous ProjectionReceipt) ([]RecordSupersededArtifact, error) {
	return d.getSupersededScrapeRuleArtifacts(ctx, ScrapeRecordRule, previous)
}
func (d *monitorConfigDAO) GetLegacySupersededRecordArtifacts(ctx context.Context, poolID int, owners []GeneratedConfigOwner) ([]RecordSupersededArtifact, error) {
	return d.getLegacySupersededScrapeRuleArtifacts(ctx, ScrapeRecordRule, poolID, owners)
}

package config

import (
	"context"
)

func validDispositionFamily(family int8) bool { return family >= 1 && family <= 5 }
func dispositionFamilies(kind int8) []int8 {
	if kind == 5 {
		return []int8{2, 5}
	}
	return []int8{kind}
}
func (d *monitorConfigDAO) GetSupersededAMArtifacts(ctx context.Context, previous ProjectionReceipt) ([]RecordSupersededArtifact, error) {
	if !validAMFamily(previous.Reservation.Family) {
		return nil, ErrJournalInvalid
	}
	return d.getSupersededScrapeRuleArtifacts(ctx, scrapePublicationScope(previous.Reservation.Family), previous)
}
func (d *monitorConfigDAO) GetLegacySupersededAMArtifacts(ctx context.Context, poolID int, owners []GeneratedConfigOwner) ([]RecordSupersededArtifact, error) {
	result := []RecordSupersededArtifact{}
	for _, kind := range []int8{2, 5} {
		group := []GeneratedConfigOwner{}
		for _, owner := range owners {
			if owner.ConfigType == kind {
				group = append(group, owner)
			} else if owner.ConfigType != 2 && owner.ConfigType != 5 {
				return nil, ErrJournalInvalid
			}
		}
		proofs, err := d.getLegacySupersededScrapeRuleArtifacts(ctx, scrapePublicationScope(kind), poolID, group)
		if err != nil {
			return nil, err
		}
		result = append(result, proofs...)
	}
	return result, nil
}
func (d *monitorConfigDAO) GetResolvedAMOwners(ctx context.Context, poolID int, owners []GeneratedConfigOwner) ([]GeneratedConfigOwner, error) {
	result := []GeneratedConfigOwner{}
	for _, kind := range []int8{2, 5} {
		group := []GeneratedConfigOwner{}
		for _, owner := range owners {
			if owner.ConfigType == kind {
				group = append(group, owner)
			} else if owner.ConfigType != 2 && owner.ConfigType != 5 {
				return nil, ErrJournalInvalid
			}
		}
		resolved, err := d.getProjectedScrapeRuleRetirements(ctx, scrapePublicationScope(kind), poolID, group)
		if err != nil {
			return nil, err
		}
		result = append(result, resolved...)
	}
	return result, nil
}
func prepareAMReplacements(req AMProjectionPlanRequest) ([]RecordSupersededArtifact, error) {
	if req.Reservation.Family == 5 && len(req.Superseded) > 0 {
		return nil, ErrJournalInvalid
	}
	return preparePublicationReplacements(RecordProjectionPlanRequest{GeneratedProjectionPlanRequest: req.GeneratedProjectionPlanRequest, History: req.History, Superseded: req.Superseded}, true)
}

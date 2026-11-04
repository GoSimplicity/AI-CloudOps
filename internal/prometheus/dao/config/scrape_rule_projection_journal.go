package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ScrapeRuleHistoryGuardOf(scope ScrapeRuleScope, instanceKey string, rows []*model.MonitorConfig) (RecordHistoryGuard, error) {
	if !validScrapeRuleScope(scope) {
		return RecordHistoryGuard{}, ErrJournalInvalid
	}
	return scrapePublicationHistoryGuardOf(scrapePublicationScope(scope), instanceKey, rows)
}
func scrapePublicationHistoryGuardOf(scope scrapePublicationScope, instanceKey string, rows []*model.MonitorConfig) (RecordHistoryGuard, error) {
	if !validScrapePublicationScope(scope) {
		return RecordHistoryGuard{}, ErrJournalInvalid
	}
	if !validConfigNamespace(configNamespaceKey{int8(scope), instanceKey}) {
		return RecordHistoryGuard{}, ErrJournalInvalid
	}
	result := RecordHistoryGuard{InstanceKey: instanceKey, Rows: make([]GeneratedConfigHistoryRow, 0, len(rows))}
	seen := map[int]bool{}
	for _, row := range rows {
		if row == nil || row.ID <= 0 || row.ConfigType != int8(scope) || row.InstanceIP != instanceKey || seen[row.ID] {
			return RecordHistoryGuard{}, ErrJournalInvalid
		}
		seen[row.ID] = true
		result.Rows = append(result.Rows, GeneratedConfigHistoryRow{
			Owner: GeneratedConfigOwnerOf(row), Status: row.Status, DeletedAt: uint64(row.DeletedAt),
			ConfigHash: row.ConfigHash, BodyDigest: sha256.Sum256([]byte(row.ConfigContent)),
		})
	}
	sort.Slice(result.Rows, func(i, j int) bool { return result.Rows[i].Owner.ID < result.Rows[j].Owner.ID })
	return result, nil
}
func prepareScrapeRuleHistory(req RecordProjectionPlanRequest) ([]RecordHistoryGuard, error) {
	if !validScrapePublicationScope(scrapePublicationScope(req.Reservation.Family)) {
		return nil, ErrJournalInvalid
	}
	retired := map[string]GeneratedConfigOwner{}
	for _, item := range req.Items {
		if item.Action == model.ConfigProjectionRetire {
			if item.Expected == nil {
				return nil, ErrJournalInvalid
			}
			retired[item.Expected.InstanceIP] = *item.Expected
		}
	}
	if len(retired) != len(req.History) {
		return nil, ErrJournalInvalid
	}
	result := make([]RecordHistoryGuard, 0, len(req.History))
	seenKeys := map[string]bool{}
	for _, guard := range req.History {
		expected, ok := retired[guard.InstanceKey]
		if !ok || seenKeys[guard.InstanceKey] || len(guard.Rows) == 0 {
			return nil, ErrJournalInvalid
		}
		seenKeys[guard.InstanceKey] = true
		copied := RecordHistoryGuard{InstanceKey: guard.InstanceKey, Rows: append([]GeneratedConfigHistoryRow(nil), guard.Rows...)}
		sort.Slice(copied.Rows, func(i, j int) bool { return copied.Rows[i].Owner.ID < copied.Rows[j].Owner.ID })
		if err := validateScrapePublicationRetirementHistory(scrapePublicationScope(req.Reservation.Family), expected, copied); err != nil {
			return nil, err
		}
		result = append(result, copied)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].InstanceKey < result[j].InstanceKey })
	return result, nil
}
func (d *monitorConfigDAO) CommitAlertRuleProjectionPlan(ctx context.Context, req ScrapeRuleProjectionPlanRequest) (ProjectionReceipt, error) {
	if req.Reservation.Family != 3 {
		return ProjectionReceipt{}, ErrJournalInvalid
	}
	return d.commitScrapeRuleProjectionPlan(ctx, req)
}
func (d *monitorConfigDAO) commitScrapeRuleProjectionPlan(ctx context.Context, req RecordProjectionPlanRequest) (ProjectionReceipt, error) {
	prepared, err := prepareProjectionPlan(req.GeneratedProjectionPlanRequest)
	if err != nil {
		return ProjectionReceipt{}, err
	}
	guards, err := prepareScrapeRuleHistory(req)
	if err != nil {
		return ProjectionReceipt{}, err
	}
	replacements, err := prepareScrapeRuleReplacements(req)
	if err != nil {
		return ProjectionReceipt{}, err
	}
	prepared.superseded = replacements
	for _, proof := range replacements {
		prepared.keys = append(prepared.keys, configNamespaceKey{req.Reservation.Family, proof.RetiredOwner.InstanceIP})
	}
	original := hex.EncodeToString(prepared.digest[:])
	raw, err := json.Marshal(struct {
		ProjectionDigest string
		History          []RecordHistoryGuard
		Superseded       []RecordSupersededArtifact
	}{original, guards, replacements})
	if err != nil {
		return ProjectionReceipt{}, ErrJournalInvalid
	}
	prepared.digest = sha256.Sum256(raw)
	prepared.history = guards
	receipt, err := d.commitGeneratedPlan(ctx, req.GeneratedPlanRequest, prepared)
	if err != nil {
		return ProjectionReceipt{}, err
	}
	return d.loadProjectionReceiptByRef(ctx, receipt.Reservation)
}

// Called only after control -> all ordered namespace guards, before the first
// metadata/event/manifest write. FOR UPDATE is a current read under MySQL RR.
func validateScrapeRuleHistoryTx(tx *gorm.DB, family int8, guards []RecordHistoryGuard) error {
	for _, guard := range guards {
		query := tx.Unscoped().Model(&model.MonitorConfig{}).Where("config_type = ?", family)
		if tx.Dialector.Name() == "mysql" {
			query = query.Where("instance_ip = ? AND BINARY instance_ip = BINARY ?", guard.InstanceKey, guard.InstanceKey)
		} else {
			query = query.Where("instance_ip COLLATE BINARY = ?", guard.InstanceKey)
		}
		var rows []*model.MonitorConfig
		if err := query.Clauses(clause.Locking{Strength: "UPDATE"}).Order("id ASC").Find(&rows).Error; err != nil {
			return err
		}
		actual, err := scrapePublicationHistoryGuardOf(scrapePublicationScope(family), guard.InstanceKey, rows)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(actual, guard) {
			return ErrJournalConflict
		}
	}
	return nil
}

// The live expected owner grants permission. Deleted history is observation,
// not a competing live owner; the transaction must still match every field.
// A tombstone-only lookup has no such live permission and stays strict.
func ValidateScrapeRuleRetirementHistory(scope ScrapeRuleScope, expected GeneratedConfigOwner, guard RecordHistoryGuard) error {
	if !validScrapeRuleScope(scope) {
		return ErrJournalInvalid
	}
	return validateScrapePublicationRetirementHistory(scrapePublicationScope(scope), expected, guard)
}
func validateScrapePublicationRetirementHistory(scope scrapePublicationScope, expected GeneratedConfigOwner, guard RecordHistoryGuard) error {
	if !validScrapePublicationScope(scope) {
		return ErrJournalInvalid
	}
	if !validGeneratedOwner(expected) || expected.ConfigType != int8(scope) || expected.InstanceIP != guard.InstanceKey || !generatedRowNameValid(scrapeRuleOwnerModel(expected)) {
		return ErrJournalConflict
	}
	var own *GeneratedConfigHistoryRow
	live := 0
	seen := map[int]bool{}
	for i := range guard.Rows {
		row := &guard.Rows[i]
		owner := row.Owner
		if owner.ID <= 0 || owner.ConfigType != int8(scope) || owner.InstanceIP != guard.InstanceKey || seen[owner.ID] {
			return ErrJournalConflict
		}
		seen[owner.ID] = true
		if row.DeletedAt == 0 {
			live++
		}
		if owner == expected {
			own = row
		}
	}
	if own == nil {
		return ErrJournalConflict
	}
	if own.DeletedAt == 0 {
		if live != 1 {
			return ErrJournalConflict
		}
		return nil
	}
	if live != 0 {
		return ErrJournalConflict
	}
	for _, row := range guard.Rows {
		if row.Owner.PoolID != expected.PoolID || !generatedRowNameValid(scrapeRuleOwnerModel(row.Owner)) {
			return ErrJournalConflict
		}
	}
	return nil
}

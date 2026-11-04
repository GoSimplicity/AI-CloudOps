package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// History guards are preconditions, not new artifact permission. They preserve
// the complete namespace, including deleted foreign/custom rows and status.
type GeneratedConfigHistoryRow struct {
	Owner      GeneratedConfigOwner
	Status     int8
	DeletedAt  uint64
	ConfigHash string
	BodyDigest [32]byte
}
type RecordHistoryGuard struct {
	InstanceKey string
	Rows        []GeneratedConfigHistoryRow
}
type RecordProjectionPlanRequest struct {
	GeneratedProjectionPlanRequest
	History    []RecordHistoryGuard
	Superseded []RecordSupersededArtifact
}
type RecordProjectionJournalDAO interface {
	ProjectionPlanJournalDAO
	ReadRecordSourceVersions(context.Context) ([]RecordSourceVersion, error)
	ReserveRecordPoolOperation(context.Context, PoolOperationRequest, uint64) (PoolReservation, error)
	CommitRecordProjectionPlan(context.Context, RecordProjectionPlanRequest) (ProjectionReceipt, error)
}

func NewRecordProjectionJournalDAO(logger *zap.Logger, db *gorm.DB) RecordProjectionJournalDAO {
	return &monitorConfigDAO{l: logger, db: db}
}

func RecordHistoryGuardOf(instanceKey string, rows []*model.MonitorConfig) (RecordHistoryGuard, error) {
	if !validConfigNamespace(configNamespaceKey{4, instanceKey}) {
		return RecordHistoryGuard{}, ErrJournalInvalid
	}
	result := RecordHistoryGuard{InstanceKey: instanceKey, Rows: make([]GeneratedConfigHistoryRow, 0, len(rows))}
	seen := map[int]bool{}
	for _, row := range rows {
		if row == nil || row.ID <= 0 || row.ConfigType != 4 || row.InstanceIP != instanceKey || seen[row.ID] {
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
func prepareRecordHistory(req RecordProjectionPlanRequest) ([]RecordHistoryGuard, error) {
	if req.Reservation.Family != model.ConfigPublicationFamilyRecordRule {
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
		if err := ValidateRecordRetirementHistory(expected, copied); err != nil {
			return nil, err
		}
		result = append(result, copied)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].InstanceKey < result[j].InstanceKey })
	return result, nil
}
func (d *monitorConfigDAO) CommitRecordProjectionPlan(ctx context.Context, req RecordProjectionPlanRequest) (ProjectionReceipt, error) {
	prepared, err := prepareProjectionPlan(req.GeneratedProjectionPlanRequest)
	if err != nil {
		return ProjectionReceipt{}, err
	}
	guards, err := prepareRecordHistory(req)
	if err != nil {
		return ProjectionReceipt{}, err
	}
	replacements, err := prepareRecordReplacements(req)
	if err != nil {
		return ProjectionReceipt{}, err
	}
	prepared.superseded = replacements
	for _, proof := range replacements {
		prepared.keys = append(prepared.keys, configNamespaceKey{4, proof.RetiredOwner.InstanceIP})
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
func validateRecordHistoryTx(tx *gorm.DB, guards []RecordHistoryGuard) error {
	for _, guard := range guards {
		query := tx.Unscoped().Model(&model.MonitorConfig{}).Where("config_type = ?", 4)
		if tx.Dialector.Name() == "mysql" {
			query = query.Where("instance_ip = ? AND BINARY instance_ip = BINARY ?", guard.InstanceKey, guard.InstanceKey)
		} else {
			query = query.Where("instance_ip COLLATE BINARY = ?", guard.InstanceKey)
		}
		var rows []*model.MonitorConfig
		if err := query.Clauses(clause.Locking{Strength: "UPDATE"}).Order("id ASC").Find(&rows).Error; err != nil {
			return err
		}
		actual, err := RecordHistoryGuardOf(guard.InstanceKey, rows)
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
func ValidateRecordRetirementHistory(expected GeneratedConfigOwner, guard RecordHistoryGuard) error {
	if !validGeneratedOwner(expected) || expected.ConfigType != 4 || expected.InstanceIP != guard.InstanceKey || !generatedRowNameValid(recordOwnerModel(expected)) {
		return ErrJournalConflict
	}
	var own *GeneratedConfigHistoryRow
	live := 0
	seen := map[int]bool{}
	for i := range guard.Rows {
		row := &guard.Rows[i]
		owner := row.Owner
		if owner.ID <= 0 || owner.ConfigType != 4 || owner.InstanceIP != guard.InstanceKey || seen[owner.ID] {
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
		if row.Owner.PoolID != expected.PoolID || !generatedRowNameValid(recordOwnerModel(row.Owner)) {
			return ErrJournalConflict
		}
	}
	return nil
}

package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AM source publication is coordinated as a pair. Counters order attempts only
// within their own family; they are never compared across family 2 and 5.
type AMSourceVersion struct {
	PoolID                     int
	FullCounter                uint64
	SubsetCounter              uint64
	FullCommittedOperationID   *uint64
	SubsetCommittedOperationID *uint64
}
type AMCommittedVector struct {
	FullOperationID   *uint64
	SubsetOperationID *uint64
}
type AMProjectionPlanRequest struct {
	GeneratedProjectionPlanRequest
	SourceVersion AMSourceVersion
	History       []RecordHistoryGuard
	Superseded    []RecordSupersededArtifact
}
type AMProjectionJournalDAO interface {
	ProjectionPlanJournalDAO
	WithAMCoverage(context.Context, AMCoverageRequest, func(AMCoverageSnapshot) error) (bool, error)
	ReadAMSourceVersions(context.Context) ([]AMSourceVersion, error)
	ReserveAMPoolOperation(context.Context, PoolOperationRequest, AMSourceVersion) (PoolReservation, error)
	CommitAMProjectionPlan(context.Context, AMProjectionPlanRequest) (ProjectionReceipt, error)
}
type AMProjectionHistoryReader interface {
	GetGeneratedConfigHistoryCandidates(context.Context, GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error)
	GetGeneratedConfigHistoryPage(context.Context, GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error)
	GetSupersededAMArtifacts(context.Context, ProjectionReceipt) ([]RecordSupersededArtifact, error)
	GetLegacySupersededAMArtifacts(context.Context, int, []GeneratedConfigOwner) ([]RecordSupersededArtifact, error)
	GetResolvedAMOwners(context.Context, int, []GeneratedConfigOwner) ([]GeneratedConfigOwner, error)
}

func NewAMProjectionJournalDAO(logger *zap.Logger, db *gorm.DB) AMProjectionJournalDAO {
	return &monitorConfigDAO{l: logger, db: db}
}
func NewAMProjectionHistoryReader(logger *zap.Logger, db *gorm.DB) AMProjectionHistoryReader {
	return &monitorConfigDAO{l: logger, db: db}
}

func AMHistoryGuardOf(kind int8, key string, rows []*model.MonitorConfig) (RecordHistoryGuard, error) {
	if kind != 2 && kind != 5 {
		return RecordHistoryGuard{}, ErrJournalInvalid
	}
	return namespaceHistoryGuardOf(kind, key, rows)
}
func ValidateAMRetirementHistory(expected GeneratedConfigOwner, guard RecordHistoryGuard) error {
	if expected.ConfigType != 2 && expected.ConfigType != 5 {
		return ErrJournalInvalid
	}
	return validateNamespaceRetirementHistory(expected.ConfigType, expected, guard)
}
func prepareAMHistory(req AMProjectionPlanRequest) ([]RecordHistoryGuard, error) {
	retired := map[configNamespaceKey]GeneratedConfigOwner{}
	for _, item := range req.Items {
		if item.Action == model.ConfigProjectionRetire {
			if item.Expected == nil {
				return nil, ErrJournalInvalid
			}
			retired[configNamespaceKey{item.Expected.ConfigType, item.Expected.InstanceIP}] = *item.Expected
		}
	}
	if len(retired) != len(req.History) {
		return nil, ErrJournalInvalid
	}
	result := make([]RecordHistoryGuard, 0, len(req.History))
	seen := map[configNamespaceKey]bool{}
	for _, guard := range req.History {
		if len(guard.Rows) == 0 {
			return nil, ErrJournalInvalid
		}
		key := configNamespaceKey{guard.Rows[0].Owner.ConfigType, guard.InstanceKey}
		expected, ok := retired[key]
		if !ok || seen[key] {
			return nil, ErrJournalInvalid
		}
		seen[key] = true
		copied := RecordHistoryGuard{InstanceKey: guard.InstanceKey, Rows: append([]GeneratedConfigHistoryRow(nil), guard.Rows...)}
		sort.Slice(copied.Rows, func(i, j int) bool { return copied.Rows[i].Owner.ID < copied.Rows[j].Owner.ID })
		if err := ValidateAMRetirementHistory(expected, copied); err != nil {
			return nil, err
		}
		result = append(result, copied)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Rows[0].Owner.ConfigType != result[j].Rows[0].Owner.ConfigType {
			return result[i].Rows[0].Owner.ConfigType < result[j].Rows[0].Owner.ConfigType
		}
		return result[i].InstanceKey < result[j].InstanceKey
	})
	return result, nil
}
func (d *monitorConfigDAO) CommitAMProjectionPlan(ctx context.Context, req AMProjectionPlanRequest) (ProjectionReceipt, error) {
	if !validAMFamily(req.Reservation.Family) || req.SourceVersion.PoolID != req.Reservation.PoolID {
		return ProjectionReceipt{}, ErrJournalInvalid
	}
	prepared, err := prepareProjectionPlan(req.GeneratedProjectionPlanRequest)
	if err != nil {
		return ProjectionReceipt{}, err
	}
	guards, err := prepareAMHistory(req)
	if err != nil {
		return ProjectionReceipt{}, err
	}
	if req.Reservation.Family == 5 {
		for _, item := range req.Items {
			if item.Action != model.ConfigProjectionRender || item.Desired == nil || item.Desired.ConfigType != 5 {
				return ProjectionReceipt{}, ErrJournalInvalid
			}
		}
		if len(guards) != 0 {
			return ProjectionReceipt{}, ErrJournalInvalid
		}
	}
	// Supersession uses the same typed namespace proof core; no raw permission is
	// acquired from an intent member or a free-form Redis key.
	replacements, err := prepareAMReplacements(req)
	if err != nil {
		return ProjectionReceipt{}, err
	}
	original := hex.EncodeToString(prepared.digest[:])
	raw, err := json.Marshal(struct {
		ProjectionDigest string
		SourceVersion    AMSourceVersion
		History          []RecordHistoryGuard
		Superseded       []RecordSupersededArtifact
	}{original, req.SourceVersion, guards, replacements})
	if err != nil {
		return ProjectionReceipt{}, ErrJournalInvalid
	}
	source := req.SourceVersion
	source.FullCommittedOperationID = cloneUint64(source.FullCommittedOperationID)
	source.SubsetCommittedOperationID = cloneUint64(source.SubsetCommittedOperationID)
	vector := AMCommittedVector{FullOperationID: cloneUint64(source.FullCommittedOperationID), SubsetOperationID: cloneUint64(source.SubsetCommittedOperationID)}
	if req.Reservation.Family == 2 {
		vector.FullOperationID = cloneUint64(&req.Reservation.OperationID)
	} else {
		vector.SubsetOperationID = cloneUint64(&req.Reservation.OperationID)
	}
	prepared.amControls = &vector

	prepared.amSource = &source
	prepared.digest = sha256.Sum256(raw)
	prepared.history = guards
	prepared.superseded = replacements
	for _, proof := range replacements {
		prepared.keys = append(prepared.keys, configNamespaceKey{proof.RetiredOwner.ConfigType, proof.RetiredOwner.InstanceIP})
	}
	receipt, err := d.commitGeneratedPlan(ctx, req.GeneratedPlanRequest, prepared)
	if err != nil {
		return ProjectionReceipt{}, err
	}
	return d.loadProjectionReceiptByRef(ctx, receipt.Reservation)
}

package config

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
	return ScrapeRuleHistoryGuardOf(ScrapeRecordRule, instanceKey, rows)
}
func (d *monitorConfigDAO) CommitRecordProjectionPlan(ctx context.Context, req RecordProjectionPlanRequest) (ProjectionReceipt, error) {
	if req.Reservation.Family != 4 {
		return ProjectionReceipt{}, ErrJournalInvalid
	}
	return d.commitScrapeRuleProjectionPlan(ctx, req)
}
func ValidateRecordRetirementHistory(expected GeneratedConfigOwner, guard RecordHistoryGuard) error {
	return ValidateScrapeRuleRetirementHistory(ScrapeRecordRule, expected, guard)
}

package config

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PromMainSourceVersion = RecordSourceVersion
type PromMainHistoryGuard = RecordHistoryGuard
type PromMainSupersededArtifact = RecordSupersededArtifact
type PromMainProjectionPlanRequest = RecordProjectionPlanRequest

type PromMainProjectionJournalDAO interface {
	ProjectionPlanJournalDAO
	ReadPromMainSourceVersions(context.Context) ([]PromMainSourceVersion, error)
	ReservePromMainPoolOperation(context.Context, PoolOperationRequest, uint64) (PoolReservation, error)
	CommitPromMainProjectionPlan(context.Context, PromMainProjectionPlanRequest) (ProjectionReceipt, error)
}
type PromMainHistoryReader interface {
	GetGeneratedConfigHistoryCandidates(context.Context, GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error)
	GetGeneratedConfigHistoryPage(context.Context, GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error)
	GetSupersededPromMainArtifacts(context.Context, ProjectionReceipt) ([]PromMainSupersededArtifact, error)
	GetLegacySupersededPromMainArtifacts(context.Context, int, []GeneratedConfigOwner) ([]PromMainSupersededArtifact, error)
	GetProjectedPromMainRetirements(context.Context, int, []GeneratedConfigOwner) ([]GeneratedConfigOwner, error)
}

func NewPromMainProjectionJournalDAO(logger *zap.Logger, db *gorm.DB) PromMainProjectionJournalDAO {
	return &monitorConfigDAO{l: logger, db: db}
}
func NewPromMainHistoryReader(logger *zap.Logger, db *gorm.DB) PromMainHistoryReader {
	return &monitorConfigDAO{l: logger, db: db}
}
func (d *monitorConfigDAO) ReadPromMainSourceVersions(ctx context.Context) ([]PromMainSourceVersion, error) {
	return d.readScrapeRuleSourceVersions(ctx, scrapePromPublication)
}
func (d *monitorConfigDAO) ReservePromMainPoolOperation(ctx context.Context, req PoolOperationRequest, counter uint64) (PoolReservation, error) {
	if req.Family != 1 {
		return PoolReservation{}, ErrJournalInvalid
	}
	return d.reservePoolOperation(ctx, req, &counter)
}
func (d *monitorConfigDAO) CommitPromMainProjectionPlan(ctx context.Context, req PromMainProjectionPlanRequest) (ProjectionReceipt, error) {
	if req.Reservation.Family != 1 {
		return ProjectionReceipt{}, ErrJournalInvalid
	}
	return d.commitScrapeRuleProjectionPlan(ctx, req)
}
func (d *monitorConfigDAO) GetSupersededPromMainArtifacts(ctx context.Context, previous ProjectionReceipt) ([]PromMainSupersededArtifact, error) {
	return d.getSupersededScrapeRuleArtifacts(ctx, scrapePromPublication, previous)
}
func (d *monitorConfigDAO) GetLegacySupersededPromMainArtifacts(ctx context.Context, poolID int, owners []GeneratedConfigOwner) ([]PromMainSupersededArtifact, error) {
	return d.getLegacySupersededScrapeRuleArtifacts(ctx, scrapePromPublication, poolID, owners)
}
func (d *monitorConfigDAO) GetProjectedPromMainRetirements(ctx context.Context, poolID int, owners []GeneratedConfigOwner) ([]GeneratedConfigOwner, error) {
	return d.getProjectedScrapeRuleRetirements(ctx, scrapePromPublication, poolID, owners)
}
func PromMainHistoryGuardOf(key string, rows []*model.MonitorConfig) (PromMainHistoryGuard, error) {
	return scrapePublicationHistoryGuardOf(scrapePromPublication, key, rows)
}
func ValidatePromMainRetirementHistory(expected GeneratedConfigOwner, guard PromMainHistoryGuard) error {
	return validateScrapePublicationRetirementHistory(scrapePromPublication, expected, guard)
}

package config

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Only the two Scrape rule publishers share this lifecycle. Prometheus main
// and Alertmanager have different rendering/control contracts.
type ScrapeRuleScope int8

const (
	ScrapeAlertRule  ScrapeRuleScope = 3
	ScrapeRecordRule ScrapeRuleScope = 4
)

func validScrapeRuleScope(scope ScrapeRuleScope) bool {
	return scope == ScrapeAlertRule || scope == ScrapeRecordRule
}

type ScrapeRuleSourceVersion = RecordSourceVersion
type ScrapeRuleHistoryGuard = RecordHistoryGuard
type ScrapeRuleSupersededArtifact = RecordSupersededArtifact
type ScrapeRuleProjectionPlanRequest = RecordProjectionPlanRequest

type AlertRuleProjectionJournalDAO interface {
	ProjectionPlanJournalDAO
	ReadAlertRuleSourceVersions(context.Context) ([]ScrapeRuleSourceVersion, error)
	ReserveAlertRulePoolOperation(context.Context, PoolOperationRequest, uint64) (PoolReservation, error)
	CommitAlertRuleProjectionPlan(context.Context, ScrapeRuleProjectionPlanRequest) (ProjectionReceipt, error)
}
type AlertRuleHistoryReader interface {
	GetGeneratedConfigHistoryCandidates(context.Context, GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error)
	GetGeneratedConfigHistoryPage(context.Context, GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error)
	GetSupersededAlertRuleArtifacts(context.Context, ProjectionReceipt) ([]ScrapeRuleSupersededArtifact, error)
	GetLegacySupersededAlertRuleArtifacts(context.Context, int, []GeneratedConfigOwner) ([]ScrapeRuleSupersededArtifact, error)
	GetProjectedAlertRuleRetirements(context.Context, int, []GeneratedConfigOwner) ([]GeneratedConfigOwner, error)
}

func NewAlertRuleProjectionJournalDAO(logger *zap.Logger, db *gorm.DB) AlertRuleProjectionJournalDAO {
	return &monitorConfigDAO{l: logger, db: db}
}
func NewAlertRuleHistoryReader(logger *zap.Logger, db *gorm.DB) AlertRuleHistoryReader {
	return &monitorConfigDAO{l: logger, db: db}
}

// Private publication scope includes exactly the three real Scrape consumers.
// The published ScrapeRuleScope and its helpers remain restricted to 3/4.
type scrapePublicationScope int8

const (
	scrapePromPublication   scrapePublicationScope = 1
	scrapeAlertPublication  scrapePublicationScope = 3
	scrapeRecordPublication scrapePublicationScope = 4
)

func validScrapePublicationScope(scope scrapePublicationScope) bool {
	return scope == scrapePromPublication || scope == scrapeAlertPublication || scope == scrapeRecordPublication
}

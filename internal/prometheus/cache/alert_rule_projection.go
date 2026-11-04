package cache

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

type AlertRuleProjectionRunner interface {
	Project(context.Context, configdao.ProjectionLease) error
}

func NewAlertRuleProjectionRunner(store *ProjectionStore) AlertRuleProjectionRunner {
	return &scrapeProjectionRunner{store: store, scope: scrapeAlertScope}
}
func (r *alertRuleConfigCache) projectionContext() (*scrapeProjection, error) {
	journal, ok := r.journal.(configdao.AlertRuleProjectionJournalDAO)
	if !ok || journal == nil || r.history == nil || r.projection == nil {
		return nil, errors.New("告警规则持久发布依赖不可用")
	}
	result := &scrapeProjection{scope: scrapeAlertScope, redis: r.redis, configDAO: r.ruleConfigDAO(), journal: alertJournalFacade{journal}, history: alertHistoryFacade{r.history}, projection: r.projection}
	return result, result.ready()
}
func (r *alertRuleConfigCache) alertRulePublicationReady() error {
	_, err := r.projectionContext()
	return err
}
func (r *alertRuleConfigCache) alertRuleSourceVersions(ctx context.Context) (map[int]uint64, error) {
	publisher, err := r.projectionContext()
	if err != nil {
		return nil, err
	}
	return publisher.sourceVersions(ctx)
}
func (r *alertRuleConfigCache) publishAlertRuleProjection(ctx context.Context, pool *model.MonitorScrapePool, generated map[string]string, targetHash string, version uint64, fail func(string) error) error {
	publisher, err := r.projectionContext()
	if err != nil {
		return err
	}
	return publisher.publish(ctx, pool, generated, targetHash, version, fail)
}

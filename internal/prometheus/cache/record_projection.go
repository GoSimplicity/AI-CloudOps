package cache

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

type RecordProjectionRunner interface {
	Project(context.Context, configdao.ProjectionLease) error
}

func NewRecordProjectionRunner(store *ProjectionStore) RecordProjectionRunner {
	return &scrapeRuleProjectionRunner{store: store, scope: scrapeRecordScope}
}
func (r *recordRuleConfigCache) projectionContext() (*scrapeRuleProjection, error) {
	journal, ok := r.journal.(configdao.RecordProjectionJournalDAO)
	if !ok || journal == nil || r.history == nil || r.projection == nil {
		return nil, errors.New("预聚合持久发布依赖不可用")
	}
	result := &scrapeRuleProjection{scope: scrapeRecordScope, redis: r.redis, configDAO: r.recordConfigDAO(), journal: recordJournalFacade{journal}, history: recordHistoryFacade{r.history}, projection: r.projection}
	return result, result.ready()
}
func (r *recordRuleConfigCache) recordPublicationReady() error {
	_, err := r.projectionContext()
	return err
}
func (r *recordRuleConfigCache) recordSourceVersions(ctx context.Context) (map[int]uint64, error) {
	publisher, err := r.projectionContext()
	if err != nil {
		return nil, err
	}
	return publisher.sourceVersions(ctx)
}
func (r *recordRuleConfigCache) publishRecordProjection(ctx context.Context, pool *model.MonitorScrapePool, generated map[string]string, targetHash string, version uint64, fail func(string) error) error {
	publisher, err := r.projectionContext()
	if err != nil {
		return err
	}
	return publisher.publish(ctx, pool, generated, targetHash, version, fail)
}

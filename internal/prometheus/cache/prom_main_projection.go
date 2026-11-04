package cache

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

type PromMainProjectionRunner interface {
	Project(context.Context, configdao.ProjectionLease) error
}

func NewPromMainProjectionRunner(store *ProjectionStore) PromMainProjectionRunner {
	return &scrapeProjectionRunner{store: store, scope: scrapePromScope}
}
func (p *prometheusConfigCache) projectionContext() (*scrapeProjection, error) {
	journal, ok := p.journal.(configdao.PromMainProjectionJournalDAO)
	if !ok || journal == nil || p.history == nil || p.projection == nil {
		return nil, errors.New("Prometheus持久发布依赖不可用")
	}
	result := &scrapeProjection{scope: scrapePromScope, redis: p.redis, configDAO: p.promConfigDAO(), journal: promJournalFacade{journal}, history: promHistoryFacade{p.history}, projection: p.projection}
	return result, result.ready()
}
func (p *prometheusConfigCache) promPublicationReady() error {
	_, err := p.projectionContext()
	return err
}
func (p *prometheusConfigCache) promSourceVersions(ctx context.Context) (map[int]uint64, error) {
	publisher, err := p.projectionContext()
	if err != nil {
		return nil, err
	}
	return publisher.sourceVersions(ctx)
}
func (p *prometheusConfigCache) publishPromProjection(ctx context.Context, pool *model.MonitorScrapePool, generated map[string]string, targetHash string, version uint64, fail func(string) error) error {
	publisher, err := p.projectionContext()
	if err != nil {
		return err
	}
	return publisher.publish(ctx, pool, generated, targetHash, version, fail)
}

package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/utils"
	pc "github.com/prometheus/prometheus/config"
	"go.uber.org/zap"
)

func (p *prometheusConfigCache) promPublicationError(poolID int, stage string) error {
	p.logger.Error(LogModuleMonitor+"Prometheus主配置发布未完成", zap.Int("pool_id", poolID), zap.String("stage", stage))
	return fmt.Errorf("Prometheus主配置发布未完成，池%d，阶段%s", poolID, stage)
}

func detachedPromInput(pool *model.MonitorScrapePool, rows []*model.MonitorScrapeJob) (*model.MonitorScrapePool, []*model.MonitorScrapeJob) {
	copiedPool := *pool
	copiedPool.PrometheusInstances = append(model.StringList(nil), pool.PrometheusInstances...)
	copiedPool.Tags = append(model.StringList(nil), pool.Tags...)
	copiedRows := make([]*model.MonitorScrapeJob, len(rows))
	for index, row := range rows {
		if row == nil {
			continue
		}
		copy := *row
		copy.TreeNodeIDs = append(model.StringList(nil), row.TreeNodeIDs...)
		copy.Tags = append(model.StringList(nil), row.Tags...)
		copiedRows[index] = &copy
	}
	return &copiedPool, copiedRows
}

func (p *prometheusConfigCache) promConfigDAO() configDao.MonitorConfigDAO {
	if p.configDAO != nil {
		return p.configDAO
	}
	if p.batchManager != nil {
		return p.batchManager.configDAO
	}
	return nil
}

type promPublicationPlan = scrapePublicationPlan

func (p *prometheusConfigCache) planPromPublication(ctx context.Context, poolID int, desired map[string]string, oldIPs []string) (promPublicationPlan, error) {
	return planScrapePublication(ctx, scrapePromScope, p.promConfigDAO(), poolID, desired, oldIPs)
}

func (p *prometheusConfigCache) readPromPoolSnapshot(ctx context.Context) ([]*model.MonitorScrapePool, error) {
	return readScrapePoolSnapshot(ctx, p.scrapePoolDAO)
}

func (p *prometheusConfigCache) GenerateMainConfig(ctx context.Context) error {
	if err := p.promPublicationReady(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	versions, err := p.promSourceVersions(ctx)
	if err != nil {
		return err
	}
	pools, err := p.readPromPoolSnapshot(ctx)
	if err != nil {
		return errors.Join(p.promPublicationError(0, "读取采集池"), ctx.Err())
	}
	var failures []error
	for _, pool := range pools {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		if err := p.publishPromPool(ctx, pool, versions[pool.ID]); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(append(failures, ctx.Err())...)
}

func (p *prometheusConfigCache) publishPromPool(ctx context.Context, sourcePool *model.MonitorScrapePool, version uint64) error {
	if err := p.promPublicationReady(); err != nil {
		return err
	}
	pool, _ := detachedPromInput(sourcePool, nil)
	fail := func(stage string) error { return p.promPublicationError(pool.ID, stage) }
	if pool.DeletedAt > 0 {
		pool.PrometheusInstances = nil
	}
	if len(pool.PrometheusInstances) > 0 {
		if err := validateInstanceIPs(pool.PrometheusInstances); err != nil {
			return fail("实例地址")
		}
	}
	desiredIPs := map[string]bool{}
	for _, ip := range pool.PrometheusInstances {
		if desiredIPs[ip] {
			return fail("实例地址重复")
		}
		desiredIPs[ip] = true
	}

	generated := map[string]string{}
	targetHash := calculateConfigHash(fmt.Sprintf("deleted:%d:%d", pool.ID, pool.DeletedAt))
	if pool.DeletedAt == 0 {
		if p.scrapeJobDAO == nil {
			return fail("采集任务存储不可用")
		}
		jobs, err := p.scrapeJobDAO.GetMonitorScrapeJobsByPoolId(ctx, pool.ID)
		if err != nil {
			return fail("读取采集任务")
		}
		pool, jobs = detachedPromInput(pool, jobs)
		base, err := p.CreateBaseConfig(pool)
		if err != nil {
			return fail("生成基础配置")
		}
		scrapes, err := p.renderScrapeJobSnapshot(jobs)
		if err != nil {
			return fail("生成采集配置")
		}
		base.ScrapeConfigs = scrapes
		// Validate even when there are no targets. An invalid source must never be
		// mistaken for an authoritative request to retire its last-good artifacts.
		raw, err := marshalPrometheusConfig(base)
		if err != nil {
			return fail("序列化基础配置")
		}
		if _, err := pc.Load(string(raw), slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
			return fail("校验基础配置")
		}
		generated = make(map[string]string, len(pool.PrometheusInstances))
		for index, ip := range pool.PrometheusInstances {
			config := base
			if len(pool.PrometheusInstances) > 1 {
				config.ScrapeConfigs = p.ApplyHashMod(scrapes, len(pool.PrometheusInstances), index)
			}
			content, err := marshalPrometheusConfig(config)
			if err != nil {
				return fail("序列化配置")
			}
			if _, err := pc.Load(string(content), slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
				return fail("校验可执行配置")
			}
			generated[ip] = string(content)
		}
		serialized, err := json.Marshal(generated)
		if err != nil {
			return fail("生成配置摘要")
		}
		// The bytes include operational credentials, discovery and per-instance
		// sharding. Public/redacted job JSON is not an adequate version fingerprint.
		targetHash = fmt.Sprintf("%x", sha256.Sum256(append([]byte(utils.CalculatePromHash(pool)), serialized...)))
	}
	return p.publishPromProjection(ctx, pool, generated, targetHash, version, fail)
}

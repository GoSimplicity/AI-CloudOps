/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertRuleDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapePoolDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/utils"
	pm "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type AlertRuleConfigCache interface {
	GetConfigByIP(ip string) string
	GenerateMainConfig(ctx context.Context) error
	RetireDeletedScrapePool(ctx context.Context, poolID int) error
	ReconcileDeletedScrapePools(ctx context.Context) error
	GenerateConfigForPool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string
}

type alertRuleConfigCache struct {
	// Redis 替代本地缓存
	redis         redis.Cmdable
	mu            sync.RWMutex
	logger        *zap.Logger
	scrapePoolDAO scrapePoolDao.ScrapePoolDAO
	alertRuleDAO  alertRuleDao.AlertManagerRuleDAO
	configDAO     configDao.MonitorConfigDAO
	batchManager  *BatchConfigManager
	journal       configDao.ProjectionPlanJournalDAO
	history       configDao.AlertRuleHistoryReader
	projection    AlertRuleProjectionRunner
}

type RuleGroup struct {
	Name  string         `yaml:"name"`
	Rules []rulefmt.Rule `yaml:"rules"`
}

type RuleGroups struct {
	Groups []RuleGroup `yaml:"groups"`
}

func NewAlertRuleConfigCache(
	logger *zap.Logger,
	scrapePoolDAO scrapePoolDao.ScrapePoolDAO,
	alertRuleDAO alertRuleDao.AlertManagerRuleDAO,
	configDAO configDao.MonitorConfigDAO,
	batchManager *BatchConfigManager,
	redisClient redis.Cmdable,
	journal configDao.ProjectionPlanJournalDAO,
	history configDao.AlertRuleHistoryReader,
	projection AlertRuleProjectionRunner,
) AlertRuleConfigCache {
	return &alertRuleConfigCache{
		logger:        logger,
		redis:         redisClient,
		mu:            sync.RWMutex{},
		scrapePoolDAO: scrapePoolDAO,
		alertRuleDAO:  alertRuleDAO,
		configDAO:     configDAO,
		batchManager:  batchManager,
		journal:       journal, history: history, projection: projection,
	}
}

// GetConfigByIP 根据IP地址获取告警规则配置
func (r *alertRuleConfigCache) GetConfigByIP(ip string) string {
	if ip == "" {
		r.logger.Warn(LogModuleMonitor + "获取配置时IP为空")
		return ""
	}
	ctx := context.Background()
	val, err := r.redis.Get(ctx, buildRedisKeyAlertRule(ip)).Result()
	if err != nil {
		r.logger.Debug(LogModuleMonitor+"缓存未命中", zap.String("ip", ip), zap.Error(err))
		return ""
	}
	r.logger.Debug(LogModuleMonitor+"缓存命中", zap.String("ip", ip))
	return val
}

// Publication identity is disjoint from legacy hashes whose suffix is a mutable name.
func alertRulePublicationKey(poolID int) string {
	return fmt.Sprintf("%s:publication:alert_rule:pool:%d", redisPrefix, poolID)
}

func (r *alertRuleConfigCache) alertRulePublicationError(poolID int, stage string) error {
	r.logger.Error(LogModuleMonitor+"告警规则发布未完成", zap.Int("pool_id", poolID), zap.String("stage", stage))
	return fmt.Errorf("告警规则发布未完成，池%d，阶段%s", poolID, stage)
}

func detachedAlertRules(pool *model.MonitorScrapePool, rows []*model.MonitorAlertRule) (*model.MonitorScrapePool, []*model.MonitorAlertRule) {
	copiedPool := *pool
	copiedPool.PrometheusInstances = append(model.StringList(nil), pool.PrometheusInstances...)
	copiedRows := make([]*model.MonitorAlertRule, len(rows))
	for index, row := range rows {
		if row == nil {
			continue
		}
		copy := *row
		copy.Labels = append(model.StringList(nil), row.Labels...)
		copy.Annotations = append(model.StringList(nil), row.Annotations...)
		copiedRows[index] = &copy
	}
	return &copiedPool, copiedRows
}

func alertRuleSnapshotHash(pool *model.MonitorScrapePool, rows []*model.MonitorAlertRule) (string, error) {
	type ruleVersion struct {
		ID                  int
		Name, Expr, ForTime string
		Labels, Annotations model.StringList
	}
	versions := make([]*ruleVersion, len(rows))
	for index, row := range rows {
		if row != nil {
			versions[index] = &ruleVersion{ID: row.ID, Name: row.Name, Expr: row.Expr, ForTime: row.ForTime, Labels: row.Labels, Annotations: row.Annotations}
		}
	}
	// Preserve ordering: target and rule order are inputs to the existing sharding.
	data, err := json.Marshal(struct {
		PoolID       int
		SupportAlert int8
		PoolName     string
		Instances    model.StringList
		Rules        []*ruleVersion
	}{PoolID: pool.ID, SupportAlert: pool.SupportAlert, PoolName: pool.Name, Instances: pool.PrometheusInstances, Rules: versions})
	if err != nil {
		return "", err
	}
	return calculateConfigHash(string(data)), nil
}

func (r *alertRuleConfigCache) ruleConfigDAO() configDao.MonitorConfigDAO {
	if r.configDAO != nil {
		return r.configDAO
	}
	if r.batchManager != nil {
		return r.batchManager.configDAO
	}
	return nil
}

type alertRulePublicationPlan = scrapeRulePublicationPlan

func (r *alertRuleConfigCache) planAlertRulePublication(ctx context.Context, poolID int, desired map[string]string, oldIPs []string) (alertRulePublicationPlan, error) {
	return planScrapeRulePublication(ctx, scrapeAlertScope, r.ruleConfigDAO(), poolID, desired, oldIPs)
}

func (r *alertRuleConfigCache) GenerateMainConfig(ctx context.Context) error {
	if err := r.alertRulePublicationReady(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Serialize this cache instance only; this is not a distributed transaction.
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	versions, err := r.alertRuleSourceVersions(ctx)
	if err != nil {
		return err
	}
	pools, err := readScrapePoolSnapshot(ctx, r.scrapePoolDAO)
	if err != nil {
		return errors.Join(r.alertRulePublicationError(0, "读取采集池"), ctx.Err())
	}
	var failures []error
	for _, pool := range pools {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		if pool == nil || pool.ID <= 0 {
			failures = append(failures, r.alertRulePublicationError(0, "采集池标识"))
			continue
		}
		if err := r.publishAlertRulePool(ctx, pool, versions[pool.ID]); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(append(failures, ctx.Err())...)
}

func (r *alertRuleConfigCache) publishAlertRulePool(ctx context.Context, sourcePool *model.MonitorScrapePool, version uint64) error {
	if err := r.alertRulePublicationReady(); err != nil {
		return err
	}
	pool, _ := detachedAlertRules(sourcePool, nil)
	fail := func(stage string) error { return r.alertRulePublicationError(pool.ID, stage) }
	if pool.DeletedAt > 0 {
		pool.SupportAlert = 2
	}
	if pool.SupportAlert != 1 && pool.SupportAlert != 2 {
		return fail("功能状态无效")
	}
	// Explicit off is authoritative. Its potentially stale target fields and rule
	// contents cannot authorize new keys or block retiring previously owned keys.
	if pool.SupportAlert == 2 {
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
	var rows []*model.MonitorAlertRule
	var err error
	generated := map[string]string{}
	if pool.SupportAlert == 1 {
		if r.alertRuleDAO == nil {
			return fail("规则存储不可用")
		}
		rows, _, err = r.alertRuleDAO.GetMonitorAlertRuleByPoolID(ctx, pool.ID)
		if err != nil {
			return fail("读取规则")
		}
		pool, rows = detachedAlertRules(pool, rows)
		generated = r.generateAlertRuleSnapshot(pool, rows)
		if generated == nil {
			return fail("构造规则")
		}
	}
	for _, ip := range pool.PrometheusInstances {
		if _, ok := generated[ip]; !ok {
			return fail("规则文件不完整")
		}
	}
	targetHash, err := alertRuleSnapshotHash(pool, rows)
	if pool.DeletedAt > 0 {
		targetHash = calculateConfigHash(fmt.Sprintf("deleted:%d:%d", pool.ID, pool.DeletedAt))
	}
	if err != nil {
		return fail("计算版本")
	}
	return r.publishAlertRuleProjection(ctx, pool, generated, targetHash, version, fail)
}

// GenerateConfigForPool 为单个采集池生成告警规则配置
func (r *alertRuleConfigCache) GenerateConfigForPool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string {
	if pool == nil || ctx.Err() != nil || r.alertRuleDAO == nil {
		return nil
	}
	rows, _, err := r.alertRuleDAO.GetMonitorAlertRuleByPoolID(ctx, pool.ID)
	if err != nil {
		r.alertRulePublicationError(pool.ID, "读取规则")
		return nil
	}
	snapshot, rows := detachedAlertRules(pool, rows)
	return r.generateAlertRuleSnapshot(snapshot, rows)
}

func (r *alertRuleConfigCache) generateAlertRuleSnapshot(pool *model.MonitorScrapePool, rules []*model.MonitorAlertRule) map[string]string {
	poolStartTime := time.Now()
	var ruleGroups RuleGroups

	seenIDs := make(map[int]struct{}, len(rules))
	for _, rule := range rules {
		if rule == nil || rule.ID <= 0 {
			r.logger.Warn(LogModuleMonitor + "跳过缺少有效ID的告警规则")
			continue
		}
		if _, duplicate := seenIDs[rule.ID]; duplicate {
			r.logger.Warn(LogModuleMonitor+"跳过重复ID告警规则", zap.Int("rule_id", rule.ID))
			continue
		}
		built, err := utils.BuildAlertingRule(rule.Name, rule.Expr, rule.ForTime, rule.Labels, rule.Annotations)
		if err != nil {
			r.logger.Warn(LogModuleMonitor+"跳过无效告警规则", zap.Int("rule_id", rule.ID))
			continue
		}
		seenIDs[rule.ID] = struct{}{}
		ruleGroups.Groups = append(ruleGroups.Groups, RuleGroup{
			Name:  fmt.Sprintf("alert-rule-%d", rule.ID),
			Rules: []rulefmt.Rule{built},
		})
	}
	// Invalid-only historical rows are not a successfully generated empty snapshot.
	// Empty publication and cache cleanup are handled separately from construction.
	if len(rules) > 0 && len(ruleGroups.Groups) == 0 {
		return nil
	}

	numInstances := len(pool.PrometheusInstances)
	if numInstances == 0 {
		// A successful source read with no targets is an authoritative empty desired
		// set, not a read/build failure. Publication still retires only owned artifacts.
		return map[string]string{}
	}

	ruleMap := make(map[string]string)
	success := true

	for i, ip := range pool.PrometheusInstances {
		var myRuleGroups RuleGroups

		for j, group := range ruleGroups.Groups {
			if numInstances > 0 && j%numInstances == i {
				myRuleGroups.Groups = append(myRuleGroups.Groups, group)
			}
		}

		// 检查分配到该IP的规则组是否为空，如果为空则跳过
		if len(myRuleGroups.Groups) == 0 {
			myRuleGroups.Groups = []RuleGroup{}
		}

		yamlData, err := yaml.Marshal(&myRuleGroups)
		if err != nil {
			r.logger.Error(LogModuleMonitor+"序列化告警规则YAML失败",
				zap.Error(err),
				zap.String("pool_name", pool.Name),
				zap.String("instance_ip", ip))
			success = false
			break
		}

		if _, validationErrors := rulefmt.Parse(yamlData, false, pm.UTF8Validation, parser.NewParser(parser.Options{})); len(validationErrors) > 0 {
			r.logger.Error(LogModuleMonitor+"生成的告警规则YAML校验失败", zap.Int("pool_id", pool.ID), zap.String("instance_ip", ip))
			success = false
			break
		}

		ruleMap[ip] = string(yamlData)
	}

	if !success {
		logCacheOperation(r.logger, "生成告警规则配置", pool.Name, poolStartTime, fmt.Errorf("序列化失败"))
		return nil
	}

	logCacheOperation(r.logger, "生成告警规则配置", pool.Name, poolStartTime, nil)
	return ruleMap
}

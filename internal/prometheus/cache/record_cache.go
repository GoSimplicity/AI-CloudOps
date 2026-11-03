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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertRecordDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapePoolDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/utils"
	prommodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type RecordRuleConfigCache interface {
	GetConfigByIP(ip string) string
	GenerateMainConfig(ctx context.Context) error
	GenerateConfigForPool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string
}

type recordRuleConfigCache struct {
	// Redis 替代本地缓存
	redis          redis.Cmdable
	mu             sync.RWMutex
	logger         *zap.Logger
	scrapePoolDAO  scrapePoolDao.ScrapePoolDAO
	alertRecordDAO alertRecordDao.AlertManagerRecordDAO
	configDAO      configDao.MonitorConfigDAO
	batchManager   *BatchConfigManager
}

type RecordGroup struct {
	Name  string         `yaml:"name"`
	Rules []rulefmt.Rule `yaml:"rules"`
}

type RecordGroups struct {
	Groups []RecordGroup `yaml:"groups"`
}

func NewRecordRuleConfigCache(
	logger *zap.Logger,
	scrapePoolDAO scrapePoolDao.ScrapePoolDAO,
	alertRecordDAO alertRecordDao.AlertManagerRecordDAO,
	configDAO configDao.MonitorConfigDAO,
	batchManager *BatchConfigManager,
	redisClient redis.Cmdable,
) RecordRuleConfigCache {
	return &recordRuleConfigCache{
		logger:         logger,
		mu:             sync.RWMutex{},
		redis:          redisClient,
		scrapePoolDAO:  scrapePoolDAO,
		alertRecordDAO: alertRecordDAO,
		configDAO:      configDAO,
		batchManager:   batchManager,
	}
}

// GetConfigByIP 根据IP地址获取Prometheus的预聚合规则配置YAML
func (r *recordRuleConfigCache) GetConfigByIP(ip string) string {
	if ip == "" {
		r.logger.Warn(LogModuleMonitor + "获取配置时IP为空")
		return ""
	}
	ctx := context.Background()
	val, err := r.redis.Get(ctx, buildRedisKeyRecordRule(ip)).Result()
	if err != nil {
		r.logger.Debug(LogModuleMonitor+"缓存未命中", zap.String("ip", ip), zap.Error(err))
		return ""
	}
	r.logger.Debug(LogModuleMonitor+"缓存命中", zap.String("ip", ip))
	return val
}

// Keep publication identity separate from mutable names and legacy name-keyed hashes.
func recordPublicationKey(poolID int) string {
	return fmt.Sprintf("%s:publication:record_rule:pool:%d", redisPrefix, poolID)
}

// GenerateMainConfig 生成并更新所有Prometheus的预聚合规则配置YAML，并同步入库
func (r *recordRuleConfigCache) GenerateMainConfig(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Serializes this instance only; consumers can still observe partial artifacts.
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	startTime := time.Now()
	pools, _, err := r.scrapePoolDAO.GetMonitorScrapePoolSupportedRecord(ctx)
	if err != nil {
		return err
	}
	type pendingCommit struct {
		poolID   int
		hashKey  string
		hash     string
		staleIPs []string
	}
	var pending []pendingCommit
	var failures []error
	reportFailure := func(poolID int, stage string) {
		// Do not include YAML, expressions, labels or backend error contents.
		r.logger.Error(LogModuleMonitor+"预聚合配置发布未完成", zap.Int("pool_id", poolID), zap.String("stage", stage))
		failures = append(failures, fmt.Errorf("预聚合配置发布未完成，池%d，阶段%s", poolID, stage))
	}
	allConfigsToSave := make(map[string]ConfigData)
	processedCount := 0
	for _, pool := range pools {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		if pool == nil || pool.ID <= 0 {
			reportFailure(0, "采集池缺失")
			continue
		}
		if len(pool.PrometheusInstances) == 0 {
			continue
		}
		if err := validateInstanceIPs(pool.PrometheusInstances); err != nil {
			reportFailure(pool.ID, "实例地址无效")
			continue
		}
		rules, _, err := r.alertRecordDAO.GetMonitorRecordRuleByPoolId(ctx, pool.ID)
		if err != nil {
			reportFailure(pool.ID, "读取规则")
			continue
		}
		currentHash := calculateRecordPoolHash(pool, rules)
		hashKey := recordPublicationKey(pool.ID)
		cachedHash, err := r.redis.Get(ctx, hashKey).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			reportFailure(pool.ID, "读取缓存版本")
			continue
		}
		if err == nil && cachedHash == currentHash {
			continue
		}
		oneMap := r.generateConfigForRules(pool, rules)
		if oneMap == nil {
			reportFailure(pool.ID, "生成规则")
			continue
		}
		// Desired membership is independent of whether this attempt can write.
		// A failed write for a current IP must never turn it into a stale IP.
		desired := make(map[string]struct{}, len(pool.PrometheusInstances))
		complete := true
		for _, ip := range pool.PrometheusInstances {
			desired[ip] = struct{}{}
			if _, exists := oneMap[ip]; !exists {
				complete = false
			}
		}
		if !complete {
			reportFailure(pool.ID, "生成结果不完整")
			continue
		}
		setKey := buildRedisSetKeyRecordRulePoolIPs(pool.ID)
		oldIPs, err := r.redis.SMembers(ctx, setKey).Result()
		if err != nil {
			reportFailure(pool.ID, "读取实例成员")
			continue
		}
		commit := pendingCommit{poolID: pool.ID, hashKey: hashKey, hash: currentHash}
		for _, oldIP := range oldIPs {
			if _, wanted := desired[oldIP]; !wanted {
				commit.staleIPs = append(commit.staleIPs, oldIP)
			}
		}
		// Invalidate the previous ACK before the first visible mutation. Retaining
		// it after a failed B write would incorrectly skip a later source reversion to A.
		if err := r.redis.Set(ctx, hashKey, "pending:"+currentHash, 0).Err(); err != nil {
			reportFailure(pool.ID, "标记发布中")
			continue
		}
		for ip, yamlContent := range oneMap {
			if err := r.redis.Set(ctx, buildRedisKeyRecordRule(ip), yamlContent, 0).Err(); err != nil {
				reportFailure(pool.ID, "写入规则缓存")
				complete = false
				continue
			}
			if err := r.redis.SAdd(ctx, setKey, ip).Err(); err != nil {
				reportFailure(pool.ID, "写入实例成员")
				complete = false
				continue
			}
			allConfigsToSave[ip] = ConfigData{
				Name: fmt.Sprintf(ConfigNameRecordRule, pool.ID, ip), PoolID: pool.ID,
				ConfigType: model.ConfigTypeRecordRule, Content: yamlContent,
			}
		}
		if complete {
			pending = append(pending, commit)
		}
		processedCount++
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(append(failures, err)...)
	}
	if len(allConfigsToSave) > 0 {
		if err := batchSaveConfigsToDatabase(ctx, r.batchManager, allConfigsToSave); err != nil {
			reportFailure(0, "持久化规则配置")
			// Cache contents may already be visible, but no pool is acknowledged.
			return errors.Join(failures...)
		}
	}
	for _, commit := range pending {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		complete := true
		setKey := buildRedisSetKeyRecordRulePoolIPs(commit.poolID)
		for _, staleIP := range commit.staleIPs {
			if err := r.redis.Del(ctx, buildRedisKeyRecordRule(staleIP)).Err(); err != nil {
				reportFailure(commit.poolID, "清理旧实例缓存")
				complete = false
				continue
			}
			// Keep the index until deletion succeeds so a failed deletion is retryable.
			if err := r.redis.SRem(ctx, setKey, staleIP).Err(); err != nil {
				reportFailure(commit.poolID, "清理旧实例成员")
				complete = false
			}
		}
		if complete {
			if err := r.redis.Set(ctx, commit.hashKey, commit.hash, 0).Err(); err != nil {
				reportFailure(commit.poolID, "确认缓存版本")
			}
		}
	}
	logBatchOperation(r.logger, "生成预聚合规则配置", processedCount, len(pools), startTime)
	return errors.Join(failures...)
}

// GenerateConfigForPool 根据单个采集池生成Prometheus的预聚合规则配置YAML
func (r *recordRuleConfigCache) GenerateConfigForPool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string {
	poolStartTime := time.Now()

	rules, _, err := r.alertRecordDAO.GetMonitorRecordRuleByPoolId(ctx, pool.ID)
	if err != nil {
		logCacheOperation(r.logger, "获取预聚合规则", pool.Name, poolStartTime, err)
		return nil
	}

	return r.generateConfigForRules(pool, rules)
}

func (r *recordRuleConfigCache) generateConfigForRules(pool *model.MonitorScrapePool, rules []*model.MonitorRecordRule) map[string]string {
	poolStartTime := time.Now()
	numInstances := len(pool.PrometheusInstances)
	if numInstances == 0 {
		r.logger.Warn(LogModuleMonitor+"采集池中没有Prometheus实例", zap.String("pool_name", pool.Name))
		return nil
	}

	ruleMap := make(map[string]string)

	// 优化：如果没有规则，返回空groups
	if len(rules) == 0 {
		emptyGroups := RecordGroups{Groups: []RecordGroup{}}
		yamlData, err := yaml.Marshal(&emptyGroups)
		if err != nil {
			r.logger.Error(LogModuleMonitor+"序列化空预聚合规则YAML失败", zap.Error(err))
			return nil
		}
		for _, ip := range pool.PrometheusInstances {
			ruleMap[ip] = string(yamlData)
		}
		logCacheOperation(r.logger, "生成空预聚合规则配置", pool.Name, poolStartTime, nil)
		return ruleMap
	}

	groups := make([]RecordGroup, 0, len(rules))
	for _, rule := range rules {
		if rule == nil {
			r.logger.Warn(LogModuleMonitor + "预聚合规则为空，已跳过")
			continue
		}
		built, err := utils.BuildRecordingRule(rule.Name, rule.Expr, rule.Labels)
		if err != nil {
			r.logger.Warn(LogModuleMonitor+"预聚合规则无效，已跳过", zap.Int("rule_id", rule.ID))
			continue
		}
		groups = append(groups, RecordGroup{Name: rule.Name, Rules: []rulefmt.Rule{built}})
	}
	if len(groups) == 0 {
		// A non-empty but invalid snapshot is not an explicit request to clear rules.
		return nil
	}
	for _, ip := range pool.PrometheusInstances {
		myRecordGroups := RecordGroups{Groups: groups}

		yamlData, err := yaml.Marshal(&myRecordGroups)
		if err != nil {
			r.logger.Error(LogModuleMonitor+"序列化预聚合规则YAML失败",
				zap.Error(err),
				zap.String("pool_name", pool.Name),
				zap.String("instance_ip", ip))
			continue
		}

		// Validate the actual consumer contract, not only YAML syntax.
		if _, errs := rulefmt.Parse(yamlData, false, prommodel.UTF8Validation, parser.NewParser(parser.Options{})); len(errs) > 0 {
			r.logger.Error(LogModuleMonitor+"生成的预聚合规则配置无效，保留现有配置",
				zap.Int("error_count", len(errs)),
				zap.String("pool_name", pool.Name),
				zap.String("instance_ip", ip))
			return nil
		}

		ruleMap[ip] = string(yamlData)
	}

	logCacheOperation(r.logger, "生成预聚合规则配置", pool.Name, poolStartTime, nil)
	return ruleMap
}

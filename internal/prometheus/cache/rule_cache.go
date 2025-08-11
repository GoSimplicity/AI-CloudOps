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
	"fmt"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertRuleDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapePoolDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	pm "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type AlertRuleConfigCache interface {
	GetConfigByIP(ip string) string
	GenerateMainConfig(ctx context.Context) error
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
) AlertRuleConfigCache {
	return &alertRuleConfigCache{
		logger:        logger,
		redis:         redisClient,
		mu:            sync.RWMutex{},
		scrapePoolDAO: scrapePoolDAO,
		alertRuleDAO:  alertRuleDAO,
		configDAO:     configDAO,
		batchManager:  batchManager,
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

// GenerateMainConfig 生成告警规则配置并入库
func (r *alertRuleConfigCache) GenerateMainConfig(ctx context.Context) error {
	startTime := time.Now()
	r.logger.Info(LogModuleMonitor + "开始生成告警规则配置")

	// 获取支持告警配置的所有采集池
	pools, _, err := r.scrapePoolDAO.GetMonitorScrapePoolSupportedAlert(ctx)
	if err != nil {
		r.logger.Error(LogModuleMonitor+"获取支持告警的采集池失败", zap.Error(err))
		return err
	}

	if len(pools) == 0 {
		r.logger.Info(LogModuleMonitor + "没有找到支持告警的采集池")
		return nil
	}

	validIPs := make(map[string]struct{})
	processedCount := 0
	allConfigsToSave := make(map[string]ConfigData)

	for _, pool := range pools {
		if err := validateInstanceIPs(pool.PrometheusInstances); err != nil {
			r.logger.Error(LogModuleMonitor+"Prometheus实例IP验证失败",
				zap.String("pool_name", pool.Name),
				zap.Error(err))
			continue
		}

		currentHash := utils.CalculatePromHash(pool)
		hashKey := buildRedisHashKeyAlertRule(pool.Name)
		cachedHash, _ := r.redis.Get(ctx, hashKey).Result()
		if cachedHash == currentHash {
			for _, ip := range pool.PrometheusInstances {
				validIPs[ip] = struct{}{}
			}
			continue
		}

		oneMap := r.GenerateConfigForPool(ctx, pool)
		if oneMap != nil {
			// Redis 旧集合
			setKey := buildRedisSetKeyAlertRulePoolIPs(pool.ID)
			oldIPs, _ := r.redis.SMembers(ctx, setKey).Result()
			oldIPSet := map[string]struct{}{}
			for _, old := range oldIPs {
				oldIPSet[old] = struct{}{}
			}

			for ip, yamlContent := range oneMap {
				configName := fmt.Sprintf(ConfigNameAlertRule, pool.ID, ip)
				validIPs[ip] = struct{}{}

				allConfigsToSave[ip] = ConfigData{
					Name:       configName,
					PoolID:     pool.ID,
					ConfigType: model.ConfigTypeAlertRule,
					Content:    yamlContent,
				}
				// 写 Redis
				if err := r.redis.Set(ctx, buildRedisKeyAlertRule(ip), yamlContent, 0).Err(); err != nil {
					r.logger.Error(LogModuleMonitor+"写入Redis失败", zap.String("pool_name", pool.Name), zap.String("ip", ip), zap.Error(err))
					continue
				}
				_ = r.redis.SAdd(ctx, setKey, ip).Err()
				delete(oldIPSet, ip)
			}

			for staleIP := range oldIPSet {
				_ = r.redis.Del(ctx, buildRedisKeyAlertRule(staleIP)).Err()
				_ = r.redis.SRem(ctx, setKey, staleIP).Err()
				r.logger.Debug(LogModuleMonitor+"删除无效IP配置", zap.String("ip", staleIP))
			}
			_ = r.redis.Set(ctx, hashKey, currentHash, 0).Err()
		}
		processedCount++
	}

	// 批量保存所有配置到数据库
	if len(allConfigsToSave) > 0 {
		if err := batchSaveConfigsToDatabase(ctx, r.batchManager, allConfigsToSave); err != nil {
			r.logger.Error(LogModuleMonitor+"批量保存告警规则配置失败", zap.Error(err))
			// 不返回错误，继续执行后续逻辑
		}
	}

	// 不再维护本地缓存

	logBatchOperation(r.logger, "生成告警规则配置", processedCount, len(pools), startTime)
	return nil
}

// GenerateConfigForPool 为单个采集池生成告警规则配置
func (r *alertRuleConfigCache) GenerateConfigForPool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string {
	poolStartTime := time.Now()

	rules, _, err := r.alertRuleDAO.GetMonitorAlertRuleByPoolID(ctx, pool.ID)
	if err != nil {
		logCacheOperation(r.logger, "获取告警规则", pool.Name, poolStartTime, err)
		return nil
	}

	if len(rules) == 0 {
		r.logger.Info(LogModuleMonitor+"没有找到告警规则", zap.String("pool_name", pool.Name))
		return nil
	}

	var ruleGroups RuleGroups

	for _, rule := range rules {
		ft, err := pm.ParseDuration(rule.ForTime)
		if err != nil {
			r.logger.Warn(LogModuleMonitor+"解析告警规则持续时间失败，使用默认值",
				zap.Error(err),
				zap.String("rule_name", rule.Name))
			ft, _ = pm.ParseDuration("5s")
		}

		oneRule := rulefmt.Rule{
			Alert:       rule.Name,
			Expr:        rule.Expr,
			For:         ft,
			Labels:      utils.FromSliceTuMap(rule.Labels),
			Annotations: utils.FromSliceTuMap(rule.Annotations),
		}

		ruleGroup := RuleGroup{
			Name:  rule.Name,
			Rules: []rulefmt.Rule{oneRule},
		}
		ruleGroups.Groups = append(ruleGroups.Groups, ruleGroup)
	}

	numInstances := len(pool.PrometheusInstances)
	if numInstances == 0 {
		r.logger.Warn(LogModuleMonitor+"采集池中没有Prometheus实例", zap.String("pool_name", pool.Name))
		return nil
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
			continue
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

		ruleMap[ip] = string(yamlData)
	}

	if !success {
		logCacheOperation(r.logger, "生成告警规则配置", pool.Name, poolStartTime, fmt.Errorf("序列化失败"))
		return nil
	}

	logCacheOperation(r.logger, "生成告警规则配置", pool.Name, poolStartTime, nil)
	return ruleMap
}

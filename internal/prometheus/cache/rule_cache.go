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
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type AlertRuleConfigCache interface {
	GetConfigByIP(ip string) string
	GenerateMainConfig(ctx context.Context) error
	GenerateConfigForPool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string
}

type alertRuleConfigCache struct {
	configMap     map[string]string
	mu            sync.RWMutex
	logger        *zap.Logger
	scrapePoolDAO scrapePoolDao.ScrapePoolDAO
	alertRuleDAO  alertRuleDao.AlertManagerRuleDAO
	ruleHashes    map[string]string
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
) AlertRuleConfigCache {
	return &alertRuleConfigCache{
		logger:        logger,
		configMap:     make(map[string]string),
		mu:            sync.RWMutex{},
		scrapePoolDAO: scrapePoolDAO,
		alertRuleDAO:  alertRuleDAO,
		ruleHashes:    make(map[string]string),
		configDAO:     configDAO,
		batchManager:  batchManager,
	}
}

// GetConfigByIP 根据IP地址获取告警规则配置
func (r *alertRuleConfigCache) GetConfigByIP(ip string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.configMap[ip]
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

	// 直接加写锁，所有操作同步进行
	r.mu.Lock()
	defer r.mu.Unlock()

	tempConfigMap := utils.CopyMap(r.configMap)
	tempPoolHashes := utils.CopyMap(r.ruleHashes)

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
		if cachedHash, ok := tempPoolHashes[pool.Name]; ok && cachedHash == currentHash {
			for _, ip := range pool.PrometheusInstances {
				validIPs[ip] = struct{}{}
			}
			continue
		}

		oneMap := r.GenerateConfigForPool(ctx, pool)
		if oneMap != nil {
			for ip, yamlContent := range oneMap {
				configName := fmt.Sprintf(ConfigNameAlertRule, pool.ID, ip)
				tempConfigMap[ip] = yamlContent
				validIPs[ip] = struct{}{}

				// 准备批量保存的配置数据
				allConfigsToSave[ip] = ConfigData{
					Name:       configName,
					PoolID:     pool.ID,
					ConfigType: model.ConfigTypeAlertRule,
					Content:    yamlContent,
				}
			}
			tempPoolHashes[pool.Name] = currentHash
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

	// 清理无效的IP配置
	cleanupInvalidIPs(tempConfigMap, validIPs, r.logger)

	r.configMap = tempConfigMap
	r.ruleHashes = tempPoolHashes

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

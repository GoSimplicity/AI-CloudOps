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
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertRecordDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapePoolDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/prometheus/prometheus/model/rulefmt"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type RecordRuleConfigCache interface {
	GetConfigByIP(ip string) string
	GenerateMainConfig(ctx context.Context) error
	GenerateConfigForPool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string
}

type recordRuleConfigCache struct {
	mu             sync.RWMutex
	logger         *zap.Logger
	configMap      map[string]string
	scrapePoolDAO  scrapePoolDao.ScrapePoolDAO
	alertRecordDAO alertRecordDao.AlertManagerRecordDAO
	recordHashes   map[string]string
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
) RecordRuleConfigCache {
	return &recordRuleConfigCache{
		logger:         logger,
		mu:             sync.RWMutex{},
		configMap:      make(map[string]string),
		scrapePoolDAO:  scrapePoolDAO,
		alertRecordDAO: alertRecordDAO,
		recordHashes:   make(map[string]string),
		configDAO:      configDAO,
		batchManager:   batchManager,
	}
}

// GetConfigByIP 根据IP地址获取Prometheus的预聚合规则配置YAML
func (r *recordRuleConfigCache) GetConfigByIP(ip string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.configMap[ip]
}

// validatePromQLExpr 验证PromQL表达式的有效性
func validatePromQLExpr(expr string) error {
	if strings.TrimSpace(expr) == "" {
		return fmt.Errorf("表达式不能为空")
	}

	// 简单的PromQL语法检查
	if strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"") {
		return fmt.Errorf("表达式不应该是字符串字面量: %s", expr)
	}

	return nil
}

// GenerateMainConfig 生成并更新所有Prometheus的预聚合规则配置YAML，并同步入库
func (r *recordRuleConfigCache) GenerateMainConfig(ctx context.Context) error {
	startTime := time.Now()
	r.logger.Info(LogModuleMonitor + "开始生成预聚合规则配置")

	// 获取支持预聚合配置的所有采集池
	pools, _, err := r.scrapePoolDAO.GetMonitorScrapePoolSupportedRecord(ctx)
	if err != nil {
		r.logger.Error(LogModuleMonitor+"获取支持预聚合的采集池失败", zap.Error(err))
		return err
	}

	if len(pools) == 0 {
		r.logger.Info(LogModuleMonitor + "没有找到支持预聚合的采集池")
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	tempConfigMap := utils.CopyMap(r.configMap)
	tempPoolHashes := utils.CopyMap(r.recordHashes)

	validIPs := make(map[string]struct{})
	processedCount := 0
	allConfigsToSave := make(map[string]ConfigData)

	for _, pool := range pools {
		if len(pool.PrometheusInstances) == 0 {
			r.logger.Warn(LogModuleMonitor+"采集池中没有Prometheus实例", zap.String("pool_name", pool.Name))
			continue
		}
		if err := validateInstanceIPs(pool.PrometheusInstances); err != nil {
			r.logger.Error(LogModuleMonitor+"Prometheus实例IP验证失败",
				zap.String("pool_name", pool.Name),
				zap.Error(err))
			continue
		}

		// 优化哈希计算，包含规则内容
		rules, _, ruleErr := r.alertRecordDAO.GetMonitorRecordRuleByPoolId(ctx, pool.ID)
		ruleHash := ""
		if ruleErr == nil && len(rules) > 0 {
			var ruleParts []string
			for _, rule := range rules {
				ruleParts = append(ruleParts, rule.Name, rule.Expr)
			}
			ruleHash = strings.Join(ruleParts, "|")
		}
		currentHash := calculateConfigHash(pool.Name + ":" + strings.Join(pool.PrometheusInstances, ",") + ":" + ruleHash)
		if cachedHash, ok := tempPoolHashes[pool.Name]; ok && cachedHash == currentHash {
			for _, ip := range pool.PrometheusInstances {
				validIPs[ip] = struct{}{}
			}
			continue
		}

		oneMap := r.GenerateConfigForPool(ctx, pool)
		if oneMap != nil {
			for ip, yamlContent := range oneMap {
				configName := fmt.Sprintf(ConfigNameRecordRule, pool.ID, ip)
				tempConfigMap[ip] = yamlContent
				validIPs[ip] = struct{}{}

				allConfigsToSave[ip] = ConfigData{
					Name:       configName,
					PoolID:     pool.ID,
					ConfigType: model.ConfigTypeRecordRule,
					Content:    yamlContent,
				}
			}
			tempPoolHashes[pool.Name] = currentHash
		}
		processedCount++
	}

	if len(allConfigsToSave) > 0 {
		if err := batchSaveConfigsToDatabase(ctx, r.batchManager, allConfigsToSave); err != nil {
			r.logger.Error(LogModuleMonitor+"批量保存预聚合规则配置失败", zap.Error(err))
		}
	}

	cleanupInvalidIPs(tempConfigMap, validIPs, r.logger)

	r.configMap = tempConfigMap
	r.recordHashes = tempPoolHashes

	logBatchOperation(r.logger, "生成预聚合规则配置", processedCount, len(pools), startTime)
	return nil
}

// GenerateConfigForPool 根据单个采集池生成Prometheus的预聚合规则配置YAML
func (r *recordRuleConfigCache) GenerateConfigForPool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string {
	poolStartTime := time.Now()

	rules, _, err := r.alertRecordDAO.GetMonitorRecordRuleByPoolId(ctx, pool.ID)
	if err != nil {
		logCacheOperation(r.logger, "获取预聚合规则", pool.Name, poolStartTime, err)
		return nil
	}

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

	for _, ip := range pool.PrometheusInstances {
		var myRecordGroups RecordGroups

		for _, rule := range rules {
			// 检查表达式和名称
			if strings.TrimSpace(rule.Name) == "" || strings.TrimSpace(rule.Expr) == "" {
				r.logger.Warn(LogModuleMonitor+"预聚合规则缺少名称或表达式，已跳过",
					zap.String("pool_name", pool.Name),
					zap.String("rule_name", rule.Name),
					zap.String("instance_ip", ip))
				continue
			}

			// 验证PromQL表达式
			if err := validatePromQLExpr(rule.Expr); err != nil {
				r.logger.Warn(LogModuleMonitor+"无效的PromQL表达式，已跳过",
					zap.Error(err),
					zap.String("rule_name", rule.Name),
					zap.String("expr", rule.Expr))
				continue
			}

			oneRule := rulefmt.Rule{
				Record: rule.Name,
				Expr:   rule.Expr,
				// Record规则不需要For字段
			}

			recordGroup := RecordGroup{
				Name:  rule.Name,
				Rules: []rulefmt.Rule{oneRule},
			}
			myRecordGroups.Groups = append(myRecordGroups.Groups, recordGroup)
		}

		yamlData, err := yaml.Marshal(&myRecordGroups)
		if err != nil {
			r.logger.Error(LogModuleMonitor+"序列化预聚合规则YAML失败",
				zap.Error(err),
				zap.String("pool_name", pool.Name),
				zap.String("instance_ip", ip))
			continue
		}

		// 验证生成的YAML是否有效
		var testGroups RecordGroups
		if err := yaml.Unmarshal(yamlData, &testGroups); err != nil {
			r.logger.Error(LogModuleMonitor+"生成的预聚合规则YAML配置无效",
				zap.Error(err),
				zap.String("pool_name", pool.Name),
				zap.String("instance_ip", ip))
			continue
		}

		ruleMap[ip] = string(yamlData)
	}

	logCacheOperation(r.logger, "生成预聚合规则配置", pool.Name, poolStartTime, nil)
	return ruleMap
}

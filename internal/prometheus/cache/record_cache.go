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
	"strings"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertRecordDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	scrapePoolDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	pm "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type RecordConfigCache interface {
	GetPrometheusRecordRuleConfigYamlByIp(ip string) string
	GenerateRecordRuleConfigYaml(ctx context.Context) error
	GeneratePrometheusRecordRuleConfigYamlOnePool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string
}

type recordConfigCache struct {
	mu             sync.RWMutex
	l              *zap.Logger
	RecordRuleMap  map[string]string
	scrapePoolDao  scrapePoolDao.ScrapePoolDAO
	alertRecordDao alertRecordDao.AlertManagerRecordDAO
	recordHashes   map[string]string
}

// RecordGroup 构造Prometheus record 结构体
type RecordGroup struct {
	Name  string         `yaml:"name"`
	Rules []rulefmt.Rule `yaml:"rules"`
}

// RecordGroups 生成Prometheus record yaml
type RecordGroups struct {
	Groups []RecordGroup `yaml:"groups"`
}

func NewRecordConfig(l *zap.Logger, scrapePoolDao scrapePoolDao.ScrapePoolDAO, alertRecordDao alertRecordDao.AlertManagerRecordDAO) RecordConfigCache {
	return &recordConfigCache{
		l:              l,
		mu:             sync.RWMutex{},
		RecordRuleMap:  make(map[string]string),
		scrapePoolDao:  scrapePoolDao,
		alertRecordDao: alertRecordDao,
		recordHashes:   make(map[string]string),
	}
}

// GetPrometheusRecordRuleConfigYamlByIp 根据IP地址获取Prometheus的预聚合规则配置YAML
func (r *recordConfigCache) GetPrometheusRecordRuleConfigYamlByIp(ip string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.RecordRuleMap[ip]
}

// GenerateRecordRuleConfigYaml 生成并更新所有Prometheus的预聚合规则配置YAML
func (r *recordConfigCache) GenerateRecordRuleConfigYaml(ctx context.Context) error {
	// 获取支持预聚合配置的所有采集池
	pools, _, err := r.scrapePoolDao.GetMonitorScrapePoolSupportedRecord(ctx)
	if err != nil {
		r.l.Error("[监控模块] 获取支持预聚合的采集池失败", zap.Error(err))
		return err
	}

	if len(pools) == 0 {

		r.l.Info("[监控模块] 没有找到支持预聚合的采集池")
		return nil
	}

	r.mu.RLock()
	// 创建当前配置的副本作为临时配置
	tempConfigMap := utils.CopyMap(r.RecordRuleMap)
	tempPoolHashes := utils.CopyMap(r.recordHashes)
	r.mu.RUnlock()

	validIPs := make(map[string]struct{})     // 记录当前所有有效的实例IP
	updatedPools := make(map[string]struct{}) // 记录需要清理旧IP的池子

	for _, pool := range pools {
		currentHash := utils.CalculatePromHash(pool)
		if cachedHash, ok := tempPoolHashes[pool.Name]; ok && cachedHash == currentHash {
			// 哈希未变化，记录有效IP并跳过
			for _, ip := range pool.PrometheusInstances {
				validIPs[ip] = struct{}{}
			}
			continue
		}

		// 标记该池子需要清理旧IP
		updatedPools[pool.Name] = struct{}{}

		oneMap := r.GeneratePrometheusRecordRuleConfigYamlOnePool(ctx, pool)
		if oneMap != nil {
			for ip, out := range oneMap {
				tempConfigMap[ip] = out
				validIPs[ip] = struct{}{}
			}
			tempPoolHashes[pool.Name] = currentHash
		}
	}

	// 清理无效的IP，只清理内存中的配置
	for ip := range tempConfigMap {
		if _, ok := validIPs[ip]; !ok {
			// 检查该IP是否属于被修改的池子
			for poolName := range updatedPools {
				if strings.Contains(ip, poolName) {
					delete(tempConfigMap, ip)
					r.l.Debug("删除无效IP配置", zap.String("ip", ip), zap.String("pool", poolName))
					break
				}
			}
		}
	}

	// 原子性更新配置和哈希
	r.mu.Lock()
	r.RecordRuleMap = tempConfigMap
	r.recordHashes = tempPoolHashes
	r.mu.Unlock()

	return nil
}

// GeneratePrometheusRecordRuleConfigYamlOnePool 根据单个采集池生成Prometheus的预聚合规则配置YAML
func (r *recordConfigCache) GeneratePrometheusRecordRuleConfigYamlOnePool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string {
	rules, _, err := r.alertRecordDao.GetMonitorRecordRuleByPoolId(ctx, pool.ID)
	if err != nil {
		r.l.Error("[监控模块] 根据采集池ID获取预聚合规则失败",
			zap.Error(err),
			zap.String("池子", pool.Name))
		return nil
	}

	if len(rules) == 0 {
		r.l.Info("[监控模块] 没有找到预聚合规则",
			zap.String("池子", pool.Name))
		return nil
	}

	var recordGroups RecordGroups

	// 构建规则组
	for _, rule := range rules {
		forD, err := pm.ParseDuration(rule.ForTime)
		if err != nil {
			r.l.Warn("[监控模块] 解析预聚合规则持续时间失败，使用默认值",
				zap.Error(err),
				zap.String("规则", rule.Name))
			forD, _ = pm.ParseDuration("5s")
		}

		oneRule := rulefmt.Rule{
			Alert: rule.Name,
			Expr:  rule.Expr,
			For:   forD,
		}

		recordGroup := RecordGroup{
			Name:  rule.Name,
			Rules: []rulefmt.Rule{oneRule},
		}
		recordGroups.Groups = append(recordGroups.Groups, recordGroup)
	}

	numInstances := len(pool.PrometheusInstances)
	if numInstances == 0 {

		r.l.Warn("[监控模块] 采集池中没有Prometheus实例",
			zap.String("池子", pool.Name))
		return nil
	}

	ruleMap := make(map[string]string)
	success := true

	// 分片逻辑，将规则分配给不同的Prometheus实例
	for i, ip := range pool.PrometheusInstances {
		var myRecordGroups RecordGroups
		for j, group := range recordGroups.Groups {
			if j%numInstances == i { // 按顺序平均分片
				myRecordGroups.Groups = append(myRecordGroups.Groups, group)
			}
		}

		yamlData, err := yaml.Marshal(&myRecordGroups)
		if err != nil {
			r.l.Error("[监控模块] 序列化预聚合规则YAML失败",
				zap.Error(err),
				zap.String("池子", pool.Name),
				zap.String("IP", ip))
			success = false
			break
		}

		// 不再写入本地文件，只保存到内存

		ruleMap[ip] = string(yamlData)
	}

	if !success {
		// 生成失败，返回nil
		return nil
	}

	return ruleMap
}

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
	"os"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertRecordDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	scrapePoolDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	pm "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/spf13/viper"
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
	localYamlDir   string
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
		localYamlDir:   viper.GetString("prometheus.local_yaml_dir"),
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
	pools, err := r.scrapePoolDao.GetMonitorScrapePoolSupportedRecord(ctx)
	if err != nil {
		r.l.Error("[监控模块] 获取支持预聚合的采集池失败", zap.Error(err))
		return err
	}

	if len(pools) == 0 {
		r.l.Info("[监控模块] 没有找到支持预聚合的采集池")
		return nil
	}

	// 创建新的配置映射
	newConfigMap := make(map[string]string)
	newHashes := make(map[string]string)

	for _, pool := range pools {
		currentHash := utils.CalculatePromHash(pool)
		// 如果缓存中存在该池子的哈希值，并且与当前哈希值相同，则跳过
		if cachedHash, ok := r.recordHashes[pool.Name]; ok && cachedHash == currentHash {
			r.l.Debug("[监控模块] 预聚合规则配置未发生变化，跳过",
				zap.String("池子", pool.Name))
			continue
		}

		oneMap := r.GeneratePrometheusRecordRuleConfigYamlOnePool(ctx, pool)
		if oneMap != nil {
			for ip, out := range oneMap {
				newConfigMap[ip] = out
				newHashes[pool.Name] = currentHash
				r.l.Debug("[监控模块] 成功生成预聚合规则配置",
					zap.String("池子", pool.Name),
					zap.String("IP", ip))
			}
		}
	}

	// 更新缓存
	r.mu.Lock()
	r.RecordRuleMap = newConfigMap
	r.recordHashes = newHashes
	r.mu.Unlock()

	return nil
}

// GeneratePrometheusRecordRuleConfigYamlOnePool 根据单个采集池生成Prometheus的预聚合规则配置YAML
func (r *recordConfigCache) GeneratePrometheusRecordRuleConfigYamlOnePool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string {
	rules, err := r.alertRecordDao.GetMonitorRecordRuleByPoolId(ctx, pool.ID)
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
			continue
		}

		// 创建Pool专属目录
		dir := fmt.Sprintf("%s/%s", r.localYamlDir, pool.Name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			r.l.Error("[监控模块] 创建目录失败",
				zap.Error(err),
				zap.String("目录路径", dir))
			continue
		}

		// 生成文件路径并写入
		fileName := fmt.Sprintf("%s/prometheus_record_%s_%d.yml", dir, pool.Name, i)
		if err := os.WriteFile(fileName, yamlData, 0644); err != nil {
			r.l.Error("[监控模块] 写入预聚合规则文件失败",
				zap.Error(err),
				zap.String("文件路径", fileName))
			continue
		}

		ruleMap[ip] = string(yamlData)
	}

	return ruleMap
}

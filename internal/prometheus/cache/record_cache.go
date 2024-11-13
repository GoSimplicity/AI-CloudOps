package cache

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

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertRecordDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	scrapePoolDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	pm "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"os"
	"sync"
)

type RecordConfigCache interface {
	// GetPrometheusRecordRuleConfigYamlByIp 根据IP地址获取Prometheus的预聚合规则配置YAML
	GetPrometheusRecordRuleConfigYamlByIp(ip string) string
	// GenerateRecordRuleConfigYaml 生成并更新所有Prometheus的预聚合规则配置YAML
	GenerateRecordRuleConfigYaml(ctx context.Context) error
	// GeneratePrometheusRecordRuleConfigYamlOnePool 根据单个采集池生成Prometheus的预聚合规则配置YAML
	GeneratePrometheusRecordRuleConfigYamlOnePool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string
}

type recordConfigCache struct {
	mu             sync.RWMutex      // 读写锁，保护缓存数据
	l              *zap.Logger       // 日志记录器
	RecordRuleMap  map[string]string // 存储预聚合规则
	localYamlDir   string            // 本地YAML目录
	scrapePoolDao  scrapePoolDao.ScrapePoolDAO
	alertRecordDao alertRecordDao.AlertManagerRecordDAO
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
	}
}

func (r *recordConfigCache) GetPrometheusRecordRuleConfigYamlByIp(ip string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.RecordRuleMap[ip]
}

func (r *recordConfigCache) GenerateRecordRuleConfigYaml(ctx context.Context) error {
	// 获取支持预聚合配置的所有采集池
	pools, err := r.scrapePoolDao.GetMonitorScrapePoolSupportedRecord(ctx)
	if err != nil {
		r.l.Error("[监控模块] 获取支持预聚合的采集池失败", zap.Error(err))
		return err
	}

	if len(pools) == 0 {
		r.l.Info("没有找到支持预聚合的采集池")
		return nil
	}

	recordConfigMap := make(map[string]string)

	// 遍历每个采集池生成对应的预聚合规则配置
	for _, pool := range pools {
		oneMap := r.GeneratePrometheusRecordRuleConfigYamlOnePool(ctx, pool)
		if oneMap != nil {
			for ip, out := range oneMap {
				recordConfigMap[ip] = out
			}
		}
	}

	r.mu.Lock()
	r.RecordRuleMap = recordConfigMap
	r.mu.Unlock()

	return nil
}

// GeneratePrometheusRecordRuleConfigYamlOnePool 根据单个采集池生成Prometheus的预聚合规则配置YAML
func (r *recordConfigCache) GeneratePrometheusRecordRuleConfigYamlOnePool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string {
	rules, err := r.alertRecordDao.GetMonitorRecordRuleByPoolId(ctx, pool.ID)
	if err != nil {
		r.l.Error("[监控模块] 根据采集池ID获取预聚合规则失败",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)

		return nil
	}

	if len(rules) == 0 {
		return nil
	}

	var recordGroups RecordGroups

	// 构建规则组
	for _, rule := range rules {
		forD, err := pm.ParseDuration(rule.ForTime)
		if err != nil {
			r.l.Warn("[监控模块] 解析预聚合规则持续时间失败，使用默认值",
				zap.Error(err),
				zap.String("规则", rule.Name),
			)
			forD, _ = pm.ParseDuration("5s")
		}
		oneRule := rulefmt.Rule{
			Alert: rule.Name, // 告警名称
			Expr:  rule.Expr, // 预聚合表达式
			For:   forD,      // 持续时间
		}

		recordGroup := RecordGroup{
			Name:  rule.Name,
			Rules: []rulefmt.Rule{oneRule},
		}
		recordGroups.Groups = append(recordGroups.Groups, recordGroup)
	}

	numInstances := len(pool.PrometheusInstances)
	if numInstances == 0 {
		r.l.Warn("[监控模块] 采集池中没有Prometheus实例", zap.String("池子", pool.Name))
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
				zap.String("IP", ip),
			)
			continue
		}
		fileName := fmt.Sprintf("%s/prometheus_record_%s_%s.yml",
			r.localYamlDir,
			pool.Name,
			ip,
		)

		if err := os.WriteFile(fileName, yamlData, 0644); err != nil {
			r.l.Error("[监控模块] 写入预聚合规则文件失败",
				zap.Error(err),
				zap.String("文件路径", fileName),
			)
			continue
		}

		ruleMap[ip] = string(yamlData)
	}

	return ruleMap
}

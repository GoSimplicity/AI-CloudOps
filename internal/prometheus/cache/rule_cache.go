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

	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"gopkg.in/yaml.v3"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertRuleDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	scrapePoolDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	pm "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type RuleConfigCache interface {
	GetPrometheusAlertRuleConfigYamlByIp(ip string) string
	GenerateAlertRuleConfigYaml(ctx context.Context) error
	GeneratePrometheusAlertRuleConfigYamlOnePool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string
}

type ruleConfigCache struct {

	AlertRuleMap  map[string]string
	mu            sync.RWMutex
	l             *zap.Logger
	localYamlDir  string
	scrapePoolDao scrapePoolDao.ScrapePoolDAO
	alertRuleDao  alertRuleDao.AlertManagerRuleDAO
	ruleHashes    map[string]string
}

// RuleGroup 构造Prometheus Rule 规则的结构体
type RuleGroup struct {
	Name  string         `yaml:"name"`
	Rules []rulefmt.Rule `yaml:"rules"`
}

// RuleGroups 生成Prometheus rule yaml
type RuleGroups struct {
	Groups []RuleGroup `yaml:"groups"`
}

func NewRuleConfigCache(l *zap.Logger, scrapePoolDao scrapePoolDao.ScrapePoolDAO, alertRuleDao alertRuleDao.AlertManagerRuleDAO) RuleConfigCache {
	return &ruleConfigCache{
		l:             l,
		AlertRuleMap:  make(map[string]string),
		localYamlDir:  viper.GetString("prometheus.local_yaml_dir"),
		mu:            sync.RWMutex{},
		scrapePoolDao: scrapePoolDao,
		alertRuleDao:  alertRuleDao,
		ruleHashes:    make(map[string]string),
	}
}

// GetPrometheusAlertRuleConfigYamlByIp 根据IP地址获取告警规则配置
func (r *ruleConfigCache) GetPrometheusAlertRuleConfigYamlByIp(ip string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.AlertRuleMap[ip]
}

// GenerateAlertRuleConfigYaml 生成告警规则配置
func (r *ruleConfigCache) GenerateAlertRuleConfigYaml(ctx context.Context) error {
	// 获取支持告警配置的所有采集池
	pools, err := r.scrapePoolDao.GetMonitorScrapePoolSupportedAlert(ctx)
	if err != nil {
		r.l.Error("[监控模块] 获取支持告警的采集池失败", zap.Error(err))
		return err
	}

	if len(pools) == 0 {

		r.l.Info("[监控模块] 没有找到支持告警的采集池")
		return nil
	}

	r.mu.RLock()
	// 创建当前配置的副本作为临时配置
	tempConfigMap := utils.CopyMap(r.AlertRuleMap)
	tempPoolHashes := utils.CopyMap(r.ruleHashes)
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

		oneMap := r.GeneratePrometheusAlertRuleConfigYamlOnePool(ctx, pool)
		if oneMap != nil {
			for ip, out := range oneMap {
				tempConfigMap[ip] = out
				validIPs[ip] = struct{}{}
			}
			tempPoolHashes[pool.Name] = currentHash
		}
	}

	// 清理被修改池子的旧IP
	utils.CleanupOldIPs(tempConfigMap, updatedPools, validIPs)

	// 原子性更新配置和哈希
	r.mu.Lock()
	r.AlertRuleMap = tempConfigMap
	r.ruleHashes = tempPoolHashes
	r.mu.Unlock()

	return nil
}

// GeneratePrometheusAlertRuleConfigYamlOnePool 为单个采集池生成告警规则配置
func (r *ruleConfigCache) GeneratePrometheusAlertRuleConfigYamlOnePool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string {
	rules, err := r.alertRuleDao.GetMonitorAlertRuleByPoolId(ctx, pool.ID)
	if err != nil {
		r.l.Error("[监控模块] 根据采集池ID获取告警规则失败",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		return nil
	}
	if len(rules) == 0 {
		return nil
	}

	var ruleGroups RuleGroups

	// 构建规则组
	for _, rule := range rules {
		ft, err := pm.ParseDuration(rule.ForTime)
		if err != nil {
			r.l.Warn("[监控模块] 解析告警规则持续时间失败，使用默认值",
				zap.Error(err),
				zap.String("规则", rule.Name),
			)
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
		r.l.Warn("[监控模块] 采集池中没有Prometheus实例", zap.String("池子", pool.Name))
		return nil
	}

	ruleMap := make(map[string]string)
	success := true


	// 分片逻辑，将规则分配给不同的Prometheus实例
	for i, ip := range pool.PrometheusInstances {
		var myRuleGroups RuleGroups

		for j, group := range ruleGroups.Groups {
			if j%numInstances == i { // 按顺序平均分片
				myRuleGroups.Groups = append(myRuleGroups.Groups, group)
			}
		}

		// 序列化规则组为YAML
		yamlData, err := yaml.Marshal(&myRuleGroups)
		if err != nil {
			r.l.Error("[监控模块] 序列化告警规则YAML失败",
				zap.Error(err),
				zap.String("池子", pool.Name),
				zap.String("IP", ip),
			)
			success = false
			break
		}


		// 生成文件路径并写入
		dir := fmt.Sprintf("%s/%s", r.localYamlDir, pool.Name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			r.l.Error("[监控模块] 创建目录失败",
				zap.Error(err),
				zap.String("目录路径", dir),
			)
			success = false
			break
		}

		fileName := fmt.Sprintf("%s/prometheus_rule_%s_%s.yml", dir, pool.Name, ip)
		if err := utils.AtomicWriteFile(fileName, yamlData); err != nil {
			r.l.Error("[监控模块] 写入告警规则文件失败",
				zap.Error(err),
				zap.String("文件路径", fileName),
			)
			success = false
			break
		}

		ruleMap[ip] = string(yamlData)
	}

	if !success {
		// 失败时删除可能已写入的临时文件
		utils.CleanupFailedPool(r.localYamlDir, pool, len(pool.PrometheusInstances))
		return nil
	}

	return ruleMap
}

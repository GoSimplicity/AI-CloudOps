package rule_cache

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertRuleDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/rule"
	scrapePoolDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape/pool"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/prometheus"
	pm "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"os"
	"sync"
)

type RuleConfigCache interface {
	// GetPrometheusAlertRuleConfigYamlByIp 根据IP地址获取Prometheus的告警规则配置YAML
	GetPrometheusAlertRuleConfigYamlByIp(ip string) string
	// GenerateAlertRuleConfigYaml 生成并更新所有Prometheus的告警规则配置YAML
	GenerateAlertRuleConfigYaml(ctx context.Context) error
	// GeneratePrometheusAlertRuleConfigYamlOnePool 根据单个采集池生成Prometheus的告警规则配置YAML
	GeneratePrometheusAlertRuleConfigYamlOnePool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string
}

type ruleConfigCache struct {
	AlertRuleMap  map[string]string // 存储告警规则
	mu            sync.RWMutex      // 读写锁，保护缓存数据
	l             *zap.Logger       // 日志记录器
	localYamlDir  string            // 本地YAML目录
	scrapePoolDao scrapePoolDao.ScrapePoolDAO
	alertRuleDao  alertRuleDao.AlertManagerRuleDAO
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
	}
}

func (r *ruleConfigCache) GetPrometheusAlertRuleConfigYamlByIp(ip string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.AlertRuleMap[ip]
}

func (r *ruleConfigCache) GenerateAlertRuleConfigYaml(ctx context.Context) error {
	// 获取支持告警配置的所有采集池
	pools, err := r.scrapePoolDao.GetMonitorScrapePoolSupportedAlert(ctx)
	if err != nil {
		r.l.Error("[监控模块] 获取支持告警的采集池失败", zap.Error(err))
		return err
	}

	if len(pools) == 0 {
		r.l.Info("没有找到支持告警的采集池")
		return nil
	}

	ruleConfigMap := make(map[string]string)

	// 遍历每个采集池生成对应的规则配置
	for _, pool := range pools {
		oneMap := r.GeneratePrometheusAlertRuleConfigYamlOnePool(ctx, pool)
		if oneMap != nil {
			for ip, out := range oneMap {
				ruleConfigMap[ip] = out
			}
		}
	}

	r.mu.Lock()
	r.AlertRuleMap = ruleConfigMap
	r.mu.Unlock()

	return nil
}

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
		lables := pkg.FromSliceTuMap(rule.Labels)
		annotations := pkg.FromSliceTuMap(rule.Annotations)

		oneRule := rulefmt.Rule{
			Alert:       rule.Name,   // 告警名称
			Expr:        rule.Expr,   // 告警表达式
			For:         ft,          // 持续时间
			Labels:      lables,      // 标签组
			Annotations: annotations, // 注解组
		}

		ruleGroup := RuleGroup{
			Name:  rule.Name,
			Rules: []rulefmt.Rule{oneRule}, // 一个规则组可以包含多个规则
		}
		ruleGroups.Groups = append(ruleGroups.Groups, ruleGroup)
	}

	numInstances := len(pool.PrometheusInstances)
	if numInstances == 0 {
		r.l.Warn("[监控模块] 采集池中没有Prometheus实例", zap.String("池子", pool.Name))
		return nil
	}

	ruleMap := make(map[string]string)

	// 分片逻辑，将规则分配给不同的Prometheus实例，以减少服务器的负载
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
			continue
		}

		fileName := fmt.Sprintf("%s/prometheus_rule_%s_%s.yml",
			r.localYamlDir,
			pool.Name,
			ip,
		)
		if err := os.WriteFile(fileName, yamlData, 0644); err != nil {
			r.l.Error("[监控模块] 写入告警规则文件失败",
				zap.Error(err),
				zap.String("文件路径", fileName),
			)
			continue
		}

		ruleMap[ip] = string(yamlData)
	}

	return ruleMap
}

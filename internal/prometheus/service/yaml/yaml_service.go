package yaml

import (
	"context"
	alertCache "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache/alert_cache"
	promCache "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache/prom_cache"
	recordCache "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache/record_cache"
	ruleCache "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache/rule_cache"
)

type ConfigYamlService interface {
	GetMonitorPrometheusYaml(ctx context.Context, ip string) string
	GetMonitorAlertManagerYaml(ctx context.Context, ip string) string
	GetMonitorPrometheusAlertRuleYaml(ctx context.Context, ip string) string
	GetMonitorPrometheusRecordYaml(ctx context.Context, ip string) string
}

type configYamlService struct {
	promCache   promCache.PromConfigCache
	alertCache  alertCache.AlertConfigCache
	ruleCache   ruleCache.RuleConfigCache
	recordCache recordCache.RecordConfigCache
}

func NewPrometheusConfigService(promCache promCache.PromConfigCache, alertCache alertCache.AlertConfigCache, ruleCache ruleCache.RuleConfigCache, recordCache recordCache.RecordConfigCache) ConfigYamlService {
	return &configYamlService{
		promCache:   promCache,
		alertCache:  alertCache,
		ruleCache:   ruleCache,
		recordCache: recordCache,
	}
}

func (c *configYamlService) GetMonitorPrometheusYaml(ctx context.Context, ip string) string {
	return c.promCache.GetPrometheusMainConfigByIP(ip)
}

func (c *configYamlService) GetMonitorAlertManagerYaml(ctx context.Context, ip string) string {
	return c.alertCache.GetAlertManagerMainConfigYamlByIP(ip)
}

func (c *configYamlService) GetMonitorPrometheusAlertRuleYaml(ctx context.Context, ip string) string {
	return c.ruleCache.GetPrometheusAlertRuleConfigYamlByIp(ip)
}

func (c *configYamlService) GetMonitorPrometheusRecordYaml(ctx context.Context, ip string) string {
	return c.recordCache.GetPrometheusRecordRuleConfigYamlByIp(ip)
}

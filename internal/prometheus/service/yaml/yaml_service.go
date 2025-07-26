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

package yaml

import (
	"context"

	alertCache "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
)

type ConfigYamlService interface {
	GetMonitorPrometheusYaml(ctx context.Context, ip string) string
	GetMonitorAlertManagerYaml(ctx context.Context, ip string) string
	GetMonitorPrometheusAlertRuleYaml(ctx context.Context, ip string) string
	GetMonitorPrometheusRecordYaml(ctx context.Context, ip string) string
}

type configYamlService struct {
	promCache   alertCache.PrometheusConfigCache
	alertCache  alertCache.AlertManagerConfigCache
	ruleCache   alertCache.AlertRuleConfigCache
	recordCache alertCache.RecordRuleConfigCache
}

func NewPrometheusConfigService(
	promCache alertCache.PrometheusConfigCache,
	alertCache alertCache.AlertManagerConfigCache,
	ruleCache alertCache.AlertRuleConfigCache,
	recordCache alertCache.RecordRuleConfigCache,
) ConfigYamlService {
	return &configYamlService{
		promCache:   promCache,
		alertCache:  alertCache,
		ruleCache:   ruleCache,
		recordCache: recordCache,
	}
}

func (c *configYamlService) GetMonitorPrometheusYaml(ctx context.Context, ip string) string {
	return c.promCache.GetConfigByIP(ip)
}

func (c *configYamlService) GetMonitorAlertManagerYaml(ctx context.Context, ip string) string {
	return c.alertCache.GetConfigByIP(ip)
}

func (c *configYamlService) GetMonitorPrometheusAlertRuleYaml(ctx context.Context, ip string) string {
	return c.ruleCache.GetConfigByIP(ip)
}

func (c *configYamlService) GetMonitorPrometheusRecordYaml(ctx context.Context, ip string) string {
	return c.recordCache.GetConfigByIP(ip)
}

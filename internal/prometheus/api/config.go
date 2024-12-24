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

package api

import (
	"net/http"

	yamlService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/yaml"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ConfigYamlHandler struct {
	yamlService yamlService.ConfigYamlService
	l           *zap.Logger
}

func NewConfigYamlHandler(l *zap.Logger, yamlService yamlService.ConfigYamlService) *ConfigYamlHandler {
	return &ConfigYamlHandler{
		l:           l,
		yamlService: yamlService,
	}
}

func (c *ConfigYamlHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")

	prometheusConfigs := monitorGroup.Group("/prometheus_configs")
	{
		prometheusConfigs.GET("/prometheus", c.GetMonitorPrometheusYaml)                // 获取单个 Prometheus 配置文件
		prometheusConfigs.GET("/prometheus_alert", c.GetMonitorPrometheusAlertRuleYaml) // 获取单个 Prometheus 告警配置文件
		prometheusConfigs.GET("/prometheus_record", c.GetMonitorPrometheusRecordYaml)   // 获取单个 Prometheus 记录配置文件
		prometheusConfigs.GET("/alertManager", c.GetMonitorAlertManagerYaml)            // 获取单个 AlertManager 配置文件
	}
}

// GetMonitorPrometheusYaml 获取单个 Prometheus 配置文件
func (c *ConfigYamlHandler) GetMonitorPrometheusYaml(ctx *gin.Context) {
	ip := ctx.Query("ip")

	yaml := c.yamlService.GetMonitorPrometheusYaml(ctx, ip)
	if yaml == "" {
		utils.ErrorWithMessage(ctx, "获取 Prometheus 配置文件失败")
		return
	}

	ctx.String(http.StatusOK, yaml)
}

// GetMonitorPrometheusAlertRuleYaml 获取单个 Prometheus 告警配置规则文件
func (c *ConfigYamlHandler) GetMonitorPrometheusAlertRuleYaml(ctx *gin.Context) {
	ip := ctx.Query("ip")

	yaml := c.yamlService.GetMonitorPrometheusAlertRuleYaml(ctx, ip)
	if yaml == "" {
		utils.ErrorWithMessage(ctx, "获取 Prometheus 告警配置文件失败")
		return
	}

	ctx.String(http.StatusOK, yaml)
}

// GetMonitorPrometheusRecordYaml 获取单个 Prometheus 记录配置文件
func (c *ConfigYamlHandler) GetMonitorPrometheusRecordYaml(ctx *gin.Context) {
	ip := ctx.Query("ip")

	yaml := c.yamlService.GetMonitorPrometheusRecordYaml(ctx, ip)
	if yaml == "" {
		utils.ErrorWithMessage(ctx, "获取 Prometheus 记录配置文件失败")
		return
	}
	ctx.String(http.StatusOK, yaml)
}

// GetMonitorAlertManagerYaml 获取单个 AlertManager 配置文件
func (c *ConfigYamlHandler) GetMonitorAlertManagerYaml(ctx *gin.Context) {
	ip := ctx.Query("ip")

	yaml := c.yamlService.GetMonitorAlertManagerYaml(ctx, ip)
	if yaml == "" {
		utils.ErrorWithMessage(ctx, "获取 AlertManager 配置文件失败")
		return
	}

	ctx.String(http.StatusOK, yaml)
}

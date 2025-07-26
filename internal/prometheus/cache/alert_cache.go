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
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertPoolDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	altconfig "github.com/prometheus/alertmanager/config"
	al "github.com/prometheus/alertmanager/pkg/labels"
	pm "github.com/prometheus/common/model"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const alertSendGroupKey = "alert_send_group"

// AlertManagerConfigCache AlertManager配置缓存接口
type AlertManagerConfigCache interface {
	GetConfigByIP(ip string) string
	GenerateMainConfig(ctx context.Context) error
	GenerateMainConfigForPool(pool *model.MonitorAlertManagerPool) *altconfig.Config
	GenerateRouteConfigForPool(ctx context.Context, pool *model.MonitorAlertManagerPool) ([]*altconfig.Route, []altconfig.Receiver)
}

// alertManagerConfigCache AlertManager配置缓存实现
type alertManagerConfigCache struct {
	configMap        map[string]string
	logger           *zap.Logger
	mu               sync.RWMutex
	alertWebhookAddr string
	alertPoolDAO     alertPoolDao.AlertManagerPoolDAO
	alertSendDAO     alertPoolDao.AlertManagerSendDAO
	configDAO        configDao.MonitorConfigDAO
	poolHashes       map[string]string
	batchManager     *BatchConfigManager
}

// NewAlertManagerConfigCache 创建AlertManager配置缓存实例
func NewAlertManagerConfigCache(
	logger *zap.Logger,
	alertPoolDAO alertPoolDao.AlertManagerPoolDAO,
	alertSendDAO alertPoolDao.AlertManagerSendDAO,
	configDAO configDao.MonitorConfigDAO,
	batchManager *BatchConfigManager,
) AlertManagerConfigCache {
	return &alertManagerConfigCache{
		configMap:        make(map[string]string),
		logger:           logger,
		alertWebhookAddr: viper.GetString("prometheus.alert_webhook_addr"),
		mu:               sync.RWMutex{},
		alertPoolDAO:     alertPoolDAO,
		alertSendDAO:     alertSendDAO,
		configDAO:        configDAO,
		poolHashes:       make(map[string]string),
		batchManager:     batchManager,
	}
}

// GetConfigByIP 根据IP地址获取AlertManager的主配置内容
func (a *alertManagerConfigCache) GetConfigByIP(ip string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.configMap[ip]
}

// GenerateMainConfig 生成所有AlertManager主配置文件
func (a *alertManagerConfigCache) GenerateMainConfig(ctx context.Context) error {
	startTime := time.Now()
	a.logger.Info(LogModuleMonitor + "开始生成AlertManager主配置")

	page := 1
	size := 100
	var allPools []*model.MonitorAlertManagerPool

	for {
		pools, total, err := a.alertPoolDAO.GetMonitorAlertManagerPoolList(ctx, &model.GetMonitorAlertManagerPoolListReq{
			ListReq: model.ListReq{
				Page: page,
				Size: size,
			},
		})
		if err != nil {
			a.logger.Error(LogModuleMonitor+"扫描数据库中的AlertManager集群失败", zap.Error(err))
			return err
		}

		if len(pools) == 0 {
			break
		}

		if err := a.processPoolBatch(ctx, pools); err != nil {
			return err
		}

		allPools = append(allPools, pools...)

		if len(allPools) >= int(total) {
			break
		}
		page++
	}

	if len(allPools) == 0 {
		a.logger.Info(LogModuleMonitor + "没有找到任何AlertManager采集池")
	}

	logCacheOperation(a.logger, "生成AlertManager主配置", "所有池", startTime, nil)
	return nil
}

// processPoolBatch 处理一批AlertManager池
func (a *alertManagerConfigCache) processPoolBatch(ctx context.Context, pools []*model.MonitorAlertManagerPool) error {
	a.mu.RLock()
	tempConfigMap := utils.CopyMap(a.configMap)
	tempPoolHashes := utils.CopyMap(a.poolHashes)
	a.mu.RUnlock()

	validIPs := make(map[string]struct{})
	allConfigsToSave := make(map[string]ConfigData)

	for _, pool := range pools {
		if err := validateInstanceIPs(pool.AlertManagerInstances); err != nil {
			a.logger.Error(LogModuleMonitor+"AlertManager实例IP验证失败",
				zap.String("pool_name", pool.Name),
				zap.Error(err))
			continue
		}

		currentHash := utils.CalculateAlertHash(pool)
		if cachedHash, ok := tempPoolHashes[pool.Name]; ok && cachedHash == currentHash {
			for _, ip := range pool.AlertManagerInstances {
				validIPs[ip] = struct{}{}
			}
			continue
		}

		oneConfig := a.GenerateMainConfigForPool(pool)
		routes, receivers := a.GenerateRouteConfigForPool(ctx, pool)
		if len(routes) == 0 {
			a.logger.Debug(LogModuleMonitor+"没有找到任何告警路由", zap.String("pool_name", pool.Name))
			continue
		}

		mainReceiver := pool.Receiver
		receiverExists := false
		for _, r := range receivers {
			if r.Name == mainReceiver {
				receiverExists = true
				break
			}
		}
		if mainReceiver == "" || !receiverExists {
			mainReceiver = routes[0].Receiver
		}

		oneConfig.Route.Receiver = mainReceiver
		oneConfig.Route.Routes = routes
		// 合并receivers，去重
		oneConfig.Receivers = mergeReceivers(oneConfig.Receivers, receivers)

		yamlData, err := yaml.Marshal(oneConfig)
		if err != nil {
			a.logger.Error(LogModuleMonitor+"生成AlertManager配置失败",
				zap.Error(err),
				zap.String("pool_name", pool.Name))
			continue
		}

		// 验证生成的YAML是否有效
		var testConfig altconfig.Config
		if err := yaml.Unmarshal(yamlData, &testConfig); err != nil {
			a.logger.Error(LogModuleMonitor+"生成的YAML配置无效",
				zap.Error(err),
				zap.String("pool_name", pool.Name))
			continue
		}

		for _, ip := range pool.AlertManagerInstances {
			configName := fmt.Sprintf(ConfigNameAlertManager, pool.ID, ip)
			tempConfigMap[ip] = string(yamlData)
			validIPs[ip] = struct{}{}

			// 准备批量保存的配置数据
			allConfigsToSave[ip] = ConfigData{
				Name:       configName,
				PoolID:     pool.ID,
				ConfigType: model.ConfigTypeAlertManager,
				Content:    string(yamlData),
			}
		}

		tempPoolHashes[pool.Name] = currentHash
	}

	// 批量保存所有配置到数据库
	if len(allConfigsToSave) > 0 {
		if err := batchSaveConfigsToDatabase(ctx, a.batchManager, allConfigsToSave); err != nil {
			a.logger.Error(LogModuleMonitor+"批量保存AlertManager配置失败", zap.Error(err))
			// 不返回错误，继续执行后续逻辑
		}
	}

	// 清理无效的IP配置
	cleanupInvalidIPs(tempConfigMap, validIPs, a.logger)

	a.mu.Lock()
	a.configMap = tempConfigMap
	a.poolHashes = tempPoolHashes
	a.mu.Unlock()

	return nil
}

// parseGroupByLabels 解析GroupBy字符串为标签切片
func parseGroupByLabels(groupByStr string) []pm.LabelName {
	if groupByStr == "" {
		return []pm.LabelName{"alertname"}
	}

	labels := strings.Split(groupByStr, ",")
	var result []pm.LabelName
	for _, label := range labels {
		if trimmed := strings.TrimSpace(label); trimmed != "" {
			result = append(result, pm.LabelName(trimmed))
		}
	}

	if len(result) == 0 {
		return []pm.LabelName{"alertname"}
	}

	return result
}

// GenerateMainConfigForPool 生成单个AlertManager池的主配置
func (a *alertManagerConfigCache) GenerateMainConfigForPool(pool *model.MonitorAlertManagerPool) *altconfig.Config {
	// 解析默认恢复时间
	resolveTimeout, err := pm.ParseDuration(pool.ResolveTimeout)
	if err != nil {
		a.logger.Warn(LogModuleMonitor+"解析ResolveTimeout失败，使用默认值",
			zap.Error(err),
			zap.String("pool_name", pool.Name))
		resolveTimeout, _ = pm.ParseDuration("5m")
	}

	// 解析分组第一次等待时间
	groupWait, err := pm.ParseDuration(pool.GroupWait)
	if err != nil {
		a.logger.Warn(LogModuleMonitor+"解析GroupWait失败，使用默认值",
			zap.Error(err),
			zap.String("pool_name", pool.Name))
		groupWait, _ = pm.ParseDuration("30s")
	}

	// 解析分组等待间隔时间
	groupInterval, err := pm.ParseDuration(pool.GroupInterval)
	if err != nil {
		a.logger.Warn(LogModuleMonitor+"解析GroupInterval失败，使用默认值",
			zap.Error(err),
			zap.String("pool_name", pool.Name))
		groupInterval, _ = pm.ParseDuration("5m")
	}

	// 解析重复发送时间
	repeatInterval, err := pm.ParseDuration(pool.RepeatInterval)
	if err != nil {
		a.logger.Warn(LogModuleMonitor+"解析RepeatInterval失败，使用默认值",
			zap.Error(err),
			zap.String("pool_name", pool.Name))
		repeatInterval, _ = pm.ParseDuration("1h")
	}

	config := &altconfig.Config{
		Global: &altconfig.GlobalConfig{
			ResolveTimeout: resolveTimeout,
		},
		Route: &altconfig.Route{
			GroupWait:      &groupWait,
			GroupInterval:  &groupInterval,
			RepeatInterval: &repeatInterval,
			GroupBy:        parseGroupByLabels(stringSliceToString(pool.GroupBy)),
		},
	}

	return config
}

// GenerateRouteConfigForPool 生成单个AlertManager池的routes和receivers配置
func (a *alertManagerConfigCache) GenerateRouteConfigForPool(ctx context.Context, pool *model.MonitorAlertManagerPool) ([]*altconfig.Route, []altconfig.Receiver) {
	sendGroups, _, err := a.alertSendDAO.GetMonitorSendGroupByPoolID(ctx, pool.ID)
	if err != nil {
		a.logger.Error(LogModuleMonitor+"根据AlertManager池ID查找所有发送组错误",
			zap.Error(err),
			zap.String("pool_name", pool.Name))
		return nil, nil
	}

	if len(sendGroups) == 0 {
		a.logger.Info(LogModuleMonitor+"没有找到发送组", zap.String("pool_name", pool.Name))
		return nil, nil
	}

	var routes []*altconfig.Route
	var receivers []altconfig.Receiver

	for _, sendGroup := range sendGroups {
		repeatInterval, err := pm.ParseDuration(sendGroup.RepeatInterval)
		if err != nil {
			a.logger.Warn(LogModuleMonitor+"解析RepeatInterval失败，使用默认值1h",
				zap.Error(err),
				zap.String("send_group", sendGroup.Name))
			repeatInterval, _ = pm.ParseDuration("1h")
		}

		matcher, err := al.NewMatcher(al.MatchEqual, alertSendGroupKey, fmt.Sprintf("%d", sendGroup.ID))
		if err != nil {
			a.logger.Error(LogModuleMonitor+"创建Matcher失败",
				zap.Error(err),
				zap.String("send_group", sendGroup.Name))
			continue
		}

		route := &altconfig.Route{
			Receiver:       sendGroup.Name,
			Continue:       false,
			Matchers:       []*al.Matcher{matcher},
			RepeatInterval: &repeatInterval,
		}

		// 构建webhook URL
		webhookURL := &url.URL{
			Scheme: "http",
			Host:   a.alertWebhookAddr,
			Path:   "/webhook",
		}

		receiver := altconfig.Receiver{
			Name: sendGroup.Name,
			WebhookConfigs: []*altconfig.WebhookConfig{
				{
					NotifierConfig: altconfig.NotifierConfig{
						VSendResolved: sendGroup.SendResolved == 1,
					},
					URL: &altconfig.SecretURL{
						URL: webhookURL,
					},
				},
			},
		}

		routes = append(routes, route)
		receivers = append(receivers, receiver)
	}

	return routes, receivers
}

// mergeReceivers 合并两个receivers slice，去重（按Name字段）
func mergeReceivers(a, b []altconfig.Receiver) []altconfig.Receiver {
	receiverMap := make(map[string]altconfig.Receiver)
	for _, r := range a {
		receiverMap[r.Name] = r
	}
	for _, r := range b {
		receiverMap[r.Name] = r
	}
	result := make([]altconfig.Receiver, 0, len(receiverMap))
	for _, r := range receiverMap {
		result = append(result, r)
	}
	return result
}

func stringSliceToString(slice model.StringList) string {
	return strings.Join(slice, ",")
}

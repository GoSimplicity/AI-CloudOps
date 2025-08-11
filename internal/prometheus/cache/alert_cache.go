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
	alertPoolDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	altconfig "github.com/prometheus/alertmanager/config"
	al "github.com/prometheus/alertmanager/pkg/labels"
	pm "github.com/prometheus/common/model"
	"github.com/redis/go-redis/v9"
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
	// 不再对外依赖本地 map
	configMap        map[string]string
	redis            redis.Cmdable
	logger           *zap.Logger
	mu               sync.RWMutex
	alertWebhookAddr string
	alertPoolDAO     alertPoolDao.AlertManagerPoolDAO
	alertSendDAO     alertPoolDao.AlertManagerSendDAO
	configDAO        configDao.MonitorConfigDAO
	batchManager     *BatchConfigManager
}

func NewAlertManagerConfigCache(
	logger *zap.Logger,
	alertPoolDAO alertPoolDao.AlertManagerPoolDAO,
	alertSendDAO alertPoolDao.AlertManagerSendDAO,
	configDAO configDao.MonitorConfigDAO,
	batchManager *BatchConfigManager,
	redisClient redis.Cmdable,
) AlertManagerConfigCache {
	return &alertManagerConfigCache{
		configMap:        make(map[string]string),
		logger:           logger,
		redis:            redisClient,
		alertWebhookAddr: viper.GetString("prometheus.alert_webhook_addr"),
		mu:               sync.RWMutex{},
		alertPoolDAO:     alertPoolDAO,
		alertSendDAO:     alertSendDAO,
		configDAO:        configDAO,
		batchManager:     batchManager,
	}
}

// GetConfigByIP 根据IP地址获取AlertManager的主配置内容
func (a *alertManagerConfigCache) GetConfigByIP(ip string) string {
	if ip == "" {
		a.logger.Warn(LogModuleMonitor + "获取配置时IP为空")
		return ""
	}
	ctx := context.Background()
	key := buildRedisKeyAlertManagerMain(ip)
	val, err := a.redis.Get(ctx, key).Result()
	if err != nil {
		a.logger.Debug(LogModuleMonitor+"缓存未命中", zap.String("ip", ip), zap.Error(err))
		return ""
	}
	a.logger.Debug(LogModuleMonitor+"缓存命中", zap.String("ip", ip))
	return val
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
		hashKey := buildRedisHashKeyAlertManager(pool.Name)
		cachedHash, _ := a.redis.Get(ctx, hashKey).Result()
		if cachedHash == currentHash {
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

		// Redis 旧集合
		setKey := buildRedisSetKeyAlertManagerMainPoolIPs(pool.ID)
		oldIPs, _ := a.redis.SMembers(ctx, setKey).Result()
		oldIPSet := map[string]struct{}{}
		for _, old := range oldIPs {
			oldIPSet[old] = struct{}{}
		}

		for _, ip := range pool.AlertManagerInstances {
			configName := fmt.Sprintf(ConfigNameAlertManager, pool.ID, ip)
			validIPs[ip] = struct{}{}

			// 入库（批量）
			allConfigsToSave[ip] = ConfigData{
				Name:       configName,
				PoolID:     pool.ID,
				ConfigType: model.ConfigTypeAlertManager,
				Content:    string(yamlData),
			}

			// 写 Redis
			key := buildRedisKeyAlertManagerMain(ip)
			if err := a.redis.Set(ctx, key, string(yamlData), 0).Err(); err != nil {
				a.logger.Error(LogModuleMonitor+"写入Redis失败", zap.String("pool_name", pool.Name), zap.String("ip", ip), zap.Error(err))
				continue
			}
			_ = a.redis.SAdd(ctx, setKey, ip).Err()
			delete(oldIPSet, ip)
		}

		// 清理失效
		for staleIP := range oldIPSet {
			_ = a.redis.Del(ctx, buildRedisKeyAlertManagerMain(staleIP)).Err()
			_ = a.redis.SRem(ctx, setKey, staleIP).Err()
			a.logger.Debug(LogModuleMonitor+"删除无效IP配置", zap.String("ip", staleIP))
		}

		// 更新池哈希
		_ = a.redis.Set(ctx, hashKey, currentHash, 0).Err()
	}

	// 批量保存所有配置到数据库
	if len(allConfigsToSave) > 0 {
		if err := batchSaveConfigsToDatabase(ctx, a.batchManager, allConfigsToSave); err != nil {
			a.logger.Error(LogModuleMonitor+"批量保存AlertManager配置失败", zap.Error(err))
			// 不返回错误，继续执行后续逻辑
		}
	}

	// 不再维护本地缓存

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

	// 为每个发送组生成 webhook file 配置
	err = a.generateWebhookFilesForSendGroups(ctx, pool, sendGroups)
	if err != nil {
		a.logger.Error(LogModuleMonitor+"生成webhook file配置失败",
			zap.Error(err),
			zap.String("pool_name", pool.Name))
		// 继续执行，不返回错误
	}

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

		// 使用 webhook file 路径而不是直接 URL
		webhookFileName := fmt.Sprintf("/data/alertmanager/webhook_%s_%d.yml", pool.Name, sendGroup.ID)

		receiver := altconfig.Receiver{
			Name: sendGroup.Name,
			WebhookConfigs: []*altconfig.WebhookConfig{
				{
					NotifierConfig: altconfig.NotifierConfig{
						VSendResolved: sendGroup.SendResolved == 1,
					},
					URLFile: webhookFileName,
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

// generateWebhookFilesForSendGroups 为发送组生成 webhook file 配置
func (a *alertManagerConfigCache) generateWebhookFilesForSendGroups(ctx context.Context, pool *model.MonitorAlertManagerPool, sendGroups []*model.MonitorSendGroup) error {
	webhookConfigsToSave := make(map[string]ConfigData)

	for _, sendGroup := range sendGroups {
		// 生成 webhook file 内容（仅包含 URL 字符串）
		webhookURL := a.generateWebhookFileContent(sendGroup)

		// 为每个 AlertManager 实例生成 webhook file 配置
		for _, ip := range pool.AlertManagerInstances {
			configName := fmt.Sprintf("webhook_%s_%d_%s", pool.Name, sendGroup.ID, ip)
			configKey := fmt.Sprintf("%s_%d", ip, sendGroup.ID)

			// 保存到待批量处理的配置映射中
			webhookConfigsToSave[configKey] = ConfigData{
				Name:       configName,
				PoolID:     pool.ID,
				ConfigType: model.ConfigTypeWebhookFile,
				Content:    webhookURL,
			}

			// 同时保存到 Redis 以供实时查询
			redisKey := buildRedisKeyWebhookFile(ip, sendGroup.ID)
			if err := a.redis.Set(ctx, redisKey, webhookURL, 0).Err(); err != nil {
				a.logger.Error(LogModuleMonitor+"保存webhook配置到Redis失败",
					zap.Error(err),
					zap.String("ip", ip),
					zap.Int("send_group_id", sendGroup.ID))
			}
		}
	}

	// 批量保存 webhook 配置到数据库
	if len(webhookConfigsToSave) > 0 {
		if err := batchSaveConfigsToDatabase(ctx, a.batchManager, webhookConfigsToSave); err != nil {
			a.logger.Error(LogModuleMonitor+"批量保存webhook配置失败", zap.Error(err))
			return err
		}
	}

	return nil
}

// buildWebhookURL 构建 webhook 接收 URL，兼容配置中填写 host:port 或完整 URL 的场景，并统一追加 send_group_id
func (a *alertManagerConfigCache) buildWebhookURL(sendGroupID int) string {
	base := strings.TrimSpace(a.alertWebhookAddr)
	if base == "" {
		// 兜底：使用本地端口，避免生成空 URL
		return fmt.Sprintf("http://localhost:%s/api/v1/alerts/receive?send_group_id=%d", viper.GetString("server.port"), sendGroupID)
	}

	hasScheme := strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "https://")
	// 如果是完整 URL，则在其后附加路径/参数；若已包含路径则只追加参数
	if hasScheme {
		u := strings.TrimRight(base, "/")
		// 判断是否已包含目标路径
		if strings.Contains(u, "/api/v1/alerts/receive") {
			sep := "?"
			if strings.Contains(u, "?") {
				sep = "&"
			}
			return fmt.Sprintf("%s%ssend_group_id=%d", u, sep, sendGroupID)
		}
		return fmt.Sprintf("%s/api/v1/alerts/receive?send_group_id=%d", u, sendGroupID)
	}

	// 否则视为 host:port
	return fmt.Sprintf("http://%s/api/v1/alerts/receive?send_group_id=%d", base, sendGroupID)
}

// generateWebhookFileContent 生成 webhook file 的内容（仅为 URL 字符串）
func (a *alertManagerConfigCache) generateWebhookFileContent(sendGroup *model.MonitorSendGroup) string {
	// 始终拼接 send_group_id，确保后端可识别发送组
	return a.buildWebhookURL(sendGroup.ID)
}

// buildRedisKeyWebhookFile 构建 webhook file 的 Redis key
func buildRedisKeyWebhookFile(ip string, sendGroupID int) string {
	return fmt.Sprintf("monitor:webhook_file:%s:%d", ip, sendGroupID)
}

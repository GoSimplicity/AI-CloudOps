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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

// calculateAlertConfigHash 计算AlertManager配置内容的哈希值
func calculateAlertConfigHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

type AlertConfigCache interface {
	GetAlertManagerMainConfigYamlByIP(ip string) string
	GenerateAlertManagerMainConfig(ctx context.Context) error
	GenerateAlertManagerMainConfigOnePool(pool *model.MonitorAlertManagerPool) *altconfig.Config
	GenerateAlertManagerRouteConfigOnePool(ctx context.Context, pool *model.MonitorAlertManagerPool) ([]*altconfig.Route, []altconfig.Receiver)
}

type alertConfigCache struct {
	AlertManagerMainConfigMap map[string]string
	l                         *zap.Logger
	mu                        sync.RWMutex
	alertWebhookAddr          string
	alertPoolDao              alertPoolDao.AlertManagerPoolDAO
	alertSendDao              alertPoolDao.AlertManagerSendDAO
	configDao                 configDao.MonitorConfigDAO
	poolHashes                map[string]string
}

func NewAlertConfigCache(l *zap.Logger, alertPoolDao alertPoolDao.AlertManagerPoolDAO, alertSendDao alertPoolDao.AlertManagerSendDAO, configDao configDao.MonitorConfigDAO) AlertConfigCache {
	return &alertConfigCache{
		AlertManagerMainConfigMap: make(map[string]string),
		l:                         l,
		alertWebhookAddr:          viper.GetString("prometheus.alert_webhook_addr"),
		mu:                        sync.RWMutex{},
		alertPoolDao:              alertPoolDao,
		alertSendDao:              alertSendDao,
		configDao:                 configDao,
		poolHashes:                make(map[string]string),
	}
}

// GetAlertManagerMainConfigYamlByIP 根据IP地址获取AlertManager的主配置内容
func (a *alertConfigCache) GetAlertManagerMainConfigYamlByIP(ip string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.AlertManagerMainConfigMap[ip]
}

// GenerateAlertManagerMainConfig 生成所有AlertManager主配置文件
func (a *alertConfigCache) GenerateAlertManagerMainConfig(ctx context.Context) error {
	page := 1
	size := 100
	var allPools []*model.MonitorAlertManagerPool
	
	// 分批次获取所有AlertManager池
	for {
		pools, total, err := a.alertPoolDao.GetMonitorAlertManagerPoolList(ctx, &model.GetMonitorAlertManagerPoolListReq{
			ListReq: model.ListReq{
				Page: page,
				Size: size,
			},
		})
		if err != nil {
			a.l.Error("[监控模块]扫描数据库中的AlertManager集群失败", zap.Error(err))
			return err
		}
		
		if len(pools) == 0 {
			break
		}
		
		// 处理当前批次的池子
		if err := a.processPoolBatch(ctx, pools); err != nil {
			return err
		}
		
		allPools = append(allPools, pools...)
		
		// 如果已获取所有数据，退出循环
		if len(allPools) >= int(total) {
			break
		}
		
		page++
	}
	
	if len(allPools) == 0 {
		a.l.Info("[监控模块]没有找到任何AlertManager采集池")
	}
	
	return nil
}

// processPoolBatch 处理一批AlertManager池
func (a *alertConfigCache) processPoolBatch(ctx context.Context, pools []*model.MonitorAlertManagerPool) error {
	a.mu.RLock()
	tempConfigMap := utils.CopyMap(a.AlertManagerMainConfigMap)
	tempPoolHashes := utils.CopyMap(a.poolHashes)
	a.mu.RUnlock()
	
	validIPs := make(map[string]struct{})
	
	for _, pool := range pools {
		currentHash := utils.CalculateAlertHash(pool)
		if cachedHash, ok := tempPoolHashes[pool.Name]; ok && cachedHash == currentHash {
			for _, ip := range pool.AlertManagerInstances {
				validIPs[ip] = struct{}{}
			}
			continue
		}
		// 生成单个AlertManager池的主配置
		oneConfig := a.GenerateAlertManagerMainConfigOnePool(pool)
		
		// 生成对应的routes和receivers配置
		routes, receivers := a.GenerateAlertManagerRouteConfigOnePool(ctx, pool)
		if len(routes) == 0 {
			a.l.Debug("[监控模块]没有找到任何告警路由", zap.String("池子", pool.Name))
			continue
		}
		
		// 更新配置
		oneConfig.Route.Routes = routes
		if oneConfig.Receivers == nil {
			oneConfig.Receivers = receivers
		} else {
			oneConfig.Receivers = append(oneConfig.Receivers, receivers...)
		}
		
		// 序列化配置为YAML格式
		yamlData, err := yaml.Marshal(oneConfig)
		if err != nil {
			a.l.Error("[监控模块]生成AlertManager配置失败",
				zap.Error(err),
				zap.String("池子", pool.Name),
			)
			continue
		}
		
		success := true
		// 为每个实例生成配置，不再写入本地文件
		for _, ip := range pool.AlertManagerInstances {
			
			tempConfigMap[ip] = string(yamlData)
			validIPs[ip] = struct{}{}
			
			// 保存配置到数据库
			if err := a.saveAlertConfigToDatabase(ctx, pool, ip, string(yamlData)); err != nil {
				a.l.Error("保存AlertManager配置到数据库失败",
					zap.String("池子", pool.Name),
					zap.String("IP", ip),
					zap.Error(err))
				// 不中断流程，只记录错误
			}
		}
		
		if success {
			tempPoolHashes[pool.Name] = currentHash
		} else {
			// 回滚该池子的哈希，下次重试
			delete(tempPoolHashes, pool.Name)
		}
	}
	
	// 清理无效的IP配置
	for ip := range tempConfigMap {
		if _, ok := validIPs[ip]; !ok {
			delete(tempConfigMap, ip)
		}
	}
	
	// 原子性更新配置和哈希
	a.mu.Lock()
	a.AlertManagerMainConfigMap = tempConfigMap
	a.poolHashes = tempPoolHashes
	a.mu.Unlock()
	
	return nil
}

// saveAlertConfigToDatabase 保存AlertManager配置到数据库
func (a *alertConfigCache) saveAlertConfigToDatabase(ctx context.Context, pool *model.MonitorAlertManagerPool, instanceIP, configContent string) error {
	configHash := calculateAlertConfigHash(configContent)
	configName := fmt.Sprintf("alertmanager-%s-%s", pool.Name, instanceIP)

	// 检查是否已存在相同的配置
	existingConfig, err := a.configDao.GetMonitorConfigByInstance(ctx, instanceIP, model.ConfigTypeAlertManager)
	if err != nil {
		// 如果不存在，创建新配置
		newConfig := &model.MonitorConfig{
			Name:              configName,
			PoolID:            pool.ID,
			InstanceIP:        instanceIP,
			ConfigType:        model.ConfigTypeAlertManager,
			ConfigContent:     configContent,
			ConfigHash:        configHash,
			Status:            model.ConfigStatusActive,
			LastGeneratedTime: time.Now().Unix(),
		}

		return a.configDao.CreateMonitorConfig(ctx, newConfig)
	}

	// 如果配置内容没有变化，不需要更新
	if existingConfig.ConfigHash == configHash {
		return nil
	}

	// 更新现有配置
	existingConfig.Name = configName
	existingConfig.ConfigContent = configContent
	existingConfig.ConfigHash = configHash
	existingConfig.Status = model.ConfigStatusActive
	existingConfig.LastGeneratedTime = time.Now().Unix()

	return a.configDao.UpdateMonitorConfig(ctx, existingConfig)
}

// GenerateAlertManagerMainConfigOnePool 生成单个AlertManager池的主配置
func (a *alertConfigCache) GenerateAlertManagerMainConfigOnePool(pool *model.MonitorAlertManagerPool) *altconfig.Config {
	// 解析默认恢复时间
	resolveTimeout, err := pm.ParseDuration(pool.ResolveTimeout)
	if err != nil {
		a.l.Warn("[监控模块]解析ResolveTimeout失败，使用默认值",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		resolveTimeout, _ = pm.ParseDuration("5s")
	}

	// 解析分组第一次等待时间
	groupWait, err := pm.ParseDuration(pool.GroupWait)
	if err != nil {
		a.l.Warn("[监控模块]解析GroupWait失败，使用默认值",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		groupWait, _ = pm.ParseDuration("5m")
	}

	// 解析分组等待间隔时间
	groupInterval, err := pm.ParseDuration(pool.GroupInterval)
	if err != nil {
		a.l.Warn("[监控模块]解析GroupInterval失败，使用默认值",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		groupInterval, _ = pm.ParseDuration("5m")
	}

	// 解析重复发送时间
	repeatInterval, err := pm.ParseDuration(pool.RepeatInterval)
	if err != nil {
		a.l.Warn("[监控模块]解析RepeatInterval失败，使用默认值",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		repeatInterval, _ = pm.ParseDuration("1h")
	}

	// 生成 Alertmanager 默认配置
	config := &altconfig.Config{
		Global: &altconfig.GlobalConfig{
			ResolveTimeout: resolveTimeout, // 设置恢复超时时间
		},
		Route: &altconfig.Route{ // 设置默认路由
			Receiver:       pool.Receiver,   // 设置默认接收者
			GroupWait:      &groupWait,      // 设置分组等待时间
			GroupInterval:  &groupInterval,  // 设置分组等待间隔
			RepeatInterval: &repeatInterval, // 设置重复发送时间
			GroupByStr:     pool.GroupBy,    // 设置分组分组标签
		},
	}

	return config
}

// GenerateAlertManagerRouteConfigOnePool 生成单个AlertManager池的routes和receivers配置
func (a *alertConfigCache) GenerateAlertManagerRouteConfigOnePool(ctx context.Context, pool *model.MonitorAlertManagerPool) ([]*altconfig.Route, []altconfig.Receiver) {
	sendGroups, _, err := a.alertSendDao.GetMonitorSendGroupByPoolID(ctx, pool.ID)
	if err != nil {
		a.l.Error("[监控模块]根据AlertManager池ID查找所有发送组错误",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		return nil, nil
	}

	if len(sendGroups) == 0 {
		a.l.Info("[监控模块]没有找到发送组", zap.String("池子", pool.Name))
		return nil, nil
	}

	var routes []*altconfig.Route
	var receivers []altconfig.Receiver

	for _, sendGroup := range sendGroups {
		// 处理 RepeatInterval
		repeatInterval, err := pm.ParseDuration(sendGroup.RepeatInterval)
		if err != nil {
			a.l.Warn("[监控模块]解析RepeatInterval失败，使用默认值1h",
				zap.Error(err),
				zap.String("发送组", sendGroup.Name),
			)
			repeatInterval, _ = pm.ParseDuration("1h")
		}

		// 创建 Matcher
		matcher, err := al.NewMatcher(al.MatchEqual, alertSendGroupKey, fmt.Sprintf("%d", sendGroup.ID))
		if err != nil {
			a.l.Error("[监控模块]创建Matcher失败",
				zap.Error(err),
				zap.String("发送组", sendGroup.Name),
			)
			continue
		}

		// 创建 Route
		route := &altconfig.Route{
			Receiver:       sendGroup.Name,
			Continue:       false, // 默认不继续匹配
			Matchers:       []*al.Matcher{matcher},
			RepeatInterval: &repeatInterval,
		}

		// 简化webhook配置以避免兼容性问题
		receiver := altconfig.Receiver{
			Name: sendGroup.Name,
			WebhookConfigs: []*altconfig.WebhookConfig{
				{
					NotifierConfig: altconfig.NotifierConfig{
						VSendResolved: sendGroup.SendResolved == 1,
					},
				},
			},
		}

		routes = append(routes, route)
		receivers = append(receivers, receiver)
	}

	return routes, receivers
}

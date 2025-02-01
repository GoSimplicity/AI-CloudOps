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
	alertPoolDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	altconfig "github.com/prometheus/alertmanager/config"
	al "github.com/prometheus/alertmanager/pkg/labels"
	pm "github.com/prometheus/common/model"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const alertSendGroupKey = "alert_send_group"

type AlertConfigCache interface {
	GetAlertManagerMainConfigYamlByIP(ip string) string
	GenerateAlertManagerMainConfig(ctx context.Context) error
	GenerateAlertManagerMainConfigOnePool(pool *model.MonitorAlertManagerPool) *altconfig.Config
	GenerateAlertManagerRouteConfigOnePool(ctx context.Context, pool *model.MonitorAlertManagerPool) ([]*altconfig.Route, []altconfig.Receiver)
}

type alertConfigCache struct {
	AlertManagerMainConfigMap map[string]string // 存储AlertManager主配置
	l                         *zap.Logger
	mu                        sync.RWMutex // 读写锁，保护缓存数据
	localYamlDir              string       // 本地YAML目录
	alertWebhookAddr          string       // Alertmanager Webhook地址
	alertPoolDao              alertPoolDao.AlertManagerPoolDAO
	alertSendDao              alertPoolDao.AlertManagerSendDAO
}

func NewAlertConfigCache(l *zap.Logger, alertPoolDao alertPoolDao.AlertManagerPoolDAO, alertSendDao alertPoolDao.AlertManagerSendDAO) AlertConfigCache {
	return &alertConfigCache{
		AlertManagerMainConfigMap: make(map[string]string),
		l:                         l,
		localYamlDir:              viper.GetString("prometheus.local_yaml_dir"),
		alertWebhookAddr:          viper.GetString("prometheus.alert_webhook_addr"),
		mu:                        sync.RWMutex{},
		alertPoolDao:              alertPoolDao,
		alertSendDao:              alertSendDao,
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
	// 从数据库中获取所有AlertManager采集池
	pools, err := a.alertPoolDao.GetAllAlertManagerPools(ctx)
	if err != nil {
		a.l.Error("[监控模块]扫描数据库中的AlertManager集群失败", zap.Error(err))
		return err
	}

	if len(pools) == 0 {
		a.l.Info("[监控模块]没有找到任何AlertManager采集池")
		return err
	}

	mainConfigMap := make(map[string]string)

	for _, pool := range pools {
		// 生成单个AlertManager池的主配置
		oneConfig := a.GenerateAlertManagerMainConfigOnePool(pool)

		// 生成对应的routes和receivers配置
		routes, receivers := a.GenerateAlertManagerRouteConfigOnePool(ctx, pool)
		if len(routes) > 0 {
			oneConfig.Route.Routes = routes
		}

		if len(receivers) > 0 {
			if oneConfig.Receivers == nil {
				oneConfig.Receivers = receivers
			} else {
				oneConfig.Receivers = append(oneConfig.Receivers, receivers...)
			}
		}

		// 序列化配置为YAML格式
		config, err := yaml.Marshal(oneConfig)
		if err != nil {
			a.l.Error("[监控模块]根据alert配置生成AlertManager主配置文件错误",
				zap.Error(err),
				zap.String("池子", pool.Name),
			)
			continue
		}
		a.l.Debug("[监控模块]根据alert配置生成AlertManager主配置文件成功",
			zap.String("池子", pool.Name),
			zap.ByteString("配置", config),
		)

		// 写入配置文件并更新缓存
		for index, ip := range pool.AlertManagerInstances {
			fileName := fmt.Sprintf("%s/alertmanager_pool_%s_%d.yaml",
				a.localYamlDir,
				pool.Name,
				index,
			)

			if err := os.WriteFile(fileName, config, 0644); err != nil {
				a.l.Error("[监控模块]写入AlertManager配置文件失败",
					zap.Error(err),
					zap.String("文件路径", fileName),
				)
				continue
			}

			// 配置存入map中
			mainConfigMap[ip] = string(config)
		}
	}

	a.mu.Lock()
	a.AlertManagerMainConfigMap = mainConfigMap
	a.mu.Unlock()

	return nil
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
	sendGroups, err := a.alertSendDao.GetMonitorSendGroupByPoolId(ctx, pool.ID)
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

		// 处理 Webhook URL 文件
		if err := os.MkdirAll(a.localYamlDir, 0755); err != nil {
			a.l.Error("[监控模块]创建目录失败",
				zap.Error(err),
				zap.String("目录", a.localYamlDir),
			)
			continue
		}

		webHookURL := fmt.Sprintf("%s?%s=%d",
			a.alertWebhookAddr,
			alertSendGroupKey,
			sendGroup.ID,
		)

		urlFilePath := fmt.Sprintf("%s/webhook_%d_%d.txt",
			a.localYamlDir,
			pool.ID,
			sendGroup.ID,
		)

		if err := os.WriteFile(urlFilePath, []byte(webHookURL), 0644); err != nil {
			a.l.Error("[监控模块]写入Webhook URL失败",
				zap.Error(err),
				zap.String("文件路径", urlFilePath),
				zap.String("发送组", sendGroup.Name),
			)
			continue
		}

		// 创建 Receiver
		receiver := altconfig.Receiver{
			Name: sendGroup.Name,
			WebhookConfigs: []*altconfig.WebhookConfig{
				{
					NotifierConfig: altconfig.NotifierConfig{
						VSendResolved: sendGroup.SendResolved,
					},
					URLFile: urlFilePath,
				},
			},
		}

		routes = append(routes, route)
		receivers = append(receivers, receiver)
	}

	return routes, receivers
}

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
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"gopkg.in/yaml.v3"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapeJobDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	pcc "github.com/prometheus/common/config"
	pm "github.com/prometheus/common/model"
	pc "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/http"
	"github.com/prometheus/prometheus/discovery/kubernetes"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const hashTmpKey = "__tmp_hash"

// calculateConfigHash 计算配置内容的哈希值
func calculateConfigHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

type PromConfigCache interface {
	GetPrometheusMainConfigByIP(ip string) string
	GeneratePrometheusMainConfig(ctx context.Context) error
	CreateBasePrometheusConfig(pool *model.MonitorScrapePool) (pc.Config, error)
	GenerateScrapeConfigs(ctx context.Context, pool *model.MonitorScrapePool) []*pc.ScrapeConfig
	ApplyHashMod(scrapeConfigs []*pc.ScrapeConfig, modNum, index int) []*pc.ScrapeConfig
}

type promConfigCache struct {
	PrometheusMainConfigMap map[string]string
	mu                      sync.RWMutex
	l                       *zap.Logger
	scrapePoolDao           scrapeJobDao.ScrapePoolDAO
	scrapeJobDao            scrapeJobDao.ScrapeJobDAO
	configDao               configDao.MonitorConfigDAO
	httpSdAPI               string
	poolHashes              map[string]string
	// 添加缓存统计信息
	cacheStats struct {
		hits   int64
		misses int64
		mu     sync.RWMutex
	}
}

func NewPromConfigCache(l *zap.Logger, scrapePoolDao scrapeJobDao.ScrapePoolDAO, scrapeJobDao scrapeJobDao.ScrapeJobDAO, configDao configDao.MonitorConfigDAO) PromConfigCache {
	return &promConfigCache{
		PrometheusMainConfigMap: make(map[string]string),
		httpSdAPI:               viper.GetString("prometheus.httpSdAPI"),
		scrapePoolDao:           scrapePoolDao,
		scrapeJobDao:            scrapeJobDao,
		configDao:               configDao,
		l:                       l,
		mu:                      sync.RWMutex{},
		poolHashes:              make(map[string]string),
	}
}

// GetPrometheusMainConfigByIP 根据IP地址获取Prometheus主配置
func (p *promConfigCache) GetPrometheusMainConfigByIP(ip string) string {
	if ip == "" {
		p.l.Warn("获取配置时IP为空")
		p.recordCacheMiss()
		return ""
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	config, exists := p.PrometheusMainConfigMap[ip]
	if exists {
		p.recordCacheHit()
		p.l.Debug("缓存命中", zap.String("ip", ip))
	} else {
		p.recordCacheMiss()
		p.l.Debug("缓存未命中", zap.String("ip", ip))
	}

	return config
}

// recordCacheHit 记录缓存命中
func (p *promConfigCache) recordCacheHit() {
	p.cacheStats.mu.Lock()
	defer p.cacheStats.mu.Unlock()
	p.cacheStats.hits++
}

// recordCacheMiss 记录缓存未命中
func (p *promConfigCache) recordCacheMiss() {
	p.cacheStats.mu.Lock()
	defer p.cacheStats.mu.Unlock()
	p.cacheStats.misses++
}

// GetCacheStats 获取缓存统计信息
func (p *promConfigCache) GetCacheStats() (hits, misses int64) {
	p.cacheStats.mu.RLock()
	defer p.cacheStats.mu.RUnlock()
	return p.cacheStats.hits, p.cacheStats.misses
}

// GeneratePrometheusMainConfig 生成Prometheus主配置
func (p *promConfigCache) GeneratePrometheusMainConfig(ctx context.Context) error {
	p.l.Info("开始生成Prometheus主配置")
	startTime := time.Now()
	defer func() {
		p.l.Info("Prometheus主配置生成完成", zap.Duration("耗时", time.Since(startTime)))
	}()

	p.mu.RLock()
	tempConfigMap := utils.CopyMap(p.PrometheusMainConfigMap)
	tempPoolHashes := utils.CopyMap(p.poolHashes)
	p.mu.RUnlock()

	validIPs := make(map[string]struct{})
	updatedPools := make(map[string]struct{}) // 记录需要清理旧IP的池子

	page := 1
	batchSize := 100
	processedCount := 0
	
	for {
		pools, total, err := p.scrapePoolDao.GetMonitorScrapePoolList(ctx, &model.GetMonitorScrapePoolListReq{
			ListReq: model.ListReq{
				Page: page,
				Size: batchSize,
			},
		})
		if err != nil {
			p.l.Error("获取采集池失败", zap.Error(err), zap.Int("page", page))
			return fmt.Errorf("获取采集池失败: %w", err)
		}

		if len(pools) == 0 {
			p.l.Info("当前批次未找到采集池", zap.Int("page", page))
			break
		}

		p.l.Info("开始处理采集池批次", zap.Int("批次", page), zap.Int("数量", len(pools)))

		for _, pool := range pools {
			currentHash := utils.CalculatePromHash(pool)
			if cachedHash, ok := tempPoolHashes[pool.Name]; ok && cachedHash == currentHash {
				for _, ip := range pool.PrometheusInstances {
					validIPs[ip] = struct{}{}
				}
				continue
			}

			// 标记该池子需要清理旧IP
			updatedPools[pool.Name] = struct{}{}

			baseConfig, err := p.CreateBasePrometheusConfig(pool)
			if err != nil {
				p.l.Error("创建基础配置失败", zap.String("池子", pool.Name), zap.Error(err))
				continue
			}

			scrapeConfigs := p.GenerateScrapeConfigs(ctx, pool)
			if len(scrapeConfigs) == 0 {
				p.l.Info("未生成采集配置", zap.String("池子", pool.Name))
				continue
			}
			baseConfig.ScrapeConfigs = scrapeConfigs

			instanceConfigs := make(map[string]string) // 暂存实例配置
			success := true

			for idx, ip := range pool.PrometheusInstances {
				configCopy := baseConfig
				if len(pool.PrometheusInstances) > 1 {
					configCopy.ScrapeConfigs = p.ApplyHashMod(scrapeConfigs, len(pool.PrometheusInstances), idx)
				}

				yamlData, err := yaml.Marshal(configCopy)
				if err != nil {
					p.l.Error("配置序列化失败", zap.String("池子", pool.Name), zap.Error(err))
					success = false
					break
				}

				// 不再写入本地文件，只保存到内存和数据库

				instanceConfigs[ip] = string(yamlData) // 暂存到内存
			}

			if success {
				// 原子性更新该池子的所有实例
				for ip, cfg := range instanceConfigs {
					tempConfigMap[ip] = cfg
					validIPs[ip] = struct{}{}

					// 保存配置到数据库
					if err := p.saveConfigToDatabase(ctx, pool, ip, cfg); err != nil {
						p.l.Error("保存配置到数据库失败",
							zap.String("池子", pool.Name),
							zap.String("IP", ip),
							zap.Error(err))
						// 不中断流程，只记录错误
					}
				}
				tempPoolHashes[pool.Name] = currentHash
			}
		}

		processedCount += len(pools)
		if processedCount >= int(total) {
			break
		}
		page++
	}

	// 清理无效的IP，只清理内存中的配置
	for ip := range tempConfigMap {
		if _, ok := validIPs[ip]; !ok {
			// 检查该IP是否属于被修改的池子
			for poolName := range updatedPools {
				if strings.Contains(ip, poolName) {
					delete(tempConfigMap, ip)
					p.l.Debug("删除无效IP配置", zap.String("ip", ip), zap.String("pool", poolName))
					break
				}
			}
		}
	}

	// 原子性更新全局配置
	p.mu.Lock()
	p.PrometheusMainConfigMap = tempConfigMap
	p.poolHashes = tempPoolHashes
	p.mu.Unlock()

	return nil
}

// saveConfigToDatabase 保存配置到数据库
func (p *promConfigCache) saveConfigToDatabase(ctx context.Context, pool *model.MonitorScrapePool, instanceIP, configContent string) error {
	configHash := calculateConfigHash(configContent)
	configName := fmt.Sprintf("prometheus-%s-%s", pool.Name, instanceIP)

	// 检查是否已存在相同的配置
	existingConfig, err := p.configDao.GetMonitorConfigByInstance(ctx, instanceIP, model.ConfigTypePrometheus)
	if err != nil {
		// 如果不存在，创建新配置
		newConfig := &model.MonitorConfig{
			Name:              configName,
			PoolID:            pool.ID,
			InstanceIP:        instanceIP,
			ConfigType:        model.ConfigTypePrometheus,
			ConfigContent:     configContent,
			ConfigHash:        configHash,
			Status:            model.ConfigStatusActive,
			LastGeneratedTime: time.Now().Unix(),
		}

		return p.configDao.CreateMonitorConfig(ctx, newConfig)
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

	return p.configDao.UpdateMonitorConfig(ctx, existingConfig)
}

// CreateBasePrometheusConfig 创建基础Prometheus配置
func (p *promConfigCache) CreateBasePrometheusConfig(pool *model.MonitorScrapePool) (pc.Config, error) {
	var config pc.Config

	// 创建prometheus global全局配置
	if pool.ScrapeInterval <= 0 || pool.ScrapeTimeout <= 0 || pool.ScrapeTimeout > pool.ScrapeInterval {
		return pc.Config{}, fmt.Errorf("采集间隔和采集超时时间不能小于等于0，且采集超时时间不能大于采集间隔")
	}
	config.GlobalConfig = pc.GlobalConfig{
		ScrapeInterval: utils.GenPromDuration(int(pool.ScrapeInterval)), // 采集间隔
		ScrapeTimeout:  utils.GenPromDuration(int(pool.ScrapeTimeout)),  // 采集超时时间
	}

	// 解析外部标签
	externalLabels := utils.ParseExternalLabels(pool.ExternalLabels)
	if len(externalLabels) > 0 {
		config.GlobalConfig.ExternalLabels = labels.FromStrings(externalLabels...)
	}

	// 解析 RemoteWrite URL
	if pool.RemoteWriteUrl != "" {
		remoteWriteURL, err := utils.ParseURL(pool.RemoteWriteUrl)
		if err != nil {
			p.l.Error("解析 RemoteWriteUrl 失败", zap.Error(err), zap.String("池名", pool.Name))
			return pc.Config{}, fmt.Errorf("解析 RemoteWriteUrl 失败: %w", err)
		}

		config.RemoteWriteConfigs = []*pc.RemoteWriteConfig{
			{
				URL:           remoteWriteURL,
				RemoteTimeout: utils.GenPromDuration(int(pool.RemoteTimeoutSeconds)),
			},
		}
	}

	// 解析 RemoteRead URL
	if pool.RemoteReadUrl != "" {
		remoteReadURL, err := utils.ParseURL(pool.RemoteReadUrl)
		if err != nil {
			p.l.Error("解析 RemoteReadUrl 失败", zap.Error(err))
			return pc.Config{}, fmt.Errorf("解析 RemoteReadUrl 失败: %w", err)
		}

		config.RemoteReadConfigs = []*pc.RemoteReadConfig{
			{
				URL:           remoteReadURL,
				RemoteTimeout: utils.GenPromDuration(int(pool.RemoteTimeoutSeconds)),
			},
		}
	}

	// 启用告警，配置 Alertmanager
	if pool.SupportAlert == 1 {
		alertConfig := &pc.AlertmanagerConfig{
			APIVersion: "v2",
			ServiceDiscoveryConfigs: []discovery.Config{ // 服务发现配置
				discovery.StaticConfig{
					{
						Targets: []pm.LabelSet{
							{
								pm.AddressLabel: pm.LabelValue(pool.AlertManagerUrl), // 配置抓取目标地址
							},
						},
					},
				},
			},
		}

		// 组装Alertmanager基础配置
		config.AlertingConfig = pc.AlertingConfig{
			AlertmanagerConfigs: []*pc.AlertmanagerConfig{alertConfig},
		}
	}

	// 启用预聚合，添加规则文件
	if pool.SupportRecord == 1 {
		config.RuleFiles = append(config.RuleFiles, pool.RecordFilePath)
	}

	return config, nil
}

// GenerateScrapeConfigs 生成 Prometheus 采集配置
func (p *promConfigCache) GenerateScrapeConfigs(ctx context.Context, pool *model.MonitorScrapePool) []*pc.ScrapeConfig {
	// 获取与指定池相关的采集任务
	scrapeJobs, err := p.scrapeJobDao.GetMonitorScrapeJobsByPoolId(ctx, pool.ID)
	if err != nil {
		p.l.Error("获取采集任务失败", zap.Error(err), zap.String("池名", pool.Name))
		return nil
	}
	if len(scrapeJobs) == 0 {
		p.l.Info("没有找到任何采集任务", zap.String("池名", pool.Name))
		return nil
	}

	var scrapeConfigs []*pc.ScrapeConfig

	for _, job := range scrapeJobs {
		sc := &pc.ScrapeConfig{
			JobName:        job.Name,
			Scheme:         job.Scheme,
			MetricsPath:    job.MetricsPath,
			ScrapeInterval: utils.GenPromDuration(job.ScrapeInterval),
			ScrapeTimeout:  utils.GenPromDuration(job.ScrapeTimeout),
		}

		// 解析 Relabel 配置
		if job.RelabelConfigsYamlString != "" {
			if err := yaml.Unmarshal([]byte(job.RelabelConfigsYamlString), &sc.RelabelConfigs); err != nil {
				p.l.Error("解析 Relabel 配置失败", zap.Error(err), zap.String("任务名", job.Name))
				continue
			}
		}

		// 根据服务发现类型配置 ServiceDiscoveryConfigs
		switch job.ServiceDiscoveryType {
		case "http":
			if p.httpSdAPI == "" { // 检查 httpSdAPI 是否为空
				p.l.Error("HTTP SD API 地址为空", zap.String("任务名", job.Name))
				continue
			}

			// 拼接 SD API URL
			sdURL := fmt.Sprintf("%s?port=%d&ipAddress=%s", p.httpSdAPI, job.Port, job.IpAddress)

			sc.ServiceDiscoveryConfigs = discovery.Configs{
				&http.SDConfig{
					URL:             sdURL,
					RefreshInterval: utils.GenPromDuration(job.RefreshInterval),
				},
			}
		case "k8s":
			sc.HTTPClientConfig = pcc.HTTPClientConfig{ // 配置 HTTP 客户端配置
				BearerTokenFile: job.BearerTokenFile, // 设置鉴权文件路径
				TLSConfig: pcc.TLSConfig{ // 配置 TLS 配置
					CAFile:             job.TlsCaFilePath, // 设置 CA 证书文件路径
					InsecureSkipVerify: true,              // 跳过证书验证
				},
			}

			sc.ServiceDiscoveryConfigs = discovery.Configs{
				&kubernetes.SDConfig{
					Role:             kubernetes.Role(job.KubernetesSdRole), // 设置k8s服务发现角色
					KubeConfig:       job.KubeConfigFilePath,                // kubeconfig文件路径
					HTTPClientConfig: pcc.DefaultHTTPClientConfig,           // 使用默认的HTTP客户端配置
				},
			}
		default:
			p.l.Warn("未知的服务发现类型", zap.String("类型", job.ServiceDiscoveryType), zap.String("任务名", job.Name))
			continue
		}

		scrapeConfigs = append(scrapeConfigs, sc)
	}

	return scrapeConfigs
}

// ApplyHashMod 根据哈希取模操作对 Prometheus 采集配置进行分组
func (p *promConfigCache) ApplyHashMod(scrapeConfigs []*pc.ScrapeConfig, modNum, index int) []*pc.ScrapeConfig {
	var modified []*pc.ScrapeConfig

	for _, sc := range scrapeConfigs {
		// 深度拷贝 ScrapeConfig
		copySc := utils.DeepCopyScrapeConfig(sc)
		// 添加新的 Relabel 配置
		newRelabelConfigs := []*relabel.Config{
			{
				Action:       relabel.HashMod,                // 使用哈希取模操作
				SourceLabels: pm.LabelNames{pm.AddressLabel}, // 使用抓取目标地址作为源标签
				Regex:        relabel.MustNewRegexp("(.*)"),  // 匹配所有字符
				Replacement:  "$1",                           // 将匹配的整个值作为替换结果
				Modulus:      uint64(modNum),                 // 设置模数
				TargetLabel:  hashTmpKey,                     // 目标标签 用于存储哈希取模后的结果
			},
			{
				Action:       relabel.Keep,                                      // 保留符合条件的目标 丢弃不符合条件的目标
				SourceLabels: pm.LabelNames{hashTmpKey},                         // 使用上一步计算出的哈希结果作为源标签
				Regex:        relabel.MustNewRegexp(fmt.Sprintf("^%d$", index)), // 只保留哈希结果等于当前实例索引 (index) 的目标
			},
		}

		copySc.RelabelConfigs = append(copySc.RelabelConfigs, newRelabelConfigs...)
		modified = append(modified, copySc)
	}

	return modified
}

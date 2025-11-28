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
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/utils"
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
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const hashTmpKey = "__tmp_hash"

type PrometheusConfigCache interface {
	GetConfigByIP(ip string) string
	GenerateMainConfig(ctx context.Context) error
	CreateBaseConfig(pool *model.MonitorScrapePool) (pc.Config, error)
	GenerateScrapeConfigs(ctx context.Context, pool *model.MonitorScrapePool) []*pc.ScrapeConfig
	ApplyHashMod(scrapeConfigs []*pc.ScrapeConfig, modNum, index int) []*pc.ScrapeConfig
}

type prometheusConfigCache struct {
	// 不再使用本地 map 作为缓存，改用 Redis
	redis         redis.Cmdable
	mu            sync.RWMutex
	logger        *zap.Logger
	scrapePoolDAO scrapeJobDao.ScrapePoolDAO
	scrapeJobDAO  scrapeJobDao.ScrapeJobDAO
	configDAO     configDao.MonitorConfigDAO
	httpSdAPI     string
	batchManager  *BatchConfigManager
	cacheStats    struct {
		hits   int64
		misses int64
		mu     sync.RWMutex
	}
}

func NewPrometheusConfigCache(
	logger *zap.Logger,
	scrapePoolDAO scrapeJobDao.ScrapePoolDAO,
	scrapeJobDAO scrapeJobDao.ScrapeJobDAO,
	configDAO configDao.MonitorConfigDAO,
	batchManager *BatchConfigManager,
	redisClient redis.Cmdable,
) PrometheusConfigCache {
	return &prometheusConfigCache{
		redis:         redisClient,
		httpSdAPI:     viper.GetString("prometheus.httpSdAPI"),
		scrapePoolDAO: scrapePoolDAO,
		scrapeJobDAO:  scrapeJobDAO,
		configDAO:     configDAO,
		logger:        logger,
		mu:            sync.RWMutex{},
		batchManager:  batchManager,
	}
}

// GetConfigByIP 根据IP地址获取Prometheus主配置
func (p *prometheusConfigCache) GetConfigByIP(ip string) string {
	if ip == "" {
		p.logger.Warn(LogModuleMonitor + "获取配置时IP为空")
		p.recordCacheMiss()
		return ""
	}

	// 从 Redis 读取配置
	ctx := context.Background()
	key := buildRedisKeyPrometheusMain(ip)
	val, err := p.redis.Get(ctx, key).Result()
	if err != nil {
		p.recordCacheMiss()
		p.logger.Debug(LogModuleMonitor+"缓存未命中", zap.String("ip", ip), zap.Error(err))
		return ""
	}
	p.recordCacheHit()
	p.logger.Debug(LogModuleMonitor+"缓存命中", zap.String("ip", ip))
	return val
}

// recordCacheHit 记录缓存命中
func (p *prometheusConfigCache) recordCacheHit() {
	p.cacheStats.mu.Lock()
	defer p.cacheStats.mu.Unlock()
	p.cacheStats.hits++
}

// recordCacheMiss 记录缓存未命中
func (p *prometheusConfigCache) recordCacheMiss() {
	p.cacheStats.mu.Lock()
	defer p.cacheStats.mu.Unlock()
	p.cacheStats.misses++
}

// GetCacheStats 获取缓存统计信息
func (p *prometheusConfigCache) GetCacheStats() (hits, misses int64) {
	p.cacheStats.mu.RLock()
	defer p.cacheStats.mu.RUnlock()
	return p.cacheStats.hits, p.cacheStats.misses
}

// GenerateMainConfig 生成Prometheus主配置并入库
func (p *prometheusConfigCache) GenerateMainConfig(ctx context.Context) error {
	startTime := time.Now()
	p.logger.Info(LogModuleMonitor + "开始生成Prometheus主配置")

	validIPs := make(map[string]struct{})
	processedCount := 0
	allConfigsToSave := make(map[string]ConfigData)

	page := 1
	batchSize := 100

	for {
		pools, total, err := p.scrapePoolDAO.GetMonitorScrapePoolList(ctx, &model.GetMonitorScrapePoolListReq{
			ListReq: model.ListReq{
				Page: page,
				Size: batchSize,
			},
		})
		if err != nil {
			p.logger.Error(LogModuleMonitor+"获取采集池失败", zap.Error(err), zap.Int("page", page))
			return fmt.Errorf("获取采集池失败: %w", err)
		}

		if len(pools) == 0 {
			p.logger.Info(LogModuleMonitor+"当前批次未找到采集池", zap.Int("page", page))
			break
		}

		p.logger.Info(LogModuleMonitor+"开始处理采集池批次", zap.Int("batch", page), zap.Int("count", len(pools)))

		for _, pool := range pools {
			if err := validateInstanceIPs(pool.PrometheusInstances); err != nil {
				p.logger.Error(LogModuleMonitor+"Prometheus实例IP验证失败",
					zap.String("pool_name", pool.Name),
					zap.Error(err))
				continue
			}

			currentHash := utils.CalculatePromHash(pool)
			// 从 Redis 读取池哈希，判断是否需要更新
			hashKey := buildRedisHashKeyPrometheus(pool.Name)
			cachedHash, _ := p.redis.Get(ctx, hashKey).Result()
			if cachedHash == currentHash {
				for _, ip := range pool.PrometheusInstances {
					validIPs[ip] = struct{}{}
				}
				continue
			}

			baseConfig, err := p.CreateBaseConfig(pool)
			if err != nil {
				p.logger.Error(LogModuleMonitor+"创建基础配置失败", zap.String("pool_name", pool.Name), zap.Error(err))
				continue
			}

			scrapeConfigs := p.GenerateScrapeConfigs(ctx, pool)
			if len(scrapeConfigs) == 0 {
				p.logger.Info(LogModuleMonitor+"未生成采集配置", zap.String("pool_name", pool.Name))
				continue
			}
			baseConfig.ScrapeConfigs = scrapeConfigs

			instanceConfigs := make(map[string]string)
			success := true

			for idx, ip := range pool.PrometheusInstances {
				configCopy := baseConfig
				if len(pool.PrometheusInstances) > 1 {
					configCopy.ScrapeConfigs = p.ApplyHashMod(scrapeConfigs, len(pool.PrometheusInstances), idx)
				}

				yamlData, err := yaml.Marshal(configCopy)
				if err != nil {
					p.logger.Error(LogModuleMonitor+"配置序列化失败", zap.String("pool_name", pool.Name), zap.Error(err))
					success = false
					break
				}

				instanceConfigs[ip] = string(yamlData)
			}

			if success {
				// 清理上一次该池的实例集合中已不存在的IP
				setKey := buildRedisSetKeyPrometheusMainPoolIPs(pool.ID)
				oldIPs, _ := p.redis.SMembers(ctx, setKey).Result()
				oldIPSet := map[string]struct{}{}
				for _, old := range oldIPs {
					oldIPSet[old] = struct{}{}
				}

				// 写入新的配置到 Redis，并更新集合
				for ip, cfg := range instanceConfigs {
					configName := fmt.Sprintf(ConfigNamePrometheus, pool.ID, ip)
					validIPs[ip] = struct{}{}

					// 保存到数据库（批量）
					allConfigsToSave[ip] = ConfigData{
						Name:       configName,
						PoolID:     pool.ID,
						ConfigType: model.ConfigTypePrometheus,
						Content:    cfg,
					}

					// 保存到 Redis
					key := buildRedisKeyPrometheusMain(ip)
					if err := p.redis.Set(ctx, key, cfg, 0).Err(); err != nil {
						p.logger.Error(LogModuleMonitor+"写入Redis失败", zap.String("pool_name", pool.Name), zap.String("ip", ip), zap.Error(err))
						continue
					}
					// 更新池IP集合
					_ = p.redis.SAdd(ctx, setKey, ip).Err()
					// 从旧集合标记中移除，剩余的是待删除
					delete(oldIPSet, ip)
				}

				// 删除已失效IP对应的 Redis 键并从集合移除
				for staleIP := range oldIPSet {
					key := buildRedisKeyPrometheusMain(staleIP)
					_ = p.redis.Del(ctx, key).Err()
					_ = p.redis.SRem(ctx, setKey, staleIP).Err()
					p.logger.Debug(LogModuleMonitor+"删除无效IP配置", zap.String("ip", staleIP))
				}

				// 更新池哈希
				_ = p.redis.Set(ctx, hashKey, currentHash, 0).Err()
			}
		}

		processedCount += len(pools)
		if processedCount >= int(total) {
			break
		}
		page++
	}

	// 批量保存所有配置到数据库
	if len(allConfigsToSave) > 0 {
		if err := batchSaveConfigsToDatabase(ctx, p.batchManager, allConfigsToSave); err != nil {
			p.logger.Error(LogModuleMonitor+"批量保存Prometheus配置失败", zap.Error(err))
			// 不返回错误，继续执行后续逻辑
		}
	}

	// 不再维护本地内存缓存，无需清理本地 map

	logBatchOperation(p.logger, "生成Prometheus主配置", processedCount, processedCount, startTime)
	return nil
}

// CreateBaseConfig 创建基础Prometheus配置
func (p *prometheusConfigCache) CreateBaseConfig(pool *model.MonitorScrapePool) (pc.Config, error) {
	var config pc.Config

	if err := utils.ValidateScrapeTiming(pool.ScrapeInterval, pool.ScrapeTimeout); err != nil {
		return pc.Config{}, err
	}
	config.GlobalConfig = pc.GlobalConfig{
		ScrapeInterval: utils.GenPromDuration(int(pool.ScrapeInterval)),
		ScrapeTimeout:  utils.GenPromDuration(int(pool.ScrapeTimeout)),
	}

	externalLabels := utils.ParseExternalLabels(pool.Tags)
	if len(externalLabels) > 0 {
		config.GlobalConfig.ExternalLabels = labels.FromStrings(externalLabels...)
	}

	if pool.RemoteWriteUrl != "" {
		remoteWriteURL, err := utils.ParseURL(pool.RemoteWriteUrl)
		if err != nil {
			p.logger.Error(LogModuleMonitor+"解析RemoteWriteUrl失败", zap.Error(err), zap.String("pool_name", pool.Name))
			return pc.Config{}, fmt.Errorf("解析RemoteWriteUrl失败: %w", err)
		}
		config.RemoteWriteConfigs = []*pc.RemoteWriteConfig{
			{
				URL:           remoteWriteURL,
				RemoteTimeout: utils.GenPromDuration(int(pool.RemoteTimeoutSeconds)),
			},
		}
	}

	if pool.RemoteReadUrl != "" {
		remoteReadURL, err := utils.ParseURL(pool.RemoteReadUrl)
		if err != nil {
			p.logger.Error(LogModuleMonitor+"解析RemoteReadUrl失败", zap.Error(err), zap.String("pool_name", pool.Name))
			return pc.Config{}, fmt.Errorf("解析RemoteReadUrl失败: %w", err)
		}
		config.RemoteReadConfigs = []*pc.RemoteReadConfig{
			{
				URL:           remoteReadURL,
				RemoteTimeout: utils.GenPromDuration(int(pool.RemoteTimeoutSeconds)),
			},
		}
	}

	if pool.SupportAlert == 1 {
		alertConfig := &pc.AlertmanagerConfig{
			APIVersion: "v2",
			ServiceDiscoveryConfigs: []discovery.Config{
				discovery.StaticConfig{
					{
						Targets: []pm.LabelSet{
							{
								pm.AddressLabel: pm.LabelValue(pool.AlertManagerUrl),
							},
						},
					},
				},
			},
		}
		config.AlertingConfig = pc.AlertingConfig{
			AlertmanagerConfigs: []*pc.AlertmanagerConfig{alertConfig},
		}
	}

	if pool.SupportRecord == 1 && pool.RecordFilePath != "" {
		config.RuleFiles = append(config.RuleFiles, pool.RecordFilePath)
	}

	return config, nil
}

// GenerateScrapeConfigs 生成Prometheus采集配置
func (p *prometheusConfigCache) GenerateScrapeConfigs(ctx context.Context, pool *model.MonitorScrapePool) []*pc.ScrapeConfig {
	scrapeJobs, err := p.scrapeJobDAO.GetMonitorScrapeJobsByPoolId(ctx, pool.ID)
	if err != nil {
		p.logger.Error(LogModuleMonitor+"获取采集任务失败", zap.Error(err), zap.String("pool_name", pool.Name))
		return nil
	}
	if len(scrapeJobs) == 0 {
		p.logger.Info(LogModuleMonitor+"没有找到任何采集任务", zap.String("pool_name", pool.Name))
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

		if job.RelabelConfigsYamlString != "" {
			if err := yaml.Unmarshal([]byte(job.RelabelConfigsYamlString), &sc.RelabelConfigs); err != nil {
				p.logger.Error(LogModuleMonitor+"解析Relabel配置失败", zap.Error(err), zap.String("job_name", job.Name))
				continue
			}
		}

		switch job.ServiceDiscoveryType {
		case model.ServiceDiscoveryTypeHttp:
			if p.httpSdAPI == "" {
				p.logger.Error(LogModuleMonitor+"HTTP SD API地址为空", zap.String("job_name", job.Name))
				continue
			}
			// HTTP SD: 传递端口与树节点ID集合
			sdURL := fmt.Sprintf("%s?port=%d&tree_node_ids=%s", p.httpSdAPI, job.Port, stringSliceToString(job.TreeNodeIDs))
			sc.ServiceDiscoveryConfigs = discovery.Configs{
				&http.SDConfig{
					URL:             sdURL,
					RefreshInterval: utils.GenPromDuration(job.RefreshInterval),
				},
			}
		case model.ServiceDiscoveryTypeK8s:
			// 采集目标的 HTTPClient 配置（用于实际抓取）
			sc.HTTPClientConfig = pcc.HTTPClientConfig{
				BearerTokenFile: job.BearerTokenFile,
				TLSConfig: pcc.TLSConfig{
					CAFile:             job.TlsCaFilePath,
					InsecureSkipVerify: true,
				},
			}
			// Kubernetes 服务发现到 APIServer 的 HTTPClient 配置（用于发现）
			sdHTTPClient := pcc.DefaultHTTPClientConfig
			if job.BearerTokenFile != "" {
				sdHTTPClient.BearerTokenFile = job.BearerTokenFile
			}
			if job.TlsCaFilePath != "" {
				sdHTTPClient.TLSConfig.CAFile = job.TlsCaFilePath
			}
			sdHTTPClient.TLSConfig.InsecureSkipVerify = true

			sc.ServiceDiscoveryConfigs = discovery.Configs{
				&kubernetes.SDConfig{
					Role:             kubernetes.Role(job.KubernetesSdRole),
					KubeConfig:       job.KubeConfigFilePath,
					HTTPClientConfig: sdHTTPClient,
				},
			}
		case model.ServiceDiscoveryTypeStatic:
			sc.ServiceDiscoveryConfigs = discovery.Configs{
				discovery.StaticConfig{
					{
						Targets: []pm.LabelSet{
							{
								pm.AddressLabel: pm.LabelValue(job.IpAddress),
							},
						},
					},
				},
			}
		default:
			p.logger.Warn(LogModuleMonitor+"未知的服务发现类型", zap.Int8("type", int8(job.ServiceDiscoveryType)), zap.String("job_name", job.Name))
			continue
		}

		scrapeConfigs = append(scrapeConfigs, sc)
	}

	return scrapeConfigs
}

// ApplyHashMod 根据哈希取模操作对Prometheus采集配置进行分组
func (p *prometheusConfigCache) ApplyHashMod(scrapeConfigs []*pc.ScrapeConfig, modNum, index int) []*pc.ScrapeConfig {
	var modified []*pc.ScrapeConfig

	for _, sc := range scrapeConfigs {
		copySc := utils.DeepCopyScrapeConfig(sc)
		newRelabelConfigs := []*relabel.Config{
			{
				Action:       relabel.HashMod,
				SourceLabels: pm.LabelNames{pm.AddressLabel},
				Regex:        relabel.MustNewRegexp("(.*)"),
				Replacement:  "$1",
				Modulus:      uint64(modNum),
				TargetLabel:  hashTmpKey,
			},
			{
				Action:       relabel.Keep,
				SourceLabels: pm.LabelNames{hashTmpKey},
				Regex:        relabel.MustNewRegexp(fmt.Sprintf("^%d$", index)),
			},
		}
		copySc.RelabelConfigs = append(copySc.RelabelConfigs, newRelabelConfigs...)
		modified = append(modified, copySc)
	}

	return modified
}

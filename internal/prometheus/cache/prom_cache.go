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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/utils"
	"gopkg.in/yaml.v3"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
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
	cfg *config.Config,
	logger *zap.Logger,
	scrapePoolDAO scrapeJobDao.ScrapePoolDAO,
	scrapeJobDAO scrapeJobDao.ScrapeJobDAO,
	configDAO configDao.MonitorConfigDAO,
	batchManager *BatchConfigManager,
	redisClient redis.Cmdable,
) (PrometheusConfigCache, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置不能为空")
	}
	return &prometheusConfigCache{
		redis:         redisClient,
		httpSdAPI:     cfg.Prometheus.HttpSdAPI,
		scrapePoolDAO: scrapePoolDAO,
		scrapeJobDAO:  scrapeJobDAO,
		configDAO:     configDAO,
		logger:        logger,
		mu:            sync.RWMutex{},
		batchManager:  batchManager,
	}, nil
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

func (p *prometheusConfigCache) recordCacheHit() {
	p.cacheStats.mu.Lock()
	defer p.cacheStats.mu.Unlock()
	p.cacheStats.hits++
}

func (p *prometheusConfigCache) recordCacheMiss() {
	p.cacheStats.mu.Lock()
	defer p.cacheStats.mu.Unlock()
	p.cacheStats.misses++
}

func (p *prometheusConfigCache) GetCacheStats() (hits, misses int64) {
	p.cacheStats.mu.RLock()
	defer p.cacheStats.mu.RUnlock()
	return p.cacheStats.hits, p.cacheStats.misses
}

// GenerateMainConfig 生成Prometheus主配置并入库
func (p *prometheusConfigCache) GenerateMainConfig(ctx context.Context) error {
	startTime := time.Now()
	type pendingCommit struct {
		poolID        int
		hashKey, hash string
		staleIPs      []string
	}
	var pending []pendingCommit
	var failures []error
	reportFailure := func(poolID int, stage string) {
		// Consumer and backend errors can contain credentials or whole configuration fragments.
		p.logger.Error(LogModuleMonitor+"Prometheus配置发布未完成", zap.Int("pool_id", poolID), zap.String("stage", stage))
		failures = append(failures, fmt.Errorf("Prometheus配置发布未完成，池%d，阶段%s", poolID, stage))
	}
	allConfigsToSave := make(map[string]ConfigData)
	processedCount := 0
	const batchSize = 100
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		pools, total, err := p.scrapePoolDAO.GetMonitorScrapePoolList(ctx, &model.GetMonitorScrapePoolListReq{
			ListReq: model.ListReq{Page: page, Size: batchSize},
		})
		if err != nil {
			reportFailure(0, "读取采集池")
			return errors.Join(failures...)
		}
		if len(pools) == 0 {
			break
		}
		for _, pool := range pools {
			if pool == nil {
				reportFailure(0, "采集池缺失")
				continue
			}
			if err := validateInstanceIPs(pool.PrometheusInstances); err != nil {
				reportFailure(pool.ID, "实例地址无效")
				continue
			}
			hashKey := buildRedisHashKeyPrometheus(pool.Name)
			cachedHash, err := p.redis.Get(ctx, hashKey).Result()
			if err != nil && !errors.Is(err, redis.Nil) {
				reportFailure(pool.ID, "读取缓存版本")
				continue
			}
			baseConfig, err := p.CreateBaseConfig(pool)
			if err != nil {
				reportFailure(pool.ID, "生成基础配置")
				continue
			}
			// One DAO read distinguishes an explicit empty active list from a failed snapshot.
			scrapeConfigs, err := p.generateScrapeConfigs(ctx, pool)
			if err != nil {
				reportFailure(pool.ID, "读取或生成采集配置")
				continue
			}
			baseConfig.ScrapeConfigs = scrapeConfigs
			instanceConfigs := make(map[string]string, len(pool.PrometheusInstances))
			complete := true
			for idx, ip := range pool.PrometheusInstances {
				configCopy := baseConfig
				if len(pool.PrometheusInstances) > 1 {
					configCopy.ScrapeConfigs = p.ApplyHashMod(scrapeConfigs, len(pool.PrometheusInstances), idx)
				}
				content, err := marshalPrometheusConfig(configCopy)
				if err != nil {
					reportFailure(pool.ID, "序列化配置")
					complete = false
					break
				}
				if _, err := pc.Load(string(content), slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
					reportFailure(pool.ID, "校验可执行配置")
					complete = false
					break
				}
				instanceConfigs[ip] = string(content)
			}
			if !complete {
				continue
			}
			serialized, err := json.Marshal(instanceConfigs)
			if err != nil {
				reportFailure(pool.ID, "生成配置摘要")
				continue
			}
			// Keep job-only credentials, discovery, and actual per-instance sharding in the hash.
			currentHash := fmt.Sprintf("%x", sha256.Sum256(append([]byte(utils.CalculatePromHash(pool)), serialized...)))
			if cachedHash == currentHash {
				continue
			}
			setKey := buildRedisSetKeyPrometheusMainPoolIPs(pool.ID)
			oldIPs, err := p.redis.SMembers(ctx, setKey).Result()
			if err != nil {
				reportFailure(pool.ID, "读取实例成员")
				continue
			}
			commit := pendingCommit{poolID: pool.ID, hashKey: hashKey, hash: currentHash}
			for _, oldIP := range oldIPs {
				if _, wanted := instanceConfigs[oldIP]; !wanted {
					commit.staleIPs = append(commit.staleIPs, oldIP)
				}
			}
			for ip, content := range instanceConfigs {
				if err := p.redis.Set(ctx, buildRedisKeyPrometheusMain(ip), content, 0).Err(); err != nil {
					reportFailure(pool.ID, "写入配置缓存")
					complete = false
					continue
				}
				if err := p.redis.SAdd(ctx, setKey, ip).Err(); err != nil {
					reportFailure(pool.ID, "写入实例成员")
					complete = false
					continue
				}
				allConfigsToSave[ip] = ConfigData{
					Name: fmt.Sprintf(ConfigNamePrometheus, pool.ID, ip), PoolID: pool.ID,
					ConfigType: model.ConfigTypePrometheus, Content: content,
				}
			}
			if complete {
				pending = append(pending, commit)
			}
		}
		processedCount += len(pools)
		if processedCount >= int(total) {
			break
		}
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(append(failures, err)...)
	}
	if len(allConfigsToSave) > 0 {
		if err := batchSaveConfigsToDatabase(ctx, p.batchManager, allConfigsToSave); err != nil {
			reportFailure(0, "持久化配置")
			return errors.Join(failures...)
		}
	}
	for _, commit := range pending {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		complete := true
		setKey := buildRedisSetKeyPrometheusMainPoolIPs(commit.poolID)
		for _, staleIP := range commit.staleIPs {
			if err := p.redis.Del(ctx, buildRedisKeyPrometheusMain(staleIP)).Err(); err != nil {
				reportFailure(commit.poolID, "清理旧实例缓存")
				complete = false
				continue
			}
			// Keep membership until deletion succeeds so a retry can still find the stale key.
			if err := p.redis.SRem(ctx, setKey, staleIP).Err(); err != nil {
				reportFailure(commit.poolID, "清理旧实例成员")
				complete = false
			}
		}
		if complete {
			if err := p.redis.Set(ctx, commit.hashKey, commit.hash, 0).Err(); err != nil {
				reportFailure(commit.poolID, "确认缓存版本")
			}
		}
	}
	logBatchOperation(p.logger, "生成Prometheus主配置", processedCount, processedCount, startTime)
	return errors.Join(failures...)
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
			p.logger.Error(LogModuleMonitor+"解析RemoteWriteUrl失败", zap.Int("pool_id", pool.ID))
			return pc.Config{}, errors.New("解析RemoteWriteUrl失败")
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
			p.logger.Error(LogModuleMonitor+"解析RemoteReadUrl失败", zap.Int("pool_id", pool.ID))
			return pc.Config{}, errors.New("解析RemoteReadUrl失败")
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

// staticScrapeAddress keeps legacy embedded ports authoritative and applies the
// separate port only when the persisted address has no port of its own.
func staticScrapeAddress(address string, port int) (string, error) {
	if address == "" || address != strings.TrimSpace(address) {
		return "", fmt.Errorf("静态采集地址不能为空或包含首尾空白")
	}
	if _, _, err := net.SplitHostPort(address); err == nil {
		return address, nil
	}
	if port < 0 || port > 65535 {
		return "", fmt.Errorf("静态采集端口超出有效范围")
	}
	host := address
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
		if _, err := netip.ParseAddr(host); err != nil {
			return "", fmt.Errorf("静态采集IP字面量无效")
		}
	}
	if host == "" {
		return "", fmt.Errorf("静态采集主机不能为空")
	}
	if port > 0 {
		return net.JoinHostPort(host, strconv.Itoa(port)), nil
	}
	if ip, err := netip.ParseAddr(host); err == nil && ip.Is6() {
		return "[" + host + "]", nil
	}
	return address, nil
}

// GenerateScrapeConfigs 生成Prometheus采集配置
func (p *prometheusConfigCache) GenerateScrapeConfigs(ctx context.Context, pool *model.MonitorScrapePool) []*pc.ScrapeConfig {
	configs, _ := p.generateScrapeConfigs(ctx, pool)
	return configs
}

func (p *prometheusConfigCache) generateScrapeConfigs(ctx context.Context, pool *model.MonitorScrapePool) ([]*pc.ScrapeConfig, error) {
	scrapeJobs, err := p.scrapeJobDAO.GetMonitorScrapeJobsByPoolId(ctx, pool.ID)
	if err != nil {
		p.logger.Error(LogModuleMonitor+"获取采集任务失败", zap.Int("pool_id", pool.ID))
		return nil, errors.New("读取采集任务失败")
	}
	if len(scrapeJobs) == 0 {
		return []*pc.ScrapeConfig{}, nil
	}

	var scrapeConfigs []*pc.ScrapeConfig

	for _, job := range scrapeJobs {
		if job == nil {
			p.logger.Warn(LogModuleMonitor + "采集任务为空，已跳过")
			continue
		}
		sc := &pc.ScrapeConfig{
			JobName:        job.Name,
			Scheme:         job.Scheme,
			MetricsPath:    job.MetricsPath,
			ScrapeInterval: utils.GenPromDuration(job.ScrapeInterval),
			ScrapeTimeout:  utils.GenPromDuration(job.ScrapeTimeout),
		}

		clientConfig, err := utils.ScrapeHTTPClientConfig(job)
		if err != nil {
			p.logger.Error(LogModuleMonitor+"采集认证或TLS配置无效", zap.Int("job_id", job.ID))
			continue
		}
		sc.HTTPClientConfig = clientConfig

		if job.RelabelConfigsYamlString != "" {
			if err := yaml.Unmarshal([]byte(job.RelabelConfigsYamlString), &sc.RelabelConfigs); err != nil {
				p.logger.Error(LogModuleMonitor+"解析Relabel配置失败", zap.Int("job_id", job.ID))
				continue
			}
		}

		switch job.ServiceDiscoveryType {
		case model.ServiceDiscoveryTypeHttp:
			if p.httpSdAPI == "" {
				p.logger.Error(LogModuleMonitor+"HTTP SD API地址为空", zap.String("job_name", job.Name))
				continue
			}
			sdURL, err := url.Parse(p.httpSdAPI)
			if err != nil || (sdURL.Scheme != "http" && sdURL.Scheme != "https") || sdURL.Host == "" {
				p.logger.Error(LogModuleMonitor+"HTTP SD API地址无效", zap.Int("job_id", job.ID))
				continue
			}
			query := sdURL.Query()
			query.Set("port", strconv.Itoa(job.Port))
			query.Set("tree_node_ids", stringSliceToString(job.TreeNodeIDs))
			sdURL.RawQuery = query.Encode()
			sc.ServiceDiscoveryConfigs = discovery.Configs{
				&http.SDConfig{
					URL:              sdURL.String(),
					HTTPClientConfig: pcc.DefaultHTTPClientConfig,
					RefreshInterval:  utils.GenPromDuration(job.RefreshInterval),
				},
			}
		case model.ServiceDiscoveryTypeK8s:
			// 服务发现使用 kubeconfig 或集群内身份，避免与自定义 HTTP 配置互斥
			sdHTTPClient := pcc.DefaultHTTPClientConfig

			sc.ServiceDiscoveryConfigs = discovery.Configs{
				&kubernetes.SDConfig{
					Role:             kubernetes.Role(job.KubernetesSdRole),
					KubeConfig:       job.KubeConfigFilePath,
					HTTPClientConfig: sdHTTPClient,
				},
			}
		case model.ServiceDiscoveryTypeStatic:
			address, err := staticScrapeAddress(job.IpAddress, job.Port)
			if err != nil {
				p.logger.Error(LogModuleMonitor+"静态采集地址配置无效", zap.Int("job_id", job.ID))
				continue
			}
			sc.ServiceDiscoveryConfigs = discovery.Configs{
				discovery.StaticConfig{
					{
						Targets: []pm.LabelSet{
							{
								pm.AddressLabel: pm.LabelValue(address),
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

	if len(scrapeConfigs) == 0 {
		return nil, errors.New("采集任务快照均无效")
	}
	return scrapeConfigs, nil
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

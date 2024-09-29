package cache

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	pcc "github.com/prometheus/common/config"
	pmodel "github.com/prometheus/common/model"
	pc "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/http"
	"github.com/prometheus/prometheus/discovery/kubernetes"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// 临时哈希键，用于分片配置的哈希操作
	hashTmpKey = "__tmp_hash"
)

// MonitorCache 管理监控相关的缓存数据
type MonitorCache struct {
	PrometheusMainConfigMap   map[string]string // 存储Prometheus主配置，键为IP地址
	AlertManagerMainConfigMap map[string]string // 存储AlertManager主配置
	AlertRuleMap              map[string]string // 存储告警规则
	RecordRuleMap             map[string]string // 存储预聚合规则
	mu                        sync.RWMutex      // 读写锁，保护缓存数据
	l                         *zap.Logger       // 日志记录器
	dao                       dao.PrometheusDao // Prometheus数据访问对象
}

// NewMonitorCache 创建新的MonitorCache实例
func NewMonitorCache(l *zap.Logger, dao dao.PrometheusDao) *MonitorCache {
	return &MonitorCache{
		PrometheusMainConfigMap:   make(map[string]string),
		AlertManagerMainConfigMap: make(map[string]string),
		AlertRuleMap:              make(map[string]string),
		RecordRuleMap:             make(map[string]string),
		mu:                        sync.RWMutex{},
		l:                         l,
		dao:                       dao,
	}
}

// MonitorCacheManager 定期更新缓存并监听退出信号
func (mc *MonitorCache) MonitorCacheManager(ctx context.Context) error {
	intervalSeconds := viper.GetInt("prometheus.interval_seconds")
	interval := time.Duration(intervalSeconds) * time.Second

	// 启动定时任务以更新不同的配置缓存
	go wait.UntilWithContext(ctx, mc.generatePrometheusMainConfig, interval)
	//go wait.UntilWithContext(ctx, mc.generateAlertManagerMainConfig, interval)
	//go wait.UntilWithContext(ctx, mc.generatePrometheusAlertRules, interval)
	//go wait.UntilWithContext(ctx, mc.generatePrometheusRecordRules, interval)

	// 等待上下文取消信号
	<-ctx.Done()
	mc.l.Info("接收到退出信号，停止缓存管理")
	return nil
}

// GetPrometheusMainConfigByIP 根据IP获取Prometheus主配置
func (mc *MonitorCache) GetPrometheusMainConfigByIP(ip string) string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.PrometheusMainConfigMap[ip]
}

// generatePrometheusMainConfig 生成所有Prometheus主配置文件
func (mc *MonitorCache) generatePrometheusMainConfig(ctx context.Context) {
	pools, err := mc.dao.GetAllMonitorScrapePool(ctx)
	if err != nil {
		mc.l.Error("获取采集池失败", zap.Error(err))
		return
	}
	if len(pools) == 0 {
		return
	}

	newConfigMap := make(map[string]string)
	localYamlDir := viper.GetString("prometheus.local_yaml_dir")

	for _, pool := range pools {
		// 创建基础配置
		baseConfig, err := mc.createBasePrometheusConfig(pool)
		if err != nil {
			mc.l.Error("创建基础 Prometheus 配置失败", zap.Error(err), zap.String("池名", pool.Name))
			continue
		}

		// 生成采集配置
		scrapeConfigs := mc.generateScrapeConfigs(ctx, pool)
		if scrapeConfigs == nil {
			continue
		}
		baseConfig.ScrapeConfigs = scrapeConfigs

		for idx, ip := range pool.PrometheusInstances {
			configCopy := baseConfig
			// 如果有多个实例，应用哈希分片
			if len(pool.PrometheusInstances) > 1 {
				configCopy.ScrapeConfigs = mc.applyHashMod(scrapeConfigs, len(pool.PrometheusInstances), idx)
			}

			// 序列化配置为 YAML
			yamlData, err := yaml.Marshal(configCopy)
			if err != nil {
				mc.l.Error("生成 Prometheus 配置失败", zap.Error(err), zap.String("池名", pool.Name))
				continue
			}

			// 写入配置文件
			filePath := fmt.Sprintf("%s/prometheus_pool_%s.yaml", localYamlDir, ip)
			if err := os.WriteFile(filePath, yamlData, 0666); err != nil {
				mc.l.Error("写入 Prometheus 配置文件失败", zap.Error(err), zap.String("文件路径", filePath))
				continue
			}

			newConfigMap[ip] = string(yamlData)
			mc.l.Debug("成功生成 Prometheus 配置", zap.String("池名", pool.Name), zap.String("IP", ip))
		}
	}

	// 更新缓存
	mc.mu.Lock()
	mc.PrometheusMainConfigMap = newConfigMap
	mc.mu.Unlock()
}

// createBasePrometheusConfig 创建基础Prometheus配置，返回错误
func (mc *MonitorCache) createBasePrometheusConfig(pool *model.MonitorScrapePool) (pc.Config, error) {
	global := pc.GlobalConfig{
		ScrapeInterval: genPromDuration(pool.ScrapeInterval),
		ScrapeTimeout:  genPromDuration(pool.ScrapeTimeout),
	}

	// 解析外部标签
	externalLabels := parseExternalLabels(pool.ExternalLabels)
	if len(externalLabels) > 0 {
		global.ExternalLabels = labels.FromStrings(externalLabels...)
	}

	// 解析 RemoteWrite URL
	remoteWriteURL, err := parseURL(pool.RemoteWriteUrl)
	if err != nil {
		return pc.Config{}, fmt.Errorf("解析 RemoteWriteUrl 失败: %w", err)
	}

	remoteWrite := &pc.RemoteWriteConfig{
		URL:           remoteWriteURL,
		RemoteTimeout: genPromDuration(pool.RemoteTimeoutSeconds),
	}

	config := pc.Config{
		GlobalConfig:       global,
		RemoteWriteConfigs: []*pc.RemoteWriteConfig{remoteWrite},
	}

	alertEnable := viper.GetInt("prometheus.enable_alert")

	if pool.SupportAlert == 1 && alertEnable == 1 {
		// 配置 RemoteRead
		remoteReadURL, err := parseURL(pool.RemoteReadUrl)
		if err != nil {
			return pc.Config{}, fmt.Errorf("解析 RemoteReadUrl 失败: %w", err)
		}

		config.RemoteReadConfigs = []*pc.RemoteReadConfig{
			{
				URL:           remoteReadURL,
				RemoteTimeout: genPromDuration(pool.RemoteTimeoutSeconds),
			},
		}

		// 配置 Alertmanager
		alertConfig := &pc.AlertmanagerConfig{
			APIVersion: "v2",
			ServiceDiscoveryConfigs: discovery.Configs{
				&discovery.StaticConfig{
					{
						Targets: []pmodel.LabelSet{
							{pmodel.AddressLabel: pmodel.LabelValue(pool.AlertManagerUrl)},
						},
					},
				},
			},
		}

		config.AlertingConfig = pc.AlertingConfig{
			AlertmanagerConfigs: []*pc.AlertmanagerConfig{alertConfig},
		}

		// 添加告警规则文件
		config.RuleFiles = append(config.RuleFiles, pool.RuleFilePath)
	}

	recordEnable := viper.GetInt("prometheus.enable_record")

	if pool.SupportRecord == 1 && recordEnable == 1 {
		// 添加预聚合规则文件
		config.RuleFiles = append(config.RuleFiles, pool.RecordFilePath)
	}

	return config, nil
}

// parseExternalLabels 解析外部标签
func parseExternalLabels(labelsList []string) []string {
	var parsed []string

	for _, label := range labelsList {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) == 2 {
			parsed = append(parsed, parts[0], parts[1])
		}
	}

	return parsed
}

// parseURL 解析字符串为URL，返回错误而非 panic
func parseURL(u string) (*pcc.URL, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("无效的URL: %s", u)
	}

	return &pcc.URL{URL: parsed}, nil
}

// genPromDuration 转换秒为Prometheus Duration
func genPromDuration(seconds int) pmodel.Duration {
	return pmodel.Duration(time.Duration(seconds) * time.Second)
}

// applyHashMod 应用HashMod和Keep Relabel配置进行分片
func (mc *MonitorCache) applyHashMod(scrapeConfigs []*pc.ScrapeConfig, modNum, index int) []*pc.ScrapeConfig {
	var modified []*pc.ScrapeConfig

	for _, sc := range scrapeConfigs {
		// 深度拷贝 ScrapeConfig
		copySc := deepCopyScrapeConfig(sc)
		// 添加新的 Relabel 配置
		newRelabelConfigs := []*relabel.Config{
			{
				Action:       relabel.HashMod,
				SourceLabels: pmodel.LabelNames{pmodel.AddressLabel},
				Regex:        relabel.MustNewRegexp("(.*)"),
				Replacement:  "$1",
				Modulus:      uint64(modNum),
				TargetLabel:  hashTmpKey,
			},
			{
				Action:       relabel.Keep,
				SourceLabels: pmodel.LabelNames{hashTmpKey},
				Regex:        relabel.MustNewRegexp(fmt.Sprintf("^%d$", index)),
			},
		}
		copySc.RelabelConfigs = append(copySc.RelabelConfigs, newRelabelConfigs...)
		modified = append(modified, copySc)
	}

	return modified
}

// deepCopyScrapeConfig 深度拷贝 ScrapeConfig
func deepCopyScrapeConfig(sc *pc.ScrapeConfig) *pc.ScrapeConfig {
	copySc := *sc

	// 深度拷贝 RelabelConfigs
	if sc.RelabelConfigs != nil {
		copySc.RelabelConfigs = make([]*relabel.Config, len(sc.RelabelConfigs))
		for i, rc := range sc.RelabelConfigs {
			copyRC := *rc
			copySc.RelabelConfigs[i] = &copyRC
		}
	}

	// 深度拷贝 ServiceDiscoveryConfigs
	if sc.ServiceDiscoveryConfigs != nil {
		copySc.ServiceDiscoveryConfigs = make(discovery.Configs, len(sc.ServiceDiscoveryConfigs))
		copy(copySc.ServiceDiscoveryConfigs, sc.ServiceDiscoveryConfigs)
	}

	return &copySc
}

// generateScrapeConfigs 生成采集配置
func (mc *MonitorCache) generateScrapeConfigs(ctx context.Context, pool *model.MonitorScrapePool) []*pc.ScrapeConfig {
	scrapeJobs, err := mc.dao.GetMonitorScrapeJobsByPoolId(ctx, pool.ID)
	if err != nil {
		mc.l.Error("获取采集任务失败", zap.Error(err), zap.String("池名", pool.Name))
		return nil
	}
	if len(scrapeJobs) == 0 {
		return nil
	}

	var scrapeConfigs []*pc.ScrapeConfig

	for _, job := range scrapeJobs {
		sc := &pc.ScrapeConfig{
			JobName:        job.Name,
			Scheme:         job.Scheme,
			MetricsPath:    job.MetricsPath,
			ScrapeInterval: genPromDuration(job.ScrapeInterval),
			ScrapeTimeout:  genPromDuration(job.ScrapeTimeout),
		}

		// 解析 Relabel 配置
		if job.RelabelConfigsYamlString != "" {
			if err := yaml.Unmarshal([]byte(job.RelabelConfigsYamlString), &sc.RelabelConfigs); err != nil {
				mc.l.Error("解析 Relabel 配置失败", zap.Error(err), zap.String("任务名", job.Name))
				continue
			}
		}

		// 根据服务发现类型配置 ServiceDiscoveryConfigs
		switch job.ServiceDiscoveryType {
		case "http":
			httpSdApi, err := mc.dao.GetHttpSdApi(ctx, job.ID)
			if err != nil {
				mc.l.Error("获取 HTTP SD API 失败", zap.Error(err), zap.String("任务名", job.Name))
				continue
			}
			sdURL := fmt.Sprintf("%s?port=%d&leafNodeIds=%s", httpSdApi, job.Port, strings.Join(job.TreeNodeIDs, ","))
			sc.ServiceDiscoveryConfigs = discovery.Configs{
				&http.SDConfig{
					URL:             sdURL,
					RefreshInterval: genPromDuration(job.RefreshInterval),
				},
			}
		case "k8s":
			sc.HTTPClientConfig = pcc.HTTPClientConfig{
				BearerTokenFile: job.BearerTokenFile,
				TLSConfig: pcc.TLSConfig{
					CAFile:             job.TlsCaFilePath,
					InsecureSkipVerify: true,
				},
			}
			sc.ServiceDiscoveryConfigs = discovery.Configs{
				&kubernetes.SDConfig{
					Role:             kubernetes.Role(job.KubernetesSdRole),
					KubeConfig:       job.KubeConfigFilePath,
					HTTPClientConfig: pcc.DefaultHTTPClientConfig,
				},
			}
		default:
			mc.l.Warn("未知的服务发现类型", zap.String("类型", job.ServiceDiscoveryType), zap.String("任务名", job.Name))
			continue
		}

		scrapeConfigs = append(scrapeConfigs, sc)
	}

	return scrapeConfigs
}

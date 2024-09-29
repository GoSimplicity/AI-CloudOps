package cache

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao"
	altconfig "github.com/prometheus/alertmanager/config"
	al "github.com/prometheus/alertmanager/pkg/labels"
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
	logger                    *zap.Logger       // 日志记录器
	dao                       dao.PrometheusDao // Prometheus数据访问对象
	localYamlDir              string            // 本地YAML目录
	alertWebhookAddr          string            // Alertmanager Webhook地址
	alertEnable               bool              // 是否启用告警
	recordEnable              bool              // 是否启用预聚合
}

// NewMonitorCache 创建新的MonitorCache实例
func NewMonitorCache(logger *zap.Logger, dao dao.PrometheusDao) *MonitorCache {
	return &MonitorCache{
		PrometheusMainConfigMap:   make(map[string]string),
		AlertManagerMainConfigMap: make(map[string]string),
		AlertRuleMap:              make(map[string]string),
		RecordRuleMap:             make(map[string]string),
		mu:                        sync.RWMutex{},
		logger:                    logger,
		dao:                       dao,
		localYamlDir:              viper.GetString("prometheus.local_yaml_dir"),
		alertWebhookAddr:          viper.GetString("prometheus.alert_webhook_addr"),
		alertEnable:               viper.GetInt("prometheus.enable_alert") == 1,
		recordEnable:              viper.GetInt("prometheus.enable_record") == 1,
	}
}

// MonitorCacheManager 定期更新缓存并监听退出信号
func (mc *MonitorCache) MonitorCacheManager(ctx context.Context) error {
	intervalSeconds := viper.GetInt("prometheus.interval_seconds")
	interval := time.Duration(intervalSeconds) * time.Second

	// 启动定时任务以更新不同的配置缓存
	go wait.UntilWithContext(ctx, func(ctx context.Context) { mc.GeneratePrometheusMainConfig(ctx) }, interval)
	go wait.UntilWithContext(ctx, func(ctx context.Context) { mc.GenerateAlertManagerMainConfig(ctx) }, interval)
	// go wait.UntilWithContext(ctx, mc.generatePrometheusAlertRules, interval)
	// go wait.UntilWithContext(ctx, mc.generatePrometheusRecordRules, interval)

	// 等待上下文取消信号
	<-ctx.Done()
	mc.logger.Info("接收到退出信号，停止缓存管理")
	return nil
}

// GetPrometheusMainConfigByIP 根据IP获取Prometheus主配置
func (mc *MonitorCache) GetPrometheusMainConfigByIP(ip string) string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.PrometheusMainConfigMap[ip]
}

// GeneratePrometheusMainConfig 生成所有Prometheus主配置文件
func (mc *MonitorCache) GeneratePrometheusMainConfig(ctx context.Context) {
	pools, err := mc.dao.GetAllMonitorScrapePool(ctx)
	if err != nil {
		mc.logger.Error("获取采集池失败", zap.Error(err))
		return
	}
	if len(pools) == 0 {
		mc.logger.Info("没有找到任何采集池")
		return
	}

	newConfigMap := make(map[string]string)

	for _, pool := range pools {
		// 创建基础配置
		baseConfig, err := mc.CreateBasePrometheusConfig(pool)
		if err != nil {
			mc.logger.Error("创建基础 Prometheus 配置失败", zap.Error(err), zap.String("池名", pool.Name))
			continue
		}

		// 生成采集配置
		scrapeConfigs := mc.GenerateScrapeConfigs(ctx, pool)
		if len(scrapeConfigs) == 0 {
			mc.logger.Warn("没有找到任何采集任务", zap.String("池名", pool.Name))
			continue
		}
		baseConfig.ScrapeConfigs = scrapeConfigs

		for idx, ip := range pool.PrometheusInstances {
			configCopy := baseConfig // 浅拷贝
			// 如果有多个实例，应用哈希分片
			if len(pool.PrometheusInstances) > 1 {
				configCopy.ScrapeConfigs = mc.ApplyHashMod(scrapeConfigs, len(pool.PrometheusInstances), idx)
			}

			// 序列化配置为 YAML
			yamlData, err := yaml.Marshal(configCopy)
			if err != nil {
				mc.logger.Error("生成 Prometheus 配置失败", zap.Error(err), zap.String("池名", pool.Name))
				continue
			}

			// 写入配置文件
			filePath := fmt.Sprintf("%s/prometheus_pool_%s.yaml", mc.localYamlDir, ip)
			if err := os.WriteFile(filePath, yamlData, 0644); err != nil { // 使用更安全的文件权限
				mc.logger.Error("写入 Prometheus 配置文件失败", zap.Error(err), zap.String("文件路径", filePath))
				continue
			}

			newConfigMap[ip] = string(yamlData)
			mc.logger.Debug("成功生成 Prometheus 配置", zap.String("池名", pool.Name), zap.String("IP", ip))
		}
	}

	// 更新缓存
	mc.mu.Lock()
	mc.PrometheusMainConfigMap = newConfigMap
	mc.mu.Unlock()
}

// CreateBasePrometheusConfig 创建基础Prometheus配置，返回错误
func (mc *MonitorCache) CreateBasePrometheusConfig(pool *model.MonitorScrapePool) (pc.Config, error) {
	globalConfig := pc.GlobalConfig{
		ScrapeInterval: GenPromDuration(pool.ScrapeInterval),
		ScrapeTimeout:  GenPromDuration(pool.ScrapeTimeout),
	}

	// 解析外部标签
	externalLabels := ParseExternalLabels(pool.ExternalLabels)
	if len(externalLabels) > 0 {
		globalConfig.ExternalLabels = labels.FromStrings(externalLabels...)
	}

	// 解析 RemoteWrite URL
	remoteWriteURL, err := ParseURL(pool.RemoteWriteUrl)
	if err != nil {
		return pc.Config{}, fmt.Errorf("解析 RemoteWriteUrl 失败: %w", err)
	}

	remoteWrite := &pc.RemoteWriteConfig{
		URL:           remoteWriteURL,
		RemoteTimeout: GenPromDuration(pool.RemoteTimeoutSeconds),
	}

	config := pc.Config{
		GlobalConfig:       globalConfig,
		RemoteWriteConfigs: []*pc.RemoteWriteConfig{remoteWrite},
	}

	if mc.alertEnable && pool.SupportAlert == 1 {
		// 配置 RemoteRead
		remoteReadURL, err := ParseURL(pool.RemoteReadUrl)
		if err != nil {
			return pc.Config{}, fmt.Errorf("解析 RemoteReadUrl 失败: %w", err)
		}

		config.RemoteReadConfigs = []*pc.RemoteReadConfig{
			{
				URL:           remoteReadURL,
				RemoteTimeout: GenPromDuration(pool.RemoteTimeoutSeconds),
			},
		}

		// 配置 Alertmanager
		alertConfig := &pc.AlertmanagerConfig{
			APIVersion: "v2",
			ServiceDiscoveryConfigs: discovery.Configs{
				&discovery.StaticConfig{
					{
						Targets: []pm.LabelSet{
							{pm.AddressLabel: pm.LabelValue(pool.AlertManagerUrl)},
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

	if mc.recordEnable && pool.SupportRecord == 1 {
		// 添加预聚合规则文件
		config.RuleFiles = append(config.RuleFiles, pool.RecordFilePath)
	}

	return config, nil
}

// ParseExternalLabels 解析外部标签
func ParseExternalLabels(labelsList []string) []string {
	var parsed []string

	for _, label := range labelsList {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) == 2 {
			parsed = append(parsed, parts[0], parts[1])
		}
	}

	return parsed
}

// ParseURL 解析字符串为URL，返回错误而非 panic
func ParseURL(u string) (*pcc.URL, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("无效的URL: %s", u)
	}

	return &pcc.URL{URL: parsed}, nil
}

// GenPromDuration 转换秒为Prometheus Duration
func GenPromDuration(seconds int) pm.Duration {
	return pm.Duration(time.Duration(seconds) * time.Second)
}

// ApplyHashMod 应用HashMod和Keep Relabel配置进行分片
func (mc *MonitorCache) ApplyHashMod(scrapeConfigs []*pc.ScrapeConfig, modNum, index int) []*pc.ScrapeConfig {
	var modified []*pc.ScrapeConfig

	for _, sc := range scrapeConfigs {
		// 深度拷贝 ScrapeConfig
		copySc := DeepCopyScrapeConfig(sc)
		// 添加新的 Relabel 配置
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

// DeepCopyScrapeConfig 深度拷贝 ScrapeConfig
func DeepCopyScrapeConfig(sc *pc.ScrapeConfig) *pc.ScrapeConfig {
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

// GenerateScrapeConfigs 生成采集配置
func (mc *MonitorCache) GenerateScrapeConfigs(ctx context.Context, pool *model.MonitorScrapePool) []*pc.ScrapeConfig {
	scrapeJobs, err := mc.dao.GetMonitorScrapeJobsByPoolId(ctx, pool.ID)
	if err != nil {
		mc.logger.Error("获取采集任务失败", zap.Error(err), zap.String("池名", pool.Name))
		return nil
	}
	if len(scrapeJobs) == 0 {
		mc.logger.Info("没有找到任何采集任务", zap.String("池名", pool.Name))
		return nil
	}

	var scrapeConfigs []*pc.ScrapeConfig

	for _, job := range scrapeJobs {
		sc := &pc.ScrapeConfig{
			JobName:        job.Name,
			Scheme:         job.Scheme,
			MetricsPath:    job.MetricsPath,
			ScrapeInterval: GenPromDuration(job.ScrapeInterval),
			ScrapeTimeout:  GenPromDuration(job.ScrapeTimeout),
		}

		// 解析 Relabel 配置
		if job.RelabelConfigsYamlString != "" {
			if err := yaml.Unmarshal([]byte(job.RelabelConfigsYamlString), &sc.RelabelConfigs); err != nil {
				mc.logger.Error("解析 Relabel 配置失败", zap.Error(err), zap.String("任务名", job.Name))
				continue
			}
		}

		// 根据服务发现类型配置 ServiceDiscoveryConfigs
		switch job.ServiceDiscoveryType {
		case "http":
			httpSdAPI, err := mc.dao.GetHttpSdApi(ctx, job.ID)
			if err != nil {
				mc.logger.Error("获取 HTTP SD API 失败", zap.Error(err), zap.String("任务名", job.Name))
				continue
			}
			sdURL := fmt.Sprintf("%s?port=%d&leafNodeIds=%s", httpSdAPI, job.Port, strings.Join(job.TreeNodeIDs, ","))
			sc.ServiceDiscoveryConfigs = discovery.Configs{
				&http.SDConfig{
					URL:             sdURL,
					RefreshInterval: GenPromDuration(job.RefreshInterval),
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
			mc.logger.Warn("未知的服务发现类型", zap.String("类型", job.ServiceDiscoveryType), zap.String("任务名", job.Name))
			continue
		}

		scrapeConfigs = append(scrapeConfigs, sc)
	}

	return scrapeConfigs
}

// GetAlertManagerMainConfigYamlByIP 根据IP获取AlertManager主配置
func (mc *MonitorCache) GetAlertManagerMainConfigYamlByIP(ip string) string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.AlertManagerMainConfigMap[ip]
}

// GenerateAlertManagerMainConfig 生成并更新所有AlertManager主配置文件
func (mc *MonitorCache) GenerateAlertManagerMainConfig(ctx context.Context) {
	// 从数据库中获取所有AlertManager采集池
	pools, err := mc.dao.GetAllAlertManagerPools(ctx)
	if err != nil {
		mc.logger.Error("[监控模块]扫描数据库中的AlertManager集群失败", zap.Error(err))
		return
	}
	if len(pools) == 0 {
		mc.logger.Info("[监控模块]没有找到任何AlertManager采集池")
		return
	}

	mainConfigMap := make(map[string]string)

	for _, pool := range pools {
		// 生成单个AlertManager池的主配置
		allConfig := mc.GenerateAlertManagerMainConfigOnePool(pool)

		// 生成对应的routes和receivers配置
		routes, receivers := mc.GenerateAlertManagerRouteConfigOnePool(ctx, pool)
		if len(routes) > 0 {
			allConfig.Route.Routes = routes
		}

		if len(receivers) > 0 {
			if allConfig.Receivers == nil {
				allConfig.Receivers = receivers
			} else {
				allConfig.Receivers = append(receivers, allConfig.Receivers...)
			}
		}

		// 序列化配置为YAML格式
		out, err := yaml.Marshal(allConfig)
		if err != nil {
			mc.logger.Error("[监控模块]根据alert配置生成AlertManager主配置文件错误",
				zap.Error(err),
				zap.String("池子", pool.Name),
			)
			continue
		}

		mc.logger.Debug("[监控模块]根据alert配置生成AlertManager主配置文件成功",
			zap.String("池子", pool.Name),
			zap.ByteString("配置", out),
		)

		// 写入配置文件并更新缓存
		for index, ip := range pool.AlertManagerInstances {
			fileName := fmt.Sprintf("%s/alertmanager_pool_%s_%s_%d.yaml",
				mc.localYamlDir,
				pool.Name,
				ip,
				index,
			)
			if err := os.WriteFile(fileName, out, 0644); err != nil { // 使用更安全的文件权限
				mc.logger.Error("[监控模块]写入AlertManager配置文件失败",
					zap.Error(err),
					zap.String("文件路径", fileName),
				)
				continue
			}
			mainConfigMap[ip] = string(out)
		}
	}

	// 更新缓存
	mc.mu.Lock()
	mc.AlertManagerMainConfigMap = mainConfigMap
	mc.mu.Unlock()
}

// GenerateAlertManagerMainConfigOnePool 生成单个AlertManager池的主配置
func (mc *MonitorCache) GenerateAlertManagerMainConfigOnePool(pool *model.MonitorAlertManagerPool) *altconfig.Config {
	// 解析持续时间配置，并处理可能的错误
	resolveTimeout, err := pm.ParseDuration(pool.ResolveTimeout)
	if err != nil {
		mc.logger.Warn("[监控模块]解析ResolveTimeout失败，使用默认值",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		resolveTimeout = 0
	}

	groupWait, err := pm.ParseDuration(pool.GroupWait)
	if err != nil {
		mc.logger.Warn("[监控模块]解析GroupWait失败，使用默认值",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		groupWait = 0
	}

	groupInterval, err := pm.ParseDuration(pool.GroupInterval)
	if err != nil {
		mc.logger.Warn("[监控模块]解析GroupInterval失败，使用默认值",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		groupInterval = 0
	}

	repeatInterval, err := pm.ParseDuration(pool.RepeatInterval)
	if err != nil {
		mc.logger.Warn("[监控模块]解析RepeatInterval失败，使用默认值",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		repeatInterval = 0
	}

	config := &altconfig.Config{
		// 设置全局配置
		Global: &altconfig.GlobalConfig{
			ResolveTimeout: resolveTimeout,
		},
		// 设置默认路由
		Route: &altconfig.Route{
			Receiver:       pool.Receiver,
			GroupWait:      &groupWait,
			GroupInterval:  &groupInterval,
			RepeatInterval: &repeatInterval,
			GroupByStr:     pool.GroupBy,
		},
	}

	// 如果有默认Receiver，则添加到Receivers列表中
	if config.Route.Receiver != "" {
		config.Receivers = []altconfig.Receiver{
			{
				Name: config.Route.Receiver,
			},
		}
	}

	return config
}

// GenerateAlertManagerRouteConfigOnePool 生成单个AlertManager池的routes和receivers配置
func (mc *MonitorCache) GenerateAlertManagerRouteConfigOnePool(ctx context.Context, pool *model.MonitorAlertManagerPool) ([]*altconfig.Route, []altconfig.Receiver) {
	// 从数据库中查找该AlertManager池的所有发送组
	sendGroups, err := mc.dao.GetMonitorSendGroupByPoolId(ctx, pool.ID)
	if err != nil {
		mc.logger.Error("[监控模块]根据AlertManager池ID查找所有发送组错误",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		return nil, nil
	}
	if len(sendGroups) == 0 {
		mc.logger.Info("[监控模块]没有找到发送组", zap.String("池子", pool.Name))
		return nil, nil
	}

	var routes []*altconfig.Route
	var receivers []altconfig.Receiver

	for _, sendGroup := range sendGroups {
		// 解析RepeatInterval
		repeatInterval, err := pm.ParseDuration(sendGroup.RepeatInterval)
		if err != nil {
			mc.logger.Warn("[监控模块]解析RepeatInterval失败，使用默认值",
				zap.Error(err),
				zap.String("发送组", sendGroup.Name),
			)
			repeatInterval = 0
		}

		// 创建Matcher
		matcher, err := al.NewMatcher(al.MatchEqual, "alert_send_group", fmt.Sprintf("%d", sendGroup.ID))
		if err != nil {
			mc.logger.Error("[监控模块]创建Matcher失败",
				zap.Error(err),
				zap.String("发送组", sendGroup.Name),
			)
			continue
		}

		// 创建Route
		route := &altconfig.Route{
			Receiver:       sendGroup.Name,
			Continue:       true,
			Matchers:       []*al.Matcher{matcher},
			RepeatInterval: &repeatInterval,
			// 其他字段保持默认
		}

		// 拼接Webhook URL
		webHookURL := fmt.Sprintf("%s?%s=%d",
			mc.alertWebhookAddr,
			"alert_send_group",
			sendGroup.ID,
		)
		parsedURL, err := url.Parse(webHookURL)
		if err != nil {
			mc.logger.Error("[监控模块]解析Webhook URL失败",
				zap.Error(err),
				zap.String("Webhook URL", webHookURL),
				zap.String("发送组", sendGroup.Name),
			)
			continue
		}

		// 设置是否发送解决通知
		sendResolved := sendGroup.SendResolved == 1

		// 创建Receiver
		receiver := altconfig.Receiver{
			Name: sendGroup.Name,
			WebhookConfigs: []*altconfig.WebhookConfig{
				{
					NotifierConfig: altconfig.NotifierConfig{
						VSendResolved: sendResolved,
					},
					URL: &altconfig.SecretURL{URL: parsedURL},
				},
			},
		}

		// 添加到routes和receivers中
		routes = append(routes, route)
		receivers = append(receivers, receiver)
	}

	return routes, receivers
}

package cache

import (
	"context"
	"fmt"
	"github.com/prometheus/prometheus/model/rulefmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/prometheus"
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
)

const (
	// 临时哈希键，用于分片配置的哈希操作
	hashTmpKey        = "__tmp_hash"
	alertSendGroupKey = "alert_send_group"
)

// MonitorCache 管理监控相关的缓存数据和配置
type MonitorCache interface {
	// MonitorCacheManager 更新缓存
	MonitorCacheManager(ctx context.Context) error
	// GetPrometheusMainConfigByIP 根据IP地址获取Prometheus的主配置内容
	GetPrometheusMainConfigByIP(ip string) string
	// GetAlertManagerMainConfigYamlByIP 根据IP地址获取AlertManager的主配置内容
	GetAlertManagerMainConfigYamlByIP(ip string) string
	// GetPrometheusAlertRuleConfigYamlByIp 根据IP地址获取Prometheus的告警规则配置YAML
	GetPrometheusAlertRuleConfigYamlByIp(ip string) string
	// GetPrometheusRecordRuleConfigYamlByIp 根据IP地址获取Prometheus的预聚合规则配置YAML
	GetPrometheusRecordRuleConfigYamlByIp(ip string) string
	// GeneratePrometheusMainConfig 生成所有Prometheus主配置文件
	GeneratePrometheusMainConfig(ctx context.Context) error
	// GenerateAlertManagerMainConfig 生成所有AlertManager主配置文件
	GenerateAlertManagerMainConfig(ctx context.Context) error
	// GenerateAlertRuleConfigYaml 生成并更新所有Prometheus的告警规则配置YAML
	GenerateAlertRuleConfigYaml(ctx context.Context) error
	// GenerateRecordRuleConfigYaml 生成并更新所有Prometheus的预聚合规则配置YAML
	GenerateRecordRuleConfigYaml(ctx context.Context) error
}

// monitorCache 管理监控相关的缓存数据
type monitorCache struct {
	PrometheusMainConfigMap   map[string]string // 存储Prometheus主配置，键为IP地址
	AlertManagerMainConfigMap map[string]string // 存储AlertManager主配置
	AlertRuleMap              map[string]string // 存储告警规则
	RecordRuleMap             map[string]string // 存储预聚合规则
	mu                        sync.RWMutex      // 读写锁，保护缓存数据
	l                         *zap.Logger       // 日志记录器
	dao                       dao.PrometheusDao // Prometheus数据访问对象
	localYamlDir              string            // 本地YAML目录
	alertWebhookAddr          string            // Alertmanager Webhook地址
	httpSdAPI                 string            // HTTP服务发现API地址
}

func NewMonitorCache(l *zap.Logger, dao dao.PrometheusDao) MonitorCache {
	return &monitorCache{
		PrometheusMainConfigMap:   make(map[string]string),
		AlertManagerMainConfigMap: make(map[string]string),
		AlertRuleMap:              make(map[string]string),
		RecordRuleMap:             make(map[string]string),
		mu:                        sync.RWMutex{},
		l:                         l,
		dao:                       dao,
		localYamlDir:              viper.GetString("prometheus.local_yaml_dir"),
		alertWebhookAddr:          viper.GetString("prometheus.alert_webhook_addr"),
		httpSdAPI:                 viper.GetString("prometheus.httpSdAPI"),
	}
}

// MonitorCacheManager 定期更新缓存并监听退出信号
func (mc *monitorCache) MonitorCacheManager(ctx context.Context) error {
	mc.l.Info("开始更新所有监控缓存配置")

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(4)

	// 创建一个通道来收集错误
	errChan := make(chan error, 4)

	// 定义一个辅助函数来执行任务
	executeTask := func(taskName string, taskFunc func(context.Context) error) {
		defer wg.Done()
		mc.l.Info(fmt.Sprintf("开始执行任务: %s", taskName))
		if err := taskFunc(ctx); err != nil {
			mc.l.Error(fmt.Sprintf("任务 %s 失败", taskName), zap.Error(err))
			errChan <- fmt.Errorf("%s: %w", taskName, err)
			return
		}
		mc.l.Info(fmt.Sprintf("任务 %s 成功完成", taskName))
	}

	// 并发执行各个配置生成任务
	go executeTask("生成 Prometheus 主配置", mc.GeneratePrometheusMainConfig)
	go executeTask("生成 AlertManager 主配置", mc.GenerateAlertManagerMainConfig)
	go executeTask("生成 Prometheus 告警规则配置", mc.GenerateAlertRuleConfigYaml)
	go executeTask("生成 Prometheus 预聚合规则配置", mc.GenerateRecordRuleConfigYaml)

	wg.Wait()
	close(errChan)

	// 收集所有错误
	var aggregatedErrors []error
	for err := range errChan {
		aggregatedErrors = append(aggregatedErrors, err)
	}

	if len(aggregatedErrors) > 0 {
		mc.l.Warn("部分任务执行失败，详情请查看日志")
		return fmt.Errorf("部分任务执行失败: %v", aggregatedErrors)
	}

	mc.l.Info("所有监控缓存配置更新完成")
	return nil
}

// GetPrometheusMainConfigByIP 根据IP获取Prometheus主配置
func (mc *monitorCache) GetPrometheusMainConfigByIP(ip string) string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.PrometheusMainConfigMap[ip]
}

// GeneratePrometheusMainConfig 生成所有Prometheus主配置文件
func (mc *monitorCache) GeneratePrometheusMainConfig(ctx context.Context) error {
	// 获取所有采集池
	pools, err := mc.dao.GetAllMonitorScrapePool(ctx)
	if err != nil {
		mc.l.Error("获取采集池失败", zap.Error(err))
		return err
	}

	if len(pools) == 0 {
		mc.l.Info("没有找到任何采集池")
		return nil
	}

	// 创建新的配置映射key为ip，val为配置
	newConfigMap := make(map[string]string)

	for _, pool := range pools {
		// 创建基础配置
		baseConfig, err := mc.CreateBasePrometheusConfig(pool)
		if err != nil {
			mc.l.Error("创建基础 Prometheus 配置失败", zap.Error(err), zap.String("池名", pool.Name))
			continue
		}

		// 生成采集配置
		scrapeConfigs := mc.GenerateScrapeConfigs(ctx, pool)
		if len(scrapeConfigs) == 0 {
			mc.l.Warn("没有找到任何采集任务", zap.String("池名", pool.Name))
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
				mc.l.Error("生成 Prometheus 配置失败", zap.Error(err), zap.String("池名", pool.Name))
				continue
			}

			// 写入配置文件
			filePath := fmt.Sprintf("%s/prometheus_pool_%s.yaml", mc.localYamlDir, ip)

			// 创建目录
			dir := filepath.Dir(filePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				mc.l.Error("创建目录失败", zap.Error(err), zap.String("目录路径", dir))
				continue
			}

			if err := os.WriteFile(filePath, yamlData, 0644); err != nil {
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

	return nil
}

// CreateBasePrometheusConfig 创建基础Prometheus配置，返回错误
func (mc *monitorCache) CreateBasePrometheusConfig(pool *model.MonitorScrapePool) (pc.Config, error) {
	// 创建prometheus global全局配置
	globalConfig := pc.GlobalConfig{
		ScrapeInterval: pkg.GenPromDuration(pool.ScrapeInterval), // 采集间隔
		ScrapeTimeout:  pkg.GenPromDuration(pool.ScrapeTimeout),  // 采集超时时间
	}

	// 解析外部标签
	externalLabels := pkg.ParseExternalLabels(pool.ExternalLabels)
	if len(externalLabels) > 0 {
		globalConfig.ExternalLabels = labels.FromStrings(externalLabels...)
	}

	// 解析 RemoteWrite URL
	remoteWriteURL, err := pkg.ParseURL(pool.RemoteWriteUrl)
	if err != nil {
		mc.l.Error("解析 RemoteWriteUrl 失败", zap.Error(err))
		return pc.Config{}, fmt.Errorf("解析 RemoteWriteUrl 失败: %w", err)
	}

	// 配置远程写入
	remoteWrite := &pc.RemoteWriteConfig{
		URL:           remoteWriteURL,
		RemoteTimeout: pkg.GenPromDuration(pool.RemoteTimeoutSeconds),
	}

	// 组装prometheus基础配置
	config := pc.Config{
		GlobalConfig:       globalConfig,
		RemoteWriteConfigs: []*pc.RemoteWriteConfig{remoteWrite},
	}

	if pool.SupportAlert == 1 { // 启用告警
		// 解析 RemoteRead URL
		remoteReadURL, err := pkg.ParseURL(pool.RemoteReadUrl)
		if err != nil {
			mc.l.Error("解析 RemoteReadUrl 失败", zap.Error(err))
			return pc.Config{}, fmt.Errorf("解析 RemoteReadUrl 失败: %w", err)
		}

		// 配置远程读取
		config.RemoteReadConfigs = []*pc.RemoteReadConfig{
			{
				URL:           remoteReadURL,
				RemoteTimeout: pkg.GenPromDuration(pool.RemoteTimeoutSeconds),
			},
		}

		// 配置 Alertmanager
		alertConfig := &pc.AlertmanagerConfig{
			APIVersion: "v2",
			ServiceDiscoveryConfigs: discovery.Configs{ // 服务发现配置
				&discovery.StaticConfig{
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

		// 添加告警规则文件
		config.RuleFiles = append(config.RuleFiles, pool.RuleFilePath)
	}

	if pool.SupportRecord == 1 { // 启用预聚合
		// 添加预聚合规则文件
		config.RuleFiles = append(config.RuleFiles, pool.RecordFilePath)
	}

	return config, nil
}

// ApplyHashMod 应用HashMod和Keep Relabel配置进行分片
func (mc *monitorCache) ApplyHashMod(scrapeConfigs []*pc.ScrapeConfig, modNum, index int) []*pc.ScrapeConfig {
	var modified []*pc.ScrapeConfig

	for _, sc := range scrapeConfigs {
		// 深度拷贝 ScrapeConfig
		copySc := pkg.DeepCopyScrapeConfig(sc)
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

// GenerateScrapeConfigs 生成采集配置
func (mc *monitorCache) GenerateScrapeConfigs(ctx context.Context, pool *model.MonitorScrapePool) []*pc.ScrapeConfig {
	// 获取与指定池相关的采集任务
	scrapeJobs, err := mc.dao.GetMonitorScrapeJobsByPoolId(ctx, pool.ID)
	if err != nil {
		mc.l.Error("获取采集任务失败", zap.Error(err), zap.String("池名", pool.Name))
		return nil
	}
	if len(scrapeJobs) == 0 {
		mc.l.Info("没有找到任何采集任务", zap.String("池名", pool.Name))
		return nil
	}

	var scrapeConfigs []*pc.ScrapeConfig

	for _, job := range scrapeJobs {
		sc := &pc.ScrapeConfig{
			JobName:        job.Name,
			Scheme:         job.Scheme,
			MetricsPath:    job.MetricsPath,
			ScrapeInterval: pkg.GenPromDuration(job.ScrapeInterval),
			ScrapeTimeout:  pkg.GenPromDuration(job.ScrapeTimeout),
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
			if err != nil {
				mc.l.Error("获取 HTTP SD API 失败", zap.Error(err), zap.String("任务名", job.Name))
				continue
			}

			// 拼接 SD API URL
			sdURL := fmt.Sprintf("%s?port=%d&leafNodeIds=%s", mc.httpSdAPI, job.Port, strings.Join(job.TreeNodeIDs, ","))

			sc.ServiceDiscoveryConfigs = discovery.Configs{
				&http.SDConfig{
					URL:             sdURL,
					RefreshInterval: pkg.GenPromDuration(job.RefreshInterval),
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
			mc.l.Warn("未知的服务发现类型", zap.String("类型", job.ServiceDiscoveryType), zap.String("任务名", job.Name))
			continue
		}

		scrapeConfigs = append(scrapeConfigs, sc)
	}

	return scrapeConfigs
}

// GetAlertManagerMainConfigYamlByIP 根据IP获取AlertManager主配置
func (mc *monitorCache) GetAlertManagerMainConfigYamlByIP(ip string) string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.AlertManagerMainConfigMap[ip]
}

// GenerateAlertManagerMainConfig 生成并更新所有AlertManager主配置文件
func (mc *monitorCache) GenerateAlertManagerMainConfig(ctx context.Context) error {
	// 从数据库中获取所有AlertManager采集池
	pools, err := mc.dao.GetAllAlertManagerPools(ctx)
	if err != nil {
		mc.l.Error("[监控模块]扫描数据库中的AlertManager集群失败", zap.Error(err))
		return err
	}

	if len(pools) == 0 {
		mc.l.Info("[监控模块]没有找到任何AlertManager采集池")
		return err
	}

	mainConfigMap := make(map[string]string)

	for _, pool := range pools {
		// 生成单个AlertManager池的主配置
		oneConfig := mc.GenerateAlertManagerMainConfigOnePool(pool)

		// 生成对应的routes和receivers配置
		routes, receivers := mc.GenerateAlertManagerRouteConfigOnePool(ctx, pool)
		if len(routes) > 0 {
			oneConfig.Route.Routes = routes
		}

		if len(receivers) > 0 {
			if oneConfig.Receivers == nil {
				oneConfig.Receivers = receivers
			} else {
				oneConfig.Receivers = append(receivers, oneConfig.Receivers...)
			}
		}

		// 序列化配置为YAML格式
		config, err := yaml.Marshal(oneConfig)
		if err != nil {
			mc.l.Error("[监控模块]根据alert配置生成AlertManager主配置文件错误",
				zap.Error(err),
				zap.String("池子", pool.Name),
			)
			continue
		}

		mc.l.Debug("[监控模块]根据alert配置生成AlertManager主配置文件成功",
			zap.String("池子", pool.Name),
			zap.ByteString("配置", config),
		)

		// 写入配置文件并更新缓存
		for index, ip := range pool.AlertManagerInstances {
			fileName := fmt.Sprintf("%s/alertmanager_pool_%s_%s_%d.yaml",
				mc.localYamlDir,
				pool.Name,
				ip,
				index,
			)

			if err := os.WriteFile(fileName, config, 0644); err != nil {
				mc.l.Error("[监控模块]写入AlertManager配置文件失败",
					zap.Error(err),
					zap.String("文件路径", fileName),
				)
				continue
			}

			// 配置存入map中
			mainConfigMap[ip] = string(config)
		}
	}

	mc.mu.Lock()
	mc.AlertManagerMainConfigMap = mainConfigMap
	mc.mu.Unlock()

	return nil
}

// GenerateAlertManagerMainConfigOnePool 生成单个AlertManager池的主配置
func (mc *monitorCache) GenerateAlertManagerMainConfigOnePool(pool *model.MonitorAlertManagerPool) *altconfig.Config {
	// 解析默认恢复时间
	resolveTimeout, err := pm.ParseDuration(pool.ResolveTimeout)
	if err != nil {
		mc.l.Warn("[监控模块]解析ResolveTimeout失败，使用默认值",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		resolveTimeout = 5
	}

	// 解析分组第一次等待时间
	groupWait, err := pm.ParseDuration(pool.GroupWait)
	if err != nil {
		mc.l.Warn("[监控模块]解析GroupWait失败，使用默认值",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		groupWait = 5
	}

	// 解析分组等待间隔时间
	groupInterval, err := pm.ParseDuration(pool.GroupInterval)
	if err != nil {
		mc.l.Warn("[监控模块]解析GroupInterval失败，使用默认值",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		groupInterval = 5
	}

	// 解析重复发送时间
	repeatInterval, err := pm.ParseDuration(pool.RepeatInterval)
	if err != nil {
		mc.l.Warn("[监控模块]解析RepeatInterval失败，使用默认值",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		repeatInterval = 5
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

	// 如果有默认rs列表中Receiver，则添加到Receive
	if config.Route.Receiver != "" {
		config.Receivers = []altconfig.Receiver{
			{
				Name: config.Route.Receiver, // 接收者名称
			},
		}
	}

	return config
}

// GenerateAlertManagerRouteConfigOnePool 生成单个AlertManager池的routes和receivers配置
func (mc *monitorCache) GenerateAlertManagerRouteConfigOnePool(ctx context.Context, pool *model.MonitorAlertManagerPool) ([]*altconfig.Route, []altconfig.Receiver) {
	// 从数据库中查找该AlertManager池的所有发送组
	sendGroups, err := mc.dao.GetMonitorSendGroupByPoolId(ctx, pool.ID)
	if err != nil {
		mc.l.Error("[监控模块]根据AlertManager池ID查找所有发送组错误",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		return nil, nil
	}
	if len(sendGroups) == 0 {
		mc.l.Info("[监控模块]没有找到发送组", zap.String("池子", pool.Name))
		return nil, nil
	}

	var routes []*altconfig.Route
	var receivers []altconfig.Receiver

	for _, sendGroup := range sendGroups {
		// 解析RepeatInterval
		repeatInterval, err := pm.ParseDuration(sendGroup.RepeatInterval)
		if err != nil {
			mc.l.Warn("[监控模块]解析RepeatInterval失败，使用默认值",
				zap.Error(err),
				zap.String("发送组", sendGroup.Name),
			)
			repeatInterval = 5
		}

		// 创建 Matcher 并设置匹配条件
		// 默认匹配条件为: alert_send_group=sendGroup.ID
		matcher, err := al.NewMatcher(al.MatchEqual, alertSendGroupKey, fmt.Sprintf("%d", sendGroup.ID))
		if err != nil {
			mc.l.Error("[监控模块]创建Matcher失败",
				zap.Error(err),
				zap.String("发送组", sendGroup.Name),
			)
			continue
		}

		// 创建Route
		route := &altconfig.Route{
			Receiver:       sendGroup.Name,         // 设置接收者
			Continue:       true,                   // 继续匹配下一个路由
			Matchers:       []*al.Matcher{matcher}, // 设置匹配条件
			RepeatInterval: &repeatInterval,        // 设置重复发送时间
		}

		// 拼接Webhook URL
		webHookURL := fmt.Sprintf("%s?%s=%d",
			mc.alertWebhookAddr,
			alertSendGroupKey,
			sendGroup.ID,
		)

		parsedURL, err := url.Parse(webHookURL)
		if err != nil {
			mc.l.Error("[监控模块]解析Webhook URL失败",
				zap.Error(err),
				zap.String("Webhook URL", webHookURL),
				zap.String("发送组", sendGroup.Name),
			)
			continue
		}

		// 创建Receiver
		receiver := altconfig.Receiver{
			Name: sendGroup.Name, // 接收者名称
			WebhookConfigs: []*altconfig.WebhookConfig{ // Webhook配置
				{
					NotifierConfig: altconfig.NotifierConfig{ // Notifier配置 用于告警通知
						VSendResolved: sendGroup.SendResolved == 1, // 在告警解决时是否发送通知
					},
					URL: &altconfig.SecretURL{URL: parsedURL}, // 告警发送的URL地址
				},
			},
		}

		// 添加到routes和receivers中
		routes = append(routes, route)
		receivers = append(receivers, receiver)
	}

	return routes, receivers
}

// RuleGroup 构造Prometheus Rule 规则的结构体
type RuleGroup struct {
	Name  string         `yaml:"name"`
	Rules []rulefmt.Rule `yaml:"rules"`
}

// RuleGroups 生成Prometheus rule yaml
type RuleGroups struct {
	Groups []RuleGroup `yaml:"groups"`
}

// GetPrometheusAlertRuleConfigYamlByIp 根据IP获取Prometheus的告警规则配置YAML
func (mc *monitorCache) GetPrometheusAlertRuleConfigYamlByIp(ip string) string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return mc.AlertRuleMap[ip]
}

// GenerateAlertRuleConfigYaml 生成并更新所有Prometheus的告警规则配置YAML
func (mc *monitorCache) GenerateAlertRuleConfigYaml(ctx context.Context) error {
	// 获取支持告警配置的所有采集池
	pools, err := mc.dao.GetMonitorScrapePoolSupportedAlert(ctx)
	if err != nil {
		mc.l.Error("[监控模块] 获取支持告警的采集池失败", zap.Error(err))
		return err
	}

	if len(pools) == 0 {
		mc.l.Info("没有找到支持告警的采集池")
		return nil
	}

	ruleConfigMap := make(map[string]string)

	// 遍历每个采集池生成对应的规则配置
	for _, pool := range pools {
		oneMap := mc.GeneratePrometheusAlertRuleConfigYamlOnePool(ctx, pool)
		if oneMap != nil {
			for ip, out := range oneMap {
				ruleConfigMap[ip] = out
			}
		}
	}

	mc.mu.Lock()
	mc.AlertRuleMap = ruleConfigMap
	mc.mu.Unlock()

	return nil
}

// GeneratePrometheusAlertRuleConfigYamlOnePool 根据单个采集池生成Prometheus的告警规则配置YAML
func (mc *monitorCache) GeneratePrometheusAlertRuleConfigYamlOnePool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string {
	rules, err := mc.dao.GetMonitorAlertRuleByPoolId(ctx, pool.ID)
	if err != nil {
		mc.l.Error("[监控模块] 根据采集池ID获取告警规则失败",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)
		return nil
	}
	if len(rules) == 0 {
		return nil
	}

	var ruleGroups RuleGroups

	// 构建规则组
	for _, rule := range rules {
		ft, err := pm.ParseDuration(rule.ForTime)
		if err != nil {
			mc.l.Warn("[监控模块] 解析告警规则持续时间失败，使用默认值",
				zap.Error(err),
				zap.String("规则", rule.Name),
			)
			ft = 15
		}
		oneRule := rulefmt.Rule{
			Alert:       rule.Name,         // 告警名称
			Expr:        rule.Expr,         // 告警表达式
			For:         ft,                // 持续时间
			Labels:      rule.LabelsM,      // 标签组
			Annotations: rule.AnnotationsM, // 注解组
		}

		ruleGroup := RuleGroup{
			Name:  rule.Name,
			Rules: []rulefmt.Rule{oneRule}, // 一个规则组可以包含多个规则
		}
		ruleGroups.Groups = append(ruleGroups.Groups, ruleGroup)
	}

	numInstances := len(pool.PrometheusInstances)
	if numInstances == 0 {
		mc.l.Warn("[监控模块] 采集池中没有Prometheus实例", zap.String("池子", pool.Name))
		return nil
	}

	ruleMap := make(map[string]string)

	// 分片逻辑，将规则分配给不同的Prometheus实例，以减少服务器的负载
	for i, ip := range pool.PrometheusInstances {
		var myRuleGroups RuleGroups

		for j, group := range ruleGroups.Groups {
			if j%numInstances == i { // 按顺序平均分片
				myRuleGroups.Groups = append(myRuleGroups.Groups, group)
			}
		}

		// 序列化规则组为YAML
		yamlData, err := yaml.Marshal(&myRuleGroups)
		if err != nil {
			mc.l.Error("[监控模块] 序列化告警规则YAML失败",
				zap.Error(err),
				zap.String("池子", pool.Name),
				zap.String("IP", ip),
			)
			continue
		}

		fileName := fmt.Sprintf("%s/prometheus_rule_%s_%s.yml",
			mc.localYamlDir,
			pool.Name,
			ip,
		)
		if err := os.WriteFile(fileName, yamlData, 0644); err != nil {
			mc.l.Error("[监控模块] 写入告警规则文件失败",
				zap.Error(err),
				zap.String("文件路径", fileName),
			)
			continue
		}

		ruleMap[ip] = string(yamlData)
	}

	return ruleMap
}

// GetPrometheusRecordRuleConfigYamlByIp 根据IP获取Prometheus的预聚合规则配置YAML
func (mc *monitorCache) GetPrometheusRecordRuleConfigYamlByIp(ip string) string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return mc.RecordRuleMap[ip]
}

// GenerateRecordRuleConfigYaml 生成并更新所有Prometheus的预聚合规则配置YAML
func (mc *monitorCache) GenerateRecordRuleConfigYaml(ctx context.Context) error {
	// 获取支持预聚合配置的所有采集池
	pools, err := mc.dao.GetMonitorScrapePoolSupportedRecord(ctx)
	if err != nil {
		mc.l.Error("[监控模块] 获取支持预聚合的采集池失败", zap.Error(err))
		return err
	}

	if len(pools) == 0 {
		mc.l.Info("没有找到支持预聚合的采集池")
		return nil
	}

	ruleConfigMap := make(map[string]string)

	// 遍历每个采集池生成对应的预聚合规则配置
	for _, pool := range pools {
		oneMap := mc.GeneratePrometheusRecordRuleConfigYamlOnePool(ctx, pool)
		if oneMap != nil {
			for ip, out := range oneMap {
				ruleConfigMap[ip] = out
			}
		}
	}

	mc.mu.Lock()
	mc.RecordRuleMap = ruleConfigMap
	mc.mu.Unlock()

	return nil
}

// GeneratePrometheusRecordRuleConfigYamlOnePool 根据单个采集池生成Prometheus的预聚合规则配置YAML
func (mc *monitorCache) GeneratePrometheusRecordRuleConfigYamlOnePool(ctx context.Context, pool *model.MonitorScrapePool) map[string]string {
	rules, err := mc.dao.GetMonitorRecordRuleByPoolId(ctx, pool.ID)
	if err != nil {
		mc.l.Error("[监控模块] 根据采集池ID获取预聚合规则失败",
			zap.Error(err),
			zap.String("池子", pool.Name),
		)

		return nil
	}

	if len(rules) == 0 {
		return nil
	}

	var ruleGroups RuleGroups

	// 构建规则组
	for _, rule := range rules {
		forD, err := pm.ParseDuration(rule.ForTime)
		if err != nil {
			mc.l.Warn("[监控模块] 解析预聚合规则持续时间失败，使用默认值",
				zap.Error(err),
				zap.String("规则", rule.Name),
			)
			forD = 15
		}
		oneRule := rulefmt.Rule{
			Alert: rule.Name, // 告警名称
			Expr:  rule.Expr, // 预聚合表达式
			For:   forD,      // 持续时间
		}

		ruleGroup := RuleGroup{
			Name:  rule.Name,
			Rules: []rulefmt.Rule{oneRule},
		}
		ruleGroups.Groups = append(ruleGroups.Groups, ruleGroup)
	}

	numInstances := len(pool.PrometheusInstances)
	if numInstances == 0 {
		mc.l.Warn("[监控模块] 采集池中没有Prometheus实例", zap.String("池子", pool.Name))
		return nil
	}

	ruleMap := make(map[string]string)

	// 分片逻辑，将规则分配给不同的Prometheus实例
	for i, ip := range pool.PrometheusInstances {
		var myRuleGroups RuleGroups
		for j, group := range ruleGroups.Groups {
			if j%numInstances == i { // 按顺序平均分片
				myRuleGroups.Groups = append(myRuleGroups.Groups, group)
			}
		}

		yamlData, err := yaml.Marshal(&myRuleGroups)
		if err != nil {
			mc.l.Error("[监控模块] 序列化预聚合规则YAML失败",
				zap.Error(err),
				zap.String("池子", pool.Name),
				zap.String("IP", ip),
			)
			continue
		}
		fileName := fmt.Sprintf("%s/prometheus_rule_%s_%s.yml",
			mc.localYamlDir,
			pool.Name,
			ip,
		)

		if err := os.WriteFile(fileName, yamlData, 0644); err != nil {
			mc.l.Error("[监控模块] 写入预聚合规则文件失败",
				zap.Error(err),
				zap.String("文件路径", fileName),
			)
			continue
		}

		ruleMap[ip] = string(yamlData)
	}

	return ruleMap
}

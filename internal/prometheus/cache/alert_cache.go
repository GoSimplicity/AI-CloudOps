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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertPoolDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	altconfig "github.com/prometheus/alertmanager/config"
	al "github.com/prometheus/alertmanager/pkg/labels"
	pm "github.com/prometheus/common/model"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const alertSendGroupKey = "alert_send_group"

// AlertManagerConfigCache AlertManager配置缓存接口
type AlertManagerConfigCache interface {
	RetireDeletedAlertManagerPool(context.Context, int) error
	ReconcileDeletedAlertManagerPools(context.Context) error
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
	webhookFileDir   string
	serverPort       string
	alertPoolDAO     alertPoolDao.AlertManagerPoolDAO
	alertSendDAO     alertPoolDao.AlertManagerSendDAO
	configDAO        configDao.MonitorConfigDAO
	batchManager     *BatchConfigManager
	journal          configDao.ProjectionPlanJournalDAO
	history          configDao.AMProjectionHistoryReader
	projection       AMProjectionRunner
}

func NewAlertManagerConfigCache(
	cfg *config.Config,
	logger *zap.Logger,
	alertPoolDAO alertPoolDao.AlertManagerPoolDAO,
	alertSendDAO alertPoolDao.AlertManagerSendDAO,
	configDAO configDao.MonitorConfigDAO,
	batchManager *BatchConfigManager,
	redisClient redis.Cmdable,
	journal configDao.ProjectionPlanJournalDAO,
	history configDao.AMProjectionHistoryReader,
	projection AMProjectionRunner,
) (AlertManagerConfigCache, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置不能为空")
	}
	return &alertManagerConfigCache{
		configMap:        make(map[string]string),
		logger:           logger,
		redis:            redisClient,
		alertWebhookAddr: cfg.Prometheus.AlertWebhookAddr,
		webhookFileDir:   cfg.Prometheus.AlertWebhookFileDir,
		serverPort:       cfg.Server.Port,
		mu:               sync.RWMutex{},
		alertPoolDAO:     alertPoolDAO,
		alertSendDAO:     alertSendDAO,
		configDAO:        configDAO,
		batchManager:     batchManager,
		journal:          journal, history: history, projection: projection,
	}, nil
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
		a.logger.Debug(LogModuleMonitor + "AlertManager缓存未命中")
		return ""
	}
	a.logger.Debug(LogModuleMonitor + "AlertManager缓存命中")
	return val
}

// GenerateMainConfig 生成所有AlertManager主配置文件
func (a *alertManagerConfigCache) GenerateMainConfig(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	versions, err := a.amSourceVersions(ctx)
	if err != nil {
		return err
	}
	pools, err := readAlertPoolSnapshot(ctx, a.alertPoolDAO, false)
	if err != nil {
		return err
	}
	failures := []error{}
	for _, pool := range pools {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		if err := a.publishAlertPool(ctx, pool, amSourceVersionFor(versions, pool.ID)); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(append(failures, ctx.Err())...)
}

// Publication identity is the database ID, never a mutable pool name. Keep this
// separate from the legacy name-key namespace (even a name such as "pool:7").
func alertPublicationStateKey(poolID int) string {
	return fmt.Sprintf("%s:publication:alertmanager:pool:%d", redisPrefix, poolID)
}

// A publication is acknowledged only after every required artifact is durable.
func (a *alertManagerConfigCache) alertPublicationError(poolID int, stage string) error {
	a.logger.Error(LogModuleMonitor+"AlertManager配置发布未完成", zap.Int("pool_id", poolID), zap.String("stage", stage))
	return fmt.Errorf("AlertManager配置发布未完成，池%d，阶段%s", poolID, stage)
}

func (a *alertManagerConfigCache) readEnabledAlertGroups(ctx context.Context, poolID int) ([]*model.MonitorSendGroup, error) {
	rows, _, err := a.alertSendDAO.GetMonitorSendGroupByPoolID(ctx, poolID)
	if err != nil {
		return nil, errors.New("发送组读取失败")
	}
	groups := make([]*model.MonitorSendGroup, 0, len(rows))
	seen := map[int]bool{}
	for _, row := range rows {
		if row == nil {
			return nil, errors.New("发送组快照无效")
		}
		if row.Enable != 1 {
			continue
		}
		if row.ID <= 0 || seen[row.ID] {
			return nil, errors.New("发送组标识无效")
		}
		seen[row.ID] = true
		copy := *row
		groups = append(groups, &copy)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].ID < groups[j].ID })
	return groups, nil
}

func (a *alertManagerConfigCache) renderAlertRoutes(pool *model.MonitorAlertManagerPool, groups []*model.MonitorSendGroup) ([]*altconfig.Route, []altconfig.Receiver, error) {
	routes := make([]*altconfig.Route, 0, len(groups))
	receivers := make([]altconfig.Receiver, 0, len(groups))
	for _, group := range groups {
		repeat, err := pm.ParseDuration(group.RepeatInterval)
		if err != nil {
			a.logger.Warn(LogModuleMonitor+"解析RepeatInterval失败，使用默认值1h", zap.Int("send_group_id", group.ID))
			repeat, _ = pm.ParseDuration("1h")
		}
		matcher, err := al.NewMatcher(al.MatchEqual, alertSendGroupKey, strconv.Itoa(group.ID))
		if err != nil {
			return nil, nil, errors.New("路由条件无效")
		}
		routes = append(routes, &altconfig.Route{Receiver: group.Name, Continue: false, Matchers: []*al.Matcher{matcher}, RepeatInterval: &repeat})
		receivers = append(receivers, altconfig.Receiver{Name: group.Name, WebhookConfigs: []*altconfig.WebhookConfig{{NotifierConfig: altconfig.NotifierConfig{VSendResolved: group.SendResolved == 1}, URLFile: fmt.Sprintf("%s/webhook_%s_%d.yml", a.webhookFileDir, pool.Name, group.ID)}}})
	}
	return routes, receivers, nil
}

func (a *alertManagerConfigCache) alertWebhookConfigs(pool *model.MonitorAlertManagerPool, groups []*model.MonitorSendGroup) map[string]ConfigData {
	rows := map[string]ConfigData{}
	for _, group := range groups {
		for _, ip := range pool.AlertManagerInstances {
			key := fmt.Sprintf("%s_%d", ip, group.ID)
			rows[key] = ConfigData{Name: fmt.Sprintf("webhook_%s_%d_%s", pool.Name, group.ID, ip), PoolID: pool.ID, ConfigType: model.ConfigTypeWebhookFile, Content: a.generateWebhookFileContent(group)}
		}
	}
	return rows
}

func (a *alertManagerConfigCache) alertConfigDAO() configDao.MonitorConfigDAO {
	if a.configDAO != nil {
		return a.configDAO
	}
	if a.batchManager != nil {
		return a.batchManager.configDAO
	}
	return nil
}

type retiredAlertWebhook struct {
	row     *model.MonitorConfig
	ip      string
	groupID int
}

func ownedAlertWebhook(row *model.MonitorConfig, poolID int) (retiredAlertWebhook, bool) {
	if row == nil || row.PoolID != poolID || row.ConfigType != model.ConfigTypeWebhookFile || row.ID <= 0 {
		return retiredAlertWebhook{}, false
	}
	split := strings.LastIndex(row.InstanceIP, "_")
	if split <= 0 {
		return retiredAlertWebhook{}, false
	}
	ip := row.InstanceIP[:split]
	id, err := strconv.Atoi(row.InstanceIP[split+1:])
	if err != nil || id <= 0 || row.InstanceIP != fmt.Sprintf("%s_%d", ip, id) {
		return retiredAlertWebhook{}, false
	}
	suffix := fmt.Sprintf("_%d_%s", id, ip)
	if !strings.HasPrefix(row.Name, "webhook_") || !strings.HasSuffix(row.Name, suffix) {
		return retiredAlertWebhook{}, false
	}
	return retiredAlertWebhook{row: row, ip: ip, groupID: id}, true
}

// Discover canonical DB-only artifacts as well as indexed targets. Complete
// the metadata scan before any writes; counters detect drift, not SQL snapshot
// isolation. A mutable pool name is never evidence of main-file ownership.
func (a *alertManagerConfigCache) checkRetiredAlertMainOwnership(ctx context.Context, poolID int, oldIPs []string, desired map[string]bool) (map[string]configDao.GeneratedConfigOwner, []string, error) {
	fail := func() (map[string]configDao.GeneratedConfigOwner, []string, error) {
		return nil, nil, errors.New("旧主配置快照或归属无效")
	}
	stale := map[string]bool{}
	for _, ip := range oldIPs {
		if err := validateInstanceIPs([]string{ip}); err != nil {
			return fail()
		}
		if !desired[ip] {
			stale[ip] = true
		}
	}
	rowsByIP := map[string][]*model.MonitorConfig{}
	kind := model.ConfigTypeAlertManager
	rows, err := a.readAlertConfigSnapshot(ctx, kind)
	if err != nil {
		return nil, nil, err
	}
	for _, row := range rows {
		rowsByIP[row.InstanceIP] = append(rowsByIP[row.InstanceIP], row)
		if row.PoolID == poolID && row.Name == fmt.Sprintf(ConfigNameAlertManager, poolID, row.InstanceIP) && !desired[row.InstanceIP] {
			if err := validateInstanceIPs([]string{row.InstanceIP}); err != nil {
				return fail()
			}
			stale[row.InstanceIP] = true
		}
	}
	owners := map[string]configDao.GeneratedConfigOwner{}
	// Current target conflicts must also fail before pending or any unrelated
	// artifact writes, even when the old membership index was missing.
	for ip := range desired {
		rows := rowsByIP[ip]
		if len(rows) > 1 {
			return fail()
		}
		if len(rows) == 1 {
			row := rows[0]
			if row.PoolID != poolID || row.Name != fmt.Sprintf(ConfigNameAlertManager, poolID, ip) {
				return fail()
			}
		}
	}
	ips := make([]string, 0, len(stale))
	for ip := range stale {
		rows := rowsByIP[ip]
		if len(rows) > 1 {
			return fail()
		}
		if len(rows) == 1 {
			row := rows[0]
			if row.PoolID != poolID || row.Name != fmt.Sprintf(ConfigNameAlertManager, poolID, ip) {
				return fail()
			}
			owners[ip] = configDao.GeneratedConfigOwnerOf(row)
		}
		ips = append(ips, ip)
	}
	sort.Strings(ips)
	return owners, ips, nil
}

func (a *alertManagerConfigCache) publishAlertPool(ctx context.Context, pool *model.MonitorAlertManagerPool, version configDao.AMSourceVersion) error {
	fail := func(stage string) error { return a.alertPublicationError(pool.ID, stage) }
	var groups []*model.MonitorSendGroup
	var content []byte
	var instances []string
	var err error
	webhooks := map[string]ConfigData{}
	if pool.DeletedAt == 0 {
		if len(pool.AlertManagerInstances) > 0 {
			if err := validateInstanceIPs(pool.AlertManagerInstances); err != nil {
				return fail("实例列表无效")
			}
		}
		if strings.TrimSpace(pool.Name) == "" {
			return fail("实例池名称无效")
		}
		groups, err = a.readEnabledAlertGroups(ctx, pool.ID)
		if err != nil {
			return fail("读取发送组")
		}
		routes, receivers, err := a.renderAlertRoutes(pool, groups)
		if err != nil {
			return fail("生成路由")
		}
		cfg := a.GenerateMainConfigForPool(pool)
		if len(routes) == 0 {
			cfg.Route.Receiver = "cloudops-disabled"
			cfg.Receivers = []altconfig.Receiver{{Name: "cloudops-disabled"}}
		} else {
			cfg.Route.Receiver = routes[0].Receiver
			for _, receiver := range receivers {
				if receiver.Name == pool.Receiver {
					cfg.Route.Receiver = pool.Receiver
					break
				}
			}
			cfg.Route.Routes = routes
			cfg.Receivers = receivers
		}
		content, err = yaml.Marshal(cfg)
		if err != nil {
			return fail("序列化配置")
		}
		if _, err = altconfig.Load(string(content)); err != nil {
			return fail("验证配置")
		}
		webhooks = a.alertWebhookConfigs(pool, groups)
		instances = append([]string(nil), pool.AlertManagerInstances...)
		sort.Strings(instances)
	}
	payload, err := json.Marshal(struct {
		Main      string
		Webhooks  map[string]ConfigData
		Instances []string
	}{string(content), webhooks, instances})
	if err != nil {
		return fail("生成版本")
	}
	hash := calculateConfigHash(string(payload))
	if pool.DeletedAt > 0 {
		hash = calculateConfigHash(fmt.Sprintf("deleted:alertmanager:%d:%d", pool.ID, pool.DeletedAt))
	}
	decoded, err := hex.DecodeString(hash)
	if err != nil || len(decoded) != 32 {
		return fail("版本指纹无效")
	}
	var sourceHash [32]byte
	copy(sourceHash[:], decoded)
	mains := map[string]ConfigData{}
	for _, ip := range instances {
		mains[ip] = ConfigData{Name: fmt.Sprintf(ConfigNameAlertManager, pool.ID, ip), PoolID: pool.ID, ConfigType: 2, Content: string(content)}
	}
	if err := a.amProject(ctx, 2, pool.ID, map[int8]map[string]ConfigData{2: mains, 5: webhooks}, sourceHash, version); err != nil {
		return errors.Join(fail("持久化投影"), ctx.Err())
	}
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
			zap.Int("pool_id", pool.ID))
		resolveTimeout, _ = pm.ParseDuration("5m")
	}

	// 解析分组第一次等待时间
	groupWait, err := pm.ParseDuration(pool.GroupWait)
	if err != nil {
		a.logger.Warn(LogModuleMonitor+"解析GroupWait失败，使用默认值",
			zap.Int("pool_id", pool.ID))
		groupWait, _ = pm.ParseDuration("30s")
	}

	// 解析分组等待间隔时间
	groupInterval, err := pm.ParseDuration(pool.GroupInterval)
	if err != nil {
		a.logger.Warn(LogModuleMonitor+"解析GroupInterval失败，使用默认值",
			zap.Int("pool_id", pool.ID))
		groupInterval, _ = pm.ParseDuration("5m")
	}

	// 解析重复发送时间
	repeatInterval, err := pm.ParseDuration(pool.RepeatInterval)
	if err != nil {
		a.logger.Warn(LogModuleMonitor+"解析RepeatInterval失败，使用默认值",
			zap.Int("pool_id", pool.ID))
		repeatInterval, _ = pm.ParseDuration("1h")
	}

	labels := parseGroupByLabels(stringSliceToString(pool.GroupBy))
	groupBy := make([]string, 0, len(labels))
	for _, label := range labels {
		groupBy = append(groupBy, string(label))
	}
	global := altconfig.DefaultGlobalConfig()
	global.ResolveTimeout = resolveTimeout
	config := &altconfig.Config{
		Global: &global,
		Route: &altconfig.Route{
			GroupWait:      &groupWait,
			GroupInterval:  &groupInterval,
			RepeatInterval: &repeatInterval,
			GroupBy:        labels,
			GroupByStr:     groupBy,
		},
	}

	return config
}

// GenerateRouteConfigForPool keeps its public signature and successful webhook
// persistence side effect. Main publication uses the private, side-effect-free renderer.
func (a *alertManagerConfigCache) GenerateRouteConfigForPool(ctx context.Context, pool *model.MonitorAlertManagerPool) ([]*altconfig.Route, []altconfig.Receiver) {
	if pool == nil || ctx.Err() != nil {
		return nil, nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	snapshot := cloneAlertPoolLifecycle(pool)
	// The released zero-output route builder has no persistence side effect.
	// It does not mint an operation or ACK and is never a publishing fallback.
	if len(snapshot.AlertManagerInstances) == 0 && a.configDAO == nil && a.redis == nil {
		groups, err := a.readEnabledAlertGroups(ctx, snapshot.ID)
		if err != nil {
			return nil, nil
		}
		routes, receivers, err := a.renderAlertRoutes(snapshot, groups)
		if err != nil {
			return nil, nil
		}
		return routes, receivers
	}
	versions, err := a.amSourceVersions(ctx)
	if err != nil || a.alertPoolDAO == nil {
		return nil, nil
	}
	current, err := a.alertPoolDAO.GetAlertPoolByID(ctx, snapshot.ID)
	if err != nil || current == nil || current.DeletedAt != 0 || current.ID != snapshot.ID || current.Name != snapshot.Name || !reflect.DeepEqual(current.AlertManagerInstances, snapshot.AlertManagerInstances) {
		return nil, nil
	}
	snapshot = cloneAlertPoolLifecycle(current)
	groups, err := a.readEnabledAlertGroups(ctx, snapshot.ID)
	if err != nil {
		return nil, nil
	}
	routes, receivers, err := a.renderAlertRoutes(snapshot, groups)
	if err != nil {
		return nil, nil
	}
	if err := a.generateWebhookFilesForSendGroups(ctx, snapshot, groups, amSourceVersionFor(versions, snapshot.ID)); err != nil {
		a.alertPublicationError(snapshot.ID, "发布webhook配置")
		return nil, nil
	}
	return routes, receivers
}
func (a *alertManagerConfigCache) generateWebhookFilesForSendGroups(ctx context.Context, pool *model.MonitorAlertManagerPool, groups []*model.MonitorSendGroup, version configDao.AMSourceVersion) error {
	rows := a.alertWebhookConfigs(pool, groups)
	encoded, err := json.Marshal(rows)
	if err != nil {
		return errors.New("webhook版本生成失败")
	}
	hash := sha256.Sum256(encoded)
	return a.amProject(ctx, 5, pool.ID, map[int8]map[string]ConfigData{5: rows}, hash, version)
}

func stringSliceToString(slice model.StringList) string {
	return strings.Join(slice, ",")
}

// buildWebhookURL 构建 webhook 接收 URL，兼容配置中填写 host:port 或完整 URL 的场景，并统一追加 send_group_id
func (a *alertManagerConfigCache) buildWebhookURL(sendGroupID int) string {
	base := strings.TrimSpace(a.alertWebhookAddr)
	if base == "" {
		// 兜底：使用本地端口，避免生成空 URL
		return fmt.Sprintf("http://localhost:%s/api/v1/alerts/receive?send_group_id=%d", a.serverPort, sendGroupID)
	}

	// scheme 大小写不敏感（RFC 3986），HTTPS://h/x 也视为完整 URL，不能拼成 http://HTTPS://…
	hasScheme := hasPrefixFold(base, "http://") || hasPrefixFold(base, "https://")
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

// hasPrefixFold 不区分大小写的前缀判断
func hasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
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

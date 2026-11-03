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
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

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
}

func NewAlertManagerConfigCache(
	cfg *config.Config,
	logger *zap.Logger,
	alertPoolDAO alertPoolDao.AlertManagerPoolDAO,
	alertSendDAO alertPoolDao.AlertManagerSendDAO,
	configDAO configDao.MonitorConfigDAO,
	batchManager *BatchConfigManager,
	redisClient redis.Cmdable,
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
	if err := ctx.Err(); err != nil {
		return err
	}
	startTime := time.Now()
	a.logger.Info(LogModuleMonitor + "开始生成AlertManager主配置")

	page := 1
	size := 100
	var allPools []*model.MonitorAlertManagerPool
	var failures []error

	for {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		pools, total, err := a.alertPoolDAO.GetMonitorAlertManagerPoolList(ctx, &model.GetMonitorAlertManagerPoolListReq{
			ListReq: model.ListReq{
				Page: page,
				Size: size,
			},
		})
		if err != nil {
			return errors.Join(append(failures, a.alertPublicationError(0, "读取实例池"), ctx.Err())...)
		}

		if len(pools) == 0 {
			break
		}

		if err := a.processPoolBatch(ctx, pools); err != nil {
			failures = append(failures, err)
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

	if err := ctx.Err(); err != nil {
		failures = append(failures, err)
	}
	result := errors.Join(failures...)
	logCacheOperation(a.logger, "生成AlertManager主配置", "所有池", startTime, result)
	return result
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

func (a *alertManagerConfigCache) processPoolBatch(ctx context.Context, pools []*model.MonitorAlertManagerPool) error {
	var failures []error
	for _, source := range pools {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		if source == nil || source.ID <= 0 {
			failures = append(failures, a.alertPublicationError(0, "实例池标识"))
			continue
		}
		pool := *source
		pool.AlertManagerInstances = append(model.StringList(nil), source.AlertManagerInstances...)
		pool.GroupBy = append(model.StringList(nil), source.GroupBy...)
		// A newly configured pool without instances is not yet a publication target.
		if len(pool.AlertManagerInstances) == 0 {
			continue
		}
		if err := a.publishAlertPool(ctx, &pool); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
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

// Save only this generator's artifacts; a content hash alone cannot detect a
// changed file name after a pool rename. Custom and foreign rows are never adopted.
func (a *alertManagerConfigCache) planAlertConfigs(ctx context.Context, desired map[string]ConfigData) (alertArtifactWrites, error) {
	if len(desired) == 0 {
		return alertArtifactWrites{}, nil
	}
	dao := a.alertConfigDAO()
	if dao == nil {
		return alertArtifactWrites{}, errors.New("配置存储不可用")
	}
	keys := make([]string, 0, len(desired))
	var kind int8
	for key, row := range desired {
		if kind == 0 {
			kind = row.ConfigType
		}
		if row.ConfigType != kind || (kind != model.ConfigTypeAlertManager && kind != model.ConfigTypeWebhookFile) {
			return alertArtifactWrites{}, errors.New("AlertManager配置类型无效")
		}
		if err := validateYAMLConfig(row.Content); err != nil {
			return alertArtifactWrites{}, errors.New("AlertManager配置内容无效")
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	saved, err := dao.GetMonitorConfigsByInstances(ctx, keys, kind)
	if err != nil {
		return alertArtifactWrites{}, errors.New("配置读取失败")
	}
	owned := map[string]*model.MonitorConfig{}
	conflicts := map[string]bool{}
	for _, row := range saved {
		if row == nil {
			return alertArtifactWrites{}, errors.New("配置快照无效")
		}
		target, wanted := desired[row.InstanceIP]
		if !wanted {
			continue
		}
		belongs := row.PoolID == target.PoolID && row.ConfigType == kind
		if belongs && kind == model.ConfigTypeWebhookFile {
			_, belongs = ownedAlertWebhook(row, target.PoolID)
		}
		if belongs && kind == model.ConfigTypeAlertManager {
			belongs = row.Name == fmt.Sprintf(ConfigNameAlertManager, target.PoolID, row.InstanceIP)
		}
		if !belongs {
			conflicts[row.InstanceIP] = true
			continue
		}
		if owned[row.InstanceIP] != nil {
			return alertArtifactWrites{}, errors.New("生成配置归属不唯一")
		}
		owned[row.InstanceIP] = row
	}
	var create []*model.MonitorConfig
	var update []configDao.GeneratedConfigUpdate
	for _, key := range keys {
		row := desired[key]
		hash := calculateConfigHash(row.Content)
		if conflicts[key] {
			return alertArtifactWrites{}, errors.New("配置键归属冲突")
		}
		previous := owned[key]
		if previous == nil {
			create = append(create, &model.MonitorConfig{Name: row.Name, PoolID: row.PoolID, InstanceIP: key, ConfigType: kind, ConfigContent: row.Content, ConfigHash: hash, Status: model.ConfigStatusActive, LastGeneratedTime: time.Now().Unix()})
			continue
		}
		if previous.Name == row.Name && previous.PoolID == row.PoolID && previous.ConfigType == kind && previous.ConfigHash == hash && previous.ConfigContent == row.Content && previous.Status == model.ConfigStatusActive {
			continue
		}
		copy := *previous
		copy.Name = row.Name
		copy.PoolID = row.PoolID
		copy.ConfigType = kind
		copy.ConfigContent = row.Content
		copy.ConfigHash = hash
		copy.Status = model.ConfigStatusActive
		copy.LastGeneratedTime = time.Now().Unix()
		update = append(update, configDao.GeneratedConfigUpdate{Expected: configDao.GeneratedConfigOwnerOf(previous), Desired: copy})
	}
	if err := ctx.Err(); err != nil {
		return alertArtifactWrites{}, err
	}
	return alertArtifactWrites{create: create, update: update}, nil
}

type alertArtifactWrites struct {
	create []*model.MonitorConfig
	update []configDao.GeneratedConfigUpdate
}

func (a *alertManagerConfigCache) applyAlertWrites(ctx context.Context, writes alertArtifactWrites) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dao := a.alertConfigDAO()
	if dao == nil {
		return errors.New("配置存储不可用")
	}
	if len(writes.create) > 0 {
		if err := dao.BatchCreateMonitorConfigs(ctx, writes.create); err != nil {
			return errors.New("配置创建失败")
		}
	}
	if len(writes.update) > 0 {
		if err := dao.UpdateGeneratedMonitorConfigs(ctx, writes.update); err != nil {
			return errors.New("配置更新失败")
		}
	}
	return nil
}
func (a *alertManagerConfigCache) saveAlertConfigs(ctx context.Context, desired map[string]ConfigData) error {
	if len(desired) == 0 {
		return nil
	}
	writes, err := a.planAlertConfigs(ctx, desired)
	if err != nil {
		return err
	}
	return a.applyAlertWrites(ctx, writes)
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
func (a *alertManagerConfigCache) retiredAlertWebhooks(ctx context.Context, poolID int, desired map[string]ConfigData) ([]retiredAlertWebhook, error) {
	dao := a.alertConfigDAO()
	if dao == nil {
		return nil, errors.New("配置存储不可用")
	}
	kind := model.ConfigTypeWebhookFile
	var retired []retiredAlertWebhook
	rowsByKey := map[string][]*model.MonitorConfig{}
	scanned := 0
	for page := 1; ; page++ {
		rows, total, err := dao.GetMonitorConfigList(ctx, &model.GetMonitorConfigListReq{ListReq: model.ListReq{Page: page, Size: 100}, ConfigType: &kind})
		if err != nil {
			return nil, errors.New("旧webhook配置读取失败")
		}
		for _, row := range rows {
			if row == nil {
				return nil, errors.New("旧配置快照无效")
			}
			if row.ConfigType == kind {
				rowsByKey[row.InstanceIP] = append(rowsByKey[row.InstanceIP], row)
			}
			if item, ok := ownedAlertWebhook(row, poolID); ok {
				if _, wanted := desired[row.InstanceIP]; !wanted {
					retired = append(retired, item)
				}
			}
		}
		scanned += len(rows)
		if len(rows) == 0 || scanned >= int(total) {
			break
		}
	}
	for _, item := range retired {
		if len(rowsByKey[item.row.InstanceIP]) != 1 {
			return nil, errors.New("旧webhook配置归属冲突")
		}
	}
	sort.Slice(retired, func(i, j int) bool { return retired[i].row.ID < retired[j].row.ID })
	return retired, nil
}

// Membership alone cannot authorize deletion of a single Redis key when its
// database projection also belongs to a custom config or another pool.
func (a *alertManagerConfigCache) checkRetiredAlertMainOwnership(ctx context.Context, poolID int, oldIPs []string, desired map[string]bool) (map[string]configDao.GeneratedConfigOwner, error) {
	var stale []string
	for _, ip := range oldIPs {
		if !desired[ip] {
			stale = append(stale, ip)
		}
	}
	if len(stale) == 0 {
		return map[string]configDao.GeneratedConfigOwner{}, nil
	}
	rows, err := a.alertConfigDAO().GetMonitorConfigsByInstances(ctx, stale, model.ConfigTypeAlertManager)
	if err != nil {
		return nil, errors.New("旧主配置读取失败")
	}
	seen := map[string]bool{}
	owners := map[string]configDao.GeneratedConfigOwner{}
	for _, row := range rows {
		if row == nil {
			return nil, errors.New("旧主配置快照无效")
		}
		if row.PoolID != poolID || row.ConfigType != model.ConfigTypeAlertManager || row.Name != fmt.Sprintf(ConfigNameAlertManager, poolID, row.InstanceIP) || seen[row.InstanceIP] {
			return nil, errors.New("旧主配置归属冲突")
		}
		seen[row.InstanceIP] = true
		owners[row.InstanceIP] = configDao.GeneratedConfigOwnerOf(row)
	}
	return owners, nil
}

func (a *alertManagerConfigCache) publishAlertPool(ctx context.Context, pool *model.MonitorAlertManagerPool) error {
	fail := func(stage string) error { return a.alertPublicationError(pool.ID, stage) }
	if err := validateInstanceIPs(pool.AlertManagerInstances); err != nil {
		return fail("实例列表无效")
	}
	if strings.TrimSpace(pool.Name) == "" {
		return fail("实例池名称无效")
	}
	groups, err := a.readEnabledAlertGroups(ctx, pool.ID)
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
	content, err := yaml.Marshal(cfg)
	if err != nil {
		return fail("序列化配置")
	}
	if _, err = altconfig.Load(string(content)); err != nil {
		return fail("验证配置")
	}
	webhooks := a.alertWebhookConfigs(pool, groups)
	instances := append([]string(nil), pool.AlertManagerInstances...)
	sort.Strings(instances)
	payload, err := json.Marshal(struct {
		Main      string
		Webhooks  map[string]ConfigData
		Instances []string
	}{string(content), webhooks, instances})
	if err != nil {
		return fail("生成版本")
	}
	hash := calculateConfigHash(string(payload))
	hashKey := alertPublicationStateKey(pool.ID)
	oldHash, err := a.redis.Get(ctx, hashKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fail("读取缓存版本")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	setKey := buildRedisSetKeyAlertManagerMainPoolIPs(pool.ID)
	oldIPs, err := a.redis.SMembers(ctx, setKey).Result()
	if err != nil {
		return fail("读取实例成员")
	}
	retired, err := a.retiredAlertWebhooks(ctx, pool.ID, webhooks)
	if err != nil {
		return fail("读取旧webhook配置")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	mains := map[string]ConfigData{}
	desired := map[string]bool{}
	for _, ip := range instances {
		desired[ip] = true
		mains[ip] = ConfigData{Name: fmt.Sprintf(ConfigNameAlertManager, pool.ID, ip), PoolID: pool.ID, ConfigType: model.ConfigTypeAlertManager, Content: string(content)}
	}
	webhookWrites, err := a.planAlertConfigs(ctx, webhooks)
	if err != nil {
		return fail("检查webhook配置归属")
	}
	mainWrites, err := a.planAlertConfigs(ctx, mains)
	if err != nil {
		return fail("检查主配置归属")
	}
	retiredMains, err := a.checkRetiredAlertMainOwnership(ctx, pool.ID, oldIPs, desired)
	if err != nil {
		return fail("检查旧实例配置归属")
	}
	intents, err := a.readAlertRetirementIntents(ctx, pool.ID, webhooks, retired)
	if err != nil {
		return fail("检查退役配置归属")
	}
	sameMembers := len(oldIPs) == len(desired)
	for _, ip := range oldIPs {
		if !desired[ip] {
			sameMembers = false
		}
	}
	if oldHash == hash && len(webhookWrites.create) == 0 && len(webhookWrites.update) == 0 && len(mainWrites.create) == 0 && len(mainWrites.update) == 0 && len(intents) == 0 && sameMembers {
		return nil
	}
	// An old ACK must stop being usable before the first artifact can change.
	// Otherwise partial B followed by source A could reuse the old A ACK while
	// serving B. A failed invalidation forbids all following artifact writes.
	if err := a.redis.Set(ctx, hashKey, "pending:"+hash, 0).Err(); err != nil {
		return fail("标记待发布版本")
	}
	for _, intent := range intents {
		if err := a.redis.SAdd(ctx, alertRetirementIntentKey(pool.ID), intent.encoded).Err(); err != nil {
			return fail("登记退役配置")
		}
	}
	if err := a.applyAlertWrites(ctx, webhookWrites); err != nil {
		return fail("持久化webhook配置")
	}
	if err := a.applyAlertWrites(ctx, mainWrites); err != nil {
		return fail("持久化主配置")
	}
	for _, group := range groups {
		for _, ip := range instances {
			if err := a.redis.Set(ctx, buildRedisKeyWebhookFile(ip, group.ID), webhooks[fmt.Sprintf("%s_%d", ip, group.ID)].Content, 0).Err(); err != nil {
				return fail("写入webhook缓存")
			}
		}
	}
	for _, ip := range instances {
		if err := a.redis.Set(ctx, buildRedisKeyAlertManagerMain(ip), string(content), 0).Err(); err != nil {
			return fail("写入主配置缓存")
		}
		if err := a.redis.SAdd(ctx, setKey, ip).Err(); err != nil {
			return fail("写入实例成员")
		}
	}
	// Desired active IPs are never retired, even when one of their SETs fails.
	for _, ip := range oldIPs {
		if !desired[ip] {
			if owner, exists := retiredMains[ip]; exists {
				if err := a.alertConfigDAO().DeleteGeneratedMonitorConfig(ctx, owner); err != nil {
					return fail("退役旧主配置")
				}
			}
			if err := a.redis.Del(ctx, buildRedisKeyAlertManagerMain(ip)).Err(); err != nil {
				return fail("清理旧实例缓存")
			}
			if err := a.redis.SRem(ctx, setKey, ip).Err(); err != nil {
				return fail("清理旧实例成员")
			}
		}
	}
	for _, intent := range intents {
		// ABA: a wanted key has already been restored by the normal owned write path.
		// Its obsolete tombstone must never authorize deleting the restored key.
		if _, wanted := webhooks[intent.owner.InstanceIP]; !wanted {
			if err := a.alertConfigDAO().DeleteGeneratedMonitorConfig(ctx, intent.retireOwner); err != nil {
				return fail("退役旧webhook配置")
			}
			if err := a.redis.Del(ctx, buildRedisKeyWebhookFile(intent.ip, intent.groupID)).Err(); err != nil {
				return fail("清理旧webhook缓存")
			}
		}
		if err := a.redis.SRem(ctx, alertRetirementIntentKey(pool.ID), intent.encoded).Err(); err != nil {
			return fail("清理退役配置索引")
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.redis.Set(ctx, hashKey, hash, 0).Err(); err != nil {
		return fail("确认缓存版本")
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
	if ctx.Err() != nil {
		return nil, nil
	}
	snapshot := *pool
	snapshot.AlertManagerInstances = append(model.StringList(nil), pool.AlertManagerInstances...)
	snapshot.GroupBy = append(model.StringList(nil), pool.GroupBy...)
	pool = &snapshot
	groups, err := a.readEnabledAlertGroups(ctx, pool.ID)
	if err != nil {
		a.alertPublicationError(pool.ID, "读取发送组")
		return nil, nil
	}
	routes, receivers, err := a.renderAlertRoutes(pool, groups)
	if err != nil {
		a.alertPublicationError(pool.ID, "生成路由")
		return nil, nil
	}
	if err := a.generateWebhookFilesForSendGroups(ctx, pool, groups); err != nil {
		a.alertPublicationError(pool.ID, "发布webhook配置")
		return nil, nil
	}
	return routes, receivers
}

func (a *alertManagerConfigCache) generateWebhookFilesForSendGroups(ctx context.Context, pool *model.MonitorAlertManagerPool, groups []*model.MonitorSendGroup) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	rows := a.alertWebhookConfigs(pool, groups)
	if err := a.saveAlertConfigs(ctx, rows); err != nil {
		return errors.New("webhook持久化失败")
	}
	for _, group := range groups {
		for _, ip := range pool.AlertManagerInstances {
			if err := a.redis.Set(ctx, buildRedisKeyWebhookFile(ip, group.ID), rows[fmt.Sprintf("%s_%d", ip, group.ID)].Content, 0).Err(); err != nil {
				return errors.New("webhook缓存写入失败")
			}
		}
	}
	return nil
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

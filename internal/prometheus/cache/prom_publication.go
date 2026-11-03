package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/utils"
	pc "github.com/prometheus/prometheus/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func (p *prometheusConfigCache) promPublicationError(poolID int, stage string) error {
	p.logger.Error(LogModuleMonitor+"Prometheus主配置发布未完成", zap.Int("pool_id", poolID), zap.String("stage", stage))
	return fmt.Errorf("Prometheus主配置发布未完成，池%d，阶段%s", poolID, stage)
}

func detachedPromInput(pool *model.MonitorScrapePool, rows []*model.MonitorScrapeJob) (*model.MonitorScrapePool, []*model.MonitorScrapeJob) {
	copiedPool := *pool
	copiedPool.PrometheusInstances = append(model.StringList(nil), pool.PrometheusInstances...)
	copiedPool.Tags = append(model.StringList(nil), pool.Tags...)
	copiedRows := make([]*model.MonitorScrapeJob, len(rows))
	for index, row := range rows {
		if row == nil {
			continue
		}
		copy := *row
		copy.TreeNodeIDs = append(model.StringList(nil), row.TreeNodeIDs...)
		copy.Tags = append(model.StringList(nil), row.Tags...)
		copiedRows[index] = &copy
	}
	return &copiedPool, copiedRows
}

func (p *prometheusConfigCache) promConfigDAO() configDao.MonitorConfigDAO {
	if p.configDAO != nil {
		return p.configDAO
	}
	if p.batchManager != nil {
		return p.batchManager.configDAO
	}
	return nil
}

func ownedPromArtifact(row *model.MonitorConfig, poolID int) bool {
	return row != nil && row.ID > 0 && row.PoolID == poolID && row.ConfigType == model.ConfigTypePrometheus &&
		validateInstanceIPs([]string{row.InstanceIP}) == nil && row.Name == fmt.Sprintf(ConfigNamePrometheus, poolID, row.InstanceIP)
}

type promPublicationPlan struct {
	create, update []*model.MonitorConfig
	stale          []string
	retired        map[string]*model.MonitorConfig
}

// Inspect every metadata page before any write. This also discovers canonical
// rows left without membership by an older failed publication.
func (p *prometheusConfigCache) planPromPublication(ctx context.Context, poolID int, desired map[string]string, oldIPs []string) (promPublicationPlan, error) {
	plan := promPublicationPlan{retired: map[string]*model.MonitorConfig{}}
	dao := p.promConfigDAO()
	if dao == nil {
		return plan, errors.New("配置存储不可用")
	}
	kind := model.ConfigTypePrometheus
	byKey := map[string][]*model.MonitorConfig{}
	stale := map[string]bool{}
	for _, ip := range oldIPs {
		if err := validateInstanceIPs([]string{ip}); err != nil {
			return plan, errors.New("实例成员无效")
		}
		if _, wanted := desired[ip]; !wanted {
			stale[ip] = true
		}
	}
	scanned := int64(0)
	expectedTotal := int64(-1)
	seenRowIDs := map[int]bool{}
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return plan, err
		}
		rows, total, err := dao.GetMonitorConfigList(ctx, &model.GetMonitorConfigListReq{ListReq: model.ListReq{Page: page, Size: 100}, ConfigType: &kind})
		if err != nil {
			return plan, errors.New("读取配置失败")
		}
		// COUNT/OFFSET is not a transactional snapshot. Reject observable drift
		// rather than treating a truncated page as proof that a key has no owner.
		if total < 0 || expectedTotal >= 0 && total != expectedTotal {
			return plan, errors.New("配置分页总数不一致")
		}
		expectedTotal = total
		if int64(len(rows)) > expectedTotal-scanned {
			return plan, errors.New("配置分页超出总数")
		}
		if len(rows) == 0 {
			if scanned != expectedTotal {
				return plan, errors.New("配置分页提前结束")
			}
			break
		}
		for _, row := range rows {
			if row == nil || row.ID <= 0 || seenRowIDs[row.ID] {
				return plan, errors.New("配置快照标识无效或重复")
			}
			seenRowIDs[row.ID] = true
			if row.ConfigType != kind {
				continue
			}
			byKey[row.InstanceIP] = append(byKey[row.InstanceIP], row)
			if ownedPromArtifact(row, poolID) {
				if _, wanted := desired[row.InstanceIP]; !wanted {
					stale[row.InstanceIP] = true
				}
			}
		}
		scanned += int64(len(rows))
		if scanned == expectedTotal {
			break
		}
		if len(rows) < 100 {
			return plan, errors.New("配置分页数量不足")
		}
	}
	keys := make([]string, 0, len(desired))
	for ip := range desired {
		keys = append(keys, ip)
	}
	sort.Strings(keys)
	for _, ip := range keys {
		rows := byKey[ip]
		if len(rows) > 1 || len(rows) == 1 && !ownedPromArtifact(rows[0], poolID) {
			return plan, errors.New("配置归属冲突")
		}
		content := desired[ip]
		hash := calculateConfigHash(content)
		name := fmt.Sprintf(ConfigNamePrometheus, poolID, ip)
		if len(rows) == 0 {
			plan.create = append(plan.create, &model.MonitorConfig{Name: name, PoolID: poolID, InstanceIP: ip, ConfigType: kind, ConfigContent: content, ConfigHash: hash, Status: model.ConfigStatusActive, LastGeneratedTime: time.Now().Unix()})
		} else {
			row := rows[0]
			if row.ConfigContent == content && row.ConfigHash == hash && row.Status == model.ConfigStatusActive {
				continue
			}
			copy := *row
			copy.ConfigContent, copy.ConfigHash, copy.Status = content, hash, model.ConfigStatusActive
			copy.LastGeneratedTime = time.Now().Unix()
			plan.update = append(plan.update, &copy)
		}
	}
	for ip := range stale {
		rows := byKey[ip]
		if len(rows) > 1 || len(rows) == 1 && !ownedPromArtifact(rows[0], poolID) {
			return plan, errors.New("旧配置归属冲突")
		}
		if len(rows) == 1 {
			plan.retired[ip] = rows[0]
		}
		plan.stale = append(plan.stale, ip)
	}
	sort.Strings(plan.stale)
	return plan, nil
}

// Collect the complete pool inventory before publication; incomplete pages do not
// authorize any writes. Count checks detect observed drift, not SQL snapshot isolation.
func (p *prometheusConfigCache) readPromPoolSnapshot(ctx context.Context) ([]*model.MonitorScrapePool, error) {
	return readScrapePoolSnapshot(ctx, p.scrapePoolDAO)
}

func (p *prometheusConfigCache) GenerateMainConfig(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	pools, err := p.readPromPoolSnapshot(ctx)
	if err != nil {
		return errors.Join(p.promPublicationError(0, "读取采集池"), ctx.Err())
	}
	var failures []error
	for _, pool := range pools {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		if err := p.publishPromPool(ctx, pool); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(append(failures, ctx.Err())...)
}

func (p *prometheusConfigCache) publishPromPool(ctx context.Context, sourcePool *model.MonitorScrapePool) error {
	pool, _ := detachedPromInput(sourcePool, nil)
	fail := func(stage string) error { return p.promPublicationError(pool.ID, stage) }
	if len(pool.PrometheusInstances) > 0 {
		if err := validateInstanceIPs(pool.PrometheusInstances); err != nil {
			return fail("实例地址")
		}
	}
	desiredIPs := map[string]bool{}
	for _, ip := range pool.PrometheusInstances {
		if desiredIPs[ip] {
			return fail("实例地址重复")
		}
		desiredIPs[ip] = true
	}

	jobs, err := p.scrapeJobDAO.GetMonitorScrapeJobsByPoolId(ctx, pool.ID)
	if err != nil {
		return fail("读取采集任务")
	}
	pool, jobs = detachedPromInput(pool, jobs)
	base, err := p.CreateBaseConfig(pool)
	if err != nil {
		return fail("生成基础配置")
	}
	scrapes, err := p.renderScrapeJobSnapshot(jobs)
	if err != nil {
		return fail("生成采集配置")
	}
	base.ScrapeConfigs = scrapes
	// Validate even when there are no targets. An invalid source must never be
	// mistaken for an authoritative request to retire its last-good artifacts.
	raw, err := marshalPrometheusConfig(base)
	if err != nil {
		return fail("序列化基础配置")
	}
	if _, err := pc.Load(string(raw), slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		return fail("校验基础配置")
	}
	generated := make(map[string]string, len(pool.PrometheusInstances))
	for index, ip := range pool.PrometheusInstances {
		config := base
		if len(pool.PrometheusInstances) > 1 {
			config.ScrapeConfigs = p.ApplyHashMod(scrapes, len(pool.PrometheusInstances), index)
		}
		content, err := marshalPrometheusConfig(config)
		if err != nil {
			return fail("序列化配置")
		}
		if _, err := pc.Load(string(content), slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
			return fail("校验可执行配置")
		}
		generated[ip] = string(content)
	}
	serialized, err := json.Marshal(generated)
	if err != nil {
		return fail("生成配置摘要")
	}
	// The bytes include operational credentials, discovery and per-instance
	// sharding. Public/redacted job JSON is not an adequate version fingerprint.
	targetHash := fmt.Sprintf("%x", sha256.Sum256(append([]byte(utils.CalculatePromHash(pool)), serialized...)))
	key := promPublicationKey(pool.ID)
	previous, err := p.redis.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fail("读取发布版本")
	}
	membersKey := buildRedisSetKeyPrometheusMainPoolIPs(pool.ID)
	oldIPs, err := p.redis.SMembers(ctx, membersKey).Result()
	if err != nil {
		return fail("读取实例成员")
	}
	plan, err := p.planPromPublication(ctx, pool.ID, generated, oldIPs)
	if err != nil {
		return fail("检查配置归属")
	}
	sameMembers := len(oldIPs) == len(desiredIPs)
	for _, ip := range oldIPs {
		if !desiredIPs[ip] {
			sameMembers = false
		}
	}
	sameArtifacts := true
	for _, ip := range pool.PrometheusInstances {
		body, err := p.redis.Get(ctx, buildRedisKeyPrometheusMain(ip)).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return fail("读取配置缓存")
		}
		if body != generated[ip] {
			sameArtifacts = false
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if previous == targetHash && sameMembers && sameArtifacts && len(plan.create) == 0 && len(plan.update) == 0 && len(plan.stale) == 0 {
		return nil
	}
	if err := p.redis.Set(ctx, key, "pending:"+targetHash, 0).Err(); err != nil {
		return fail("标记待发布")
	}
	// Register all intended targets before DB or configuration writes. Failed
	// attempts remain discoverable even when no configuration row was created.
	for _, ip := range pool.PrometheusInstances {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := p.redis.SAdd(ctx, membersKey, ip).Err(); err != nil {
			return fail("登记实例成员")
		}
	}
	dao := p.promConfigDAO()
	if len(plan.create) > 0 {
		if err := dao.BatchCreateMonitorConfigs(ctx, plan.create); err != nil {
			return fail("创建配置")
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(plan.update) > 0 {
		if err := dao.BatchUpdateMonitorConfigs(ctx, plan.update); err != nil {
			return fail("更新配置")
		}
	}
	for _, ip := range pool.PrometheusInstances {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := p.redis.Set(ctx, buildRedisKeyPrometheusMain(ip), generated[ip], 0).Err(); err != nil {
			return fail("写入配置缓存")
		}
	}
	for _, ip := range plan.stale {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := p.redis.Del(ctx, buildRedisKeyPrometheusMain(ip)).Err(); err != nil {
			return fail("清理旧配置缓存")
		}
		if row := plan.retired[ip]; row != nil {
			if err := dao.DeleteMonitorConfig(ctx, row.ID); err != nil {
				return fail("清理旧配置")
			}
		}
		if err := p.redis.SRem(ctx, membersKey, ip).Err(); err != nil {
			return fail("清理旧实例成员")
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := p.redis.Set(ctx, key, targetHash, 0).Err(); err != nil {
		return fail("确认发布版本")
	}
	return nil
}

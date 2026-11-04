package cache

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"go.uber.org/zap"
)

func (r *recordRuleConfigCache) recordPublicationError(poolID int, stage string) error {
	r.logger.Error(LogModuleMonitor+"预聚合规则发布未完成", zap.Int("pool_id", poolID), zap.String("stage", stage))
	return fmt.Errorf("预聚合规则发布未完成，池%d，阶段%s", poolID, stage)
}

func detachedRecordRules(pool *model.MonitorScrapePool, rows []*model.MonitorRecordRule) (*model.MonitorScrapePool, []*model.MonitorRecordRule) {
	copiedPool := *pool
	copiedPool.PrometheusInstances = append(model.StringList(nil), pool.PrometheusInstances...)
	copiedRows := make([]*model.MonitorRecordRule, len(rows))
	for index, row := range rows {
		if row == nil {
			continue
		}
		copy := *row
		copy.Labels = append(model.StringList(nil), row.Labels...)
		copiedRows[index] = &copy
	}
	return &copiedPool, copiedRows
}

func (r *recordRuleConfigCache) recordConfigDAO() configDao.MonitorConfigDAO {
	if r.configDAO != nil {
		return r.configDAO
	}
	if r.batchManager != nil {
		return r.batchManager.configDAO
	}
	return nil
}

func ownedRecordArtifact(row *model.MonitorConfig, poolID int) bool {
	return row != nil && row.ID > 0 && row.PoolID == poolID && row.ConfigType == model.ConfigTypeRecordRule &&
		validateInstanceIPs([]string{row.InstanceIP}) == nil && row.Name == fmt.Sprintf(ConfigNameRecordRule, poolID, row.InstanceIP)
}

type recordPublicationPlan struct {
	create  []*model.MonitorConfig
	update  []configDao.GeneratedConfigUpdate
	stale   []string
	retired map[string]configDao.GeneratedConfigOwner
	owners  map[string]*model.MonitorConfig
}

// Inspect every metadata page before any write. This also discovers canonical
// rows left without membership by an older failed publication.
func (r *recordRuleConfigCache) planRecordPublication(ctx context.Context, poolID int, desired map[string]string, oldIPs []string) (recordPublicationPlan, error) {
	plan := recordPublicationPlan{retired: map[string]configDao.GeneratedConfigOwner{}, owners: map[string]*model.MonitorConfig{}}
	dao := r.recordConfigDAO()
	if dao == nil {
		return plan, errors.New("配置存储不可用")
	}
	kind := model.ConfigTypeRecordRule
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
	candidates, err := readPoolConfigCandidates(ctx, dao, poolID, kind)
	if err != nil {
		return plan, err
	}
	queryKeys := make([]string, 0, len(desired)+len(oldIPs)+len(candidates))
	for ip := range desired {
		queryKeys = append(queryKeys, ip)
	}
	queryKeys = append(queryKeys, oldIPs...)
	var locators []configDao.GeneratedConfigOwner
	for _, row := range candidates {
		if ownedRecordArtifact(row, poolID) {
			queryKeys = append(queryKeys, row.InstanceIP)
			locators = append(locators, configDao.GeneratedConfigOwnerOf(row))
			if _, wanted := desired[row.InstanceIP]; !wanted {
				stale[row.InstanceIP] = true
			}
		}
	}
	rows, err := readConfigKeyNamespace(ctx, dao, kind, queryKeys)
	if err != nil {
		return plan, err
	}
	if err := validateConfigLocatorOwners(locators, rows); err != nil {
		return plan, err
	}
	for _, row := range rows {
		byKey[row.InstanceIP] = append(byKey[row.InstanceIP], row)
	}
	keys := make([]string, 0, len(desired))
	for ip := range desired {
		keys = append(keys, ip)
	}
	sort.Strings(keys)
	for _, ip := range keys {
		rows := byKey[ip]
		if len(rows) > 1 || len(rows) == 1 && !ownedRecordArtifact(rows[0], poolID) {
			return plan, errors.New("规则配置归属冲突")
		}
		content := desired[ip]
		hash := calculateConfigHash(content)
		name := fmt.Sprintf(ConfigNameRecordRule, poolID, ip)
		if len(rows) == 0 {
			plan.create = append(plan.create, &model.MonitorConfig{Name: name, PoolID: poolID, InstanceIP: ip, ConfigType: kind, ConfigContent: content, ConfigHash: hash, Status: model.ConfigStatusActive, LastGeneratedTime: time.Now().Unix()})
		} else {
			row := rows[0]
			copyOfRow := *row
			plan.owners[ip] = &copyOfRow
			if row.ConfigContent == content && row.ConfigHash == hash && row.Status == model.ConfigStatusActive {
				continue
			}
			copy := *row
			copy.ConfigContent, copy.ConfigHash, copy.Status = content, hash, model.ConfigStatusActive
			copy.LastGeneratedTime = time.Now().Unix()
			plan.update = append(plan.update, configDao.GeneratedConfigUpdate{Expected: configDao.GeneratedConfigOwnerOf(row), Desired: copy})
		}
	}
	for ip := range stale {
		rows := byKey[ip]
		if len(rows) > 1 || len(rows) == 1 && !ownedRecordArtifact(rows[0], poolID) {
			return plan, errors.New("旧规则配置归属冲突")
		}
		if len(rows) == 1 {
			plan.retired[ip] = configDao.GeneratedConfigOwnerOf(rows[0])
		}
		plan.stale = append(plan.stale, ip)
	}
	sort.Strings(plan.stale)
	return plan, nil
}

func (r *recordRuleConfigCache) GenerateMainConfig(ctx context.Context) error {
	if err := r.recordPublicationReady(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Serialize this cache instance only; this is not a distributed transaction.
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	sourceVersions, err := r.recordSourceVersions(ctx)
	if err != nil {
		return err
	}
	pools, err := readScrapePoolSnapshot(ctx, r.scrapePoolDAO)
	if err != nil {
		return errors.Join(r.recordPublicationError(0, "读取采集池"), ctx.Err())
	}
	var failures []error
	for _, pool := range pools {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		if pool == nil || pool.ID <= 0 {
			failures = append(failures, r.recordPublicationError(0, "采集池标识"))
			continue
		}
		if err := r.publishRecordPool(ctx, pool, sourceVersions[pool.ID]); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(append(failures, ctx.Err())...)
}

func (r *recordRuleConfigCache) publishRecordPool(ctx context.Context, sourcePool *model.MonitorScrapePool, sourceVersion uint64) error {
	if err := r.recordPublicationReady(); err != nil {
		return err
	}
	pool, _ := detachedRecordRules(sourcePool, nil)
	fail := func(stage string) error { return r.recordPublicationError(pool.ID, stage) }
	if pool.DeletedAt > 0 {
		pool.SupportRecord = 2
	}
	if pool.SupportRecord != 1 && pool.SupportRecord != 2 {
		return fail("功能状态无效")
	}
	// Explicit off is authoritative. Its potentially stale target fields and rule
	// contents cannot authorize new keys or block retiring previously owned keys.
	if pool.SupportRecord == 2 {
		pool.PrometheusInstances = nil
	}
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
	var rows []*model.MonitorRecordRule
	var err error
	generated := map[string]string{}
	if pool.SupportRecord == 1 {
		if r.alertRecordDAO == nil {
			return fail("规则存储不可用")
		}
		rows, _, err = r.alertRecordDAO.GetMonitorRecordRuleByPoolId(ctx, pool.ID)
		if err != nil {
			return fail("读取规则")
		}
		pool, rows = detachedRecordRules(pool, rows)
		generated = r.generateConfigForRules(pool, rows)
		if generated == nil {
			return fail("构造规则")
		}
	}
	for _, ip := range pool.PrometheusInstances {
		if _, ok := generated[ip]; !ok {
			return fail("规则文件不完整")
		}
	}
	targetHash := calculateConfigHash(fmt.Sprintf("record-support-v1:%d:%s", pool.SupportRecord, calculateRecordPoolHash(pool, rows)))
	if pool.DeletedAt > 0 {
		targetHash = calculateConfigHash(fmt.Sprintf("deleted:%d:%d", pool.ID, pool.DeletedAt))
	}
	return r.publishRecordProjection(ctx, pool, generated, targetHash, sourceVersion, fail)
}

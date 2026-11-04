package cache

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

type scrapePublicationPlan struct {
	create  []*model.MonitorConfig
	update  []configDao.GeneratedConfigUpdate
	stale   []string
	retired map[string]configDao.GeneratedConfigOwner
	owners  map[string]*model.MonitorConfig
}

// Inspect every metadata page before any write. This also discovers canonical
// rows left without membership by an older failed publication.
func planScrapePublication(ctx context.Context, scope scrapePublicationScope, dao configDao.MonitorConfigDAO, poolID int, desired map[string]string, oldIPs []string) (scrapePublicationPlan, error) {
	if !scope.valid() {
		return scrapePublicationPlan{}, errors.New("规则发布作用域无效")
	}
	rows := map[string]ConfigData{}
	for key, body := range desired {
		rows[key] = ConfigData{Name: fmt.Sprintf(scope.pattern(), poolID, key), PoolID: poolID, ConfigType: int8(scope), Content: body}
	}
	return planNamespacePublication(ctx, int8(scope), dao, poolID, rows, oldIPs)
}
func planNamespacePublication(ctx context.Context, kind int8, dao configDao.MonitorConfigDAO, poolID int, desired map[string]ConfigData, oldIPs []string) (scrapePublicationPlan, error) {
	return planNamespaceProjection(ctx, kind, dao, poolID, desired, oldIPs, true)
}
func planNamespaceProjection(ctx context.Context, kind int8, dao configDao.MonitorConfigDAO, poolID int, desired map[string]ConfigData, oldIPs []string, retire bool) (scrapePublicationPlan, error) {
	if kind < 1 || kind > 5 {
		return scrapePublicationPlan{}, errors.New("配置命名空间类型无效")
	}
	plan := scrapePublicationPlan{retired: map[string]configDao.GeneratedConfigOwner{}, owners: map[string]*model.MonitorConfig{}}

	if dao == nil {
		return plan, errors.New("配置存储不可用")
	}
	byKey := map[string][]*model.MonitorConfig{}
	stale := map[string]bool{}
	for _, ip := range oldIPs {
		if _, err := configDao.ProjectionBindingForNamespace(kind, ip); err != nil {
			return plan, errors.New("实例成员无效")
		}
		if _, wanted := desired[ip]; !wanted {
			stale[ip] = true
		}
	}
	var candidates []*model.MonitorConfig
	var err error
	if retire {
		candidates, err = readPoolConfigCandidates(ctx, dao, poolID, kind)
		if err != nil {
			return plan, err
		}
	}
	queryKeys := make([]string, 0, len(desired)+len(oldIPs)+len(candidates))
	for ip := range desired {
		queryKeys = append(queryKeys, ip)
	}
	queryKeys = append(queryKeys, oldIPs...)
	var locators []configDao.GeneratedConfigOwner
	for _, row := range candidates {
		if ownedNamespaceArtifact(row, kind, poolID) {
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
		if len(rows) > 1 || len(rows) == 1 && !ownedNamespaceArtifact(rows[0], kind, poolID) {
			return plan, errors.New("规则配置归属冲突")
		}
		content := desired[ip].Content
		hash := calculateConfigHash(content)
		name := desired[ip].Name
		if len(rows) == 0 {
			plan.create = append(plan.create, &model.MonitorConfig{Name: name, PoolID: poolID, InstanceIP: ip, ConfigType: kind, ConfigContent: content, ConfigHash: hash, Status: model.ConfigStatusActive, LastGeneratedTime: time.Now().Unix()})
		} else {
			row := rows[0]
			copyOfRow := *row
			plan.owners[ip] = &copyOfRow
			if row.ConfigContent == content && row.ConfigHash == hash && row.Status == model.ConfigStatusActive && row.Name == name {
				continue
			}
			copy := *row
			copy.ConfigContent, copy.ConfigHash, copy.Status, copy.Name = content, hash, model.ConfigStatusActive, name
			copy.LastGeneratedTime = time.Now().Unix()
			plan.update = append(plan.update, configDao.GeneratedConfigUpdate{Expected: configDao.GeneratedConfigOwnerOf(row), Desired: copy})
		}
	}
	for ip := range stale {
		rows := byKey[ip]
		if len(rows) > 1 || len(rows) == 1 && !ownedNamespaceArtifact(rows[0], kind, poolID) {
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

func ownedNamespaceArtifact(row *model.MonitorConfig, kind int8, poolID int) bool {
	if row == nil || row.PoolID != poolID || row.ConfigType != kind {
		return false
	}
	if _, err := configDao.ProjectionBindingForNamespace(kind, row.InstanceIP); err != nil {
		return false
	}
	return configDao.GeneratedProjectionOwnerValid(configDao.GeneratedConfigOwnerOf(row))
}

package cache

import (
	"context"
	"errors"
	"sort"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

// Candidates locate possible old keys; their content is never the final plan's
// authority. Read them afresh and validate the private DAO's declared scope.
func readPoolConfigCandidates(ctx context.Context, dao configdao.MonitorConfigDAO, poolID int, kind int8) ([]*model.MonitorConfig, error) {
	if dao == nil {
		return nil, errors.New("配置元数据存储不可用")
	}
	rows, err := readConfigMetadataPages(ctx, func(page, size int) ([]*model.MonitorConfig, int64, error) {
		return dao.GetGeneratedConfigPoolCandidates(ctx, configdao.GeneratedConfigPoolQuery{PoolID: poolID, ConfigType: kind, Page: page, Size: size})
	})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.PoolID != poolID || row.ConfigType != kind || row.DeletedAt != 0 {
			return nil, errors.New("配置候选超出查询范围")
		}
	}
	return rows, nil
}

// Every key batch reads the entire namespace, including foreign/custom/inactive
// duplicates. No batch can authorize writes until every batch/page has passed.
func readConfigKeyNamespace(ctx context.Context, dao configdao.MonitorConfigDAO, kind int8, keys []string) ([]*model.MonitorConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if dao == nil {
		return nil, errors.New("配置元数据存储不可用")
	}
	unique := map[string]bool{}
	ordered := make([]string, 0, len(keys))
	for _, key := range keys {
		if key == "" {
			return nil, errors.New("配置查询键无效")
		}
		if !unique[key] {
			unique[key] = true
			ordered = append(ordered, key)
		}
	}
	sort.Strings(ordered)
	var snapshot []*model.MonitorConfig
	seen := map[int]bool{}
	for start := 0; start < len(ordered); start += configdao.GeneratedConfigQueryLimit {
		end := start + configdao.GeneratedConfigQueryLimit
		if end > len(ordered) {
			end = len(ordered)
		}
		batch := append([]string(nil), ordered[start:end]...)
		wanted := map[string]bool{}
		for _, key := range batch {
			wanted[key] = true
		}
		rows, err := readConfigMetadataPages(ctx, func(page, size int) ([]*model.MonitorConfig, int64, error) {
			return dao.GetGeneratedConfigNamespacePage(ctx, configdao.GeneratedConfigKeysQuery{ConfigType: kind, InstanceKeys: batch, Page: page, Size: size})
		})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if seen[row.ID] || row.ConfigType != kind || row.DeletedAt != 0 || !wanted[row.InstanceIP] {
				return nil, errors.New("配置命名空间标识重复或超出查询范围")
			}
			seen[row.ID] = true
			snapshot = append(snapshot, row)
		}
	}
	return snapshot, nil
}

// A row disappearing or changing immutable owner between the locator read and
// namespace read is observed drift, never proof of absence/create authority.
// Same-owner body/hash/status changes are resolved from the fresh namespace.
func validateConfigLocatorOwners(locators []configdao.GeneratedConfigOwner, current []*model.MonitorConfig) error {
	byID := map[int]configdao.GeneratedConfigOwner{}
	for _, row := range current {
		byID[row.ID] = configdao.GeneratedConfigOwnerOf(row)
	}
	for _, owner := range locators {
		if byID[owner.ID] != owner {
			return errors.New("配置候选归属在读取期间变化")
		}
	}
	return nil
}

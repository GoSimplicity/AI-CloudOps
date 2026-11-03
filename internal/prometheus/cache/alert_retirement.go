package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

// Only immutable metadata is retained here. Config bodies, URLs and credentials
// must never enter this retry index. It locates work; the SQL owner guard remains
// the authority for deletion, including when a prior attempt soft-deleted a row.
type alertRetirementIntent struct {
	owner       configdao.GeneratedConfigOwner
	retireOwner configdao.GeneratedConfigOwner
	encoded     string
	ip          string
	groupID     int
}

func alertRetirementIntentKey(poolID int) string {
	return fmt.Sprintf("%s:retirement:alertmanager:webhook:pool:%d", redisPrefix, poolID)
}
func (a *alertManagerConfigCache) readAlertRetirementIntents(ctx context.Context, poolID int, desired map[string]ConfigData, retired []retiredAlertWebhook) ([]alertRetirementIntent, error) {
	entries, err := a.redis.SMembers(ctx, alertRetirementIntentKey(poolID)).Result()
	if err != nil {
		return nil, errors.New("退役配置索引读取失败")
	}
	byEntry := map[string]alertRetirementIntent{}
	currentRetired := map[int]configdao.GeneratedConfigOwner{}
	for _, item := range retired {
		currentRetired[item.row.ID] = configdao.GeneratedConfigOwnerOf(item.row)
	}
	add := func(owner configdao.GeneratedConfigOwner, encoded string) error {
		row := &model.MonitorConfig{Model: model.Model{ID: owner.ID}, Name: owner.Name, PoolID: owner.PoolID, ConfigType: owner.ConfigType, InstanceIP: owner.InstanceIP}
		item, ok := ownedAlertWebhook(row, poolID)
		if !ok {
			return errors.New("退役配置索引无效")
		}
		byEntry[encoded] = alertRetirementIntent{owner: owner, retireOwner: owner, encoded: encoded, ip: item.ip, groupID: item.groupID}
		return nil
	}
	for _, encoded := range entries {
		var owner configdao.GeneratedConfigOwner
		decoder := json.NewDecoder(strings.NewReader(encoded))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&owner); err != nil {
			return nil, errors.New("退役配置索引无效")
		}
		canonical, err := json.Marshal(owner)
		if err != nil || string(canonical) != encoded {
			return nil, errors.New("退役配置索引无效")
		}
		if err := add(owner, encoded); err != nil {
			return nil, err
		}
	}
	// Decode the whole index before validating aliases: after a renamed row is
	// retired, its exact-owner intent can justify cancelling an older name's
	// retry locator, regardless of Redis set enumeration order.
	for encoded, intent := range byEntry {
		owner := intent.owner
		// Never treat the Redis JSON itself as proof of ownership, even on ABA.
		row, err := a.alertConfigDAO().GetGeneratedMonitorConfigByID(ctx, owner.ID)
		if err != nil || row == nil {
			return nil, errors.New("退役配置归属冲突")
		}
		if configdao.GeneratedConfigOwnerOf(row) != owner {
			// A previous ABA restoration may have legitimately renamed this
			// still-active generated row before SREM failed. Only the current
			// desired/current canonical owner can authorize cancelling the old locator.
			wanted, ok := desired[owner.InstanceIP]
			current := configdao.GeneratedConfigOwnerOf(row)
			wantedOwner := ok && row.PoolID == wanted.PoolID && row.ConfigType == wanted.ConfigType && row.InstanceIP == owner.InstanceIP && row.Name == wanted.Name
			retiredOwner := currentRetired[row.ID] == current && row.PoolID == owner.PoolID && row.ConfigType == owner.ConfigType && row.InstanceIP == owner.InstanceIP
			currentJSON, _ := json.Marshal(current)
			_, indexedCurrent := byEntry[string(currentJSON)]
			_, currentCanonical := ownedAlertWebhook(row, poolID)
			sameAddress := row.PoolID == owner.PoolID && row.ConfigType == owner.ConfigType && row.InstanceIP == owner.InstanceIP
			exactIndexedTombstone := row.DeletedAt != 0 && sameAddress && currentCanonical && indexedCurrent
			if !exactIndexedTombstone && (row.DeletedAt != 0 || (!wantedOwner && !retiredOwner)) {
				return nil, errors.New("退役配置归属冲突")
			}
			intent.retireOwner = current
			byEntry[encoded] = intent
		}
	}
	for _, item := range retired {
		owner := configdao.GeneratedConfigOwnerOf(item.row)
		encoded, err := json.Marshal(owner)
		if err != nil {
			return nil, errors.New("退役配置索引无效")
		}
		if err := add(owner, string(encoded)); err != nil {
			return nil, err
		}
	}
	result := make([]alertRetirementIntent, 0, len(byEntry))
	for _, intent := range byEntry {
		if _, wanted := desired[intent.owner.InstanceIP]; !wanted {
			rows, err := a.alertConfigDAO().GetMonitorConfigsByInstances(ctx, []string{intent.owner.InstanceIP}, model.ConfigTypeWebhookFile)
			if err != nil {
				return nil, errors.New("退役配置读取失败")
			}
			if len(rows) > 1 {
				return nil, errors.New("退役配置归属冲突")
			}
			if len(rows) == 1 {
				row := rows[0]
				current := configdao.GeneratedConfigOwnerOf(row)
				// A failed cancellation can leave an older intent beside a
				// restored canonical row at this address. The current complete
				// preflight snapshot, not the obsolete JSON, supplies the SQL
				// authority. Foreign/custom/replacement ambiguity still fails.
				if row == nil || currentRetired[row.ID] != current {
					return nil, errors.New("退役配置归属冲突")
				}
				intent.retireOwner = current
			}
		}
		result = append(result, intent)
	}
	// Stable order keeps retries and fault injection deterministic.
	sort.Slice(result, func(i, j int) bool {
		if result[i].owner.ID != result[j].owner.ID {
			return result[i].owner.ID < result[j].owner.ID
		}
		return result[i].encoded < result[j].encoded
	})
	return result, nil
}

// Clone and validate the complete metadata inventory before any visible write.
// This detects observed COUNT/OFFSET drift; it is not a SQL snapshot transaction.
func (a *alertManagerConfigCache) readAlertConfigSnapshot(ctx context.Context, kind int8) ([]*model.MonitorConfig, error) {
	if kind != model.ConfigTypeAlertManager && kind != model.ConfigTypeWebhookFile {
		return nil, errors.New("AM元数据类型无效")
	}
	if a.alertConfigDAO() == nil {
		return nil, errors.New("AM元数据存储不可用")
	}
	var result []*model.MonitorConfig
	expected, scanned := int64(-1), int64(0)
	seen := map[int]bool{}
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rows, total, err := a.alertConfigDAO().GetMonitorConfigList(ctx, &model.GetMonitorConfigListReq{ListReq: model.ListReq{Page: page, Size: 100}, ConfigType: &kind})
		if err != nil || total < 0 || expected >= 0 && total != expected {
			return nil, errors.Join(errors.New("AM元数据分页读取失败"), ctx.Err())
		}
		expected = total
		if int64(len(rows)) > expected-scanned {
			return nil, errors.New("AM元数据分页计数无效")
		}
		if len(rows) == 0 {
			if scanned != expected {
				return nil, errors.New("AM元数据分页提前结束")
			}
			break
		}
		for _, source := range rows {
			if source == nil || source.ID <= 0 || seen[source.ID] || source.ConfigType != kind {
				return nil, errors.New("AM元数据标识无效")
			}
			seen[source.ID] = true
			copy := *source
			result = append(result, &copy)
		}
		scanned += int64(len(rows))
		if scanned == expected {
			break
		}
		if len(rows) < 100 {
			return nil, errors.New("AM元数据分页不完整")
		}
	}
	return result, nil
}

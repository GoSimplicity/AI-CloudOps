package cache

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

// Read every page afresh for one caller. This function owns only pagination,
// identity validation and detachment; each publisher keeps its type/ownership
// policy and creation preflight. There is no cross-pool or cross-run cache.
// COUNT/OFFSET checks detect observed drift, not SQL snapshot isolation.
func readConfigMetadataSnapshot(ctx context.Context, dao configdao.MonitorConfigDAO, kind int8) ([]*model.MonitorConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if dao == nil {
		return nil, errors.New("配置元数据存储不可用")
	}
	var snapshot []*model.MonitorConfig
	expected, scanned := int64(-1), int64(0)
	seen := map[int]bool{}
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rows, total, err := dao.GetMonitorConfigList(ctx, &model.GetMonitorConfigListReq{ListReq: model.ListReq{Page: page, Size: 100}, ConfigType: &kind})
		if err != nil {
			return nil, errors.Join(errors.New("配置元数据读取失败"), ctx.Err())
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if total < 0 || expected >= 0 && total != expected {
			return nil, errors.New("配置元数据分页总数不一致")
		}
		expected = total
		if int64(len(rows)) > expected-scanned {
			return nil, errors.New("配置元数据分页超出总数")
		}
		if len(rows) == 0 {
			if scanned != expected {
				return nil, errors.New("配置元数据分页提前结束")
			}
			break
		}
		for _, source := range rows {
			if source == nil || source.ID <= 0 || seen[source.ID] {
				return nil, errors.New("配置元数据标识无效或重复")
			}
			seen[source.ID] = true
			copy := *source
			snapshot = append(snapshot, &copy)
		}
		scanned += int64(len(rows))
		if scanned == expected {
			break
		}
		if len(rows) < 100 {
			return nil, errors.New("配置元数据分页数量不足")
		}
	}
	return snapshot, nil
}

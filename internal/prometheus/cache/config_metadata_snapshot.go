package cache

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// Read every page afresh for one caller. The fetch closure supplies its query
// scope; this pager owns only completeness, positive unique IDs and detachment.
// COUNT/OFFSET detects observed drift, not SQL snapshot isolation.
func readConfigMetadataPages(ctx context.Context, fetch func(page, size int) ([]*model.MonitorConfig, int64, error)) ([]*model.MonitorConfig, error) {
	if fetch == nil {
		return nil, errors.New("配置元数据读取器不可用")
	}
	var snapshot []*model.MonitorConfig
	expected, scanned := int64(-1), int64(0)
	seen := map[int]bool{}
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rows, total, err := fetch(page, 100)
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

package cache

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
)

func cloneScrapePoolSnapshot(source *model.MonitorScrapePool) *model.MonitorScrapePool {
	copy := *source
	copy.PrometheusInstances = append(model.StringList(nil), source.PrometheusInstances...)
	copy.Tags = append(model.StringList(nil), source.Tags...)
	return &copy
}

// Read the full unfiltered inventory before any pool is published. The counters
// reject observed drift; COUNT/OFFSET does not provide transaction isolation.
func readScrapePoolSnapshot(ctx context.Context, dao scrapedao.ScrapePoolDAO) ([]*model.MonitorScrapePool, error) {
	if dao == nil {
		return nil, errors.New("采集池存储不可用")
	}
	return readScrapePoolPages(ctx, dao.GetMonitorScrapePoolList)
}

func readScrapePoolLifecycleSnapshot(ctx context.Context, dao scrapedao.ScrapePoolDAO) ([]*model.MonitorScrapePool, error) {
	if dao == nil {
		return nil, errors.New("采集池存储不可用")
	}
	return readScrapePoolPages(ctx, dao.GetMonitorScrapePoolLifecycleList)
}

func readScrapePoolPages(ctx context.Context, readPage func(context.Context, *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error)) ([]*model.MonitorScrapePool, error) {
	if readPage == nil {
		return nil, errors.New("采集池存储不可用")
	}
	var pools []*model.MonitorScrapePool
	expected, scanned := int64(-1), int64(0)
	seen := map[int]bool{}
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rows, total, err := readPage(ctx, &model.GetMonitorScrapePoolListReq{ListReq: model.ListReq{Page: page, Size: 100}})
		if err != nil || total < 0 || expected >= 0 && total != expected {
			return nil, errors.New("采集池分页读取失败")
		}
		expected = total
		if int64(len(rows)) > expected-scanned {
			return nil, errors.New("采集池分页总数无效")
		}
		if len(rows) == 0 {
			if scanned != expected {
				return nil, errors.New("采集池分页提前结束")
			}
			break
		}
		for _, row := range rows {
			if row == nil || row.ID <= 0 || seen[row.ID] {
				return nil, errors.New("采集池快照标识无效")
			}
			seen[row.ID] = true
			snapshot := cloneScrapePoolSnapshot(row)
			pools = append(pools, snapshot)
		}
		scanned += int64(len(rows))
		if scanned == expected {
			break
		}
		if len(rows) < 100 {
			return nil, errors.New("采集池分页数量不足")
		}
	}
	return pools, nil
}

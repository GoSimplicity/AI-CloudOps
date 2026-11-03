package cache

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
)

func cloneAlertPoolLifecycle(source *model.MonitorAlertManagerPool) *model.MonitorAlertManagerPool {
	copy := *source
	copy.AlertManagerInstances = append(model.StringList(nil), source.AlertManagerInstances...)
	copy.GroupBy = append(model.StringList(nil), source.GroupBy...)
	return &copy
}
func readAlertPoolLifecycleSnapshot(ctx context.Context, dao alertdao.AlertManagerPoolDAO) ([]*model.MonitorAlertManagerPool, error) {
	if dao == nil {
		return nil, errors.New("AM生命周期存储不可用")
	}
	var pools []*model.MonitorAlertManagerPool
	expected, scanned := int64(-1), int64(0)
	seen := map[int]bool{}
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rows, total, err := dao.GetMonitorAlertManagerPoolLifecycleList(ctx, &model.GetMonitorAlertManagerPoolListReq{ListReq: model.ListReq{Page: page, Size: 100}})
		if err != nil || total < 0 || expected >= 0 && total != expected {
			return nil, errors.Join(errors.New("AM生命周期分页读取失败"), ctx.Err())
		}
		expected = total
		if int64(len(rows)) > expected-scanned {
			return nil, errors.New("AM生命周期分页计数无效")
		}
		if len(rows) == 0 {
			if scanned != expected {
				return nil, errors.New("AM生命周期分页提前结束")
			}
			break
		}
		for _, row := range rows {
			if row == nil || row.ID <= 0 || seen[row.ID] {
				return nil, errors.New("AM生命周期标识无效")
			}
			seen[row.ID] = true
			pools = append(pools, cloneAlertPoolLifecycle(row))
		}
		scanned += int64(len(rows))
		if scanned == expected {
			break
		}
		if len(rows) < 100 {
			return nil, errors.New("AM生命周期分页不完整")
		}
	}
	return pools, nil
}
func (a *alertManagerConfigCache) RetireDeletedAlertManagerPool(ctx context.Context, poolID int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.alertPoolDAO == nil || poolID <= 0 {
		return errors.New("缺少AM删除授权")
	}
	pool, err := a.alertPoolDAO.GetMonitorAlertManagerPoolTombstone(ctx, poolID)
	if err != nil {
		return errors.Join(errors.New("无法确认AM删除记录"), ctx.Err())
	}
	if pool == nil || pool.ID != poolID || pool.DeletedAt == 0 {
		return errors.New("缺少真实AM删除授权")
	}
	return a.publishAlertPool(ctx, cloneAlertPoolLifecycle(pool))
}
func (a *alertManagerConfigCache) ReconcileDeletedAlertManagerPools(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	pools, err := readAlertPoolLifecycleSnapshot(ctx, a.alertPoolDAO)
	if err != nil {
		return err
	}
	var failures []error
	for _, pool := range pools {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		if pool.DeletedAt == 0 {
			continue
		}
		if err := a.publishAlertPool(ctx, pool); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(append(failures, ctx.Err())...)
}

package handler

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
)

var errCronLeaseBusy = errors.New("任务已有活跃执行租约，稍后重试")
var errCronLeaseLost = errors.New("任务执行租约已失效，当前尝试不得写入结果")

func (h *CronHandlers) maintainLease(ctx context.Context, cancelExecution context.CancelFunc, id int, owner string) func() error {
	stopped := make(chan struct{})
	finished := make(chan struct{})
	var once sync.Once
	var leaseError error
	go func() {
		defer close(finished)
		ticks := h.leaseTicks
		if ticks == nil {
			ticker := time.NewTicker(dao.CronLeaseRenewInterval)
			defer ticker.Stop()
			ticks = ticker.C
		}
		for {
			select {
			case <-stopped:
				return
			case <-ctx.Done():
				return
			case <-ticks:
				probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
				renewed, err := h.cronDAO.RenewCronJobLease(probeCtx, id, owner)
				cancel()
				if ctx.Err() != nil {
					return
				}
				if err != nil || !renewed {
					leaseError = errCronLeaseLost
					cancelExecution()
					return
				}
			}
		}
	}()
	return func() error { once.Do(func() { close(stopped) }); <-finished; return leaseError }
}

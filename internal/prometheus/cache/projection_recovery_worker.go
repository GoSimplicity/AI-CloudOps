package cache

import (
	"context"
	"errors"
	"sync"
	"time"

	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"go.uber.org/zap"
)

type PoolProjectionRecoverer interface {
	RecoverCommittedPool(context.Context, configdao.PoolReservation) error
}
type ProjectionRecoveryDispatcher interface {
	Recover(context.Context, configdao.PoolReservation) error
}
type projectionRecoveryDispatcher struct {
	families map[int8]PoolProjectionRecoverer
}

func NewProjectionRecoveryDispatcher(prom PrometheusConfigCache, am AlertManagerConfigCache, rule AlertRuleConfigCache, record RecordRuleConfigCache) (ProjectionRecoveryDispatcher, error) {
	result := &projectionRecoveryDispatcher{families: map[int8]PoolProjectionRecoverer{}}
	for family, value := range map[int8]interface{}{1: prom, 2: am, 3: rule, 4: record, 5: am} {
		recoverer, ok := value.(PoolProjectionRecoverer)
		if !ok || recoverer == nil {
			return nil, errors.New("持久投影恢复依赖不可用")
		}
		result.families[family] = recoverer
	}
	return result, nil
}
func (d *projectionRecoveryDispatcher) Recover(ctx context.Context, ref configdao.PoolReservation) error {
	recoverer := d.families[ref.Family]
	if recoverer == nil {
		return configdao.ErrJournalInvalid
	}
	return recoverer.RecoverCommittedPool(ctx, ref)
}

type ProjectionRecoveryOptions struct {
	Interval     time.Duration
	ItemTimeout  time.Duration
	CycleTimeout time.Duration
	PageSize     int
	MaxPerCycle  int
}

func DefaultProjectionRecoveryOptions() ProjectionRecoveryOptions {
	return ProjectionRecoveryOptions{Interval: 15 * time.Second, ItemTimeout: 30 * time.Second, CycleTimeout: time.Minute, PageSize: 100, MaxPerCycle: 1000}
}

type ProjectionRecoveryWorker struct {
	journal    configdao.ProjectionRecoveryJournalDAO
	dispatcher ProjectionRecoveryDispatcher
	logger     *zap.Logger
	options    ProjectionRecoveryOptions
	mu         sync.Mutex
	cursor     uint64
	through    uint64
	scanning   bool
}

func NewProjectionRecoveryWorker(journal configdao.ProjectionRecoveryJournalDAO, dispatcher ProjectionRecoveryDispatcher, logger *zap.Logger, options ProjectionRecoveryOptions) (*ProjectionRecoveryWorker, error) {
	if options.CycleTimeout == 0 {
		options.CycleTimeout = DefaultProjectionRecoveryOptions().CycleTimeout
	}
	if journal == nil || dispatcher == nil || logger == nil || options.Interval <= 0 || options.Interval > time.Hour || options.ItemTimeout <= 0 || options.ItemTimeout > 5*time.Minute || options.CycleTimeout <= 0 || options.CycleTimeout > 5*time.Minute || options.PageSize < 1 || options.PageSize > 100 || options.MaxPerCycle < 1 || options.MaxPerCycle > 10000 {
		return nil, errors.New("持久投影恢复配置无效")
	}
	return &ProjectionRecoveryWorker{journal: journal, dispatcher: dispatcher, logger: logger, options: options}, nil
}
func (w *ProjectionRecoveryWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.options.Interval)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		if err := w.RunOnce(ctx); err != nil && ctx.Err() == nil {
			w.logger.Warn("持久投影恢复轮次未全部完成", zap.String("category", "recovery_pending"))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
func (w *ProjectionRecoveryWorker) RunOnce(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, w.options.CycleTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	if !w.scanning {
		high, err := w.journal.CurrentProjectionRecoveryHighWater(ctx)
		if err != nil {
			return err
		}
		w.through = high
		w.cursor = 0
		w.scanning = true
	}
	failures := []error{}
	processed := 0
	for processed < w.options.MaxPerCycle {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		size := w.options.PageSize
		if remaining := w.options.MaxPerCycle - processed; size > remaining {
			size = remaining
		}
		refs, err := w.journal.ListCurrentUnprojectedPoolOperations(ctx, w.cursor, w.through, size)
		if err != nil {
			return errors.Join(append(failures, err)...)
		}
		if len(refs) > size {
			return errors.Join(append(failures, configdao.ErrJournalConflict)...)
		}
		// Check the whole page before dispatching any entry from it.
		last := w.cursor
		for _, ref := range refs {
			if ref.OperationID <= last || ref.OperationID > w.through || ref.Family < 1 || ref.Family > 5 || ref.PoolID <= 0 {
				return errors.Join(append(failures, configdao.ErrJournalConflict)...)
			}
			last = ref.OperationID
		}
		if len(refs) == 0 {
			w.scanning = false
			return errors.Join(append(failures, ctx.Err())...)
		}
		for _, ref := range refs {
			if err := ctx.Err(); err != nil {
				return errors.Join(append(failures, err)...)
			}
			// An attempted poison item cannot monopolise the next cycle's first page.
			w.cursor = ref.OperationID
			processed++
			itemCtx, itemCancel := context.WithTimeout(ctx, w.options.ItemTimeout)
			err := w.dispatcher.Recover(itemCtx, ref)
			itemCancel()
			if err != nil {
				failures = append(failures, errors.New("持久投影恢复条目未完成"))
				w.logger.Warn("持久投影恢复条目未完成", zap.Int8("family", ref.Family), zap.Int("pool_id", ref.PoolID), zap.Uint64("operation_id", ref.OperationID), zap.String("category", "recovery_pending"))
			}
		}
		if len(refs) < size || w.cursor == w.through {
			w.scanning = false
			return errors.Join(append(failures, ctx.Err())...)
		}
	}
	return errors.Join(append(failures, ctx.Err())...)
}

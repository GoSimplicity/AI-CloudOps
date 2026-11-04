package cache

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"gorm.io/gorm"
)

// This private call-scoped expectation carries no projection authority. It lets
// the unchanged renderer pipeline bind its eventual Reserve and Commit to the
// operation selected by recovery. SQL validates that expectation under control.
type projectionRecoveryExpectationKey struct{}

func recoveryExpectedOperation(ctx context.Context, family int8, poolID int) *uint64 {
	ref, ok := ctx.Value(projectionRecoveryExpectationKey{}).(configdao.PoolReservation)
	if !ok || ref.Family != family || ref.PoolID != poolID {
		return nil
	}
	id := ref.OperationID
	return &id
}
func recoveryCurrent(ctx context.Context, journal configdao.ProjectionPlanJournalDAO, ref configdao.PoolReservation) (bool, error) {
	receipt, err := journal.LoadLastCommittedProjection(ctx, ref.Family, ref.PoolID)
	if err != nil {
		return false, err
	}
	return receipt != nil && receipt.Reservation == ref && receipt.Phase == model.ConfigPublicationCommitted, nil
}
func recoveryScrapePool(ctx context.Context, dao scrapedao.ScrapePoolDAO, poolID int) (*model.MonitorScrapePool, error) {
	pool, err := dao.GetMonitorScrapePoolById(ctx, poolID)
	if err == nil && pool != nil && pool.ID == poolID && pool.DeletedAt == 0 {
		return cloneScrapePoolSnapshot(pool), nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("恢复源池读取失败")
	}
	tombstone, err := dao.GetMonitorScrapePoolTombstone(ctx, poolID)
	if err != nil || tombstone == nil || tombstone.ID != poolID || tombstone.DeletedAt == 0 {
		return nil, errors.New("恢复源池缺失且无真实删除授权")
	}
	return cloneScrapePoolSnapshot(tombstone), nil
}
func recoverScrapeCommitted(ctx context.Context, journal configdao.ProjectionPlanJournalDAO, dao scrapedao.ScrapePoolDAO, ref configdao.PoolReservation, publish func(context.Context, *model.MonitorScrapePool, uint64) error) error {
	reader, ok := journal.(configdao.ProjectionRecoveryJournalDAO)
	if !ok || reader == nil || dao == nil {
		return errors.New("持久投影恢复源依赖不可用")
	}
	current, err := recoveryCurrent(ctx, journal, ref)
	if err != nil || !current {
		return err
	}
	version, err := reader.ReadPoolProjectionSourceVersion(ctx, ref.Family, ref.PoolID)
	if err != nil {
		return err
	}
	pool, err := recoveryScrapePool(ctx, dao, ref.PoolID)
	if err != nil {
		return err
	}
	return publish(context.WithValue(ctx, projectionRecoveryExpectationKey{}, ref), pool, version)
}
func (p *prometheusConfigCache) RecoverCommittedPool(ctx context.Context, ref configdao.PoolReservation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ref.Family != 1 {
		return configdao.ErrJournalInvalid
	}
	if err := lockProjectionRecovery(ctx, &p.mu); err != nil {
		return err
	}
	defer p.mu.Unlock()
	return recoverScrapeCommitted(ctx, p.journal, p.scrapePoolDAO, ref, p.publishPromPool)
}
func (r *alertRuleConfigCache) RecoverCommittedPool(ctx context.Context, ref configdao.PoolReservation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ref.Family != 3 {
		return configdao.ErrJournalInvalid
	}
	if err := lockProjectionRecovery(ctx, &r.mu); err != nil {
		return err
	}
	defer r.mu.Unlock()
	return recoverScrapeCommitted(ctx, r.journal, r.scrapePoolDAO, ref, r.publishAlertRulePool)
}
func (r *recordRuleConfigCache) RecoverCommittedPool(ctx context.Context, ref configdao.PoolReservation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ref.Family != 4 {
		return configdao.ErrJournalInvalid
	}
	if err := lockProjectionRecovery(ctx, &r.mu); err != nil {
		return err
	}
	defer r.mu.Unlock()
	return recoverScrapeCommitted(ctx, r.journal, r.scrapePoolDAO, ref, r.publishRecordPool)
}
func (a *alertManagerConfigCache) RecoverCommittedPool(ctx context.Context, ref configdao.PoolReservation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ref.Family != 2 && ref.Family != 5 {
		return configdao.ErrJournalInvalid
	}
	if err := lockProjectionRecovery(ctx, &a.mu); err != nil {
		return err
	}
	defer a.mu.Unlock()
	reader, ok := a.journal.(configdao.ProjectionRecoveryJournalDAO)
	if !ok || reader == nil || a.alertPoolDAO == nil {
		return errors.New("AM恢复源依赖不可用")
	}
	current, err := recoveryCurrent(ctx, a.journal, ref)
	if err != nil || !current {
		return err
	}
	version, err := reader.ReadAMPoolSourceVersion(ctx, ref.PoolID)
	if err != nil {
		return err
	}
	pool, err := a.alertPoolDAO.GetAlertPoolByID(ctx, ref.PoolID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("AM恢复源池读取失败")
	}
	if pool == nil || err != nil {
		pool, err = a.alertPoolDAO.GetMonitorAlertManagerPoolTombstone(ctx, ref.PoolID)
		if err != nil || pool == nil || pool.DeletedAt == 0 {
			return errors.New("AM恢复源池缺失且无真实删除授权")
		}
	}
	if pool.ID != ref.PoolID {
		return configdao.ErrJournalConflict
	}
	pool = cloneAlertPoolLifecycle(pool)
	ctx = context.WithValue(ctx, projectionRecoveryExpectationKey{}, ref)
	if ref.Family == 2 {
		return a.publishAlertPool(ctx, pool, version)
	}
	// A subset never retires full state. A deleted pool has no desired webhook
	// subset; its full-family tombstone controller owns the actual raw retirement.
	groups := []*model.MonitorSendGroup{}
	if pool.DeletedAt == 0 {
		groups, err = a.readEnabledAlertGroups(ctx, pool.ID)
		if err != nil {
			return err
		}
		if _, _, err = a.renderAlertRoutes(pool, groups); err != nil {
			return err
		}
	}
	return a.generateWebhookFilesForSendGroups(ctx, pool, groups, version)
}

// Recovery has a bounded item deadline even when a normal publisher owns this
// mutex. A blocked Lock goroutine would outlive that deadline and later acquire
// authority, so acquisition itself polls TryLock under the caller's context.
func lockProjectionRecovery(ctx context.Context, mutex *sync.RWMutex) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if mutex.TryLock() {
			if err := ctx.Err(); err != nil {
				mutex.Unlock()
				return err
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

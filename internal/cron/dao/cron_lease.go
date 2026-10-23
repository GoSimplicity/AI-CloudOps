package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

const CronLeaseTTL = 30 * time.Second
const CronLeaseRenewInterval = 10 * time.Second

func (d *cronJobDAO) leaseClock() string {
	if d.db.Dialector.Name() == "sqlite" {
		return "CAST(strftime('%s','now') AS INTEGER)"
	}
	return "UNIX_TIMESTAMP()"
}

func (d *cronJobDAO) AcquireCronJobLease(ctx context.Context, id int, owner string) (bool, error) {
	if owner == "" || len(owner) > 64 {
		return false, fmt.Errorf("执行租约归属无效")
	}
	clock := d.leaseClock()
	result := d.db.WithContext(ctx).Model(&model.CronJob{}).
		Where("id = ? AND status IN ?", id, []model.CronJobStatus{model.CronJobStatusEnabled, model.CronJobStatusRunning}).
		Where("((status = ? AND lease_owner = '') OR (lease_owner <> '' AND lease_expires_at <= "+clock+"))", model.CronJobStatusEnabled).
		Updates(map[string]interface{}{"status": model.CronJobStatusRunning, "lease_owner": owner, "lease_expires_at": gorm.Expr(clock+" + ?", int64(CronLeaseTTL/time.Second))})
	return result.RowsAffected == 1, result.Error
}

func (d *cronJobDAO) ownedLease(ctx context.Context, id int, owner string) *gorm.DB {
	return d.db.WithContext(ctx).Model(&model.CronJob{}).Where("id = ? AND lease_owner = ? AND lease_expires_at > "+d.leaseClock(), id, owner)
}

func (d *cronJobDAO) RenewCronJobLease(ctx context.Context, id int, owner string) (bool, error) {
	query := d.ownedLease(ctx, id, owner)
	result := query.Update("lease_expires_at", gorm.Expr(d.leaseClock()+" + ?", int64(CronLeaseTTL/time.Second)))
	if result.Error != nil || result.RowsAffected > 0 {
		return result.RowsAffected == 1, result.Error
	}
	// MySQL默认报告变化行数，同一秒续租可能仍有效但未改变值
	var count int64
	err := d.ownedLease(ctx, id, owner).Count(&count).Error
	return count == 1, err
}

func (d *cronJobDAO) CompleteCronJobLease(ctx context.Context, id int, owner string, started *time.Time, status int8, duration int, output, errorMsg string) (bool, error) {
	updates := map[string]interface{}{
		"lease_owner": "", "lease_expires_at": 0,
		"status":          gorm.Expr("CASE WHEN status = ? THEN ? ELSE status END", model.CronJobStatusRunning, model.CronJobStatusEnabled),
		"last_run_status": status, "last_run_duration": duration, "last_run_output": output, "last_run_error": errorMsg,
		"run_count": gorm.Expr("run_count + 1"),
	}
	if started != nil {
		updates["last_run_time"] = started
	}
	if status == 1 {
		updates["success_count"] = gorm.Expr("success_count + 1")
	} else {
		updates["failure_count"] = gorm.Expr("failure_count + 1")
	}
	result := d.ownedLease(ctx, id, owner).Updates(updates)
	return result.RowsAffected == 1, result.Error
}

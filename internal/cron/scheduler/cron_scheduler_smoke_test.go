//go:build smoke

package scheduler

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestReloadPreservesRunningJobsAndSharedEntries(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	// 仅建立本场景所需的持久字段，使用实际DAO查询和状态条件更新
	if err := db.Exec(`CREATE TABLE cl_cron_jobs (id INTEGER PRIMARY KEY, name TEXT, schedule TEXT, job_type INTEGER, status INTEGER, deleted_at INTEGER DEFAULT 0, updated_at DATETIME)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO cl_cron_jobs (id,name,schedule,job_type,status) VALUES (1,'daily','@every 1h',2,1)`).Error; err != nil {
		t.Fatal(err)
	}
	store := dao.NewCronJobDAO(zap.NewNop(), db)
	shared := asynq.NewScheduler(asynq.RedisClientOpt{Addr: "127.0.0.1:1"}, nil)
	foreign, err := shared.Register("@every 1h", asynq.NewTask("other:task", nil))
	if err != nil {
		t.Fatal(err)
	}
	scheduler := NewCronScheduler(zap.NewNop(), store, shared, nil)
	scheduler.localExecAllowed = func() bool { return true } // job_type=2 需打开本地执行开关
	ctx := context.Background()
	if err := scheduler.loadAndScheduleJobs(ctx); err != nil {
		t.Fatal(err)
	}
	original := scheduler.entries[1].id
	for _, status := range []model.CronJobStatus{model.CronJobStatusRunning, model.CronJobStatusEnabled} {
		if err := store.UpdateCronJobStatus(ctx, 1, status); err != nil {
			t.Fatal(err)
		}
		if err := scheduler.loadAndScheduleJobs(ctx); err != nil {
			t.Fatal(err)
		}
		if scheduler.entries[1].id != original {
			t.Fatal("running job lost scheduling cadence")
		}
	}
	claimed, err := store.CompareAndSwapCronJobStatus(ctx, 1, model.CronJobStatusEnabled, model.CronJobStatusRunning)
	if err != nil || !claimed {
		t.Fatalf("claim=%v error=%v", claimed, err)
	}
	if claimed, err := store.CompareAndSwapCronJobStatus(ctx, 1, model.CronJobStatusEnabled, model.CronJobStatusRunning); err != nil || claimed {
		t.Fatalf("duplicate claim=%v error=%v", claimed, err)
	}
	if err := store.UpdateCronJobStatus(ctx, 1, model.CronJobStatusDisabled); err != nil {
		t.Fatal(err)
	}
	if restored, err := store.CompareAndSwapCronJobStatus(ctx, 1, model.CronJobStatusRunning, model.CronJobStatusEnabled); err != nil || restored {
		t.Fatalf("disabled job restored=%v error=%v", restored, err)
	}
	if err := scheduler.loadAndScheduleJobs(ctx); err != nil {
		t.Fatal(err)
	}
	if len(scheduler.entries) != 0 {
		t.Fatal("disabled job remains scheduled")
	}
	if err := shared.Unregister(original); err == nil {
		t.Fatal("disabled entry still registered")
	}
	if err := shared.Unregister(foreign); err != nil {
		t.Fatalf("shared scheduler entry was removed: %v", err)
	}
}

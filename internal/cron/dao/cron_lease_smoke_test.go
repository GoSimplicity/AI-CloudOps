//go:build smoke

package dao

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func leaseDatabase(t *testing.T) (*gorm.DB, CronJobDAO) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.Exec(`CREATE TABLE cl_cron_jobs(id INTEGER PRIMARY KEY,status INTEGER,deleted_at INTEGER DEFAULT 0,updated_at DATETIME,last_run_time DATETIME,last_run_status INTEGER,last_run_duration INTEGER,last_run_output TEXT,last_run_error TEXT,run_count INTEGER DEFAULT 0,success_count INTEGER DEFAULT 0,failure_count INTEGER DEFAULT 0);`).Error; err != nil {
		t.Fatal(err)
	}
	// 使用实际GORM为旧表补充新增列，默认归属为空且不改旧Running状态
	for _, field := range []string{"LeaseOwner", "LeaseExpiresAt"} {
		if err := db.Migrator().AddColumn(&model.CronJob{}, field); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Exec(`INSERT INTO cl_cron_jobs(id,status) VALUES(1,1);`).Error; err != nil {
		t.Fatal(err)
	}
	return db, NewCronJobDAO(zap.NewNop(), db)
}

func TestLeaseOnlyOneReplicaAcquires(t *testing.T) {
	db, store := leaseDatabase(t)
	var acquired atomic.Int32
	var failures atomic.Int32
	var wait sync.WaitGroup
	for i := 0; i < 24; i++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			client := NewCronJobDAO(zap.NewNop(), db)
			ok, err := client.AcquireCronJobLease(context.Background(), 1, fmt.Sprintf("replica-%d", index))
			if err != nil {
				failures.Add(1)
			}
			if ok {
				acquired.Add(1)
			}
		}(i)
	}
	wait.Wait()
	if acquired.Load() != 1 || failures.Load() != 0 {
		t.Fatalf("acquired=%d failures=%d", acquired.Load(), failures.Load())
	}
	job, err := store.GetCronJob(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != model.CronJobStatusRunning || job.LeaseOwner == "" || job.LeaseExpiresAt <= 0 {
		t.Fatal("lease was not persisted atomically")
	}
	if ok, err := store.RenewCronJobLease(context.Background(), 1, job.LeaseOwner); err != nil || !ok {
		t.Fatalf("same-second renew failed: %v", err)
	}
}

func TestExpiredLeaseRecoversAndOldOwnerCannotWrite(t *testing.T) {
	db, store := leaseDatabase(t)
	ctx := context.Background()
	if ok, err := store.AcquireCronJobLease(ctx, 1, "crashed-owner"); err != nil || !ok {
		t.Fatal("initial lease failed")
	}
	if ok, err := store.AcquireCronJobLease(ctx, 1, "new-owner"); err != nil || ok {
		t.Fatal("active lease was stolen")
	}
	if err := db.Exec(`UPDATE cl_cron_jobs SET lease_expires_at=CAST(strftime('%s','now') AS INTEGER)-1 WHERE id=1;`).Error; err != nil {
		t.Fatal(err)
	}
	if ok, err := store.AcquireCronJobLease(ctx, 1, "new-owner"); err != nil || !ok {
		t.Fatal("expired hard-crash lease could not recover")
	}
	if ok, err := store.RenewCronJobLease(ctx, 1, "crashed-owner"); err != nil || ok {
		t.Fatal("old owner renewed new lease")
	}
	now := time.Now()
	if ok, err := store.CompleteCronJobLease(ctx, 1, "crashed-owner", &now, 1, 1, "stale output", ""); err != nil || ok {
		t.Fatal("old owner completed new lease")
	}
	job, err := store.GetCronJob(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if job.LeaseOwner != "new-owner" || job.RunCount != 0 || job.LastRunOutput != "" {
		t.Fatal("stale completion changed current owner/results")
	}
	if ok, err := store.CompleteCronJobLease(ctx, 1, "new-owner", &now, 1, 1, "current output", ""); err != nil || !ok {
		t.Fatal("current owner could not finish")
	}
	job, err = store.GetCronJob(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if job.LeaseOwner != "" || job.LeaseExpiresAt != 0 || job.RunCount != 1 || job.LastRunOutput != "current output" || job.Status != model.CronJobStatusEnabled {
		t.Fatal("completion did not atomically release and persist")
	}
}

func TestAdministrativeStatusSurvivesLeaseCompletion(t *testing.T) {
	for _, status := range []model.CronJobStatus{model.CronJobStatusDisabled, model.CronJobStatusEnabled} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			_, store := leaseDatabase(t)
			ctx := context.Background()
			if ok, err := store.AcquireCronJobLease(ctx, 1, "owner"); err != nil || !ok {
				t.Fatal("claim failed")
			}
			if err := store.UpdateCronJobStatus(ctx, 1, model.CronJobStatusDisabled); err != nil {
				t.Fatal(err)
			}
			if status == model.CronJobStatusEnabled {
				if err := store.UpdateCronJobStatus(ctx, 1, status); err != nil {
					t.Fatal(err)
				}
			}
			if ok, err := store.AcquireCronJobLease(ctx, 1, "competitor"); err != nil || ok {
				t.Fatal("administrative enable bypassed active owner")
			}
			if ok, err := store.RenewCronJobLease(ctx, 1, "owner"); err != nil || !ok {
				t.Fatal("administrative status lost lease")
			}
			if ok, err := store.CompleteCronJobLease(ctx, 1, "owner", nil, 1, 1, "ok", ""); err != nil || !ok {
				t.Fatal("completion failed")
			}
			job, err := store.GetCronJob(ctx, 1)
			if err != nil {
				t.Fatal(err)
			}
			if job.Status != status {
				t.Fatal("completion overwrote administrative state")
			}
		})
	}
}

func TestLegacyRunningIsNotBlindlyTakenOver(t *testing.T) {
	db, store := leaseDatabase(t)
	if err := db.Exec(`UPDATE cl_cron_jobs SET status=3 WHERE id=1;`).Error; err != nil {
		t.Fatal(err)
	}
	if ok, err := store.AcquireCronJobLease(context.Background(), 1, "new-owner"); err != nil || ok {
		t.Fatal("unattributed legacy running job was stolen")
	}
}

//go:build smoke

package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUnownedRunningTaskIsNotAcknowledgedAsSuccessful(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	pool.SetMaxOpenConns(1)
	if err := db.Exec(`CREATE TABLE cl_cron_jobs(id INTEGER PRIMARY KEY,status INTEGER,deleted_at INTEGER DEFAULT 0);`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO cl_cron_jobs(id,status) VALUES (1,3);`).Error; err != nil {
		t.Fatal(err)
	}
	handler := &CronHandlers{logger: zap.NewNop(), cronDAO: dao.NewCronJobDAO(zap.NewNop(), db)}
	if err := handler.ProcessTask(context.Background(), asynq.NewTask("cron:task", []byte(`{"job_id":1}`))); err == nil {
		t.Fatal("hard-crash Running record was silently acknowledged without execution/recovery")
	}
}

func handlerLeaseDatabase(t *testing.T) (*gorm.DB, dao.CronJobDAO) {
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
	if err := db.Exec(`CREATE TABLE cl_cron_jobs(id INTEGER PRIMARY KEY,status INTEGER,deleted_at INTEGER DEFAULT 0,updated_at DATETIME,job_type INTEGER,http_method TEXT,http_url TEXT,timeout INTEGER DEFAULT 0,lease_owner TEXT NOT NULL DEFAULT '',lease_expires_at BIGINT NOT NULL DEFAULT 0,last_run_time DATETIME,last_run_status INTEGER,last_run_duration INTEGER,last_run_output TEXT,last_run_error TEXT,run_count INTEGER DEFAULT 0,success_count INTEGER DEFAULT 0,failure_count INTEGER DEFAULT 0);`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO cl_cron_jobs(id,status,job_type,http_method,http_url) VALUES(1,1,3,'GET','http://controlled.invalid');`).Error; err != nil {
		t.Fatal(err)
	}
	return db, dao.NewCronJobDAO(zap.NewNop(), db)
}

func TestExpiredHardCrashAttemptRunsAgainInsteadOfAcknowledging(t *testing.T) {
	db, store := handlerLeaseDatabase(t)
	ctx := context.Background()
	if ok, err := store.AcquireCronJobLease(ctx, 1, "hard-crashed-attempt"); err != nil || !ok {
		t.Fatal("failed to prepare crash lease")
	}
	if err := db.Exec(`UPDATE cl_cron_jobs SET lease_expires_at=CAST(strftime('%s','now') AS INTEGER)-1 WHERE id=1;`).Error; err != nil {
		t.Fatal(err)
	}
	executed := false
	executor := NewHTTPExecutor(zap.NewNop())
	executor.client.Transport = memoryTransport(func(*http.Request) (*http.Response, error) {
		executed = true
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("recovered")), Header: make(http.Header)}, nil
	})
	handler := &CronHandlers{logger: zap.NewNop(), cronDAO: store, httpExecutor: executor}
	if err := handler.ProcessTask(ctx, asynq.NewTask("cron:task", []byte(`{"job_id":1}`))); err != nil {
		t.Fatal(err)
	}
	job, err := store.GetCronJob(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !executed || job.Status != model.CronJobStatusEnabled || job.RunCount != 1 || job.LeaseOwner != "" || job.LastRunOutput != "recovered" {
		t.Fatal("hard-crash retry was not recovered and finalized")
	}
}

func TestActiveLeaseDeliveryRemainsRetryable(t *testing.T) {
	_, store := handlerLeaseDatabase(t)
	ctx := context.Background()
	if ok, err := store.AcquireCronJobLease(ctx, 1, "active-attempt"); err != nil || !ok {
		t.Fatal("failed to prepare lease")
	}
	handler := &CronHandlers{logger: zap.NewNop(), cronDAO: store}
	if err := handler.ProcessTask(ctx, asynq.NewTask("cron:task", []byte(`{"job_id":1}`))); !errors.Is(err, errCronLeaseBusy) {
		t.Fatalf("active retry was acknowledged/lost: %v", err)
	}
}

type renewalStore struct {
	dao.CronJobDAO
	renewed chan bool
	failure error
}

func (s renewalStore) RenewCronJobLease(ctx context.Context, id int, owner string) (bool, error) {
	if s.failure != nil {
		s.renewed <- false
		return false, s.failure
	}
	ok, err := s.CronJobDAO.RenewCronJobLease(ctx, id, owner)
	s.renewed <- ok
	return ok, err
}

func TestHeartbeatKeepsLeaseAndCancelsOnOwnershipLoss(t *testing.T) {
	for _, mode := range []string{"renew active", "lose owner", "database renewal failure"} {
		lose := mode == "lose owner"
		failed := mode == "database renewal failure"
		t.Run(mode, func(t *testing.T) {
			db, store := handlerLeaseDatabase(t)
			ticks := make(chan time.Time, 1)
			renewed := make(chan bool, 1)
			started := make(chan struct{})
			release := make(chan struct{})
			done := make(chan error, 1)
			executor := NewHTTPExecutor(zap.NewNop())
			executor.client.Transport = memoryTransport(func(request *http.Request) (*http.Response, error) {
				close(started)
				select {
				case <-release:
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("current result")), Header: make(http.Header)}, nil
				case <-request.Context().Done():
					return nil, request.Context().Err()
				}
			})
			var renewalError error
			if failed {
				renewalError = errors.New("private database credential marker")
			}
			handler := &CronHandlers{logger: zap.NewNop(), cronDAO: renewalStore{CronJobDAO: store, renewed: renewed, failure: renewalError}, httpExecutor: executor, leaseTicks: ticks}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() { done <- handler.ProcessTask(ctx, asynq.NewTask("cron:task", []byte(`{"job_id":1}`))) }()
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("execution did not start")
			}
			if lose {
				if err := db.Exec(`UPDATE cl_cron_jobs SET lease_expires_at=CAST(strftime('%s','now') AS INTEGER)-1 WHERE id=1;`).Error; err != nil {
					t.Fatal(err)
				}
				if ok, err := store.AcquireCronJobLease(ctx, 1, "replacement-owner"); err != nil || !ok {
					t.Fatal("replacement did not acquire expired lease")
				}
			} else {
				if err := db.Exec(`UPDATE cl_cron_jobs SET lease_expires_at=CAST(strftime('%s','now') AS INTEGER)+5 WHERE id=1;`).Error; err != nil {
					t.Fatal(err)
				}
			}
			ticks <- time.Now()
			select {
			case ok := <-renewed:
				if ok == (lose || failed) {
					t.Fatal("unexpected renewal ownership result")
				}
			case <-time.After(time.Second):
				t.Fatal("controlled heartbeat did not run")
			}
			if !lose && !failed {
				if ok, err := store.AcquireCronJobLease(ctx, 1, "competitor"); err != nil || ok {
					t.Fatal("heartbeat let competitor steal live lease")
				}
				close(release)
			}
			select {
			case err := <-done:
				if (lose || failed) && !errors.Is(err, errCronLeaseLost) {
					t.Fatalf("ownership loss was not retryable: %v", err)
				}
				if !lose && !failed && err != nil {
					t.Fatal(err)
				}
			case <-time.After(time.Second):
				t.Fatal("execution did not stop/finalize")
			}
			job, err := store.GetCronJob(ctx, 1)
			if err != nil {
				t.Fatal(err)
			}
			if lose {
				if job.LeaseOwner != "replacement-owner" || job.RunCount != 0 || job.LastRunOutput != "" {
					t.Fatal("old worker changed replacement owner/results")
				}
			} else if failed {
				if job.RunCount != 0 || job.LeaseOwner == "" {
					t.Fatal("failed heartbeat committed or released its uncertain lease")
				}
			} else if job.RunCount != 1 || job.LeaseOwner != "" {
				t.Fatal("healthy worker did not finalize lease")
			}
		})
	}
}

package scheduler

import (
	"context"
	"errors"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type jobsStub struct {
	dao.CronJobDAO
	jobs []*model.CronJob
	err  error
}

func (s *jobsStub) GetEnabledCronJobs(context.Context) ([]*model.CronJob, error) {
	return s.jobs, s.err
}

func TestReloadKeepsSingleEntryAndRemovesDeletedJob(t *testing.T) {
	// Register/Unregister 只操作内存，不启动调度器或访问 Redis
	scheduler := asynq.NewScheduler(asynq.RedisClientOpt{Addr: "127.0.0.1:1"}, nil)
	data := &jobsStub{jobs: []*model.CronJob{{Model: model.Model{ID: 1}, Name: "daily", Schedule: "@every 1h", JobType: model.CronJobTypeCommand}}}
	cs := NewCronScheduler(zap.NewNop(), data, scheduler, nil)
	for i := 0; i < 3; i++ {
		if err := cs.loadAndScheduleJobs(context.Background()); err != nil {
			t.Fatal(err)
		}
		if len(cs.entries) != 1 {
			t.Fatalf("unexpected entries: %d", len(cs.entries))
		}
	}
	original := cs.entries[1].id
	if err := cs.loadAndScheduleJobs(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cs.entries[1].id != original {
		t.Fatal("unchanged job was registered again")
	}
	data.jobs[0].Schedule = "@every 2h"
	if err := cs.loadAndScheduleJobs(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cs.entries[1].id == original {
		t.Fatal("changed schedule not replaced")
	}
	if err := scheduler.Unregister(original); err == nil {
		t.Fatal("old schedule still exists")
	}
	current := cs.entries[1].id
	data.err = errors.New("database unavailable")
	if err := cs.loadAndScheduleJobs(context.Background()); err == nil {
		t.Fatal("expected DAO error")
	}
	if cs.entries[1].id != current {
		t.Fatal("DAO failure changed active schedule")
	}
	data.err = nil
	data.jobs = nil
	if err := cs.loadAndScheduleJobs(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(cs.entries) != 0 {
		t.Fatal("deleted job remains")
	}
	if err := scheduler.Unregister(current); err == nil {
		t.Fatal("deleted job still registered")
	}
}

func TestRemoveUsesSchedulerEntryID(t *testing.T) {
	scheduler := asynq.NewScheduler(asynq.RedisClientOpt{Addr: "127.0.0.1:1"}, nil)
	data := &jobsStub{jobs: []*model.CronJob{{Model: model.Model{ID: 5}, Name: "job", Schedule: "@every 1h", JobType: model.CronJobTypeCommand}}}
	cs := NewCronScheduler(zap.NewNop(), data, scheduler, nil)
	if err := cs.loadAndScheduleJobs(context.Background()); err != nil {
		t.Fatal(err)
	}
	id := cs.entries[5].id
	if err := cs.RemoveScheduledJob(5); err != nil {
		t.Fatal(err)
	}
	if err := scheduler.Unregister(id); err == nil {
		t.Fatal("removed schedule still exists")
	}
	if err := cs.RemoveScheduledJob(5); err != nil {
		t.Fatal("removal should be idempotent")
	}
}

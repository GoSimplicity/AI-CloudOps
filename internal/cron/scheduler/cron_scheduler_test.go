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
	cs.localExecAllowed = func() bool { return true } // 本用例使用命令行任务，需打开开关
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
	cs.localExecAllowed = func() bool { return true }
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

func TestSchedulerSkipsLocalExecJobsWhenDisabled(t *testing.T) {
	scheduler := asynq.NewScheduler(asynq.RedisClientOpt{Addr: "127.0.0.1:1"}, nil)
	data := &jobsStub{jobs: []*model.CronJob{
		{Model: model.Model{ID: 1}, Name: "cmd", Schedule: "@every 1h", JobType: model.CronJobTypeCommand},
		{Model: model.Model{ID: 2}, Name: "http", Schedule: "@every 1h", JobType: model.CronJobTypeHTTP},
		{Model: model.Model{ID: 3}, Name: "script", Schedule: "@every 1h", JobType: model.CronJobTypeScript},
		{Model: model.Model{ID: 4}, Name: "ssh", Schedule: "@every 1h", JobType: model.CronJobTypeSSH},
		{Model: model.Model{ID: 5}, Name: "system", Schedule: "@every 1h", JobType: model.CronJobTypeSystem},
	}}
	allowed := false
	cs := NewCronScheduler(zap.NewNop(), data, scheduler, nil)
	cs.localExecAllowed = func() bool { return allowed }
	assertEntries := func(want ...int) {
		t.Helper()
		if len(cs.entries) != len(want) {
			t.Fatalf("allowed=%v: entries=%v, want ids %v", allowed, cs.entries, want)
		}
		for _, id := range want {
			if _, ok := cs.entries[id]; !ok {
				t.Fatalf("allowed=%v: job %d not scheduled", allowed, id)
			}
		}
	}

	// 开关关闭：命令行(2)/脚本(4)不登记，HTTP/SSH 照常
	if err := cs.loadAndScheduleJobs(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertEntries(2, 4)

	// 开关打开：全部非系统任务登记
	allowed = true
	if err := cs.loadAndScheduleJobs(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertEntries(1, 2, 3, 4)
	cmdEntry, scriptEntry := cs.entries[1].id, cs.entries[3].id

	// 运行中关闭开关：下一轮重载移除已登记的命令行/脚本任务
	allowed = false
	if err := cs.loadAndScheduleJobs(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertEntries(2, 4)
	for _, id := range []string{cmdEntry, scriptEntry} {
		if err := scheduler.Unregister(id); err == nil {
			t.Fatal("关闭开关后命令行/脚本任务仍在调度器中")
		}
	}
}

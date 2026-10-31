package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type recoveringDAO struct {
	dao.CronJobDAO
	mu       sync.Mutex
	calls    int
	failures int
}

func (d *recoveringDAO) GetEnabledCronJobs(ctx context.Context) ([]*model.CronJob, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls++
	if d.calls <= d.failures {
		return nil, errors.New("private database failure")
	}
	return []*model.CronJob{{Model: model.Model{ID: 1}, Name: "owned", Schedule: "@every 1h", JobType: model.CronJobTypeHTTP}}, nil
}

func TestInitialLoadRetriesAndReportsRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := &recoveringDAO{failures: 2}
	actual := asynq.NewScheduler(asynq.RedisClientOpt{Addr: "127.0.0.1:1"}, nil)
	cs := NewCronScheduler(zap.NewNop(), store, actual, nil, nil)
	waits := make(chan time.Duration, 4)
	proceed := make(chan struct{})
	cs.retryWait = func(ctx context.Context, duration time.Duration) error {
		waits <- duration
		select {
		case <-proceed:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	done := make(chan error, 1)
	go func() { done <- cs.StartScheduler(ctx) }()
	for i, expected := range []time.Duration{time.Second, 2 * time.Second, 30 * time.Second} {
		select {
		case got := <-waits:
			if got != expected {
				t.Fatalf("retry %d = %v, want %v", i, got, expected)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("调度器未进入预期等待")
		}
		status := cs.Status()
		if !status.Running {
			t.Fatal("调度器应在运行")
		}
		if i < 2 {
			if status.Ready || status.ConsecutiveFailures != i+1 || !status.LastSuccess.IsZero() {
				t.Fatalf("错误状态: %+v", status)
			}
			proceed <- struct{}{}
		} else if !status.Ready || status.ConsecutiveFailures != 0 || status.LastSuccess.IsZero() {
			t.Fatalf("恢复状态: %+v", status)
		}
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("等待未随取消退出")
	}
	if status := cs.Status(); status.Running || status.Ready {
		t.Fatalf("停止状态错误: %+v", status)
	}
	if err := actual.Unregister(cs.entries[1].id); err != nil {
		t.Fatal("恢复后任务未登记")
	}
}

func TestRetryDelayIsBoundedAndCancelable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := &recoveringDAO{failures: 100}
	cs := NewCronScheduler(zap.NewNop(), store, asynq.NewScheduler(asynq.RedisClientOpt{Addr: "127.0.0.1:1"}, nil), nil, nil)
	var delays []time.Duration
	cs.retryWait = func(_ context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		if len(delays) == 8 {
			cancel()
			return context.Canceled
		}
		return nil
	}
	if err := cs.StartScheduler(ctx); err != nil {
		t.Fatal(err)
	}
	expected := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 30 * time.Second, 30 * time.Second, 30 * time.Second}
	for i := range expected {
		if delays[i] != expected[i] {
			t.Fatalf("delay %d=%v", i, delays[i])
		}
	}
}

type replacementScheduler struct {
	active         map[string]string
	next           int
	failRegister   map[string]bool
	failUnregister bool
}

func (f *replacementScheduler) Register(spec string, _ *asynq.Task, _ ...asynq.Option) (string, error) {
	if f.failRegister[spec] {
		return "", errors.New("register failed")
	}
	if len(f.active) != 0 {
		return "", errors.New("overlapping registration")
	}
	f.next++
	id := fmt.Sprint(f.next)
	f.active[id] = spec
	return id, nil
}
func (f *replacementScheduler) Unregister(id string) error {
	if f.failUnregister {
		return errors.New("unregister failed")
	}
	if _, ok := f.active[id]; !ok {
		return errors.New("unknown id")
	}
	delete(f.active, id)
	return nil
}
func TestReplacementFailurePreservesPriorSchedule(t *testing.T) {
	for _, tc := range []struct {
		name, replacement                    string
		failUnregister, failNew, failRestore bool
	}{
		{name: "invalid-expression", replacement: "not-a-cron"},
		{name: "old-unregister-failed", replacement: "@every 2h", failUnregister: true},
		{name: "new-register-failed-restore", replacement: "@every 2h", failNew: true},
		{name: "new-and-restore-failed", replacement: "@every 2h", failNew: true, failRestore: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &replacementScheduler{active: map[string]string{}, failRegister: map[string]bool{}}
			cs := &CronScheduler{logger: zap.NewNop(), scheduler: fake}
			job := &model.CronJob{Model: model.Model{ID: 1}, Name: "owned", Schedule: "@every 1h", JobType: model.CronJobTypeHTTP}
			if err := cs.scheduleJob(job); err != nil {
				t.Fatal(err)
			}
			old := cs.entries[1]
			fake.failUnregister = tc.failUnregister
			if tc.failNew {
				fake.failRegister[tc.replacement] = true
			}
			if tc.failRestore {
				fake.failRegister[old.spec] = true
			}
			job.Schedule = tc.replacement
			if err := cs.scheduleJob(job); err == nil {
				t.Fatal("应拒绝替换失败")
			}
			if tc.failRestore {
				if len(fake.active) != 0 || len(cs.entries) != 0 {
					t.Fatal("恢复失败必须如实暴露缺失调度")
				}
				return
			}
			current, ok := cs.entries[1]
			if !ok || len(fake.active) != 1 || fake.active[current.id] != old.spec || current.signature != old.signature {
				t.Fatalf("旧任务未保留: %+v %+v", current, fake.active)
			}
		})
	}
}

// 使用真实 Asynq 内存调度器验证无效更新不会取消已登记条目
func TestInvalidReplacementKeepsActualSchedulerEntry(t *testing.T) {
	actual := asynq.NewScheduler(asynq.RedisClientOpt{Addr: "127.0.0.1:1"}, nil)
	cs := &CronScheduler{logger: zap.NewNop(), scheduler: actual}
	job := &model.CronJob{Model: model.Model{ID: 12}, Name: "owned", Schedule: "@every 1h", JobType: model.CronJobTypeHTTP}
	if err := cs.scheduleJob(job); err != nil {
		t.Fatal(err)
	}
	old := cs.entries[job.ID].id
	job.Schedule = "TZ=Asia/Shanghai"
	if err := cs.scheduleJob(job); err == nil {
		t.Fatal("无效表达式应被拒绝")
	}
	if cs.entries[job.ID].id != old {
		t.Fatal("旧条目标识发生变化")
	}
	if err := actual.Unregister(old); err != nil {
		t.Fatal("旧条目不再可用")
	}
}

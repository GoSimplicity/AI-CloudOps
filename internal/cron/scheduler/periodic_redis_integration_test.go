//go:build integration

package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/task"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// 验证真实周期调度在失败归档后仍能产生独立的新任务
func TestPeriodicTaskContinuesAfterArchive(t *testing.T) {
	if os.Getenv("CLOUDOPS_TEST_CRON_ASYNC") != "true" {
		t.Skip("requires isolated Redis acceptance flag")
	}
	name := fmt.Sprintf("cloudops-cron-periodic-%d", time.Now().UnixNano())
	command := func(args ...string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	if out, err := command("run", "-d", "--pull=never", "--name", name, "--label", "cloudops.test.run="+name, "-p", "127.0.0.1::6379", "redis:7.4.11-alpine", "redis-server", "--save", "", "--appendonly", "no"); err != nil {
		t.Fatalf("start owned Redis: %s: %v", out, err)
	}
	t.Cleanup(func() {
		label, err := command("inspect", "--format", `{{index .Config.Labels "cloudops.test.run"}}`, name)
		if err != nil || label != name {
			t.Error("owned Redis label mismatch; cleanup refused")
			return
		}
		if out, err := command("rm", "-f", name); err != nil {
			t.Errorf("remove owned Redis: %s: %v", out, err)
		}
	})
	address, err := command("port", name, "6379/tcp")
	if err != nil {
		t.Fatal(err)
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil || host != "127.0.0.1" {
		t.Fatal("Redis must bind loopback")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	client := redis.NewClient(&redis.Options{Addr: address})
	defer client.Close()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for client.Ping(ctx).Err() != nil {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatal("Redis not ready")
		}
	}
	opt := asynq.RedisClientOpt{Addr: address}
	shared := asynq.NewScheduler(opt, nil)
	cron := &CronScheduler{logger: zap.NewNop(), scheduler: shared}
	job := &model.CronJob{Model: model.Model{ID: 42}, Name: "local-owned-test", Schedule: "@every 1s", JobType: model.CronJobTypeHTTP, MaxRetry: 0}
	if err := cron.scheduleJob(job); err != nil {
		t.Fatal(err)
	}
	server := asynq.NewServer(opt, asynq.Config{Concurrency: 1, ShutdownTimeout: time.Second})
	if err := server.Start(asynq.HandlerFunc(func(_ context.Context, task *asynq.Task) error { return errors.New("local simulated failure") })); err != nil {
		t.Fatal(err)
	}
	defer server.Shutdown()
	if err := shared.Start(); err != nil {
		t.Fatal(err)
	}
	defer shared.Shutdown()
	inspector := asynq.NewInspector(opt)
	defer inspector.Close()
	for {
		archived, err := inspector.ListArchivedTasks("default")
		if err == nil && len(archived) >= 2 {
			ids := map[string]bool{}
			for _, entry := range archived {
				var payload task.CronTaskPayload
				if err := json.Unmarshal(entry.Payload, &payload); err != nil {
					t.Fatal(err)
				}
				if payload.JobID != job.ID || entry.MaxRetry != 0 {
					t.Fatal("周期载荷或重试预算错误")
				}
				ids[entry.ID] = true
			}
			if len(ids) < 2 {
				t.Fatal("归档后没有独立任务 ID")
			}
			return
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatal("归档后下一周期未生成独立任务")
		}
	}
}

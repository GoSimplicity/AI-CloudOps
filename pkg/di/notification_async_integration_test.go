//go:build integration

package di

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/app"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// 独立 Redis 必须由本测试创建；不接受外部 Redis 地址以免误碰业务队列
func TestNotificationAsyncIsolatedRedis(t *testing.T) {
	if os.Getenv("CLOUDOPS_TEST_NOTIFICATION_ASYNC") != "true" {
		t.Skip("requires explicit isolated Redis acceptance flag")
	}
	nonce := fmt.Sprintf("cloudops-notify-async-%d", time.Now().UnixNano())
	command := func(args ...string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		output, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
		return strings.TrimSpace(string(output)), err
	}
	if output, err := command("run", "-d", "--pull=never", "--name", nonce, "--label", "cloudops.test=notification-async", "--label", "cloudops.test.run="+nonce, "-p", "127.0.0.1::6379", "redis:7.4.11-alpine", "redis-server", "--save", "", "--appendonly", "no"); err != nil {
		t.Fatalf("start owned Redis: %s: %v", output, err)
	}
	t.Cleanup(func() {
		owner, err := command("inspect", "--format", `{{index .Config.Labels "cloudops.test.run"}}`, nonce)
		if err != nil || owner != nonce {
			t.Errorf("refuse cleanup: Redis owner label mismatch: %v", err)
			return
		}
		if output, err := command("rm", "-f", nonce); err != nil {
			t.Errorf("remove owned Redis: %s: %v", output, err)
		}
	})
	address, err := command("port", nonce, "6379/tcp")
	if err != nil {
		t.Fatal(err)
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil || host != "127.0.0.1" {
		t.Fatal("owned Redis must bind loopback only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	redisClient := redis.NewClient(&redis.Options{Addr: address})
	defer redisClient.Close()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for redisClient.Ping(ctx).Err() != nil {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatal("owned Redis not ready")
		}
	}
	var mu sync.Mutex
	received := map[string]int{}
	inbox := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/token":
			_, _ = io.WriteString(w, `{"code":0,"tenant_access_token":"local-fake-token","expire":7200}`)
		case "/messages":
			var message struct {
				Recipient string `json:"receive_id"`
			}
			if json.NewDecoder(r.Body).Decode(&message) != nil {
				w.WriteHeader(400)
				return
			}
			mu.Lock()
			received[message.Recipient]++
			mu.Unlock()
			_, _ = io.WriteString(w, `{"code":999,"msg":"owned fake rejection"}`)
		default:
			t.Error("unexpected fake provider endpoint")
			w.WriteHeader(404)
		}
	}))
	defer inbox.Close()
	option := asynq.RedisClientOpt{Addr: address}
	client := asynq.NewClient(option)
	manager, err := notification.NewManager(asyncInboxConfig{inbox.URL}, client, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	cfg := workerConfig(manager)
	// 只缩短测试队列扫描周期，实际重试间隔直接使用生产 RetryDelayFunc，不替换为假函数
	cfg.DelayedTaskCheckInterval = 50 * time.Millisecond
	cfg.ShutdownTimeout = time.Second
	worker := asynq.NewServer(option, cfg)
	processor := notification.NewProcessor(manager, zap.NewNop())
	mux := app.NewTaskMux(asynq.HandlerFunc(func(context.Context, *asynq.Task) error { t.Error("notification routed to Cron"); return nil }), processor)
	if err := worker.Start(mux); err != nil {
		t.Fatal(err)
	}
	defer worker.Shutdown()
	inspector := asynq.NewInspector(option)
	defer inspector.Close()
	for _, retries := range []int{0, 2} {
		t.Run(fmt.Sprintf("extra-retries-%d", retries), func(t *testing.T) {
			interval := 0
			recipient := fmt.Sprintf("ou_retry%d", retries)
			request := &notification.SendRequest{MessageID: fmt.Sprintf("%s-%d", nonce, retries), Channel: "feishu", RecipientAddr: recipient, Content: "owned local inbox acceptance", MaxRetries: &retries, RetryIntervalMinutes: &interval}
			if err := manager.SendNotificationAsync(ctx, request, 0); err != nil {
				t.Fatal(err)
			}
			for {
				info, err := inspector.GetTaskInfo("default", request.MessageID)
				if err == nil && info.State == asynq.TaskStateArchived {
					mu.Lock()
					count := received[recipient]
					mu.Unlock()
					if info.ID != request.MessageID || info.MaxRetry != retries || info.Retried != retries || count != retries+1 {
						t.Fatalf("retry contract: max=%d retried=%d inbox=%d", info.MaxRetry, info.Retried, count)
					}
					break
				}
				select {
				case <-ticker.C:
				case <-ctx.Done():
					t.Fatal("notification did not finish in owned queue")
				}
			}
			// 生产 hook 在同一真实请求上独立核实分钟单位，无需等待一分钟
			minute := 1
			request.RetryIntervalMinutes = &minute
			task, err := notification.CreateSendNotificationTask(request, nil)
			if err != nil {
				t.Fatal(err)
			}
			if delay := cfg.RetryDelayFunc(0, fmt.Errorf("fake"), task); delay != time.Minute {
				t.Fatalf("minute conversion=%s", delay)
			}
		})
	}
}

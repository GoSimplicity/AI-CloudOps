package retry

import (
	"context"
	"errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sync/atomic"
	"testing"
	"time"
)

// 自定义错误类型
var (
	errRetryable    = errors.New("retryable")
	errNonRetryable = errors.New("non-retryable")
)

func TestRunRetryWithConcurrency(t *testing.T) {
	t.Run("all tasks succeed without retry", func(t *testing.T) {
		var executed int32
		tasks := []WrapperTask{
			{
				Backoff: DefaultBackoff,
				Task: func(ctx context.Context) error {
					atomic.AddInt32(&executed, 1)
					return nil
				},
				RetryCheck: func(err error) bool { return false },
			},
			{
				Backoff: DefaultBackoff,
				Task: func(ctx context.Context) error {
					atomic.AddInt32(&executed, 1)
					return nil
				},
				RetryCheck: func(err error) bool { return false },
			},
		}

		err := RunRetryWithConcurrency(context.Background(), 2, tasks)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if executed != int32(len(tasks)) {
			t.Fatalf("expected executed=%d, got %d", len(tasks), executed)
		}
	})

	t.Run("retry when error is retryable", func(t *testing.T) {
		var attempts int32
		tasks := []WrapperTask{
			{
				Backoff: DefaultBackoff,
				Task: func(ctx context.Context) error {
					atomic.AddInt32(&attempts, 1)
					if attempts < 3 {
						return errRetryable
					}
					return nil
				},
				RetryCheck: func(err error) bool { return err == errRetryable },
			},
		}
		err := RunRetryWithConcurrency(context.Background(), 1, tasks)
		if err != nil {
			t.Fatalf("expected success after retries, got %v", err)
		}
		if attempts != 3 {
			t.Fatalf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("fail fast on non-retryable error", func(t *testing.T) {
		var attempts int32
		tasks := []WrapperTask{
			{
				Backoff: DefaultBackoff,
				Task: func(ctx context.Context) error {
					atomic.AddInt32(&attempts, 1)
					return errNonRetryable
				},
				RetryCheck: func(err error) bool { return err == errRetryable },
			},
		}

		err := RunRetryWithConcurrency(context.Background(), 1, tasks)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if attempts != 1 {
			t.Fatalf("expected only 1 attempt, got %d", attempts)
		}
	})

	t.Run("context cancellation stops execution", func(t *testing.T) {
		var attempts int32
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		tasks := []WrapperTask{
			{
				Backoff: DefaultBackoff,
				Task: func(ctx context.Context) error {
					atomic.AddInt32(&attempts, 1)
					select {
					case <-time.After(300 * time.Millisecond):
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				},
				RetryCheck: func(err error) bool { return false },
			},
		}

		err := RunRetryWithConcurrency(ctx, 1, tasks)

		if !wait.Interrupted(err) {
			t.Fatalf("expected wait to be interrupted, got %v", err)
		}
		if attempts != 1 {
			t.Fatalf("expected only 1 attempt, got %d", attempts)
		}
	})

	t.Run("concurrency limit works", func(t *testing.T) {
		var concurrent int32
		var maxConcurrent int32

		taskFunc := func(ctx context.Context) error {
			c := atomic.AddInt32(&concurrent, 1)
			if c > maxConcurrent {
				atomic.StoreInt32(&maxConcurrent, c)
			}
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&concurrent, -1)
			return nil
		}

		tasks := []WrapperTask{}
		for i := 0; i < 5; i++ {
			tasks = append(tasks, WrapperTask{
				Backoff: DefaultBackoff,
				Task:    taskFunc,
			})
		}

		err := RunRetryWithConcurrency(context.Background(), 2, tasks)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if maxConcurrent > 2 {
			t.Fatalf("expected max concurrent <= 2, got %d", maxConcurrent)
		}
	})
}

// 测试用短 backoff
func waitBackoffForTest() wait.Backoff {
	return wait.Backoff{
		Steps:    5,
		Duration: 50 * time.Millisecond,
		Factor:   1.0,
		Jitter:   0.0,
	}
}

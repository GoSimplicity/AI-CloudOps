package retry

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/wait"
)

var DefaultBackoff = wait.Backoff{
	Steps:    5,
	Duration: 200 * time.Millisecond,
	Factor:   2.0,
	Jitter:   0.1,
}

type WrapperTask struct {
	Task       func(ctx context.Context) error
	Backoff    wait.Backoff
	RetryCheck func(error) bool
}

func RunRetryWithConcurrency(ctx context.Context, concurrency int, tasks []WrapperTask) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(concurrency)

	for i := 0; i < len(tasks); i++ {
		task := tasks[i]

		eg.Go(func() error {
			// 兜底 DefaultBackoff
			backoffCfg := task.Backoff
			if backoffCfg.Steps == 0 {
				backoffCfg = DefaultBackoff
			}

			err := wait.ExponentialBackoffWithContext(ctx, backoffCfg,
				func(ctx context.Context) (done bool, err error) {
					select {
					case <-ctx.Done():
						return false, ctx.Err()
					default:
					}

					if e := task.Task(ctx); e != nil {
						if task.RetryCheck != nil && task.RetryCheck(e) {
							return false, nil
						}
						return false, e
					}
					return true, nil
				},
			)
			return err
		})
	}
	return eg.Wait()
}

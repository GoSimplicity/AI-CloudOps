package scheduler

import (
	"fmt"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type retryScheduler struct {
	options [][]asynq.Option
	removed []string
}

func (s *retryScheduler) Register(_ string, _ *asynq.Task, options ...asynq.Option) (string, error) {
	s.options = append(s.options, options)
	return fmt.Sprint(len(s.options)), nil
}
func (s *retryScheduler) Unregister(id string) error { s.removed = append(s.removed, id); return nil }
func TestScheduledRetriesReachSDKAndReloadOnChange(t *testing.T) {
	for _, count := range []int{0, 3, 10} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			target := &retryScheduler{}
			scheduler := &CronScheduler{logger: zap.NewNop(), scheduler: target}
			job := &model.CronJob{Model: model.Model{ID: 1}, Schedule: "@every 1h", JobType: model.CronJobTypeCommand, MaxRetry: count}
			if err := scheduler.scheduleJob(job); err != nil {
				t.Fatal(err)
			}
			found := false
			for _, option := range target.options[0] {
				if option.Type() == asynq.MaxRetryOpt {
					found = true
					if option.Value().(int) != count {
						t.Fatal("wrong scheduled retry budget")
					}
				}
			}
			if !found {
				t.Fatal("scheduled task would use SDK default 25")
			}
			if err := scheduler.scheduleJob(job); err != nil {
				t.Fatal(err)
			}
			if len(target.options) != 1 {
				t.Fatal("unchanged budget replaced registration")
			}
			job.MaxRetry = count + 1
			if err := scheduler.scheduleJob(job); err != nil {
				t.Fatal(err)
			}
			if len(target.options) != 2 || len(target.removed) != 1 {
				t.Fatal("changed budget did not replace registration")
			}
		})
	}
}

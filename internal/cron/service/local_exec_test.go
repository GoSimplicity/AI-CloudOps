package service

import (
	"context"
	"errors"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/task"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type localExecStore struct {
	dao.CronJobDAO
	created, updated, job *model.CronJob
}

func (s *localExecStore) CreateCronJob(_ context.Context, job *model.CronJob) error {
	s.created = job
	return nil
}
func (s *localExecStore) UpdateCronJob(_ context.Context, job *model.CronJob, _ *int) error {
	s.updated = job
	return nil
}
func (s *localExecStore) GetCronJob(context.Context, int) (*model.CronJob, error) { return s.job, nil }

func newLocalExecService(allowed bool, store *localExecStore, queue *retryQueue) *cronService {
	return &cronService{logger: zap.NewNop(), cronDAO: store, client: queue, localExecAllowed: func() bool { return allowed }}
}

// 各类型满足 validateBasicJobConfig 的最小字段
func jobFields(jobType model.CronJobType) (command, httpURL, script string, sshID *int, sshCommand string) {
	id := 1
	switch jobType {
	case model.CronJobTypeCommand:
		return "true", "", "", nil, ""
	case model.CronJobTypeScript:
		return "", "", "true", nil, ""
	case model.CronJobTypeHTTP:
		return "", "http://fake.invalid", "", nil, ""
	case model.CronJobTypeSSH:
		return "", "", "", &id, "true"
	}
	return "", "", "", nil, ""
}

func createReq(jobType model.CronJobType) *model.CreateCronJobReq {
	command, httpURL, script, sshID, sshCommand := jobFields(jobType)
	return &model.CreateCronJobReq{Name: "job", JobType: jobType, Schedule: "@every 1h", Command: command, HTTPUrl: httpURL, ScriptType: "bash", ScriptContent: script, SSHResourceID: sshID, SSHCommand: sshCommand}
}

func updateReq(jobType model.CronJobType) *model.UpdateCronJobReq {
	command, httpURL, script, sshID, sshCommand := jobFields(jobType)
	return &model.UpdateCronJobReq{ID: 1, Name: "job", JobType: jobType, Schedule: "@every 1h", Command: command, HTTPUrl: httpURL, ScriptType: "bash", ScriptContent: script, SSHResourceID: sshID, SSHCommand: sshCommand}
}

var allJobTypes = []model.CronJobType{model.CronJobTypeCommand, model.CronJobTypeHTTP, model.CronJobTypeScript, model.CronJobTypeSSH}

func TestCreateCronJobLocalExecSwitch(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		for _, jobType := range allJobTypes {
			store := &localExecStore{}
			err := newLocalExecService(allowed, store, nil).CreateCronJob(context.Background(), createReq(jobType))
			blocked := !allowed && task.IsLocalExecJobType(jobType)
			if blocked {
				if !errors.Is(err, task.ErrLocalExecDisabled) || store.created != nil {
					t.Fatalf("allowed=%v type=%d: want rejection without persisting, err=%v", allowed, jobType, err)
				}
				continue
			}
			if err != nil || store.created == nil || store.created.JobType != jobType {
				t.Fatalf("allowed=%v type=%d: create behaviour changed, err=%v", allowed, jobType, err)
			}
		}
	}
}

func TestUpdateCronJobLocalExecSwitch(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		for _, jobType := range allJobTypes {
			store := &localExecStore{}
			err := newLocalExecService(allowed, store, nil).UpdateCronJob(context.Background(), updateReq(jobType))
			blocked := !allowed && task.IsLocalExecJobType(jobType)
			if blocked {
				if !errors.Is(err, task.ErrLocalExecDisabled) || store.updated != nil {
					t.Fatalf("allowed=%v type=%d: want rejection without persisting, err=%v", allowed, jobType, err)
				}
				continue
			}
			if err != nil || store.updated == nil || store.updated.JobType != jobType {
				t.Fatalf("allowed=%v type=%d: update behaviour changed, err=%v", allowed, jobType, err)
			}
		}
	}
}

func TestCreateCronJobRejectsMissingJobType(t *testing.T) {
	store := &localExecStore{}
	if err := newLocalExecService(true, store, nil).CreateCronJob(context.Background(), createReq(0)); err == nil || store.created != nil {
		t.Fatal("job_type 0 must be rejected so the DB column default can never apply")
	}
}

func TestTriggerCronJobLocalExecSwitch(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		for _, jobType := range allJobTypes {
			queue := &retryQueue{}
			store := &localExecStore{job: &model.CronJob{Model: model.Model{ID: 1}, Status: model.CronJobStatusEnabled, JobType: jobType}}
			err := newLocalExecService(allowed, store, queue).TriggerCronJob(context.Background(), 1)
			blocked := !allowed && task.IsLocalExecJobType(jobType)
			if blocked {
				if !errors.Is(err, task.ErrLocalExecDisabled) || queue.options != nil {
					t.Fatalf("allowed=%v type=%d: manual trigger must not enqueue, err=%v", allowed, jobType, err)
				}
				continue
			}
			if err != nil || queue.options == nil {
				t.Fatalf("allowed=%v type=%d: trigger behaviour changed, err=%v", allowed, jobType, err)
			}
		}
	}
}

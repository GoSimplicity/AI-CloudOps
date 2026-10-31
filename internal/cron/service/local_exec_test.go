package service

import (
	"context"
	"errors"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/task"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type localExecStore struct {
	dao.CronJobDAO
	created, updated, job *model.CronJob
	status                *model.CronJobStatus
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
func (s *localExecStore) UpdateCronJobStatus(_ context.Context, _ int, status model.CronJobStatus) error {
	s.status = &status
	return nil
}

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

func TestGetCronJobListReportsAllowLocalExec(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		allowed := allowed
		svc := &cronService{
			logger:           zap.NewNop(),
			cronDAO:          &listOnlyDAO{},
			localExecAllowed: func() bool { return allowed },
		}
		resp, err := svc.GetCronJobList(context.Background(), &model.GetCronJobListReq{})
		if err != nil {
			t.Fatal(err)
		}
		if resp.AllowLocalExec != allowed || resp.Total != 0 {
			t.Fatalf("allowed=%v got %+v", allowed, resp)
		}
	}
}

type listOnlyDAO struct{ dao.CronJobDAO }

func (listOnlyDAO) GetCronJobList(context.Context, *model.GetCronJobListReq) ([]*model.CronJob, int64, error) {
	return nil, 0, nil
}

func TestEnableCronJobLocalExecSwitch(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		for _, jobType := range append(allJobTypes, model.CronJobTypeSystem) {
			store := &localExecStore{job: &model.CronJob{Model: model.Model{ID: 1}, Status: model.CronJobStatusDisabled, JobType: jobType, Schedule: "@every 1h"}}
			err := newLocalExecService(allowed, store, nil).EnableCronJob(context.Background(), 1)
			if !allowed && task.IsLocalExecJobType(jobType) {
				if !errors.Is(err, task.ErrLocalExecDisabled) || store.status != nil {
					t.Fatalf("allowed=%v type=%d: enable must be rejected without status change, err=%v", allowed, jobType, err)
				}
				continue
			}
			if err != nil || store.status == nil || *store.status != model.CronJobStatusEnabled {
				t.Fatalf("allowed=%v type=%d: enable behaviour changed, err=%v", allowed, jobType, err)
			}
		}
	}
}

// P3-3：开关打开时，命令行任务的自定义环境变量拒绝 LD_*、DYLD_*、BASH_ENV、ENV、IFS（不区分大小写）
func TestCronJobRejectsDangerousTaskEnv(t *testing.T) {
	for _, name := range []string{"LD_PRELOAD", "ld_library_path", "DYLD_INSERT_LIBRARIES", "BASH_ENV", "env", "IFS", "LD_PRELOAD=/tmp/x.so"} {
		env := model.KeyValueList{{Key: name, Value: "fake-value"}}

		store := &localExecStore{}
		req := createReq(model.CronJobTypeCommand)
		req.Environment = env
		if err := newLocalExecService(true, store, nil).CreateCronJob(context.Background(), req); !errors.Is(err, task.ErrForbiddenTaskEnv) || store.created != nil {
			t.Fatalf("create %q: want ErrForbiddenTaskEnv and no write, got %v", name, err)
		}

		store = &localExecStore{job: &model.CronJob{JobType: model.CronJobTypeCommand}}
		ureq := updateReq(model.CronJobTypeCommand)
		ureq.Environment = env
		if err := newLocalExecService(true, store, nil).UpdateCronJob(context.Background(), ureq); !errors.Is(err, task.ErrForbiddenTaskEnv) || store.updated != nil {
			t.Fatalf("update %q: want ErrForbiddenTaskEnv and no write, got %v", name, err)
		}
	}

	store := &localExecStore{}
	req := createReq(model.CronJobTypeCommand)
	req.Environment = model.KeyValueList{{Key: "APP_MODE", Value: "x"}, {Key: "ENVIRONMENT", Value: "dev"}}
	if err := newLocalExecService(true, store, nil).CreateCronJob(context.Background(), req); err != nil || store.created == nil {
		t.Fatalf("普通变量名应允许创建, got %v", err)
	}
}

// P2-2：前端新增环境变量会插入 {key:”,value:”} 空行，保存时丢弃而不是拒绝（创建、更新两条路径）
func TestCronJobSaveDropsBlankEnvRows(t *testing.T) {
	env := func() model.KeyValueList {
		return model.KeyValueList{{Key: "", Value: ""}, {Key: "APP_MODE", Value: "x"}, {Key: "  ", Value: ""}}
	}
	store := &localExecStore{}
	req := createReq(model.CronJobTypeCommand)
	req.Environment = env()
	if err := newLocalExecService(true, store, nil).CreateCronJob(context.Background(), req); err != nil || store.created == nil {
		t.Fatalf("create: 空行不应导致拒绝, got %v", err)
	}
	if got := store.created.Environment; len(got) != 1 || got[0].Key != "APP_MODE" {
		t.Fatalf("create: 空行应在保存前丢弃, got %v", got)
	}

	store = &localExecStore{job: &model.CronJob{JobType: model.CronJobTypeCommand}}
	ureq := updateReq(model.CronJobTypeCommand)
	ureq.Environment = env()
	if err := newLocalExecService(true, store, nil).UpdateCronJob(context.Background(), ureq); err != nil || store.updated == nil {
		t.Fatalf("update: 空行不应导致拒绝, got %v", err)
	}
	if got := store.updated.Environment; len(got) != 1 || got[0].Key != "APP_MODE" {
		t.Fatalf("update: 空行应在保存前丢弃, got %v", got)
	}

	// key 为空但 value 非空仍拒绝
	store = &localExecStore{}
	req = createReq(model.CronJobTypeCommand)
	req.Environment = model.KeyValueList{{Key: "", Value: "fake-value"}}
	if err := newLocalExecService(true, store, nil).CreateCronJob(context.Background(), req); !errors.Is(err, task.ErrForbiddenTaskEnv) || store.created != nil {
		t.Fatalf("key 为空、value 非空应拒绝, got %v", err)
	}
}

// P2-2：只有命令行任务校验 Environment；HTTP、SSH 任务即使带“不合规”环境变量也不被拦
func TestCronJobEnvOnlyValidatedForCommand(t *testing.T) {
	bad := model.KeyValueList{{Key: "LD_PRELOAD", Value: "fake-value"}, {Key: "", Value: "fake-value"}, {Key: "A=B", Value: ""}}
	for _, jobType := range []model.CronJobType{model.CronJobTypeHTTP, model.CronJobTypeSSH} {
		store := &localExecStore{}
		req := createReq(jobType)
		req.Environment = bad
		if err := newLocalExecService(true, store, nil).CreateCronJob(context.Background(), req); err != nil || store.created == nil {
			t.Fatalf("create type=%d: 不应因 Environment 被拦, got %v", jobType, err)
		}
		store = &localExecStore{job: &model.CronJob{JobType: jobType}}
		ureq := updateReq(jobType)
		ureq.Environment = bad
		if err := newLocalExecService(true, store, nil).UpdateCronJob(context.Background(), ureq); err != nil || store.updated == nil {
			t.Fatalf("update type=%d: 不应因 Environment 被拦, got %v", jobType, err)
		}
	}
}

func TestConstructedCronServiceReadsRevokedPolicy(t *testing.T) {
	for _, jobType := range []model.CronJobType{model.CronJobTypeCommand, model.CronJobTypeScript} {
		for _, missingPolicy := range []bool{false, true} {
			store := &localExecStore{job: &model.CronJob{Model: model.Model{ID: 1}, JobType: jobType, Status: model.CronJobStatusEnabled, Schedule: "@every 1h"}}
			allowed := true
			var policy config.CronLocalExecPolicy
			if !missingPolicy {
				policy = func() bool { return allowed }
			}
			svc := NewCronService(zap.NewNop(), store, nil, nil, nil, policy)
			// 构造后撤销；保存、启用、手动入队均不能使用启动时的 true。
			allowed = false
			for _, invoke := range []func() error{
				func() error { return svc.CreateCronJob(context.Background(), createReq(jobType)) },
				func() error { return svc.UpdateCronJob(context.Background(), updateReq(jobType)) },
				func() error { return svc.EnableCronJob(context.Background(), 1) },
				func() error { return svc.TriggerCronJob(context.Background(), 1) },
			} {
				if err := invoke(); !errors.Is(err, task.ErrLocalExecDisabled) {
					t.Fatalf("type=%d nil=%v: want revoked permission, got %v", jobType, missingPolicy, err)
				}
			}
			if store.created != nil || store.updated != nil || store.status != nil {
				t.Fatal("revoked policy allowed a persistent mutation")
			}
		}
	}
}

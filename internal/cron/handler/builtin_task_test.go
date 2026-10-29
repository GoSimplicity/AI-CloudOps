package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type builtinExecutionStore struct {
	*executionStore
	job *model.CronJob
}

func (s *builtinExecutionStore) GetCronJob(context.Context, int) (*model.CronJob, error) {
	return s.job, nil
}

type builtinDutyDAO struct {
	alertdao.AlertManagerOnDutyDAO
	called int
	fail   bool
}

func (d *builtinDutyDAO) GetMonitorOnDutyList(_ context.Context, _ *model.GetMonitorOnDutyGroupListReq) ([]*model.MonitorOnDutyGroup, int64, error) {
	return []*model.MonitorOnDutyGroup{{Model: model.Model{ID: 1}, Enable: 1, Users: []*model.User{{Model: model.Model{ID: 7}}}}}, 1, nil
}
func (d *builtinDutyDAO) ExistsMonitorOnDutyHistory(context.Context, int, string) (bool, error) {
	d.called++
	if d.fail {
		return false, errors.New("isolated failure")
	}
	return true, nil
}
func (d *builtinDutyDAO) GetMonitorOnDutyChangesByGroupAndTimeRange(context.Context, int, string, string) ([]*model.MonitorOnDutyChange, int64, error) {
	return nil, 0, nil
}
func TestManualBuiltinUsesRealBusinessLogicAndLease(t *testing.T) {
	for _, fail := range []bool{false, true} {
		base := &executionStore{status: model.CronJobStatusEnabled}
		job := &model.CronJob{Model: model.Model{ID: 1}, Status: model.CronJobStatusEnabled, JobType: model.CronJobTypeSystem, IsBuiltIn: 1, Command: "on_duty_history"}
		store := &builtinExecutionStore{executionStore: base, job: job}
		duty := &builtinDutyDAO{fail: fail}
		handler := &CronHandlers{logger: zap.NewNop(), cronDAO: store, onDutyDAO: duty}
		err := handler.ProcessTask(context.Background(), asynq.NewTask("cron:task", []byte(`{"job_id":1,"trigger_by":"manual"}`)))
		if (err != nil) != fail || duty.called != 1 || !base.stored {
			t.Fatalf("fail=%v err=%v actual_calls=%d stored=%v", fail, err, duty.called, base.stored)
		}
	}
}
func TestBuiltinRejectsNonBuiltinOrStaleSchedule(t *testing.T) {
	base := &executionStore{status: model.CronJobStatusEnabled}
	job := &model.CronJob{Model: model.Model{ID: 1}, Status: model.CronJobStatusEnabled, JobType: model.CronJobTypeSystem, IsBuiltIn: 1, Command: "on_duty_history", Schedule: "@every 1h"}
	store := &builtinExecutionStore{executionStore: base, job: job}
	duty := &builtinDutyDAO{}
	handler := &CronHandlers{logger: zap.NewNop(), cronDAO: store, onDutyDAO: duty}
	err := handler.ProcessTask(context.Background(), asynq.NewTask("cron:task", []byte(`{"job_id":1,"trigger_by":"builtin_scheduled","data":{"schedule":"@every 2h","command":"on_duty_history"}}`)))
	if err != nil || duty.called != 0 || base.stored {
		t.Fatal("obsolete schedule executed")
	}
	job.IsBuiltIn = 0
	if _, err = handler.executeSystemTask(context.Background(), job); err == nil {
		t.Fatal("non-builtin system task allowed")
	}
}

package service

import (
	"context"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/task"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestScheduleValidationPreventsInvalidPersistenceAndEnable(t *testing.T) {
	for _, test := range []struct {
		name, expression string
		valid            bool
	}{
		{"standard", "0 9 * * MON-FRI", true},
		{"descriptor", "@every 1h", true},
		{"explicit timezone", "CRON_TZ=Asia/Shanghai 0 9 * * *", true},
		{"range", "61 * * * *", false},
		{"six fields", "0 0 9 * * *", false},
		{"bad timezone prefix", "TZ=UTC", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			store := &localExecStore{job: &model.CronJob{Model: model.Model{ID: 1}, JobType: model.CronJobTypeHTTP, Status: model.CronJobStatusDisabled, Schedule: test.expression}}
			service := newLocalExecService(true, store, nil)
			preview, err := service.ValidateSchedule(ctx, &model.ValidateScheduleReq{Schedule: test.expression})
			if err != nil || preview.Valid != test.valid {
				t.Fatalf("preview: %+v, err=%v", preview, err)
			}
			create := createReq(model.CronJobTypeHTTP)
			create.Schedule = test.expression
			if err := service.CreateCronJob(ctx, create); (err == nil) != test.valid || (store.created != nil) != test.valid {
				t.Fatalf("create: err=%v persisted=%v", err, store.created != nil)
			}
			update := updateReq(model.CronJobTypeHTTP)
			update.Schedule = test.expression
			if err := service.UpdateCronJob(ctx, update); (err == nil) != test.valid || (store.updated != nil) != test.valid {
				t.Fatalf("update: err=%v persisted=%v", err, store.updated != nil)
			}
			if err := service.EnableCronJob(ctx, 1); (err == nil) != test.valid || (store.status != nil) != test.valid {
				t.Fatalf("enable: err=%v status updated=%v", err, store.status != nil)
			}
		})
	}
}

func TestSchedulePreviewUsesAsynqUTC(t *testing.T) {
	old := time.Local
	time.Local = time.FixedZone("LOCAL", 8*60*60)
	defer func() { time.Local = old }()
	schedule, err := task.ParseSchedule("0 9 * * *")
	if err != nil {
		t.Fatal(err)
	}
	service := newLocalExecService(true, &localExecStore{}, nil)
	before := time.Now().UTC()
	preview := service.generateNextRunTimes(schedule, 1)
	after := time.Now().UTC()
	wantBefore := schedule.Next(before).In(time.Local).Format("2006-01-02 15:04:05 MST")
	wantAfter := schedule.Next(after).In(time.Local).Format("2006-01-02 15:04:05 MST")
	if preview[0] != wantBefore && preview[0] != wantAfter {
		t.Fatalf("preview=%v differs from UTC scheduler next run", preview[0])
	}
}

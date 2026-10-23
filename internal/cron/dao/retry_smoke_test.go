//go:build smoke

package dao

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExplicitRetryZeroPersistsAndConfigUpdatePreservesLease(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	defer pool.Close()
	if err := db.AutoMigrate(&model.CronJob{}); err != nil {
		t.Fatal(err)
	}
	store := NewCronJobDAO(zap.NewNop(), db)
	ctx := context.Background()
	for _, count := range []int{0, 3} {
		name := "zero"
		if count == 3 {
			name = "default"
		}
		job := &model.CronJob{Name: name, Schedule: "@every 1h", JobType: model.CronJobTypeCommand, Command: "true", MaxRetry: count}
		if err := store.CreateCronJob(ctx, job); err != nil {
			t.Fatal(err)
		}
		stored, err := store.GetCronJob(ctx, job.ID)
		if err != nil || stored.MaxRetry != count {
			t.Fatal("SQL default overwrote explicit retry count")
		}
	}
	var job model.CronJob
	if err := db.Where("name = ?", "default").First(&job).Error; err != nil {
		t.Fatal(err)
	}
	if ok, err := store.AcquireCronJobLease(ctx, job.ID, "active-owner"); err != nil || !ok {
		t.Fatal("claim failed")
	}
	if err := db.Model(&model.CronJob{}).Where("id = ?", job.ID).Update("last_run_output", "prior result").Error; err != nil {
		t.Fatal(err)
	}
	config := &model.CronJob{Model: model.Model{ID: job.ID}, Name: "edited", Schedule: "@every 2h", JobType: model.CronJobTypeCommand, Command: "true"}
	if err := store.UpdateCronJob(ctx, config, nil); err != nil {
		t.Fatal(err)
	}
	stored, err := store.GetCronJob(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.MaxRetry != 3 || stored.Status != model.CronJobStatusRunning || stored.LeaseOwner != "active-owner" || stored.LeaseExpiresAt == 0 || stored.LastRunOutput != "prior result" {
		t.Fatal("configuration edit overwrote retry/runtime/lease fields")
	}
	zero := 0
	if err := store.UpdateCronJob(ctx, config, &zero); err != nil {
		t.Fatal(err)
	}
	stored, err = store.GetCronJob(ctx, job.ID)
	if err != nil || stored.MaxRetry != 0 || stored.LeaseOwner != "active-owner" {
		t.Fatal("explicit zero update failed or changed owner")
	}
}

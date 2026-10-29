package dao

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"path/filepath"
	"testing"
	"time"
)

func TestCronJobListPaginationStableForEqualTimestamps(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "page.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err = db.AutoMigrate(&model.CronJob{}); err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 205; i++ {
		job := &model.CronJob{Model: model.Model{CreatedAt: created}, Name: fmt.Sprintf("fixture-%03d", i), Schedule: "@every 1h", JobType: model.CronJobTypeSystem, Status: model.CronJobStatusEnabled}
		if err = db.Create(job).Error; err != nil {
			t.Fatal(err)
		}
	}
	store := NewCronJobDAO(zap.NewNop(), db)
	seen := map[int]bool{}
	previous := int(^uint(0) >> 1)
	for page := 1; page <= 3; page++ {
		rows, total, err := store.GetCronJobList(context.Background(), &model.GetCronJobListReq{ListReq: model.ListReq{Page: page, Size: 100}})
		if err != nil || total != 205 {
			t.Fatalf("page %d total=%d error=%v", page, total, err)
		}
		for _, job := range rows {
			if seen[job.ID] || job.ID >= previous || !job.CreatedAt.Equal(created) {
				t.Fatal("unstable or duplicate equal-timestamp pagination")
			}
			seen[job.ID] = true
			previous = job.ID
		}
	}
	if len(seen) != 205 {
		t.Fatalf("only %d jobs visited", len(seen))
	}
}

//go:build smoke

package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestScheduleHTTPAndRealDAOPreserveExistingJob(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.AutoMigrate(&model.CronJob{}); err != nil {
		t.Fatal(err)
	}
	dao := dao.NewCronJobDAO(zap.NewNop(), db)
	svc := service.NewCronService(zap.NewNop(), dao, nil, nil, nil, nil)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 1, Username: "fixture"}); c.Next() })
	api.NewCronJobHandler(zap.NewNop(), svc).RegisterRouters(router)
	call := func(method, path string, body map[string]any) int {
		t.Helper()
		var data []byte
		if body != nil {
			data, err = json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(data))
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		var decoded struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("bad response: %s", resp.Body.String())
		}
		return decoded.Code
	}
	payload := func(schedule string) map[string]any {
		return map[string]any{"name": "fixture", "job_type": 3, "schedule": schedule, "http_url": "http://example.invalid"}
	}
	if got := call(http.MethodPost, "/api/cron/job/create", payload("61 * * * *")); got == 0 {
		t.Fatal("invalid create was accepted")
	}
	var count int64
	if err := db.Model(&model.CronJob{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("invalid create persisted: count=%d err=%v", count, err)
	}
	if got := call(http.MethodPost, "/api/cron/job/create", payload("@every 1h")); got != 0 {
		t.Fatal("valid create failed")
	}
	var job model.CronJob
	if err := db.Where("name = ?", "fixture").First(&job).Error; err != nil {
		t.Fatal(err)
	}
	updatePath := fmt.Sprintf("/api/cron/job/%d/update", job.ID)
	if got := call(http.MethodPut, updatePath, payload("TZ=UTC")); got == 0 {
		t.Fatal("invalid update was accepted")
	}
	stored, err := dao.GetCronJob(context.Background(), job.ID)
	if err != nil || stored.Schedule != "@every 1h" {
		t.Fatalf("invalid update changed schedule: %+v err=%v", stored, err)
	}
	if err := db.Model(&model.CronJob{}).Where("id = ?", job.ID).Updates(map[string]any{"schedule": "61 * * * *", "status": model.CronJobStatusDisabled}).Error; err != nil {
		t.Fatal(err)
	}
	if got := call(http.MethodPost, fmt.Sprintf("/api/cron/job/%d/enable", job.ID), nil); got == 0 {
		t.Fatal("invalid existing job was enabled")
	}
	stored, err = dao.GetCronJob(context.Background(), job.ID)
	if err != nil || stored.Status != model.CronJobStatusDisabled || stored.Schedule != "61 * * * *" {
		t.Fatalf("rejected enable changed stored job: %+v err=%v", stored, err)
	}
	if got := call(http.MethodPut, updatePath, payload("CRON_TZ=UTC 0 9 * * *")); got != 0 {
		t.Fatal("repair of invalid job failed")
	}
	if got := call(http.MethodPost, fmt.Sprintf("/api/cron/job/%d/enable", job.ID), nil); got != 0 {
		t.Fatal("repaired job failed to enable")
	}
}

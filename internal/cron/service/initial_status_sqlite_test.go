package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCronInitialStatusPersistsThroughHTTP(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
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
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "fixture"}) })
	store := dao.NewCronJobDAO(zap.NewNop(), db)
	api.NewCronJobHandler(zap.NewNop(), service.NewCronService(zap.NewNop(), store, nil, nil, nil)).RegisterRouters(r)
	for i, tc := range []struct {
		field string
		want  model.CronJobStatus
	}{{"", model.CronJobStatusEnabled}, {`,"status":1`, model.CronJobStatusEnabled}, {`,"status":2`, model.CronJobStatusDisabled}} {
		body := fmt.Sprintf(`{"name":"status-%d","job_type":3,"schedule":"@every 87600h","http_method":"GET","http_url":"http://127.0.0.1:9/fixture"%s}`, i, tc.field)
		req := httptest.NewRequest("POST", "/api/cron/job/create", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		var response struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil || response.Code != 0 {
			t.Fatal("HTTP creation failed")
		}
		var saved model.CronJob
		if err := db.Where("name = ?", fmt.Sprintf("status-%d", i)).First(&saved).Error; err != nil {
			t.Fatal(err)
		}
		if saved.Status != tc.want || saved.LastRunStatus != 0 || saved.RunCount != 0 {
			t.Fatalf("unexpected persisted/executed state: status=%d", saved.Status)
		}
	}
	for index, tc := range []struct {
		name, target string
		valid        bool
	}{
		{"loopback single digit port", "http://127.0.0.1:9/fixture", true},
		{"https", "https://example.invalid/path", true},
		{"wrong protocol", "ftp://example.invalid/path", false},
		{"empty host", "http:///path", false},
		{"invalid port", "http://localhost:65536", false},
		{"zero port", "http://localhost:0", false},
		{"control character", "http://local\nhost:9", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := map[string]any{"name": fmt.Sprintf("url-%d", index), "job_type": 3, "status": 2, "schedule": "@every 87600h", "http_method": "GET", "http_url": tc.target}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/api/cron/job/create", strings.NewReader(string(body)))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()
			r.ServeHTTP(res, req)
			var response struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil || (response.Code == 0) != tc.valid {
				t.Fatal("HTTP URL validation mismatch")
			}
			var count int64
			if err := db.Model(&model.CronJob{}).Where("name = ?", payload["name"]).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if (count == 1) != tc.valid {
				t.Fatal("invalid HTTP URL reached persistence")
			}
		})
	}
	enabled, err := store.GetEnabledCronJobs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(enabled) != 2 {
		t.Fatal("scheduler query included a disabled task or lost compatible enabled defaults")
	}
	for _, job := range enabled {
		if job.Status != model.CronJobStatusEnabled {
			t.Fatal("disabled task became eligible for scheduler")
		}
	}
}

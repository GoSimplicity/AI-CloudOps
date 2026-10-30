package service_test

import (
	"encoding/json"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
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

func TestCloudAccountInitialStatusPersistsThroughHTTP(t *testing.T) {
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
	if err := db.AutoMigrate(&model.CloudAccount{}, &model.CloudAccountRegion{}); err != nil {
		t.Fatal(err)
	}
	encryptionKey := strings.Repeat("k", 32)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "fixture"}) })
	accountService, err := service.NewCloudAccountService(&config.Config{Tree: config.TreeConfig{PasswordEncryptionKey: encryptionKey}}, zap.NewNop(), dao.NewCloudAccountDAO(db, zap.NewNop()))
	if err != nil {
		t.Fatal(err)
	}
	api.NewCloudAccountHandler(accountService).RegisterRouters(r)
	for i, tc := range []struct {
		field string
		want  model.CloudAccountStatus
	}{{"", model.CloudAccountEnabled}, {`,"status":1`, model.CloudAccountEnabled}, {`,"status":2`, model.CloudAccountDisabled}} {
		body := fmt.Sprintf(`{"name":"status-%d","provider":1,"access_key":"fixture-access-key","secret_key":"fixture-secret-key","regions":[{"region":"cn-hangzhou","is_default":true}]%s}`, i, tc.field)
		req := httptest.NewRequest("POST", "/api/tree/cloud/account/create", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		var response struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil || response.Code != 0 {
			t.Fatal("HTTP creation failed")
		}
		var saved model.CloudAccount
		if err := db.Where("name = ?", fmt.Sprintf("status-%d", i)).First(&saved).Error; err != nil {
			t.Fatal(err)
		}
		if saved.Status != tc.want {
			t.Fatalf("status=%d want=%d", saved.Status, tc.want)
		}
	}
}

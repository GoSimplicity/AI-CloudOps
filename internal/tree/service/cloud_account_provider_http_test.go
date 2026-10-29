package service_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestProviderAccountHTTPBindingAndRedaction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "http.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err := db.AutoMigrate(&model.CloudAccount{}, &model.CloudAccountRegion{}, &model.TreeCloudResource{}); err != nil {
		t.Fatal(err)
	}
	old := viper.Get("tree.password_encryption_key")
	viper.Set("tree.password_encryption_key", strings.Repeat("k", 32))
	defer viper.Set("tree.password_encryption_key", old)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "fixture"}); c.Next() })
	api.NewCloudAccountHandler(service.NewCloudAccountService(zap.NewNop(), dao.NewCloudAccountDAO(db, zap.NewNop()))).RegisterRouters(engine)
	credential := map[string]any{"tenant_id": "11111111-1111-1111-1111-111111111111", "client_id": "22222222-2222-2222-2222-222222222222", "subscription_id": "33333333-3333-3333-3333-333333333333", "client_secret": "private-http-canary"}
	request := func(method, path string, body any) *httptest.ResponseRecorder {
		t.Helper()
		var data []byte
		if body != nil {
			data, _ = json.Marshal(body)
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if strings.Contains(w.Body.String(), "private-http-canary") {
			t.Fatal("secret appeared in HTTP response")
		}
		return w
	}
	payload := map[string]any{"name": "azure-http", "provider": 5, "provider_credentials": credential, "regions": []any{map[string]any{"region": "eastus", "is_default": true}}}
	created := request(http.MethodPost, "/api/tree/cloud/account/create", payload)
	if created.Code != http.StatusOK {
		t.Fatalf("create status %d", created.Code)
	}
	var saved model.CloudAccount
	if err := db.Where("name = ?", "azure-http").First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/tree/cloud/account/list?page=1&size=10", fmt.Sprintf("/api/tree/cloud/account/%d/detail", saved.ID)} {
		response := request(http.MethodGet, path, nil)
		if response.Code != http.StatusOK || strings.Contains(response.Body.String(), saved.ProviderCredentials) {
			t.Fatal("response exposed credentials")
		}
	}
	bad := map[string]any{"provider_credentials": map[string]any{"client_secret": "private-http-canary"}}
	before := saved.ProviderCredentials
	result := request(http.MethodPut, fmt.Sprintf("/api/tree/cloud/account/%d/update", saved.ID), bad)
	var envelope map[string]any
	if err := json.Unmarshal(result.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope["code"] == float64(0) {
		t.Fatal("incomplete credential rotation accepted")
	}
	if err := db.First(&saved, saved.ID).Error; err != nil || saved.ProviderCredentials != before {
		t.Fatal("incomplete rotation modified state")
	}
}

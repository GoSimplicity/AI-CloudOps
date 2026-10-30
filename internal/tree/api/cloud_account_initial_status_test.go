package api

import (
	"context"
	"encoding/json"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

type initialAccountRecorder struct {
	service.CloudAccountService
	status *int
	called bool
}

func (s *initialAccountRecorder) CreateCloudAccount(_ context.Context, req *model.CreateCloudAccountReq, _ int, _ string) error {
	s.status = req.Status
	s.called = true
	return nil
}

func TestCreateCloudAccountHTTPInitialStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, field string
		accepted    bool
		expected    *int
	}{
		{"omitted", "", true, nil}, {"null", `,"status":null`, true, nil}, {"enabled", `,"status":1`, true, initialStatusNumber(1)}, {"disabled", `,"status":2`, true, initialStatusNumber(2)},
		{"zero", `,"status":0`, false, nil}, {"unknown", `,"status":3`, false, nil}, {"negative", `,"status":-1`, false, nil}, {"overflow", `,"status":99999`, false, nil}, {"string", `,"status":"2"`, false, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &initialAccountRecorder{}
			handler := NewCloudAccountHandler(recorder)
			engine := gin.New()
			engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "fixture"}) })
			engine.POST("/create", handler.CreateCloudAccount)
			body := `{"name":"fixture","provider":1,"access_key":"fixture-access-key","secret_key":"fixture-secret-key","regions":[{"region":"cn-hangzhou","is_default":true}]` + tc.field + `}`
			req := httptest.NewRequest("POST", "/create", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, req)
			var payload struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if recorder.called != tc.accepted || (payload.Code == 0) != tc.accepted {
				t.Fatalf("accepted=%v code=%d called=%v", tc.accepted, payload.Code, recorder.called)
			}
			if tc.accepted && ((recorder.status == nil) != (tc.expected == nil) || (recorder.status != nil && *recorder.status != *tc.expected)) {
				t.Fatal("status was not preserved")
			}
		})
	}
}
func initialStatusNumber(value int) *int { return &value }

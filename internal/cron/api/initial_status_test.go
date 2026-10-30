package api

import (
	"encoding/json"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateCronJobHTTPInitialStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, field string
		accepted    bool
		expected    *int
	}{
		{"omitted", "", true, nil}, {"null", `,"status":null`, true, nil},
		{"enabled", `,"status":1`, true, statusNumber(1)}, {"disabled", `,"status":2`, true, statusNumber(2)},
		{"zero", `,"status":0`, false, nil}, {"running", `,"status":3`, false, nil}, {"negative", `,"status":-1`, false, nil}, {"overflow", `,"status":99999`, false, nil}, {"string", `,"status":"2"`, false, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &createJobRecorder{}
			handler := NewCronJobHandler(zap.NewNop(), recorder)
			engine := gin.New()
			engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "fixture"}) })
			engine.POST("/create", handler.CreateCronJob)
			body := `{"name":"fixture","job_type":3,"schedule":"@every 1h","http_url":"http://127.0.0.1:9"` + tc.field + `}`
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
			if (recorder.request != nil) != tc.accepted || (payload.Code == 0) != tc.accepted {
				t.Fatalf("accepted=%v code=%d called=%v", tc.accepted, payload.Code, recorder.request != nil)
			}
			if tc.accepted {
				actual := recorder.request.Status
				if (actual == nil) != (tc.expected == nil) || (actual != nil && *actual != *tc.expected) {
					t.Fatal("initial status was not preserved")
				}
			}
		})
	}
}
func statusNumber(value int) *int { return &value }

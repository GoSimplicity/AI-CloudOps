package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/config"
	"github.com/gin-gonic/gin"
)

type routeIdentityConfigService struct {
	configService.MonitorConfigService
	calls int
	id    int
}

func (s *routeIdentityConfigService) UpdateMonitorConfig(_ context.Context, req *model.UpdateMonitorConfigReq) error {
	s.calls++
	s.id = req.ID
	return nil
}

func TestConfigPathIDCannotBeReplaced(t *testing.T) {
	for _, tc := range []struct {
		path    string
		allowed bool
	}{
		{"0", false}, {"-0", false}, {"000", false}, {"-1", false},
		{"invalid", false}, {"7", true}, {"0007", true},
	} {
		t.Run(tc.path, func(t *testing.T) {
			svc := &routeIdentityConfigService{}
			engine := gin.New()
			NewMonitorConfigHandler(svc).RegisterRouters(engine)
			request := httptest.NewRequest(http.MethodPut, "/api/monitor/configs/update/"+tc.path, strings.NewReader(`{"id":999,"name":"fixture"}`))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			var envelope struct{ Code int }
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatal("handler must return the normal response envelope")
			}
			if tc.allowed {
				if svc.calls != 1 || svc.id != 7 || envelope.Code != 0 {
					t.Fatal("the positive path ID must remain authoritative over a forged body ID")
				}
			} else if svc.calls != 0 || envelope.Code == 0 {
				t.Fatal("invalid path ID must never borrow a valid body ID or reach the service")
			}
		})
	}
}

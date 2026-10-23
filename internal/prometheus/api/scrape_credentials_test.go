package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	scrapeService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/scrape"
	systemService "github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type credentialAPIService struct {
	scrapeService.ScrapeJobService
	created string
	updated *string
}

func (s *credentialAPIService) CreateMonitorScrapeJob(_ context.Context, req *model.CreateMonitorScrapeJobReq) error {
	s.created = req.BearerToken
	return nil
}
func (s *credentialAPIService) UpdateMonitorScrapeJob(_ context.Context, req *model.UpdateMonitorScrapeJobReq) error {
	s.updated = req.BearerToken
	return nil
}
func (*credentialAPIService) GetMonitorScrapeJobDetail(context.Context, *model.GetMonitorScrapeJobDetailReq) (*model.MonitorScrapeJob, error) {
	return &model.MonitorScrapeJob{BearerToken: "private-api-fixture", BearerTokenFile: "/visible/path", TlsCaContent: "public-ca"}, nil
}
func (*credentialAPIService) GetMonitorScrapeJobList(context.Context, *model.GetMonitorScrapeJobListReq) (model.ListResp[*model.MonitorScrapeJob], error) {
	return model.ListResp[*model.MonitorScrapeJob]{Items: []*model.MonitorScrapeJob{{BearerToken: "private-api-fixture"}}}, nil
}

type credentialAudit struct {
	systemService.AuditService
	requests []*model.CreateAuditLogRequest
}

func (s *credentialAudit) CreateAuditLogAsync(_ context.Context, req *model.CreateAuditLogRequest) {
	s.requests = append(s.requests, req)
}
func TestCredentialHTTPResponsesAndMetadataLogs(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	svc := &credentialAPIService{}
	audit := &credentialAudit{}
	engine := gin.New()
	engine.Use(middleware.NewLogMiddleware(logger).Log(), middleware.NewAuditLogMiddleware(audit, logger).AuditLog(), func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 2, Username: "fixture"}); c.Next() })
	NewScrapeJobHandler(svc).RegisterRouters(engine)
	for _, request := range []struct{ method, path string }{
		{http.MethodPost, "/api/monitor/scrape_jobs/create"},
		{http.MethodPut, "/api/monitor/scrape_jobs/update/1"},
	} {
		response := httptest.NewRecorder()
		req := httptest.NewRequest(request.method, request.path, strings.NewReader(`{"name":"fixture","pool_id":1,"bearer_token":"private-api-fixture"}`))
		req.Header.Set("Content-Type", "application/json")
		engine.ServeHTTP(response, req)
		if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "private-api-fixture") {
			t.Fatal("credential write response changed or exposed token")
		}
	}
	if svc.created != "private-api-fixture" || svc.updated == nil || *svc.updated != "private-api-fixture" {
		t.Fatal("request binding lost creation string or update pointer credential")
	}
	if len(audit.requests) != 2 {
		t.Fatal("metadata audit events missing")
	}
	for _, entry := range audit.requests {
		if len(entry.RequestBody) != 0 || len(entry.ResponseBody) != 0 || strings.Contains(entry.ErrorMsg, "private-api-fixture") {
			t.Fatal("credential stored in audit body/error")
		}
	}
	for _, path := range []string{"/api/monitor/scrape_jobs/detail/1", "/api/monitor/scrape_jobs/list?page=1&size=10"} {
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if strings.Contains(response.Body.String(), "private-api-fixture") || strings.Contains(response.Body.String(), `"bearer_token":`) || !strings.Contains(response.Body.String(), `"has_bearer_token":true`) {
			t.Fatal("HTTP read token redaction/presence contract failed")
		}
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
	}
	for _, entry := range logs.All() {
		if strings.Contains(entry.Message+fmt.Sprint(entry.ContextMap()), "private-api-fixture") {
			t.Fatal("access/audit middleware logged request credential")
		}
	}
	empty, err := json.Marshal(model.MonitorScrapeJob{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(empty), `"has_bearer_token":false`) {
		t.Fatal("empty credential lacks explicit false presence flag")
	}
}

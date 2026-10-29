package api

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	systemservice "github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	treeservice "github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type accountAuthRoles struct {
	systemservice.RoleService
	grants []*model.Api
}

func (r accountAuthRoles) GetUserRoles(context.Context, int) (*model.ListResp[*model.Role], error) {
	return &model.ListResp[*model.Role]{Items: []*model.Role{{Status: 1, Apis: r.grants}}}, nil
}

type accountAuthStub struct {
	treeservice.CloudAccountService
	id, calls int
}

type accountCredentialAudit struct {
	systemservice.AuditService
	requests []*model.CreateAuditLogRequest
}

func (a *accountCredentialAudit) CreateAuditLogAsync(_ context.Context, req *model.CreateAuditLogRequest) {
	a.requests = append(a.requests, req)
}

func (s *accountAuthStub) UpdateCloudAccount(_ context.Context, req *model.UpdateCloudAccountReq) error {
	s.calls++
	s.id = req.ID
	return nil
}

func TestTemporaryCredentialUpdateRequiresGrantAndUsesPathID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, allow := range []bool{false, true} {
		stub := &accountAuthStub{}
		grants := []*model.Api{}
		if allow {
			grants = append(grants, &model.Api{Method: 3, Path: "/api/tree/cloud/account/:id/update"})
		}
		engine := gin.New()
		core, logs := observer.New(zap.DebugLevel)
		log := zap.New(core)
		audit := &accountCredentialAudit{}
		engine.Use(middleware.NewLogMiddleware(log).Log(), middleware.NewAuditLogMiddleware(audit, log).AuditLog())
		engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7}); c.Next() })
		engine.Use(middleware.NewAuthMiddleware(accountAuthRoles{grants: grants}).CheckAuth())
		NewCloudAccountHandler(stub).RegisterRouters(engine)
		req := httptest.NewRequest("PUT", "/api/tree/cloud/account/7/update", strings.NewReader(`{"id":999,"credential_type":"temporary","access_key":"ASIAFAKEKEY123456","secret_key":"secret-canary","session_token":"token-canary","credential_expires_at":"2099-01-01T00:00:00Z"}`))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		engine.ServeHTTP(res, req)
		if !allow && (res.Code != 403 || stub.calls != 0) {
			t.Fatalf("ungranted update: %d", res.Code)
		}
		if allow && (res.Code != 200 || stub.calls != 1 || stub.id != 7) {
			t.Fatalf("forged account ID accepted: status=%d id=%d", res.Code, stub.id)
		}
		if strings.Contains(res.Body.String(), "token-canary") || strings.Contains(res.Body.String(), "secret-canary") {
			t.Fatal("credential in response")
		}
		for _, entry := range audit.requests {
			if len(entry.RequestBody) != 0 || len(entry.ResponseBody) != 0 || strings.Contains(entry.ErrorMsg, "canary") {
				t.Fatal("credential in audit")
			}
		}
		for _, entry := range logs.All() {
			output := entry.Message + fmt.Sprint(entry.ContextMap())
			if strings.Contains(output, "canary") || strings.Contains(output, "ASIAFAKEKEY123456") {
				t.Fatal("credential in log")
			}
		}
	}
}

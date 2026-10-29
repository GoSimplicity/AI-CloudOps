package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type cloudActorCapture struct {
	service.TreeCloudService
	id, actor int
	name      string
	calls     int
	result    *model.SyncCloudResourceResp
}

func (s *cloudActorCapture) SyncTreeCloudResource(_ context.Context, r *model.SyncTreeCloudResourceReq) (*model.SyncCloudResourceResp, error) {
	s.actor, s.name, s.calls = r.OperatorID, r.OperatorName, s.calls+1
	if s.result != nil {
		return s.result, nil
	}
	return &model.SyncCloudResourceResp{}, nil
}

func TestCloudSyncPartialResponsePreservesRegionStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &cloudActorCapture{result: &model.SyncCloudResourceResp{Status: "partial", Regions: []model.CloudRegionSyncResult{{RegionID: 1, Status: "success", HistoryID: 7, RetrySafe: true}, {RegionID: 2, Status: "failed", ErrorCategory: "audit_unavailable", HistoryID: 8, RetrySafe: false}}}}
	app := gin.New()
	app.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 5, Username: "trusted"}); c.Next() })
	app.POST("/sync", NewTreeCloudHandler(stub, nil, nil, zap.NewNop()).SyncTreeCloudResource)
	request := httptest.NewRequest("POST", "/sync", strings.NewReader(`{"cloud_account_id":1}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)
	var body struct {
		Code int                         `json:"code"`
		Data model.SyncCloudResourceResp `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Code != 0 || body.Data.Status != "partial" || len(body.Data.Regions) != 2 || body.Data.Regions[1].RetrySafe || body.Data.Regions[1].HistoryID != 8 {
		t.Fatalf("invalid response: %+v err=%v", body, err)
	}
}
func (s *cloudActorCapture) UpdateTreeCloudResource(_ context.Context, r *model.UpdateTreeCloudResourceReq) error {
	s.id, s.actor, s.name, s.calls = r.ID, r.OperatorID, r.OperatorName, s.calls+1
	return nil
}
func (s *cloudActorCapture) DeleteTreeCloudResource(_ context.Context, r *model.DeleteTreeCloudResourceReq) error {
	s.id, s.actor, s.name, s.calls = r.ID, r.OperatorID, r.OperatorName, s.calls+1
	return nil
}
func (s *cloudActorCapture) BatchDeleteTreeCloudResource(_ context.Context, r *model.BatchDeleteTreeCloudResourceReq) error {
	s.actor, s.name, s.calls = r.OperatorID, r.OperatorName, s.calls+1
	return nil
}
func (s *cloudActorCapture) BatchUpdateCloudResourceStatus(_ context.Context, r *model.BatchUpdateCloudResourceStatusReq) error {
	s.actor, s.name, s.calls = r.OperatorID, r.OperatorName, s.calls+1
	return nil
}

func TestCloudMutationAuditUsesTrustedIdentityAndPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, method, path, route, body string
		id                              int
	}{
		{"sync", "POST", "/sync", "/sync", `{"cloud_account_id":1,"operator_id":999,"operator_name":"spoof"}`, 0},
		{"update", "PUT", "/cloud/7", "/cloud/:id", `{"id":999,"description":"fixture","operator_id":999,"operator_name":"spoof"}`, 7},
		{"delete", "DELETE", "/cloud/7", "/cloud/:id", `{"id":999,"operator_id":999,"operator_name":"spoof"}`, 7},
		{"batch delete", "POST", "/batch/delete", "/batch/delete", `{"ids":[7],"operator_id":999,"operator_name":"spoof"}`, 0},
		{"batch status", "PUT", "/batch/status", "/batch/status", `{"ids":[7],"status":1,"operator_id":999,"operator_name":"spoof"}`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &cloudActorCapture{}
			handler := NewTreeCloudHandler(stub, nil, nil, zap.NewNop())
			app := gin.New()
			app.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 5, Username: "trusted"}); c.Next() })
			switch tc.name {
			case "sync":
				app.POST(tc.route, handler.SyncTreeCloudResource)
			case "update":
				app.PUT(tc.route, handler.UpdateTreeCloudResource)
			case "delete":
				app.DELETE(tc.route, handler.DeleteTreeCloudResource)
			case "batch delete":
				app.POST(tc.route, handler.BatchDeleteTreeCloudResource)
			case "batch status":
				app.PUT(tc.route, handler.BatchUpdateCloudResourceStatus)
			}
			request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			app.ServeHTTP(response, request)
			if response.Code != 200 || stub.calls != 1 || stub.actor != 5 || stub.name != "trusted" || stub.id != tc.id {
				t.Fatalf("response=%d calls=%d actor=%d id=%d", response.Code, stub.calls, stub.actor, stub.id)
			}
		})
	}
}

func TestCloudSyncRejectsMissingTrustedIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &cloudActorCapture{}
	app := gin.New()
	app.POST("/sync", NewTreeCloudHandler(stub, nil, nil, zap.NewNop()).SyncTreeCloudResource)
	req := httptest.NewRequest("POST", "/sync", strings.NewReader(`{"cloud_account_id":1}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	app.ServeHTTP(response, req)
	if response.Code != 403 || stub.calls != 0 {
		t.Fatalf("status=%d calls=%d", response.Code, stub.calls)
	}
}

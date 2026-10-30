package api

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	systemservice "github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type managedAPIStub struct {
	service.ManagedCloudService
	actor       int
	historyID   int
	operationID string
	calls       int
}

func (s *managedAPIStub) ResolveManagedHistory(_ context.Context, req *model.ResolveCloudManagedHistoryReq) error {
	s.actor = req.OperatorID
	s.historyID = req.HistoryID
	s.calls++
	return nil
}
func (s *managedAPIStub) GetManagedOperation(_ context.Context, id string, owner int) (*model.CloudManagedOperationView, error) {
	s.actor = owner
	s.calls++
	return &model.CloudManagedOperationView{OperationID: id, State: "running"}, nil
}
func (s *managedAPIStub) ReviewManagedOperation(_ context.Context, req *model.ReviewCloudManagedOperationReq) (*model.CloudManagedOperationView, error) {
	s.actor = req.OperatorID
	s.operationID = req.OperationID
	s.calls++
	return &model.CloudManagedOperationView{OperationID: req.OperationID, State: "reviewed"}, nil
}
func (s *managedAPIStub) ListManagedOperations(_ context.Context, req *model.ListCloudManagedOperationsReq) (model.ListResp[*model.CloudManagedOperationView], error) {
	s.actor = req.ViewerID
	s.calls++
	return model.ListResp[*model.CloudManagedOperationView]{}, nil
}
func (s *managedAPIStub) SyncManaged(_ context.Context, req *model.SyncCloudManagedResourcesReq) (*model.CloudManagedSyncResult, error) {
	s.actor = req.OperatorID
	s.calls++
	return &model.CloudManagedSyncResult{Status: "success"}, nil
}
func (s *managedAPIStub) ListManaged(_ context.Context, _ *model.ListCloudManagedResourcesReq) (model.ListResp[*model.CloudManagedResource], error) {
	s.calls++
	return model.ListResp[*model.CloudManagedResource]{Items: []*model.CloudManagedResource{}, Total: 0}, nil
}
func (s *managedAPIStub) GetManaged(_ context.Context, _ *model.ListCloudManagedResourcesReq, id int) (*model.CloudManagedResource, error) {
	s.calls++
	return &model.CloudManagedResource{Model: model.Model{ID: id}}, nil
}
func (s *managedAPIStub) ListManagedHistory(_ context.Context, _ *model.ListCloudManagedResourcesReq) (model.ListResp[*model.CloudManagedSyncHistory], error) {
	s.calls++
	return model.ListResp[*model.CloudManagedSyncHistory]{Items: []*model.CloudManagedSyncHistory{}, Total: 0}, nil
}

type managedRoles struct {
	systemservice.RoleService
	grants []*model.Api
	super  bool
}

func (m managedRoles) GetUserRoles(context.Context, int) (*model.ListResp[*model.Role], error) {
	if m.super {
		return &model.ListResp[*model.Role]{Items: []*model.Role{{Status: 1, IsSystem: 1, Code: permission.SuperAdminRoleCode}}}, nil
	}
	return &model.ListResp[*model.Role]{Items: []*model.Role{{Status: 1, Apis: m.grants}}}, nil
}

func TestManagedCloudRoutesRequirePermissionsAndTrustedActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, method, path, body  string
		grants                    []*model.Api
		uid, status, calls, actor int
	}{
		{"denied sync", "POST", "/api/tree/cloud/managed/sync", `{"cloud_account_id":1,"cloud_account_region_id":2,"kind":"rds","resource_ids":["rm-test"]}`, nil, 7, 403, 0, 0},
		{"trusted actor", "POST", "/api/tree/cloud/managed/sync", `{"cloud_account_id":1,"cloud_account_region_id":2,"kind":"rds","resource_ids":["rm-test"],"operator_id":999}`, []*model.Api{{Method: 2, Path: "/api/tree/cloud/managed/sync"}}, 7, 200, 1, 7},
		{"wrong method grant", "POST", "/api/tree/cloud/managed/sync", `{"cloud_account_id":1,"cloud_account_region_id":2,"kind":"rds","resource_ids":["rm-test"]}`, []*model.Api{{Method: 1, Path: "/api/tree/cloud/managed/sync"}}, 7, 403, 0, 0},
		{"review trusted actor", "POST", "/api/tree/cloud/managed/history/12/resolve", `{"cloud_account_id":1,"cloud_account_region_id":2,"kind":"rds","resources_verified":true,"reason":"resources checked and reconciled","operator_id":999,"history_id":999}`, []*model.Api{{Method: 2, Path: model.CloudManagedResolveRoute}}, 7, 200, 1, 7},
		{"review wildcard denied", "POST", "/api/tree/cloud/managed/history/12/resolve", `{}`, []*model.Api{{Method: 2, Path: "/api/tree/cloud/*"}}, 7, 403, 0, 0},
		{"operation owner", "GET", "/api/tree/cloud/managed/operations/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa?operator_id=999", "", []*model.Api{{Method: 1, Path: "/api/tree/cloud/managed/operations/:operation_id"}}, 7, 200, 1, 7},
		{"operation permission denied", "GET", "/api/tree/cloud/managed/operations/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "", []*model.Api{{Method: 1, Path: "/api/tree/cloud/managed/history"}}, 7, 403, 0, 0},
		{"review operation owner", "POST", "/api/tree/cloud/managed/operations/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/resolve", `{"cloud_account_id":1,"cloud_account_region_id":2,"kind":"rds","resource_ids":["rm-one"],"resources_verified":true,"reason":"resources checked and reconciled","operator_id":999,"operation_id":"forged"}`, []*model.Api{{Method: 2, Path: model.CloudManagedOperationReviewRoute}}, 7, 200, 1, 7},
		{"review operation wildcard denied", "POST", "/api/tree/cloud/managed/operations/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/resolve", `{}`, []*model.Api{{Method: 2, Path: "/api/tree/cloud/*"}}, 7, 403, 0, 0},
		{"own operation list", "GET", "/api/tree/cloud/managed/operations?cloud_account_id=1&cloud_account_region_id=2&kind=rds&viewer_id=999&trusted_super=true", "", []*model.Api{{Method: 1, Path: "/api/tree/cloud/managed/operations"}}, 7, 200, 1, 7},
		{"cross-owner list denied", "GET", "/api/tree/cloud/managed/operations?cloud_account_id=1&cloud_account_region_id=2&kind=rds&all_owners=true&trusted_super=true", "", []*model.Api{{Method: 1, Path: "/api/tree/cloud/managed/operations"}}, 7, 403, 0, 0},
		{"cross-owner get denied", "GET", "/api/tree/cloud/managed/operations/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa?owner_id=8&trusted_super=true", "", []*model.Api{{Method: 1, Path: "/api/tree/cloud/managed/operations/:operation_id"}}, 7, 403, 0, 0},
		{"cross-owner review denied", "POST", "/api/tree/cloud/managed/operations/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/resolve", `{"cloud_account_id":1,"cloud_account_region_id":2,"kind":"rds","resource_ids":["rm-one"],"resources_verified":true,"reason":"verified actual resource state","owner_id":8,"trusted_super":true}`, []*model.Api{{Method: 2, Path: model.CloudManagedOperationReviewRoute}}, 7, 403, 0, 0},
		{"super list owners", "GET", "/api/tree/cloud/managed/operations?cloud_account_id=1&cloud_account_region_id=2&kind=rds&all_owners=true", "", nil, 7, 200, 1, 7},
		{"super get owner", "GET", "/api/tree/cloud/managed/operations/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa?owner_id=8", "", nil, 7, 200, 1, 8},
		{"super review operation", "POST", "/api/tree/cloud/managed/operations/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/resolve", `{"cloud_account_id":1,"cloud_account_region_id":2,"kind":"rds","resource_ids":["rm-one"],"resources_verified":true,"reason":"verified actual resource state","owner_id":8}`, nil, 7, 200, 1, 7},
		{"list allowed", "GET", "/api/tree/cloud/managed/list?cloud_account_id=1&cloud_account_region_id=2&kind=rds", "", []*model.Api{{Method: 1, Path: "/api/tree/cloud/managed/list"}}, 7, 200, 1, 0},
		{"detail wrong grant", "GET", "/api/tree/cloud/managed/detail/9?cloud_account_id=1&cloud_account_region_id=2&kind=rds", "", []*model.Api{{Method: 1, Path: "/api/tree/cloud/managed/list"}}, 7, 403, 0, 0},
		{"detail allowed", "GET", "/api/tree/cloud/managed/detail/9?cloud_account_id=1&cloud_account_region_id=2&kind=rds", "", []*model.Api{{Method: 1, Path: "/api/tree/cloud/managed/detail/:id"}}, 7, 200, 1, 0},
		{"history denied", "GET", "/api/tree/cloud/managed/history?cloud_account_id=1&cloud_account_region_id=2&kind=rds", "", []*model.Api{{Method: 1, Path: "/api/tree/cloud/managed/list"}}, 7, 403, 0, 0},
		{"invalid identity", "POST", "/api/tree/cloud/managed/sync", `{"cloud_account_id":1,"cloud_account_region_id":2,"kind":"rds","resource_ids":["rm-test"]}`, []*model.Api{{Method: 2, Path: "/api/tree/cloud/managed/sync"}}, 0, 403, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &managedAPIStub{}
			app := gin.New()
			app.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: tc.uid}); c.Next() })
			app.Use(middleware.NewAuthMiddleware(managedRoles{grants: tc.grants, super: strings.HasPrefix(tc.name, "super ")}).CheckAuth())
			NewManagedCloudHandler(stub).RegisterRouters(app)
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			app.ServeHTTP(response, req)
			if tc.name == "review trusted actor" && stub.historyID != 12 {
				t.Fatal("body overrode history route identity")
			}
			if tc.name == "review operation owner" && stub.operationID != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" {
				t.Fatal("body overrode operation identity")
			}
			if response.Code != tc.status || stub.calls != tc.calls || stub.actor != tc.actor {
				t.Fatalf("status=%d calls=%d actor=%d", response.Code, stub.calls, stub.actor)
			}
		})
	}
}

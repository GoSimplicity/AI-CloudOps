package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	k8sapi "github.com/GoSimplicity/AI-CloudOps/internal/k8s/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	treeapi "github.com/GoSimplicity/AI-CloudOps/internal/tree/api"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// userRolesDAO 模拟 roleDAO.GetRoles：与线上一致只返回启用中的角色。
type userRolesDAO struct {
	dao.RoleDAO
	roles []*model.Role
	err   error
}

func (d userRolesDAO) GetRoles(context.Context, int) ([]*model.Role, error) {
	if d.err != nil {
		return nil, d.err
	}
	var enabled []*model.Role
	for _, role := range d.roles {
		if role.Status == 1 {
			enabled = append(enabled, role)
		}
	}
	return enabled, nil
}

// middlewareRoles 让鉴权中间件读取与权限接口相同的角色来源。
type middlewareRoles struct {
	service.RoleService
	dao userRolesDAO
}

func (r middlewareRoles) GetUserRoles(ctx context.Context, uid int) (*model.ListResp[*model.Role], error) {
	roles, err := r.dao.GetRoles(ctx, uid)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.Role]{Items: roles}, nil
}

var testAIOpsCatalog = []permission.Route{
	{Method: "GET", Path: "/api/v1/assistant/config"},
	{Method: "GET", Path: "/api/v1/assistant/session/:session_id"},
	{Method: "POST", Path: "/api/v1/assistant/query"},
	{Method: "POST", Path: "/api/v1/autofix"},
	{Method: "POST", Path: "/api/v1/autofix/"},
}

// permissionsRouter 按线上顺序装配：登录态 -> 鉴权 -> 真实 handler 路由（业务 handler 由终止中间件代替）。
func permissionsRouter(t *testing.T, roles userRolesDAO, uid int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	previous := permission.ProxyCatalog(permission.AIOpsProxyTemplate)
	permission.SetProxyCatalog(permission.AIOpsProxyTemplate, testAIOpsCatalog)
	t.Cleanup(func() { permission.SetProxyCatalog(permission.AIOpsProxyTemplate, previous) })

	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: uid, Username: "operator"}); c.Next() })
	engine.Use(middleware.NewAuthMiddleware(middlewareRoles{dao: roles}).CheckAuth())
	engine.Use(func(c *gin.Context) {
		if c.FullPath() != "/api/user/permissions" {
			c.Header("X-Matched", c.FullPath())
			c.AbortWithStatus(204)
		}
	})
	NewUserHandler(service.NewUserService(nil, roles, zap.NewNop()), nil).RegisterRoutes(engine)
	k8sapi.NewK8sEventHandler(nil).RegisterRouters(engine)
	treeapi.NewCloudAccountHandler(nil).RegisterRouters(engine)
	treeapi.NewTreeCloudHandler(nil, nil, nil, nil).RegisterRouters(engine)
	NewDepartmentHandler(nil).RegisterRouters(engine)
	// 与 registerAIOps 一样在其他路由之后注册代理
	engine.Any(permission.AIOpsProxyTemplate, func(c *gin.Context) { c.Status(204) })
	return engine
}

type permissionsBody struct {
	Code int                       `json:"code"`
	Data model.UserPermissionsResp `json:"data"`
	Raw  map[string]json.RawMessage
}

func fetchPermissions(t *testing.T, engine *gin.Engine) (permissionsBody, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest("GET", "/api/user/permissions", nil))
	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body permissionsBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body, rec.Body.String()
}

func TestUserPermissionsExpandsGrantsToMethodTemplates(t *testing.T) {
	roles := userRolesDAO{roles: []*model.Role{
		{Status: 1, Apis: []*model.Api{
			{Method: 1, Path: "/api/k8s/clusters/:cluster_id/events/:namespace/:name"},
			{Method: 3, Path: "/api/tree/cloud/account/:id/status"},
			{Method: 1, Path: "/api/v1/assistant/*"},
			{Method: 1, Path: "/api/tree/*/list"},         // 中间通配：不再支持
			{Method: 2, Path: "/api/department/*"},        // 超管专属：授予无效
			{Method: 2, Path: "/api/user/signup"},         // 超管专属：授予无效
			{Method: 1, Path: "/api/user/codes"},          // 不经授权的接口不列出
			{Method: 1, Path: "/api/tree/cloud/9/detail"}, // 真实路径不是模板
		}},
		{Status: 0, Apis: []*model.Api{{Method: 4, Path: "/*"}}}, // 禁用角色
	}}
	body, raw := fetchPermissions(t, permissionsRouter(t, roles, 5))
	want := []string{
		"GET /api/k8s/clusters/:cluster_id/events/:namespace/:name",
		"GET /api/v1/assistant/config",
		"GET /api/v1/assistant/session/:session_id",
		"PUT /api/tree/cloud/account/:id/status",
	}
	if body.Code != 0 || body.Data.Version != 2 || body.Data.SuperAdmin || strings.Join(body.Data.Permissions, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected response: %s", raw)
	}
}

func TestUserPermissionsForSuperAdminAndEmptyUser(t *testing.T) {
	super := userRolesDAO{roles: []*model.Role{{Code: permission.SuperAdminRoleCode, IsSystem: 1, Status: 1}}}
	body, raw := fetchPermissions(t, permissionsRouter(t, super, 1))
	if !body.Data.SuperAdmin || len(body.Data.Permissions) != 1 || body.Data.Permissions[0] != permission.SuperCode {
		t.Fatalf("super admin: %s", raw)
	}

	// 借用 super_admin 编码的普通角色不是超管
	impostor := userRolesDAO{roles: []*model.Role{{Code: permission.SuperAdminRoleCode, IsSystem: 0, Status: 1}}}
	body, raw = fetchPermissions(t, permissionsRouter(t, impostor, 3))
	if body.Data.SuperAdmin || body.Data.Permissions == nil || len(body.Data.Permissions) != 0 {
		t.Fatalf("impostor: %s", raw)
	}
	if !strings.Contains(raw, `"permissions":[]`) {
		t.Fatalf("permissions must serialise as [] not null: %s", raw)
	}
}

func TestUserPermissionsReportsRoleLookupFailure(t *testing.T) {
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 5}); c.Next() })
	NewUserHandler(service.NewUserService(nil, userRolesDAO{err: errors.New("db down")}, zap.NewNop()), nil).RegisterRoutes(engine)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest("GET", "/api/user/permissions", nil))
	var body permissionsBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Code == 0 || len(body.Data.Permissions) != 0 {
		t.Fatalf("lookup failure must not return permissions: %s", rec.Body.String())
	}
}

// 权限接口展开的列表必须与鉴权中间件的实际判定逐条一致。
func TestUserPermissionsAgreeWithAuthorizationMiddleware(t *testing.T) {
	for name, apis := range map[string][]*model.Api{
		"exact":        {{Method: 1, Path: "/api/k8s/clusters/:cluster_id/events/:namespace/:name"}, {Method: 3, Path: "/api/tree/cloud/:id/status"}},
		"module":       {{Method: 3, Path: "/api/tree/*"}, {Method: 1, Path: "/api/k8s/clusters/:cluster_id/events/*"}},
		"proxy prefix": {{Method: 2, Path: "/api/v1/autofix*"}, {Method: 1, Path: "/api/v1/assistant/session/:session_id"}},
		"whole proxy":  {{Method: 2, Path: permission.AIOpsProxyTemplate}},
		"global":       {{Method: 1, Path: "/*"}, {Method: 4, Path: "/*"}},
	} {
		t.Run(name, func(t *testing.T) {
			roles := userRolesDAO{roles: []*model.Role{{Status: 1, Apis: apis}}}
			engine := permissionsRouter(t, roles, 5)
			body, _ := fetchPermissions(t, engine)
			granted := map[string]bool{}
			for _, code := range body.Data.Permissions {
				granted[code] = true
			}
			checked := 0
			for _, entry := range permission.BuildCatalog(engine.Routes()) {
				if strings.Contains(entry.Path, "*") {
					continue // gin 通配模板本身没有唯一的具体请求路径
				}
				concrete := entry.Path
				for _, part := range strings.Split(entry.Path, "/") {
					if strings.HasPrefix(part, ":") {
						concrete = strings.Replace(concrete, part, "v"+part[1:], 1)
					}
				}
				rec := httptest.NewRecorder()
				engine.ServeHTTP(rec, httptest.NewRequest(permission.MethodName(entry.Method), concrete, nil))
				if matched := rec.Header().Get("X-Matched"); rec.Code == 204 && entry.Via == "" && matched != entry.Path {
					t.Fatalf("%s routed to %s", concrete, matched)
				}
				allowed := rec.Code == 204
				if allowed != granted[entry.Code()] {
					t.Errorf("%s: middleware allowed=%v, permissions listed=%v", entry.Code(), allowed, granted[entry.Code()])
				}
				checked++
			}
			if checked < 40 {
				t.Fatalf("catalog too small: %d", checked)
			}
		})
	}
}

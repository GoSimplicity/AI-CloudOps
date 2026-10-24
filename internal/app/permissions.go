package app

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
)

//go:embed aiops_permissions.json
var aiopsPermissionCatalog []byte

// namedPermission 需要稳定权限名和说明的接口（敏感接口），其余接口按路由自动生成名称
type namedPermission struct {
	Name        string
	Description string
}

// namedPermissions 以 "METHOD 路由" 为键登记的权限码
var namedPermissions = map[string]namedPermission{
	"GET " + model.K8sSecretRevealRoute: {
		Name:        model.K8sSecretRevealPermissionName,
		Description: "查看 K8s Secret 明文（SEC-25，每次调用写审计；仅显式授权生效，通配符授权不覆盖）",
	},
	"GET " + model.K8sServiceAccountTokenRoute: {
		Name:        model.K8sServiceAccountTokenPermissionName,
		Description: "读取 K8s ServiceAccount token Secret（SEC-29，每次调用写审计；仅显式授权生效，通配符授权不覆盖）",
	},
	"POST " + model.K8sServiceAccountTokenRoute: {
		Name:        model.K8sServiceAccountTokenCreatePermissionName,
		Description: "通过 TokenRequest 签发 K8s ServiceAccount token（SEC-29，每次调用写审计；仅显式授权生效，通配符授权不覆盖）",
	},
}

// autoPermissionName 未登记权限码的接口按路由自动生成的名称
func autoPermissionName(route gin.RouteInfo) string {
	digest := sha256.Sum256([]byte(route.Method + " " + route.Path))
	handler := route.Handler[strings.LastIndex(route.Handler, ".")+1:]
	if len(handler) > 25 {
		handler = handler[:25]
	}
	return fmt.Sprintf("%s %s %x", route.Method, handler, digest[:4])
}

// autoPermissionDescription 未登记权限码的接口自动生成的说明："<METHOD> <path>"
func autoPermissionDescription(route gin.RouteInfo) string {
	return route.Method + " " + route.Path
}

// parseAIOpsCatalog 解析内嵌的 AI 接口目录。该目录既用于补充接口元数据，
// 也登记为 /api/v1/*path 代理的鉴权目录（见 permission.ResolveProxyTemplate）。
func parseAIOpsCatalog() ([]permission.Route, error) {
	var routes []permission.Route
	if err := json.Unmarshal(aiopsPermissionCatalog, &routes); err != nil {
		return nil, fmt.Errorf("解析AI接口权限目录: %w", err)
	}
	prefix := strings.TrimSuffix(permission.AIOpsProxyTemplate, "*path")
	for _, route := range routes {
		if _, ok := permission.MethodCode(route.Method); !ok || !strings.HasPrefix(route.Path, prefix) {
			return nil, fmt.Errorf("AI接口权限目录包含非法条目: %s %s", route.Method, route.Path)
		}
	}
	return routes, nil
}

// bootstrapPermissionCatalog 补充可发现接口，不授予角色或覆盖管理员维护的元数据
func (a *App) bootstrapPermissionCatalog(ctx context.Context) error {
	methods := map[string]int8{"GET": 1, "POST": 2, "PUT": 3, "DELETE": 4}
	public := map[string]bool{"/api/user/login": true, "/api/user/logout": true, "/api/user/refresh_token": true, "/api/not_auth/getBindIps": true, "/api/not_auth/getTreeNodeBindIps": true}
	var existing []model.Api
	if err := a.db.WithContext(ctx).Select("id", "name", "path", "method", "description").Find(&existing).Error; err != nil {
		return err
	}
	seen := make(map[string]bool, len(existing))
	existingByKey := make(map[string]model.Api, len(existing))
	key := func(method int8, path string) string { return fmt.Sprintf("%d %s", method, path) }
	for _, api := range existing {
		seen[key(api.Method, api.Path)] = true
		existingByKey[key(api.Method, api.Path)] = api
	}
	routes := a.server.Routes()
	for _, route := range routes {
		if route.Path == permission.AIOpsProxyTemplate {
			aiRoutes, err := parseAIOpsCatalog()
			if err != nil {
				return err
			}
			for _, aiRoute := range aiRoutes {
				routes = append(routes, gin.RouteInfo{Method: aiRoute.Method, Path: aiRoute.Path, Handler: "AIOpsProxy"})
			}
			break
		}
	}
	pending := make([]model.Api, 0)
	for _, route := range routes {
		method, ok := methods[route.Method]
		if !ok || !strings.HasPrefix(route.Path, "/api/") || public[route.Path] {
			continue
		}
		if seen[key(method, route.Path)] {
			a.renameAutoNamedPermission(ctx, route, existingByKey[key(method, route.Path)])
			continue
		}
		entry := model.Api{Name: autoPermissionName(route), Path: route.Path, Method: method, Description: autoPermissionDescription(route), Version: "v1", Category: 2, IsPublic: 0}
		if named, ok := namedPermissions[route.Method+" "+route.Path]; ok {
			entry.Name, entry.Description = named.Name, named.Description
		}
		pending = append(pending, entry)
		seen[key(method, route.Path)] = true
	}
	if len(pending) == 0 {
		return nil
	}
	if err := a.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(pending, 100).Error; err != nil {
		return err
	}
	a.warnMissingNamedPermissions(ctx, pending)
	return nil
}

// warnMissingNamedPermissions OnConflict DoNothing 遇到唯一键冲突（例如权限码已被其他接口或已软删除的行占用）会静默跳过插入。
// 插入后逐个核对本次尝试登记的权限码：表中不存在 name + path + method 均一致的行时记一条 Warn（只记权限码），不影响启动。
func (a *App) warnMissingNamedPermissions(ctx context.Context, pending []model.Api) {
	expected := make(map[string]model.Api)
	for _, entry := range pending {
		for _, named := range namedPermissions {
			if entry.Name == named.Name {
				expected[entry.Name] = entry
			}
		}
	}
	if len(expected) == 0 {
		return
	}
	names := make([]string, 0, len(expected))
	for name := range expected {
		names = append(names, name)
	}
	sort.Strings(names)
	var rows []model.Api
	if err := a.db.WithContext(ctx).Select("id", "name", "path", "method").Where("name IN ?", names).Find(&rows).Error; err != nil {
		if a.logger != nil {
			a.logger.Warn("核对敏感接口权限码失败", zap.Strings("permissions", names), zap.Error(err))
		}
		return
	}
	present := make(map[string]bool, len(rows))
	for _, row := range rows {
		if want, ok := expected[row.Name]; ok && row.Path == want.Path && row.Method == want.Method {
			present[row.Name] = true
		}
	}
	for _, name := range names {
		if !present[name] && a.logger != nil {
			a.logger.Warn("敏感接口权限码未登记到接口目录（插入因唯一键冲突被跳过），请人工核对 cl_system_apis",
				zap.String("permission", name))
		}
	}
}

// renameAutoNamedPermission 已登记过的敏感接口（如 SEC-29 之前就存在的 SA token 路由）在目录中仍是自动生成的
// 名称与说明时，改为稳定的权限码与说明。只在名称和说明都仍等于自动生成值时更新
// （WHERE id = ? AND name = <自动名> AND description = "<METHOD> <path>"），管理员改过名称或说明（任一项）的行整行不动；
// 更新失败（例如权限码已被其他记录占用）只记日志，不影响启动。
func (a *App) renameAutoNamedPermission(ctx context.Context, route gin.RouteInfo, row model.Api) {
	named, ok := namedPermissions[route.Method+" "+route.Path]
	if !ok || row.ID == 0 || row.Name == named.Name {
		return
	}
	auto := autoPermissionName(route)
	autoDescription := autoPermissionDescription(route)
	if row.Name != auto {
		return
	}
	if row.Description != autoDescription {
		if a.logger != nil {
			a.logger.Info("敏感接口权限行的说明已被管理员修改，保留原名称与说明",
				zap.String("path", route.Path), zap.String("method", route.Method), zap.String("permission", named.Name))
		}
		return
	}
	err := a.db.WithContext(ctx).Model(&model.Api{}).
		Where("id = ? AND name = ? AND description = ?", row.ID, auto, autoDescription).
		Updates(map[string]interface{}{"name": named.Name, "description": named.Description}).Error
	if err != nil && a.logger != nil {
		a.logger.Warn("更新敏感接口权限名失败，保留原名称",
			zap.String("path", route.Path), zap.String("method", route.Method), zap.Error(err))
	}
}

package app

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
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

// bootstrapPermissionCatalog 补充可发现接口，不授予角色或覆盖管理员维护的元数据
func (a *App) bootstrapPermissionCatalog(ctx context.Context) error {
	methods := map[string]int8{"GET": 1, "POST": 2, "PUT": 3, "DELETE": 4}
	public := map[string]bool{"/api/user/login": true, "/api/user/logout": true, "/api/user/refresh_token": true, "/api/not_auth/getBindIps": true, "/api/not_auth/getTreeNodeBindIps": true}
	var existing []model.Api
	if err := a.db.WithContext(ctx).Select("id", "name", "path", "method").Find(&existing).Error; err != nil {
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
		if route.Path == "/api/v1/*path" {
			var aiRoutes []struct {
				Method string `json:"method"`
				Path   string `json:"path"`
			}
			if err := json.Unmarshal(aiopsPermissionCatalog, &aiRoutes); err != nil {
				return fmt.Errorf("解析AI接口权限目录: %w", err)
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
		entry := model.Api{Name: autoPermissionName(route), Path: route.Path, Method: method, Description: route.Method + " " + route.Path, Version: "v1", Category: 2, IsPublic: 0}
		if named, ok := namedPermissions[route.Method+" "+route.Path]; ok {
			entry.Name, entry.Description = named.Name, named.Description
		}
		pending = append(pending, entry)
		seen[key(method, route.Path)] = true
	}
	if len(pending) == 0 {
		return nil
	}
	return a.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(pending, 100).Error
}

// renameAutoNamedPermission 已登记过的敏感接口（如 SEC-29 之前就存在的 SA token 路由）在目录中仍是自动生成的名称时，
// 改为稳定的权限码。只在名称仍等于自动生成值时更新（WHERE name = 自动名），管理员改过的名称不覆盖；
// 更新失败（例如权限码已被其他记录占用）只记日志，不影响启动。
func (a *App) renameAutoNamedPermission(ctx context.Context, route gin.RouteInfo, row model.Api) {
	named, ok := namedPermissions[route.Method+" "+route.Path]
	if !ok || row.ID == 0 || row.Name == named.Name {
		return
	}
	auto := autoPermissionName(route)
	if row.Name != auto {
		return
	}
	err := a.db.WithContext(ctx).Model(&model.Api{}).
		Where("id = ? AND name = ?", row.ID, auto).
		Updates(map[string]interface{}{"name": named.Name, "description": named.Description}).Error
	if err != nil && a.logger != nil {
		a.logger.Warn("更新敏感接口权限名失败，保留原名称",
			zap.String("path", route.Path), zap.String("method", route.Method), zap.Error(err))
	}
}

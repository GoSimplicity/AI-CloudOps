package app

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
)

//go:embed aiops_permissions.json
var aiopsPermissionCatalog []byte

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
	if err := a.db.WithContext(ctx).Select("path", "method").Find(&existing).Error; err != nil {
		return err
	}
	seen := make(map[string]bool, len(existing))
	key := func(method int8, path string) string { return fmt.Sprintf("%d %s", method, path) }
	for _, api := range existing {
		seen[key(api.Method, api.Path)] = true
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
		if !ok || !strings.HasPrefix(route.Path, "/api/") || public[route.Path] || seen[key(method, route.Path)] {
			continue
		}
		digest := sha256.Sum256([]byte(route.Method + " " + route.Path))
		handler := route.Handler[strings.LastIndex(route.Handler, ".")+1:]
		if len(handler) > 25 {
			handler = handler[:25]
		}
		pending = append(pending, model.Api{Name: fmt.Sprintf("%s %s %x", route.Method, handler, digest[:4]), Path: route.Path, Method: method, Description: route.Method + " " + route.Path, Version: "v1", Category: 2, IsPublic: 0})
		seen[key(method, route.Path)] = true
	}
	if len(pending) == 0 {
		return nil
	}
	return a.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(pending, 100).Error
}

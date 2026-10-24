package permission

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

var skipAuthorizationPaths = map[string]bool{
	"/api/user/login":                  true,
	"/api/user/logout":                 true,
	"/api/user/refresh_token":          true,
	"/api/user/profile":                true,
	"/api/user/codes":                  true,
	"/api/user/permissions":            true,
	"/api/not_auth/getBindIps":         true,
	"/api/not_auth/getTreeNodeBindIps": true,
	"/favicon.ico":                     true,
}

// SkipsAuthorization 判断路径是否不经角色授权（登录态仍由登录中间件负责）。
func SkipsAuthorization(path string) bool {
	return skipAuthorizationPaths[path]
}

// RequiresSuperAdmin 判断路由模板是否只允许超管访问（角色授予对其无效）。
func RequiresSuperAdmin(method, template string) bool {
	if method == "POST" && template == "/api/user/signup" {
		return true
	}
	return template == "/api/department" || strings.HasPrefix(template, "/api/department/")
}

// CatalogEntry 是可授权接口目录中的一项。Via 非空表示该接口经由 gin 通配模板
// （代理）提供，授予覆盖 Via 本身也视为覆盖该接口。
type CatalogEntry struct {
	Method int8
	Path   string
	Via    string
}

// Code 返回 "METHOD /path/:param" 形式的权限码。
func (e CatalogEntry) Code() string {
	return MethodName(e.Method) + " " + e.Path
}

// BuildCatalog 根据 gin 已注册路由与已登记的代理目录构建可授权接口目录：
// 只保留 GET/POST/PUT/DELETE、/api/ 前缀、需要角色授权且非超管专属的接口。
func BuildCatalog(routes gin.RoutesInfo) []CatalogEntry {
	seen := map[string]bool{}
	var entries []CatalogEntry
	add := func(method, path, via string) {
		code, ok := MethodCode(method)
		if !ok || !strings.HasPrefix(path, "/api/") || SkipsAuthorization(path) || RequiresSuperAdmin(method, path) {
			return
		}
		key := method + " " + path
		if seen[key] {
			return
		}
		seen[key] = true
		entries = append(entries, CatalogEntry{Method: code, Path: path, Via: via})
	}
	var proxies []string
	for _, route := range routes {
		add(route.Method, route.Path, "")
		if !contains(proxies, route.Path) && len(ProxyCatalog(route.Path)) > 0 {
			proxies = append(proxies, route.Path)
		}
	}
	for _, template := range proxies {
		registered := map[string]bool{}
		for _, route := range routes {
			if route.Path == template {
				registered[route.Method] = true
			}
		}
		for _, route := range ProxyCatalog(template) {
			if registered[route.Method] {
				add(route.Method, route.Path, template)
			}
		}
	}
	return entries
}

// Expand 把授予展开为目录中被覆盖的权限码，结果已排序去重，且不为 nil。
// 判定与鉴权中间件一致：方法相同，且授予覆盖接口模板或其代理通配模板。
func Expand(grants []Grant, catalog []CatalogEntry) []string {
	codes := map[string]bool{}
	for _, entry := range catalog {
		if Allowed(grants, entry.Method, entry.Path) || (entry.Via != "" && Allowed(grants, entry.Method, entry.Via)) {
			codes[entry.Code()] = true
		}
	}
	result := make([]string, 0, len(codes))
	for code := range codes {
		result = append(result, code)
	}
	sort.Strings(result)
	return result
}

func contains(items []string, item string) bool {
	for _, existing := range items {
		if existing == item {
			return true
		}
	}
	return false
}

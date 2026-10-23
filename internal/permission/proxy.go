package permission

import (
	"strings"
	"sync"
)

// AIOpsProxyTemplate 是 AIOps 反向代理在 gin 中注册的通配模板。
const AIOpsProxyTemplate = "/api/v1/*path"

// Route 是代理目录中的一个接口模板。
type Route struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

var (
	proxyMu       sync.RWMutex
	proxyCatalogs = map[string][]Route{}
)

// SetProxyCatalog 为某个 gin 通配模板登记下游接口目录（覆盖原有登记）。
// 代理请求的 c.FullPath() 只有通配模板，鉴权需要借助目录还原出具体接口模板。
func SetProxyCatalog(template string, routes []Route) {
	copied := append([]Route(nil), routes...)
	proxyMu.Lock()
	defer proxyMu.Unlock()
	if len(copied) == 0 {
		delete(proxyCatalogs, template)
		return
	}
	proxyCatalogs[template] = copied
}

// ProxyCatalog 返回某个通配模板登记的下游接口目录副本。
func ProxyCatalog(template string) []Route {
	proxyMu.RLock()
	defer proxyMu.RUnlock()
	return append([]Route(nil), proxyCatalogs[template]...)
}

// ResolveProxyTemplate 把命中代理通配模板的真实请求路径解析为目录中最具体的接口模板。
// fullPath 不是已登记的代理模板、或目录中没有匹配项时返回 ""。
//
// 匹配逐段进行：段数必须相同，静态段必须完全相等，:param 段只匹配一个非空且不为
// "." / ".." 的段。多个模板同时匹配时，最先出现静态段的模板胜出（静态优先于参数，
// 与 gin 路由优先级一致）；完全并列时取目录中靠前的一条。
func ResolveProxyTemplate(method, fullPath, requestPath string) string {
	proxyMu.RLock()
	routes := proxyCatalogs[fullPath]
	proxyMu.RUnlock()
	if len(routes) == 0 {
		return ""
	}
	requested := strings.Split(requestPath, "/")
	best, bestShape := "", []bool(nil)
	for _, route := range routes {
		if route.Method != method {
			continue
		}
		shape, ok := matchSegments(strings.Split(route.Path, "/"), requested)
		if !ok {
			continue
		}
		if best == "" || moreSpecific(shape, bestShape) {
			best, bestShape = route.Path, shape
		}
	}
	return best
}

// matchSegments 返回每一段是否为静态段。
func matchSegments(template, requested []string) ([]bool, bool) {
	if len(template) != len(requested) {
		return nil, false
	}
	shape := make([]bool, len(template))
	for i, part := range template {
		if strings.HasPrefix(part, ":") && len(part) > 1 {
			if requested[i] == "" || requested[i] == "." || requested[i] == ".." {
				return nil, false
			}
			continue
		}
		if part != requested[i] {
			return nil, false
		}
		shape[i] = true
	}
	return shape, true
}

func moreSpecific(a, b []bool) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i]
		}
	}
	return false
}

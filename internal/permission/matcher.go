package permission

import (
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// SuperCode 是 /api/user/permissions 返回给超管的唯一权限码，表示全部放行。
const SuperCode = "*"

var methodCodes = map[string]int8{"GET": 1, "POST": 2, "PUT": 3, "DELETE": 4}
var methodNames = map[int8]string{1: "GET", 2: "POST", 3: "PUT", 4: "DELETE"}

// MethodCode 把 HTTP 方法转换为 cl_system_apis.method 的取值。
func MethodCode(method string) (int8, bool) {
	code, ok := methodCodes[method]
	return code, ok
}

// MethodName 把 cl_system_apis.method 的取值转换为 HTTP 方法；未知取值返回 ""。
func MethodName(code int8) string {
	return methodNames[code]
}

// Grant 是角色授予的一条接口权限（方法 + 授予路径）。
type Grant struct {
	Method int8
	Path   string
}

// GrantsFromRoles 收集启用中角色的接口授予。
func GrantsFromRoles(roles []*model.Role) []Grant {
	var grants []Grant
	for _, role := range roles {
		if role == nil || role.Status != 1 {
			continue
		}
		for _, api := range role.Apis {
			if api == nil {
				continue
			}
			grants = append(grants, Grant{Method: api.Method, Path: api.Path})
		}
	}
	return grants
}

// Covers 判断授予路径是否覆盖某个 gin 路由模板。
//
//   - 模板为空（请求没有命中路由）时永远不覆盖；
//   - 与模板完全相同则覆盖（包括 /api/v1/*path 这类 gin 通配模板本身）；
//   - 只含一个 * 且位于末尾的授予（如 /*、/api/k8s/*）对模板字符串做前缀匹配；
//   - 其他写法（开头或中间的 *、多个 *）不覆盖任何模板；
//   - :param 只是模板里的字面量，不会被当作通配符。
func Covers(granted, template string) bool {
	if template == "" || granted == "" {
		return false
	}
	if granted == template {
		return true
	}
	// Secret 明文读取和 ServiceAccount 令牌操作必须显式授予。
	if template == model.K8sSecretRevealRoute || template == model.K8sServiceAccountTokenRoute {
		return false
	}
	if strings.Count(granted, "*") == 1 && strings.HasSuffix(granted, "*") {
		return strings.HasPrefix(template, strings.TrimSuffix(granted, "*"))
	}
	return false
}

// Allowed 判断授予集合中是否存在方法相同且覆盖任一候选模板的授予。
// 空模板会被忽略。
func Allowed(grants []Grant, method int8, templates ...string) bool {
	for _, grant := range grants {
		if grant.Method != method {
			continue
		}
		for _, template := range templates {
			if Covers(grant.Path, template) {
				return true
			}
		}
	}
	return false
}

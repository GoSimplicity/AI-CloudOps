// Package permission 集中定义 RBAC 的身份判定与授权匹配规则，供鉴权中间件、
// 权限码接口与启动引导共用，避免各处各写一套导致语义漂移。
package permission

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/gin-gonic/gin"
)

const (
	// SuperAdminRoleCode 内置超级管理员角色编码。身份判定同时要求 is_system=1，
	// 普通角色即使使用同样的编码也不会获得超管身份（CreateRole 强制 is_system=0）。
	SuperAdminRoleCode = "super_admin"
	// SuperAdminRoleName 内置超级管理员角色名称。
	SuperAdminRoleName = "超级管理员"
	// SuperAdminRoleDescription 内置超级管理员角色描述。
	SuperAdminRoleDescription = "内置超级管理员角色，由启动引导维护，不可修改或删除"
	// BootstrapAdminUsername 启动引导会绑定到超管角色的内置账号。它只用于引导，
	// 运行期授权从不按用户名判断。
	BootstrapAdminUsername = "admin"

	superAdminContextKey = "cloudops.rbac.super_admin"
)

// IsSuperAdminIdentity 判断角色是否是内置超管角色本身（不看启用状态）。
func IsSuperAdminIdentity(role *model.Role) bool {
	return role != nil && role.IsSystem == 1 && role.Code == SuperAdminRoleCode
}

// IsSuperAdminRole 判断角色是否是启用中的内置超管角色。
func IsSuperAdminRole(role *model.Role) bool {
	return IsSuperAdminIdentity(role) && role.Status == 1
}

// HasSuperAdminRole 判断角色集合中是否包含启用中的内置超管角色。
func HasSuperAdminRole(roles []*model.Role) bool {
	for _, role := range roles {
		if IsSuperAdminRole(role) {
			return true
		}
	}
	return false
}

// MarkSuperAdmin 由鉴权中间件在读取用户角色后写入请求上下文。
func MarkSuperAdmin(c *gin.Context, super bool) {
	c.Set(superAdminContextKey, super)
}

// IsSuperAdmin 读取鉴权中间件写入的超管标记。未经过鉴权中间件（没有标记）时
// 一律视为非超管，保证 fail closed。
func IsSuperAdmin(c *gin.Context) bool {
	if c == nil {
		return false
	}
	value, ok := c.Get(superAdminContextKey)
	if !ok {
		return false
	}
	super, ok := value.(bool)
	return ok && super
}

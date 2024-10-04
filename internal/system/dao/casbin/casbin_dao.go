package casbin

import (
	"context"
	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"
)

// CasbinDAO 定义了与 Casbin 相关的权限管理操作
type CasbinDAO interface {
	// CheckPermission 检查角色是否具有某资源的访问权限
	CheckPermission(ctx context.Context, roleName, path, method string) (bool, error)
	// AddPolicies 批量添加权限策略
	AddPolicies(ctx context.Context, rules [][]string) (bool, error)
	// AddPolicy 添加单条权限策略
	AddPolicy(ctx context.Context, sub, obj, act string) (bool, error)
}

type casbinDAO struct {
	casbin *casbin.Enforcer
	l      *zap.Logger
}

func NewCasbinDAO(casbin *casbin.Enforcer, l *zap.Logger) CasbinDAO {
	return &casbinDAO{
		casbin: casbin,
		l:      l,
	}
}

// CheckPermission 检查指定的角色 roleName 是否有权访问某个资源 path 并执行特定操作 method
func (c *casbinDAO) CheckPermission(_ context.Context, roleName, path, method string) (bool, error) {
	// 使用 Casbin 的 Enforce 方法进行权限检查
	enforce, err := c.casbin.Enforce(roleName, path, method)
	if err != nil {
		c.l.Error("casbin check permission failed", zap.Error(err))
		return false, err
	}

	return enforce, nil
}

// AddPolicies 批量添加多条权限规则
func (c *casbinDAO) AddPolicies(_ context.Context, rules [][]string) (bool, error) {
	// 使用 Casbin 的 AddPolicies 方法批量添加权限规则
	added, err := c.casbin.AddPolicies(rules)
	if err != nil {
		c.l.Error("casbin add policies failed", zap.Error(err))
		return false, err
	}

	return added, nil
}

// AddPolicy 添加单条权限规则
func (c *casbinDAO) AddPolicy(_ context.Context, sub, obj, act string) (bool, error) {
	// 使用 Casbin 的 AddPolicy 方法添加单条权限规则
	added, err := c.casbin.AddPolicy(sub, obj, act)
	if err != nil {
		c.l.Error("casbin add policy failed", zap.Error(err))
		return false, err
	}

	return added, nil
}

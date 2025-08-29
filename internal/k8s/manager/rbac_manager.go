package manager

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	authorizationv1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// RBACManager RBAC 权限管理器（统一管理所有 RBAC 资源）
type RBACManager interface {
	// Role 操作
	CreateRole(ctx context.Context, clusterID int, namespace string, role *rbacv1.Role) error
	GetRole(ctx context.Context, clusterID int, namespace, name string) (*rbacv1.Role, error)
	GetRoleList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*rbacv1.RoleList, error)
	UpdateRole(ctx context.Context, clusterID int, namespace string, role *rbacv1.Role) error
	DeleteRole(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error

	// ClusterRole 操作
	CreateClusterRole(ctx context.Context, clusterID int, clusterRole *rbacv1.ClusterRole) error
	GetClusterRole(ctx context.Context, clusterID int, name string) (*rbacv1.ClusterRole, error)
	GetClusterRoleList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) (*rbacv1.ClusterRoleList, error)
	UpdateClusterRole(ctx context.Context, clusterID int, clusterRole *rbacv1.ClusterRole) error
	DeleteClusterRole(ctx context.Context, clusterID int, name string, deleteOptions metav1.DeleteOptions) error

	// RoleBinding 操作
	CreateRoleBinding(ctx context.Context, clusterID int, namespace string, roleBinding *rbacv1.RoleBinding) error
	GetRoleBinding(ctx context.Context, clusterID int, namespace, name string) (*rbacv1.RoleBinding, error)
	GetRoleBindingList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*rbacv1.RoleBindingList, error)
	UpdateRoleBinding(ctx context.Context, clusterID int, namespace string, roleBinding *rbacv1.RoleBinding) error
	DeleteRoleBinding(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error

	// ClusterRoleBinding 操作
	CreateClusterRoleBinding(ctx context.Context, clusterID int, clusterRoleBinding *rbacv1.ClusterRoleBinding) error
	GetClusterRoleBinding(ctx context.Context, clusterID int, name string) (*rbacv1.ClusterRoleBinding, error)
	GetClusterRoleBindingList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) (*rbacv1.ClusterRoleBindingList, error)
	UpdateClusterRoleBinding(ctx context.Context, clusterID int, clusterRoleBinding *rbacv1.ClusterRoleBinding) error
	DeleteClusterRoleBinding(ctx context.Context, clusterID int, name string, deleteOptions metav1.DeleteOptions) error

	// 批量操作
	BatchDeleteRoles(ctx context.Context, clusterID int, namespace string, roleNames []string) error
	BatchDeleteClusterRoles(ctx context.Context, clusterID int, clusterRoleNames []string) error
	BatchDeleteRoleBindings(ctx context.Context, clusterID int, namespace string, roleBindingNames []string) error
	BatchDeleteClusterRoleBindings(ctx context.Context, clusterID int, clusterRoleBindingNames []string) error

	// 高级功能
	PatchRole(ctx context.Context, clusterID int, namespace, name string, data []byte, patchType string) (*rbacv1.Role, error)
	PatchClusterRole(ctx context.Context, clusterID int, name string, data []byte, patchType string) (*rbacv1.ClusterRole, error)

	// RBAC 权限查询
	GetRolesBySubject(ctx context.Context, clusterID int, namespace, subjectKind, subjectName string) (*rbacv1.RoleList, error)
	GetClusterRolesBySubject(ctx context.Context, clusterID int, subjectKind, subjectName string) (*rbacv1.ClusterRoleList, error)
	CheckUserPermissions(ctx context.Context, clusterID int, username, namespace string, resources []string, verbs []string) (map[string]bool, error)

	// 高级 RBAC 功能（来自 RBACService）
	GetRBACStatistics(ctx context.Context, clusterID int) (*model.RBACStatistics, error)
	CheckPermissions(ctx context.Context, req *model.CheckPermissionsReq) ([]model.PermissionResult, error)
	GetSubjectPermissions(ctx context.Context, req *model.SubjectPermissionsReq) (*model.SubjectPermissionsResponse, error)
	GetResourceVerbs(ctx context.Context) (*model.ResourceVerbsResponse, error)
}

type rbacManager struct {
	logger *zap.Logger
	client client.K8sClient
}

// NewRBACManager 创建新的 RBACManager 实例
func NewRBACManager(logger *zap.Logger, client client.K8sClient) RBACManager {
	return &rbacManager{
		logger: logger,
		client: client,
	}
}

// CreateRole 创建Role
func (r *rbacManager) CreateRole(ctx context.Context, clusterID int, namespace string, role *rbacv1.Role) error {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil {
		r.logger.Error("创建Role失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", role.Name),
			zap.Error(err))
		return err
	}

	r.logger.Info("成功创建Role",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", role.Name))

	return nil
}

// GetRole 获取指定Role
func (r *rbacManager) GetRole(ctx context.Context, clusterID int, namespace, name string) (*rbacv1.Role, error) {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	role, err := kubeClient.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("获取Role失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, err
	}

	return role, nil
}

// GetRoleList 获取Role列表
func (r *rbacManager) GetRoleList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*rbacv1.RoleList, error) {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	roleList, err := kubeClient.RbacV1().Roles(namespace).List(ctx, listOptions)
	if err != nil {
		r.logger.Error("获取Role列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, err
	}

	r.logger.Debug("成功获取Role列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(roleList.Items)))

	return roleList, nil
}

// UpdateRole 更新Role
func (r *rbacManager) UpdateRole(ctx context.Context, clusterID int, namespace string, role *rbacv1.Role) error {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.RbacV1().Roles(namespace).Update(ctx, role, metav1.UpdateOptions{})
	if err != nil {
		r.logger.Error("更新Role失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", role.Name),
			zap.Error(err))
		return err
	}

	r.logger.Info("成功更新Role",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", role.Name))

	return nil
}

// DeleteRole 删除Role
func (r *rbacManager) DeleteRole(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	err = kubeClient.RbacV1().Roles(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		r.logger.Error("删除Role失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return err
	}

	r.logger.Info("成功删除Role",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	return nil
}

// CreateClusterRole 创建ClusterRole
func (r *rbacManager) CreateClusterRole(ctx context.Context, clusterID int, clusterRole *rbacv1.ClusterRole) error {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
	if err != nil {
		r.logger.Error("创建ClusterRole失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", clusterRole.Name),
			zap.Error(err))
		return err
	}

	r.logger.Info("成功创建ClusterRole",
		zap.Int("clusterID", clusterID),
		zap.String("name", clusterRole.Name))

	return nil
}

// GetClusterRole 获取指定ClusterRole
func (r *rbacManager) GetClusterRole(ctx context.Context, clusterID int, name string) (*rbacv1.ClusterRole, error) {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	clusterRole, err := kubeClient.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("获取ClusterRole失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", name),
			zap.Error(err))
		return nil, err
	}

	return clusterRole, nil
}

// GetClusterRoleList 获取ClusterRole列表
func (r *rbacManager) GetClusterRoleList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) (*rbacv1.ClusterRoleList, error) {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	clusterRoleList, err := kubeClient.RbacV1().ClusterRoles().List(ctx, listOptions)
	if err != nil {
		r.logger.Error("获取ClusterRole列表失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	r.logger.Debug("成功获取ClusterRole列表",
		zap.Int("clusterID", clusterID),
		zap.Int("count", len(clusterRoleList.Items)))

	return clusterRoleList, nil
}

// UpdateClusterRole 更新ClusterRole
func (r *rbacManager) UpdateClusterRole(ctx context.Context, clusterID int, clusterRole *rbacv1.ClusterRole) error {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.RbacV1().ClusterRoles().Update(ctx, clusterRole, metav1.UpdateOptions{})
	if err != nil {
		r.logger.Error("更新ClusterRole失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", clusterRole.Name),
			zap.Error(err))
		return err
	}

	r.logger.Info("成功更新ClusterRole",
		zap.Int("clusterID", clusterID),
		zap.String("name", clusterRole.Name))

	return nil
}

// DeleteClusterRole 删除ClusterRole
func (r *rbacManager) DeleteClusterRole(ctx context.Context, clusterID int, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	err = kubeClient.RbacV1().ClusterRoles().Delete(ctx, name, deleteOptions)
	if err != nil {
		r.logger.Error("删除ClusterRole失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", name),
			zap.Error(err))
		return err
	}

	r.logger.Info("成功删除ClusterRole",
		zap.Int("clusterID", clusterID),
		zap.String("name", name))

	return nil
}

// CreateRoleBinding 创建RoleBinding
func (r *rbacManager) CreateRoleBinding(ctx context.Context, clusterID int, namespace string, roleBinding *rbacv1.RoleBinding) error {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		r.logger.Error("创建RoleBinding失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", roleBinding.Name),
			zap.Error(err))
		return err
	}

	r.logger.Info("成功创建RoleBinding",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", roleBinding.Name))

	return nil
}

// GetRoleBinding 获取指定RoleBinding
func (r *rbacManager) GetRoleBinding(ctx context.Context, clusterID int, namespace, name string) (*rbacv1.RoleBinding, error) {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	roleBinding, err := kubeClient.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("获取RoleBinding失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, err
	}

	return roleBinding, nil
}

// GetRoleBindingList 获取RoleBinding列表
func (r *rbacManager) GetRoleBindingList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*rbacv1.RoleBindingList, error) {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	roleBindingList, err := kubeClient.RbacV1().RoleBindings(namespace).List(ctx, listOptions)
	if err != nil {
		r.logger.Error("获取RoleBinding列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, err
	}

	r.logger.Debug("成功获取RoleBinding列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(roleBindingList.Items)))

	return roleBindingList, nil
}

// UpdateRoleBinding 更新RoleBinding
func (r *rbacManager) UpdateRoleBinding(ctx context.Context, clusterID int, namespace string, roleBinding *rbacv1.RoleBinding) error {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.RbacV1().RoleBindings(namespace).Update(ctx, roleBinding, metav1.UpdateOptions{})
	if err != nil {
		r.logger.Error("更新RoleBinding失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", roleBinding.Name),
			zap.Error(err))
		return err
	}

	r.logger.Info("成功更新RoleBinding",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", roleBinding.Name))

	return nil
}

// DeleteRoleBinding 删除RoleBinding
func (r *rbacManager) DeleteRoleBinding(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	err = kubeClient.RbacV1().RoleBindings(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		r.logger.Error("删除RoleBinding失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return err
	}

	r.logger.Info("成功删除RoleBinding",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	return nil
}

// CreateClusterRoleBinding 创建ClusterRoleBinding
func (r *rbacManager) CreateClusterRoleBinding(ctx context.Context, clusterID int, clusterRoleBinding *rbacv1.ClusterRoleBinding) error {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
	if err != nil {
		r.logger.Error("创建ClusterRoleBinding失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", clusterRoleBinding.Name),
			zap.Error(err))
		return err
	}

	r.logger.Info("成功创建ClusterRoleBinding",
		zap.Int("clusterID", clusterID),
		zap.String("name", clusterRoleBinding.Name))

	return nil
}

// GetClusterRoleBinding 获取指定ClusterRoleBinding
func (r *rbacManager) GetClusterRoleBinding(ctx context.Context, clusterID int, name string) (*rbacv1.ClusterRoleBinding, error) {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	clusterRoleBinding, err := kubeClient.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("获取ClusterRoleBinding失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", name),
			zap.Error(err))
		return nil, err
	}

	return clusterRoleBinding, nil
}

// GetClusterRoleBindingList 获取ClusterRoleBinding列表
func (r *rbacManager) GetClusterRoleBindingList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) (*rbacv1.ClusterRoleBindingList, error) {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	clusterRoleBindingList, err := kubeClient.RbacV1().ClusterRoleBindings().List(ctx, listOptions)
	if err != nil {
		r.logger.Error("获取ClusterRoleBinding列表失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	r.logger.Debug("成功获取ClusterRoleBinding列表",
		zap.Int("clusterID", clusterID),
		zap.Int("count", len(clusterRoleBindingList.Items)))

	return clusterRoleBindingList, nil
}

// UpdateClusterRoleBinding 更新ClusterRoleBinding
func (r *rbacManager) UpdateClusterRoleBinding(ctx context.Context, clusterID int, clusterRoleBinding *rbacv1.ClusterRoleBinding) error {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.RbacV1().ClusterRoleBindings().Update(ctx, clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		r.logger.Error("更新ClusterRoleBinding失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", clusterRoleBinding.Name),
			zap.Error(err))
		return err
	}

	r.logger.Info("成功更新ClusterRoleBinding",
		zap.Int("clusterID", clusterID),
		zap.String("name", clusterRoleBinding.Name))

	return nil
}

// DeleteClusterRoleBinding 删除ClusterRoleBinding
func (r *rbacManager) DeleteClusterRoleBinding(ctx context.Context, clusterID int, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	err = kubeClient.RbacV1().ClusterRoleBindings().Delete(ctx, name, deleteOptions)
	if err != nil {
		r.logger.Error("删除ClusterRoleBinding失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", name),
			zap.Error(err))
		return err
	}

	r.logger.Info("成功删除ClusterRoleBinding",
		zap.Int("clusterID", clusterID),
		zap.String("name", name))

	return nil
}

// BatchDeleteRoles 批量删除Role
func (r *rbacManager) BatchDeleteRoles(ctx context.Context, clusterID int, namespace string, roleNames []string) error {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	deleteOptions := metav1.DeleteOptions{}
	var failedDeletions []string

	for _, name := range roleNames {
		err := kubeClient.RbacV1().Roles(namespace).Delete(ctx, name, deleteOptions)
		if err != nil {
			r.logger.Error("删除Role失败",
				zap.Int("clusterID", clusterID),
				zap.String("namespace", namespace),
				zap.String("name", name),
				zap.Error(err))
			failedDeletions = append(failedDeletions, name)
		} else {
			r.logger.Info("成功删除Role",
				zap.Int("clusterID", clusterID),
				zap.String("namespace", namespace),
				zap.String("name", name))
		}
	}

	if len(failedDeletions) > 0 {
		r.logger.Warn("部分Role删除失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Strings("failedDeletions", failedDeletions))
		return err // 返回最后一个错误
	}

	r.logger.Info("批量删除Role完成",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(roleNames)))

	return nil
}

// BatchDeleteClusterRoles 批量删除ClusterRole
func (r *rbacManager) BatchDeleteClusterRoles(ctx context.Context, clusterID int, clusterRoleNames []string) error {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	deleteOptions := metav1.DeleteOptions{}
	var failedDeletions []string

	for _, name := range clusterRoleNames {
		err := kubeClient.RbacV1().ClusterRoles().Delete(ctx, name, deleteOptions)
		if err != nil {
			r.logger.Error("删除ClusterRole失败",
				zap.Int("clusterID", clusterID),
				zap.String("name", name),
				zap.Error(err))
			failedDeletions = append(failedDeletions, name)
		} else {
			r.logger.Info("成功删除ClusterRole",
				zap.Int("clusterID", clusterID),
				zap.String("name", name))
		}
	}

	if len(failedDeletions) > 0 {
		r.logger.Warn("部分ClusterRole删除失败",
			zap.Int("clusterID", clusterID),
			zap.Strings("failedDeletions", failedDeletions))
		return err // 返回最后一个错误
	}

	r.logger.Info("批量删除ClusterRole完成",
		zap.Int("clusterID", clusterID),
		zap.Int("count", len(clusterRoleNames)))

	return nil
}

// BatchDeleteRoleBindings 批量删除RoleBinding
func (r *rbacManager) BatchDeleteRoleBindings(ctx context.Context, clusterID int, namespace string, roleBindingNames []string) error {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	deleteOptions := metav1.DeleteOptions{}
	var failedDeletions []string

	for _, name := range roleBindingNames {
		err := kubeClient.RbacV1().RoleBindings(namespace).Delete(ctx, name, deleteOptions)
		if err != nil {
			r.logger.Error("删除RoleBinding失败",
				zap.Int("clusterID", clusterID),
				zap.String("namespace", namespace),
				zap.String("name", name),
				zap.Error(err))
			failedDeletions = append(failedDeletions, name)
		} else {
			r.logger.Info("成功删除RoleBinding",
				zap.Int("clusterID", clusterID),
				zap.String("namespace", namespace),
				zap.String("name", name))
		}
	}

	if len(failedDeletions) > 0 {
		r.logger.Warn("部分RoleBinding删除失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Strings("failedDeletions", failedDeletions))
		return err // 返回最后一个错误
	}

	r.logger.Info("批量删除RoleBinding完成",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(roleBindingNames)))

	return nil
}

// BatchDeleteClusterRoleBindings 批量删除ClusterRoleBinding
func (r *rbacManager) BatchDeleteClusterRoleBindings(ctx context.Context, clusterID int, clusterRoleBindingNames []string) error {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	deleteOptions := metav1.DeleteOptions{}
	var failedDeletions []string

	for _, name := range clusterRoleBindingNames {
		err := kubeClient.RbacV1().ClusterRoleBindings().Delete(ctx, name, deleteOptions)
		if err != nil {
			r.logger.Error("删除ClusterRoleBinding失败",
				zap.Int("clusterID", clusterID),
				zap.String("name", name),
				zap.Error(err))
			failedDeletions = append(failedDeletions, name)
		} else {
			r.logger.Info("成功删除ClusterRoleBinding",
				zap.Int("clusterID", clusterID),
				zap.String("name", name))
		}
	}

	if len(failedDeletions) > 0 {
		r.logger.Warn("部分ClusterRoleBinding删除失败",
			zap.Int("clusterID", clusterID),
			zap.Strings("failedDeletions", failedDeletions))
		return err // 返回最后一个错误
	}

	r.logger.Info("批量删除ClusterRoleBinding完成",
		zap.Int("clusterID", clusterID),
		zap.Int("count", len(clusterRoleBindingNames)))

	return nil
}

// PatchRole 部分更新Role
func (r *rbacManager) PatchRole(ctx context.Context, clusterID int, namespace, name string, data []byte, patchType string) (*rbacv1.Role, error) {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	// 转换 patch 类型
	pt := types.PatchType(patchType)
	role, err := kubeClient.RbacV1().Roles(namespace).Patch(ctx, name, pt, data, metav1.PatchOptions{})
	if err != nil {
		r.logger.Error("Patch Role失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.String("patchType", patchType),
			zap.Error(err))
		return nil, err
	}

	r.logger.Info("成功Patch Role",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	return role, nil
}

// PatchClusterRole 部分更新ClusterRole
func (r *rbacManager) PatchClusterRole(ctx context.Context, clusterID int, name string, data []byte, patchType string) (*rbacv1.ClusterRole, error) {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	// 转换 patch 类型
	pt := types.PatchType(patchType)
	clusterRole, err := kubeClient.RbacV1().ClusterRoles().Patch(ctx, name, pt, data, metav1.PatchOptions{})
	if err != nil {
		r.logger.Error("Patch ClusterRole失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", name),
			zap.String("patchType", patchType),
			zap.Error(err))
		return nil, err
	}

	r.logger.Info("成功Patch ClusterRole",
		zap.Int("clusterID", clusterID),
		zap.String("name", name))

	return clusterRole, nil
}

// GetRolesBySubject 根据主体获取关联的Role
func (r *rbacManager) GetRolesBySubject(ctx context.Context, clusterID int, namespace, subjectKind, subjectName string) (*rbacv1.RoleList, error) {
	// 首先获取所有RoleBinding
	roleBindings, err := r.GetRoleBindingList(ctx, clusterID, namespace, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var roleNames []string
	for _, rb := range roleBindings.Items {
		for _, subject := range rb.Subjects {
			if subject.Kind == subjectKind && subject.Name == subjectName {
				roleNames = append(roleNames, rb.RoleRef.Name)
				break
			}
		}
	}

	// 获取相关的Role
	var roles []rbacv1.Role
	for _, roleName := range roleNames {
		role, err := r.GetRole(ctx, clusterID, namespace, roleName)
		if err != nil {
			r.logger.Warn("获取Role失败",
				zap.Int("clusterID", clusterID),
				zap.String("namespace", namespace),
				zap.String("roleName", roleName),
				zap.Error(err))
			continue
		}
		roles = append(roles, *role)
	}

	roleList := &rbacv1.RoleList{
		Items: roles,
	}

	r.logger.Debug("根据主体获取Role列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("subjectKind", subjectKind),
		zap.String("subjectName", subjectName),
		zap.Int("count", len(roles)))

	return roleList, nil
}

// GetClusterRolesBySubject 根据主体获取关联的ClusterRole
func (r *rbacManager) GetClusterRolesBySubject(ctx context.Context, clusterID int, subjectKind, subjectName string) (*rbacv1.ClusterRoleList, error) {
	// 首先获取所有ClusterRoleBinding
	clusterRoleBindings, err := r.GetClusterRoleBindingList(ctx, clusterID, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var clusterRoleNames []string
	for _, crb := range clusterRoleBindings.Items {
		for _, subject := range crb.Subjects {
			if subject.Kind == subjectKind && subject.Name == subjectName {
				clusterRoleNames = append(clusterRoleNames, crb.RoleRef.Name)
				break
			}
		}
	}

	// 获取相关的ClusterRole
	var clusterRoles []rbacv1.ClusterRole
	for _, clusterRoleName := range clusterRoleNames {
		clusterRole, err := r.GetClusterRole(ctx, clusterID, clusterRoleName)
		if err != nil {
			r.logger.Warn("获取ClusterRole失败",
				zap.Int("clusterID", clusterID),
				zap.String("clusterRoleName", clusterRoleName),
				zap.Error(err))
			continue
		}
		clusterRoles = append(clusterRoles, *clusterRole)
	}

	clusterRoleList := &rbacv1.ClusterRoleList{
		Items: clusterRoles,
	}

	r.logger.Debug("根据主体获取ClusterRole列表",
		zap.Int("clusterID", clusterID),
		zap.String("subjectKind", subjectKind),
		zap.String("subjectName", subjectName),
		zap.Int("count", len(clusterRoles)))

	return clusterRoleList, nil
}

// CheckUserPermissions 检查用户权限
func (r *rbacManager) CheckUserPermissions(ctx context.Context, clusterID int, username, namespace string, resources []string, verbs []string) (map[string]bool, error) {
	// 获取用户相关的Role和ClusterRole
	roles, err := r.GetRolesBySubject(ctx, clusterID, namespace, "User", username)
	if err != nil {
		r.logger.Error("获取用户Role失败", zap.Error(err))
		return nil, err
	}

	clusterRoles, err := r.GetClusterRolesBySubject(ctx, clusterID, "User", username)
	if err != nil {
		r.logger.Error("获取用户ClusterRole失败", zap.Error(err))
		return nil, err
	}

	// 构建权限映射
	permissions := make(map[string]bool)

	// 初始化所有权限为false
	for _, resource := range resources {
		for _, verb := range verbs {
			key := resource + ":" + verb
			permissions[key] = false
		}
	}

	// 检查Role权限
	for _, role := range roles.Items {
		for _, rule := range role.Rules {
			for _, resource := range resources {
				for _, verb := range verbs {
					if r.hasPermission(rule, resource, verb) {
						key := resource + ":" + verb
						permissions[key] = true
					}
				}
			}
		}
	}

	// 检查ClusterRole权限
	for _, clusterRole := range clusterRoles.Items {
		for _, rule := range clusterRole.Rules {
			for _, resource := range resources {
				for _, verb := range verbs {
					if r.hasPermission(rule, resource, verb) {
						key := resource + ":" + verb
						permissions[key] = true
					}
				}
			}
		}
	}

	r.logger.Debug("检查用户权限完成",
		zap.Int("clusterID", clusterID),
		zap.String("username", username),
		zap.String("namespace", namespace),
		zap.Any("permissions", permissions))

	return permissions, nil
}

// hasPermission 检查规则是否包含指定的资源和动作权限
func (r *rbacManager) hasPermission(rule rbacv1.PolicyRule, resource, verb string) bool {
	// 检查动作权限
	hasVerb := false
	for _, v := range rule.Verbs {
		if v == "*" || v == verb {
			hasVerb = true
			break
		}
	}
	if !hasVerb {
		return false
	}

	// 检查资源权限
	hasResource := false
	for _, res := range rule.Resources {
		if res == "*" || res == resource {
			hasResource = true
			break
		}
	}

	return hasResource
}

// ========== 高级 RBAC 功能实现（来自 RBACService）==========

// GetRBACStatistics 获取RBAC统计信息
func (r *rbacManager) GetRBACStatistics(ctx context.Context, clusterID int) (*model.RBACStatistics, error) {
	kubeClient, err := r.client.GetKubeClient(clusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	stats := &model.RBACStatistics{}

	// 统计Roles
	roles, err := kubeClient.RbacV1().Roles("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.TotalRoles = len(roles.Items)
	}

	// 统计ClusterRoles
	clusterRoles, err := kubeClient.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.TotalClusterRoles = len(clusterRoles.Items)
		// 统计系统和自定义角色
		for _, cr := range clusterRoles.Items {
			if strings.HasPrefix(cr.Name, "system:") {
				stats.SystemRoles++
			} else {
				stats.CustomRoles++
			}
		}
	}

	// 统计RoleBindings
	roleBindings, err := kubeClient.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.TotalRoleBindings = len(roleBindings.Items)
	}

	// 统计ClusterRoleBindings
	clusterRoleBindings, err := kubeClient.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.TotalClusterRoleBindings = len(clusterRoleBindings.Items)
	}

	// 统计活跃主体
	activeSubjects := make(map[string]bool)
	if roleBindings != nil {
		for _, rb := range roleBindings.Items {
			for _, subject := range rb.Subjects {
				key := fmt.Sprintf("%s:%s:%s", subject.Kind, subject.Name, subject.Namespace)
				activeSubjects[key] = true
			}
		}
	}
	if clusterRoleBindings != nil {
		for _, crb := range clusterRoleBindings.Items {
			for _, subject := range crb.Subjects {
				key := fmt.Sprintf("%s:%s:%s", subject.Kind, subject.Name, subject.Namespace)
				activeSubjects[key] = true
			}
		}
	}
	stats.ActiveSubjects = len(activeSubjects)

	r.logger.Info("获取RBAC统计信息成功",
		zap.Int("clusterID", clusterID),
		zap.Int("totalRoles", stats.TotalRoles),
		zap.Int("totalClusterRoles", stats.TotalClusterRoles),
		zap.Int("activeSubjects", stats.ActiveSubjects))

	return stats, nil
}

// CheckPermissions 检查权限
func (r *rbacManager) CheckPermissions(ctx context.Context, req *model.CheckPermissionsReq) ([]model.PermissionResult, error) {
	kubeClient, err := r.client.GetKubeClient(req.ClusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", req.ClusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	var results []model.PermissionResult

	for _, resource := range req.Resources {
		// 构建SubjectAccessReview
		sar := &authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User:   req.Subject.Name,
				Groups: []string{req.Subject.APIGroup},
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Namespace: resource.Namespace,
					Verb:      resource.Verb,
					Resource:  resource.Resource,
				},
			},
		}

		// 如果主体是ServiceAccount，设置UID
		if req.Subject.Kind == "ServiceAccount" {
			sar.Spec.UID = req.Subject.Name
		}

		// 执行权限检查
		result, err := kubeClient.AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
		if err != nil {
			r.logger.Error("权限检查失败",
				zap.String("subject", req.Subject.Name),
				zap.String("resource", resource.Resource),
				zap.String("verb", resource.Verb),
				zap.Error(err))
			results = append(results, model.PermissionResult{
				Namespace: resource.Namespace,
				Resource:  resource.Resource,
				Verb:      resource.Verb,
				Allowed:   false,
				Reason:    fmt.Sprintf("权限检查失败: %v", err),
			})
			continue
		}

		results = append(results, model.PermissionResult{
			Namespace: resource.Namespace,
			Resource:  resource.Resource,
			Verb:      resource.Verb,
			Allowed:   result.Status.Allowed,
			Reason:    result.Status.Reason,
		})
	}

	r.logger.Info("权限检查完成",
		zap.String("subject", req.Subject.Name),
		zap.Int("totalChecks", len(results)))

	return results, nil
}

// GetSubjectPermissions 获取主体的有效权限列表
func (r *rbacManager) GetSubjectPermissions(ctx context.Context, req *model.SubjectPermissionsReq) (*model.SubjectPermissionsResponse, error) {
	kubeClient, err := r.client.GetKubeClient(req.ClusterID)
	if err != nil {
		r.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", req.ClusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	response := &model.SubjectPermissionsResponse{
		Subject:      req.Subject,
		Permissions:  []model.PolicyRule{},
		Roles:        []string{},
		ClusterRoles: []string{},
	}

	// 获取所有RoleBindings
	roleBindings, err := kubeClient.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{})
	if err != nil {
		r.logger.Error("获取RoleBindings失败",
			zap.Int("clusterID", req.ClusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取RoleBindings失败: %w", err)
	}

	// 检查RoleBindings中的权限
	for _, rb := range roleBindings.Items {
		if r.isSubjectInBinding(req.Subject, rb.Subjects) {
			// 获取对应的Role
			if rb.RoleRef.Kind == "Role" {
				role, err := kubeClient.RbacV1().Roles(rb.Namespace).Get(ctx, rb.RoleRef.Name, metav1.GetOptions{})
				if err == nil {
					response.Roles = append(response.Roles, fmt.Sprintf("%s/%s", rb.Namespace, role.Name))
					for _, rule := range role.Rules {
						response.Permissions = append(response.Permissions, model.PolicyRule{
							APIGroups:       rule.APIGroups,
							Resources:       rule.Resources,
							Verbs:           rule.Verbs,
							ResourceNames:   rule.ResourceNames,
							NonResourceURLs: rule.NonResourceURLs,
						})
					}
				}
			} else if rb.RoleRef.Kind == "ClusterRole" {
				clusterRole, err := kubeClient.RbacV1().ClusterRoles().Get(ctx, rb.RoleRef.Name, metav1.GetOptions{})
				if err == nil {
					response.ClusterRoles = append(response.ClusterRoles, clusterRole.Name)
					for _, rule := range clusterRole.Rules {
						response.Permissions = append(response.Permissions, model.PolicyRule{
							APIGroups:       rule.APIGroups,
							Resources:       rule.Resources,
							Verbs:           rule.Verbs,
							ResourceNames:   rule.ResourceNames,
							NonResourceURLs: rule.NonResourceURLs,
						})
					}
				}
			}
		}
	}

	// 获取所有ClusterRoleBindings
	clusterRoleBindings, err := kubeClient.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		r.logger.Error("获取ClusterRoleBindings失败",
			zap.Int("clusterID", req.ClusterID),
			zap.Error(err))
		return nil, fmt.Errorf("获取ClusterRoleBindings失败: %w", err)
	}

	// 检查ClusterRoleBindings中的权限
	for _, crb := range clusterRoleBindings.Items {
		if r.isSubjectInBinding(req.Subject, crb.Subjects) {
			clusterRole, err := kubeClient.RbacV1().ClusterRoles().Get(ctx, crb.RoleRef.Name, metav1.GetOptions{})
			if err == nil {
				response.ClusterRoles = append(response.ClusterRoles, clusterRole.Name)
				for _, rule := range clusterRole.Rules {
					response.Permissions = append(response.Permissions, model.PolicyRule{
						APIGroups:       rule.APIGroups,
						Resources:       rule.Resources,
						Verbs:           rule.Verbs,
						ResourceNames:   rule.ResourceNames,
						NonResourceURLs: rule.NonResourceURLs,
					})
				}
			}
		}
	}

	r.logger.Info("获取主体权限完成",
		zap.String("subject", req.Subject.Name),
		zap.Int("totalRoles", len(response.Roles)),
		zap.Int("totalClusterRoles", len(response.ClusterRoles)),
		zap.Int("totalPermissions", len(response.Permissions)))

	return response, nil
}

// GetResourceVerbs 获取预定义的资源和动作列表
func (r *rbacManager) GetResourceVerbs(ctx context.Context) (*model.ResourceVerbsResponse, error) {
	// 预定义常用的Kubernetes资源和动作
	resources := []model.ResourceInfo{
		// Core resources
		{APIGroup: "", Resource: "pods", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "po"},
		{APIGroup: "", Resource: "services", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "svc"},
		{APIGroup: "", Resource: "configmaps", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "cm"},
		{APIGroup: "", Resource: "secrets", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}},
		{APIGroup: "", Resource: "persistentvolumes", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "pv"},
		{APIGroup: "", Resource: "persistentvolumeclaims", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "pvc"},
		{APIGroup: "", Resource: "namespaces", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "ns"},
		{APIGroup: "", Resource: "nodes", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}},
		{APIGroup: "", Resource: "serviceaccounts", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "sa"},
		{APIGroup: "", Resource: "events", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}},

		// Apps resources
		{APIGroup: "apps", Resource: "deployments", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "deploy"},
		{APIGroup: "apps", Resource: "statefulsets", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "sts"},
		{APIGroup: "apps", Resource: "daemonsets", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "ds"},
		{APIGroup: "apps", Resource: "replicasets", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "rs"},

		// RBAC resources
		{APIGroup: "rbac.authorization.k8s.io", Resource: "roles", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}},
		{APIGroup: "rbac.authorization.k8s.io", Resource: "clusterroles", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}},
		{APIGroup: "rbac.authorization.k8s.io", Resource: "rolebindings", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}},
		{APIGroup: "rbac.authorization.k8s.io", Resource: "clusterrolebindings", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}},

		// Networking resources
		{APIGroup: "networking.k8s.io", Resource: "ingresses", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "ing"},
		{APIGroup: "networking.k8s.io", Resource: "networkpolicies", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "netpol"},

		// Batch resources
		{APIGroup: "batch", Resource: "jobs", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}},
		{APIGroup: "batch", Resource: "cronjobs", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "cj"},

		// Autoscaling resources
		{APIGroup: "autoscaling", Resource: "horizontalpodautoscalers", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "hpa"},

		// Metrics resources
		{APIGroup: "metrics.k8s.io", Resource: "nodes", Verbs: []string{"get", "list"}},
		{APIGroup: "metrics.k8s.io", Resource: "pods", Verbs: []string{"get", "list"}},

		// Custom resources (example)
		{APIGroup: "*", Resource: "*", Verbs: []string{"*"}},
	}

	response := &model.ResourceVerbsResponse{
		Resources: resources,
	}

	r.logger.Info("获取资源动作列表成功", zap.Int("resourceCount", len(resources)))

	return response, nil
}

// isSubjectInBinding 辅助方法：检查主体是否在绑定列表中
func (r *rbacManager) isSubjectInBinding(subject model.Subject, subjects []rbacv1.Subject) bool {
	for _, s := range subjects {
		if s.Kind == subject.Kind && s.Name == subject.Name {
			// 对于ServiceAccount，还需要检查命名空间
			if subject.Kind == "ServiceAccount" {
				return s.Namespace == subject.Namespace
			}
			return true
		}
	}
	return false
}

package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RBACService RBAC（基于角色的访问控制）服务接口
// 提供对 Kubernetes RBAC 资源的完整管理功能，包括 Role、ClusterRole、RoleBinding、ClusterRoleBinding
type RBACService interface {
	// Role 管理接口
	// Role 是命名空间级别的权限定义，只在特定命名空间内生效

	// GetRolesByNamespace 获取指定命名空间下的所有 Role
	// @param ctx 上下文
	// @param clusterID 集群ID
	// @param namespace 命名空间名称
	// @return []model.K8sRole Role列表
	// @return error 错误信息
	GetRolesByNamespace(ctx context.Context, clusterID int, namespace string) ([]model.K8sRole, error)

	// GetRole 获取指定的 Role 详细信息
	// @param ctx 上下文
	// @param clusterID 集群ID
	// @param namespace 命名空间名称
	// @param name Role名称
	// @return *model.K8sRole Role详细信息
	// @return error 错误信息
	GetRole(ctx context.Context, clusterID int, namespace, name string) (*model.K8sRole, error)

	// CreateRole 创建新的 Role
	// @param ctx 上下文
	// @param req 创建请求参数
	// @return error 错误信息
	CreateRole(ctx context.Context, req model.CreateK8sRoleRequest) error

	// UpdateRole 更新现有的 Role
	// @param ctx 上下文
	// @param req 更新请求参数
	// @return error 错误信息
	UpdateRole(ctx context.Context, req model.UpdateK8sRoleRequest) error

	// DeleteRole 删除指定的 Role
	// @param ctx 上下文
	// @param req 删除请求参数
	// @return error 错误信息
	DeleteRole(ctx context.Context, req model.DeleteK8sRoleRequest) error

	// ClusterRole 管理接口
	// ClusterRole 是集群级别的权限定义，可以跨命名空间使用

	// GetClusterRoles 获取集群中的所有 ClusterRole
	// @param ctx 上下文
	// @param clusterID 集群ID
	// @return []model.K8sClusterRole ClusterRole列表
	// @return error 错误信息
	GetClusterRoles(ctx context.Context, clusterID int) ([]model.K8sClusterRole, error)

	// GetClusterRole 获取指定的 ClusterRole 详细信息
	// @param ctx 上下文
	// @param clusterID 集群ID
	// @param name ClusterRole名称
	// @return *model.K8sClusterRole ClusterRole详细信息
	// @return error 错误信息
	GetClusterRole(ctx context.Context, clusterID int, name string) (*model.K8sClusterRole, error)

	// CreateClusterRole 创建新的 ClusterRole
	// @param ctx 上下文
	// @param req 创建请求参数
	// @return error 错误信息
	CreateClusterRole(ctx context.Context, req model.CreateClusterRoleRequest) error

	// UpdateClusterRole 更新现有的 ClusterRole
	// @param ctx 上下文
	// @param req 更新请求参数
	// @return error 错误信息
	UpdateClusterRole(ctx context.Context, req model.UpdateClusterRoleRequest) error

	// DeleteClusterRole 删除指定的 ClusterRole
	// @param ctx 上下文
	// @param req 删除请求参数
	// @return error 错误信息
	DeleteClusterRole(ctx context.Context, req model.DeleteClusterRoleRequest) error

	// RoleBinding 管理接口
	// RoleBinding 将 Role 与用户、组或服务账户绑定，在指定命名空间内生效

	// GetRoleBindingsByNamespace 获取指定命名空间下的所有 RoleBinding
	// @param ctx 上下文
	// @param clusterID 集群ID
	// @param namespace 命名空间名称
	// @return []model.K8sRoleBinding RoleBinding列表
	// @return error 错误信息
	GetRoleBindingsByNamespace(ctx context.Context, clusterID int, namespace string) ([]model.K8sRoleBinding, error)

	// GetRoleBinding 获取指定的 RoleBinding 详细信息
	// @param ctx 上下文
	// @param clusterID 集群ID
	// @param namespace 命名空间名称
	// @param name RoleBinding名称
	// @return *model.K8sRoleBinding RoleBinding详细信息
	// @return error 错误信息
	GetRoleBinding(ctx context.Context, clusterID int, namespace, name string) (*model.K8sRoleBinding, error)

	// CreateRoleBinding 创建新的 RoleBinding
	// @param ctx 上下文
	// @param req 创建请求参数
	// @return error 错误信息
	CreateRoleBinding(ctx context.Context, req model.CreateRoleBindingRequest) error

	// UpdateRoleBinding 更新现有的 RoleBinding
	// @param ctx 上下文
	// @param req 更新请求参数
	// @return error 错误信息
	UpdateRoleBinding(ctx context.Context, req model.UpdateRoleBindingRequest) error

	// DeleteRoleBinding 删除指定的 RoleBinding
	// @param ctx 上下文
	// @param req 删除请求参数
	// @return error 错误信息
	DeleteRoleBinding(ctx context.Context, req model.DeleteRoleBindingRequest) error

	// ClusterRoleBinding 管理接口
	// ClusterRoleBinding 将 ClusterRole 与用户、组或服务账户绑定，在整个集群范围内生效

	// GetClusterRoleBindings 获取集群中的所有 ClusterRoleBinding
	// @param ctx 上下文
	// @param clusterID 集群ID
	// @return []model.K8sClusterRoleBinding ClusterRoleBinding列表
	// @return error 错误信息
	GetClusterRoleBindings(ctx context.Context, clusterID int) ([]model.K8sClusterRoleBinding, error)

	// GetClusterRoleBinding 获取指定的 ClusterRoleBinding 详细信息
	// @param ctx 上下文
	// @param clusterID 集群ID
	// @param name ClusterRoleBinding名称
	// @return *model.K8sClusterRoleBinding ClusterRoleBinding详细信息
	// @return error 错误信息
	GetClusterRoleBinding(ctx context.Context, clusterID int, name string) (*model.K8sClusterRoleBinding, error)

	// CreateClusterRoleBinding 创建新的 ClusterRoleBinding
	// @param ctx 上下文
	// @param req 创建请求参数
	// @return error 错误信息
	CreateClusterRoleBinding(ctx context.Context, req model.CreateClusterRoleBindingRequest) error

	// UpdateClusterRoleBinding 更新现有的 ClusterRoleBinding
	// @param ctx 上下文
	// @param req 更新请求参数
	// @return error 错误信息
	UpdateClusterRoleBinding(ctx context.Context, req model.UpdateClusterRoleBindingRequest) error

	// DeleteClusterRoleBinding 删除指定的 ClusterRoleBinding
	// @param ctx 上下文
	// @param req 删除请求参数
	// @return error 错误信息
	DeleteClusterRoleBinding(ctx context.Context, req model.DeleteClusterRoleBindingRequest) error
}

// rbacService RBAC服务实现结构体
type rbacService struct {
	logger    *zap.Logger      // 日志记录器
	k8sClient client.K8sClient // Kubernetes客户端
}

// NewRBACService 创建新的RBAC服务实例
// 参数:
//
//	logger: 日志记录器
//	k8sClient: Kubernetes客户端
//
// 返回: RBACService RBAC服务接口实例
func NewRBACService(logger *zap.Logger, k8sClient client.K8sClient) RBACService {
	return &rbacService{
		logger:    logger,
		k8sClient: k8sClient,
	}
}

// ========== Role 管理实现 ==========

// GetRolesByNamespace 获取指定命名空间下的所有Role
func (r *rbacService) GetRolesByNamespace(ctx context.Context, clusterID int, namespace string) ([]model.K8sRole, error) {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(clusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API获取Role列表
	roles, err := clientset.RbacV1().Roles(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		r.logger.Error("获取 Role 列表失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID),
			zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取 Role 列表失败: %w", err)
	}

	// 转换Kubernetes原生Role对象为内部模型
	var result []model.K8sRole
	for _, role := range roles.Items {
		result = append(result, r.convertRole(&role))
	}

	return result, nil
}

// GetRole 获取指定的Role详细信息
func (r *rbacService) GetRole(ctx context.Context, clusterID int, namespace, name string) (*model.K8sRole, error) {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(clusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API获取指定Role
	role, err := clientset.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("获取 Role 失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name))
		return nil, fmt.Errorf("获取 Role 失败: %w", err)
	}

	// 转换为内部模型并返回
	result := r.convertRole(role)
	return &result, nil
}

// CreateRole 创建新的Role
func (r *rbacService) CreateRole(ctx context.Context, req model.CreateK8sRoleRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 构建Kubernetes Role对象
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      r.convertStringListToMap(req.Labels),      // 转换标签
			Annotations: r.convertStringListToMap(req.Annotations), // 转换注解
		},
		Rules: r.convertPolicyRules(req.Rules), // 转换策略规则
	}

	// 调用Kubernetes API创建Role
	_, err = clientset.RbacV1().Roles(req.Namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil {
		r.logger.Error("创建 Role 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建 Role 失败: %w", err)
	}

	r.logger.Info("Role 创建成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// UpdateRole 更新现有的Role
func (r *rbacService) UpdateRole(ctx context.Context, req model.UpdateK8sRoleRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 首先获取现有的Role
	existingRole, err := clientset.RbacV1().Roles(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("获取现有 Role 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("获取现有 Role 失败: %w", err)
	}

	// 更新Role的字段
	existingRole.Labels = r.convertStringListToMap(req.Labels)
	existingRole.Annotations = r.convertStringListToMap(req.Annotations)
	existingRole.Rules = r.convertPolicyRules(req.Rules)

	// 调用Kubernetes API更新Role
	_, err = clientset.RbacV1().Roles(req.Namespace).Update(ctx, existingRole, metav1.UpdateOptions{})
	if err != nil {
		r.logger.Error("更新 Role 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新 Role 失败: %w", err)
	}

	r.logger.Info("Role 更新成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// DeleteRole 删除指定的Role
func (r *rbacService) DeleteRole(ctx context.Context, req model.DeleteK8sRoleRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API删除Role
	err = clientset.RbacV1().Roles(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{})
	if err != nil {
		r.logger.Error("删除 Role 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除 Role 失败: %w", err)
	}

	r.logger.Info("Role 删除成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// ========== ClusterRole 管理实现 ==========

// GetClusterRoles 获取集群中的所有ClusterRole
func (r *rbacService) GetClusterRoles(ctx context.Context, clusterID int) ([]model.K8sClusterRole, error) {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(clusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API获取ClusterRole列表
	clusterRoles, err := clientset.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		r.logger.Error("获取 ClusterRole 列表失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取 ClusterRole 列表失败: %w", err)
	}

	// 转换Kubernetes原生ClusterRole对象为内部模型
	var result []model.K8sClusterRole
	for _, clusterRole := range clusterRoles.Items {
		result = append(result, r.convertClusterRole(&clusterRole))
	}

	return result, nil
}

// GetClusterRole 获取指定的ClusterRole详细信息
func (r *rbacService) GetClusterRole(ctx context.Context, clusterID int, name string) (*model.K8sClusterRole, error) {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(clusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API获取指定ClusterRole
	clusterRole, err := clientset.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("获取 ClusterRole 失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID),
			zap.String("name", name))
		return nil, fmt.Errorf("获取 ClusterRole 失败: %w", err)
	}

	// 转换为内部模型并返回
	result := r.convertClusterRole(clusterRole)
	return &result, nil
}

// CreateClusterRole 创建新的ClusterRole
func (r *rbacService) CreateClusterRole(ctx context.Context, req model.CreateClusterRoleRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 构建Kubernetes ClusterRole对象
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Labels:      r.convertStringListToMap(req.Labels),      // 转换标签
			Annotations: r.convertStringListToMap(req.Annotations), // 转换注解
		},
		Rules: r.convertPolicyRules(req.Rules), // 转换策略规则
	}

	// 调用Kubernetes API创建ClusterRole
	_, err = clientset.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
	if err != nil {
		r.logger.Error("创建 ClusterRole 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("创建 ClusterRole 失败: %w", err)
	}

	r.logger.Info("ClusterRole 创建成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("name", req.Name))
	return nil
}

// UpdateClusterRole 更新现有的ClusterRole
func (r *rbacService) UpdateClusterRole(ctx context.Context, req model.UpdateClusterRoleRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 首先获取现有的ClusterRole
	existingClusterRole, err := clientset.RbacV1().ClusterRoles().Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("获取现有 ClusterRole 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("获取现有 ClusterRole 失败: %w", err)
	}

	// 更新ClusterRole的字段
	existingClusterRole.Labels = r.convertStringListToMap(req.Labels)
	existingClusterRole.Annotations = r.convertStringListToMap(req.Annotations)
	existingClusterRole.Rules = r.convertPolicyRules(req.Rules)

	// 调用Kubernetes API更新ClusterRole
	_, err = clientset.RbacV1().ClusterRoles().Update(ctx, existingClusterRole, metav1.UpdateOptions{})
	if err != nil {
		r.logger.Error("更新 ClusterRole 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("更新 ClusterRole 失败: %w", err)
	}

	r.logger.Info("ClusterRole 更新成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("name", req.Name))
	return nil
}

// DeleteClusterRole 删除指定的ClusterRole
func (r *rbacService) DeleteClusterRole(ctx context.Context, req model.DeleteClusterRoleRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API删除ClusterRole
	err = clientset.RbacV1().ClusterRoles().Delete(ctx, req.Name, metav1.DeleteOptions{})
	if err != nil {
		r.logger.Error("删除 ClusterRole 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("删除 ClusterRole 失败: %w", err)
	}

	r.logger.Info("ClusterRole 删除成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("name", req.Name))
	return nil
}

// ========== RoleBinding 管理实现 ==========

// GetRoleBindingsByNamespace 获取指定命名空间下的所有RoleBinding
func (r *rbacService) GetRoleBindingsByNamespace(ctx context.Context, clusterID int, namespace string) ([]model.K8sRoleBinding, error) {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(clusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API获取RoleBinding列表
	roleBindings, err := clientset.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		r.logger.Error("获取 RoleBinding 列表失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID),
			zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取 RoleBinding 列表失败: %w", err)
	}

	// 转换Kubernetes原生RoleBinding对象为内部模型
	var result []model.K8sRoleBinding
	for _, roleBinding := range roleBindings.Items {
		result = append(result, r.convertRoleBinding(&roleBinding))
	}

	return result, nil
}

// GetRoleBinding 获取指定的RoleBinding详细信息
func (r *rbacService) GetRoleBinding(ctx context.Context, clusterID int, namespace, name string) (*model.K8sRoleBinding, error) {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(clusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API获取指定RoleBinding
	roleBinding, err := clientset.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("获取 RoleBinding 失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name))
		return nil, fmt.Errorf("获取 RoleBinding 失败: %w", err)
	}

	// 转换为内部模型并返回
	result := r.convertRoleBinding(roleBinding)
	return &result, nil
}

// CreateRoleBinding 创建新的RoleBinding
func (r *rbacService) CreateRoleBinding(ctx context.Context, req model.CreateRoleBindingRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 构建Kubernetes RoleBinding对象
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      r.convertStringListToMap(req.Labels),      // 转换标签
			Annotations: r.convertStringListToMap(req.Annotations), // 转换注解
		},
		Subjects: r.convertSubjects(req.Subjects), // 转换绑定主体
		RoleRef:  r.convertRoleRef(req.RoleRef),   // 转换角色引用
	}

	// 调用Kubernetes API创建RoleBinding
	_, err = clientset.RbacV1().RoleBindings(req.Namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		r.logger.Error("创建 RoleBinding 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建 RoleBinding 失败: %w", err)
	}

	r.logger.Info("RoleBinding 创建成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// UpdateRoleBinding 更新现有的RoleBinding
func (r *rbacService) UpdateRoleBinding(ctx context.Context, req model.UpdateRoleBindingRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 首先获取现有的RoleBinding
	existingRoleBinding, err := clientset.RbacV1().RoleBindings(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("获取现有 RoleBinding 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("获取现有 RoleBinding 失败: %w", err)
	}

	// 更新RoleBinding的字段
	existingRoleBinding.Labels = r.convertStringListToMap(req.Labels)
	existingRoleBinding.Annotations = r.convertStringListToMap(req.Annotations)
	existingRoleBinding.Subjects = r.convertSubjects(req.Subjects)
	existingRoleBinding.RoleRef = r.convertRoleRef(req.RoleRef)

	// 调用Kubernetes API更新RoleBinding
	_, err = clientset.RbacV1().RoleBindings(req.Namespace).Update(ctx, existingRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		r.logger.Error("更新 RoleBinding 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新 RoleBinding 失败: %w", err)
	}

	r.logger.Info("RoleBinding 更新成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// DeleteRoleBinding 删除指定的RoleBinding
func (r *rbacService) DeleteRoleBinding(ctx context.Context, req model.DeleteRoleBindingRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API删除RoleBinding
	err = clientset.RbacV1().RoleBindings(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{})
	if err != nil {
		r.logger.Error("删除 RoleBinding 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除 RoleBinding 失败: %w", err)
	}

	r.logger.Info("RoleBinding 删除成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// ========== ClusterRoleBinding 管理实现 ==========

// GetClusterRoleBindings 获取集群中的所有ClusterRoleBinding
func (r *rbacService) GetClusterRoleBindings(ctx context.Context, clusterID int) ([]model.K8sClusterRoleBinding, error) {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(clusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API获取ClusterRoleBinding列表
	clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		r.logger.Error("获取 ClusterRoleBinding 列表失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取 ClusterRoleBinding 列表失败: %w", err)
	}

	// 转换Kubernetes原生ClusterRoleBinding对象为内部模型
	var result []model.K8sClusterRoleBinding
	for _, clusterRoleBinding := range clusterRoleBindings.Items {
		result = append(result, r.convertClusterRoleBinding(&clusterRoleBinding))
	}

	return result, nil
}

// GetClusterRoleBinding 获取指定的ClusterRoleBinding详细信息
func (r *rbacService) GetClusterRoleBinding(ctx context.Context, clusterID int, name string) (*model.K8sClusterRoleBinding, error) {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(clusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API获取指定ClusterRoleBinding
	clusterRoleBinding, err := clientset.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("获取 ClusterRoleBinding 失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID),
			zap.String("name", name))
		return nil, fmt.Errorf("获取 ClusterRoleBinding 失败: %w", err)
	}

	// 转换为内部模型并返回
	result := r.convertClusterRoleBinding(clusterRoleBinding)
	return &result, nil
}

// CreateClusterRoleBinding 创建新的ClusterRoleBinding
func (r *rbacService) CreateClusterRoleBinding(ctx context.Context, req model.CreateClusterRoleBindingRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 构建Kubernetes ClusterRoleBinding对象
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Labels:      r.convertStringListToMap(req.Labels),      // 转换标签
			Annotations: r.convertStringListToMap(req.Annotations), // 转换注解
		},
		Subjects: r.convertSubjects(req.Subjects), // 转换绑定主体
		RoleRef:  r.convertRoleRef(req.RoleRef),   // 转换角色引用
	}

	// 调用Kubernetes API创建ClusterRoleBinding
	_, err = clientset.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
	if err != nil {
		r.logger.Error("创建 ClusterRoleBinding 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("创建 ClusterRoleBinding 失败: %w", err)
	}

	r.logger.Info("ClusterRoleBinding 创建成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("name", req.Name))
	return nil
}

// UpdateClusterRoleBinding 更新现有的ClusterRoleBinding
func (r *rbacService) UpdateClusterRoleBinding(ctx context.Context, req model.UpdateClusterRoleBindingRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 首先获取现有的ClusterRoleBinding
	existingClusterRoleBinding, err := clientset.RbacV1().ClusterRoleBindings().Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("获取现有 ClusterRoleBinding 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("获取现有 ClusterRoleBinding 失败: %w", err)
	}

	// 更新ClusterRoleBinding的字段
	existingClusterRoleBinding.Labels = r.convertStringListToMap(req.Labels)
	existingClusterRoleBinding.Annotations = r.convertStringListToMap(req.Annotations)
	existingClusterRoleBinding.Subjects = r.convertSubjects(req.Subjects)
	existingClusterRoleBinding.RoleRef = r.convertRoleRef(req.RoleRef)

	// 调用Kubernetes API更新ClusterRoleBinding
	_, err = clientset.RbacV1().ClusterRoleBindings().Update(ctx, existingClusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		r.logger.Error("更新 ClusterRoleBinding 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("更新 ClusterRoleBinding 失败: %w", err)
	}

	r.logger.Info("ClusterRoleBinding 更新成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("name", req.Name))
	return nil
}

// DeleteClusterRoleBinding 删除指定的ClusterRoleBinding
func (r *rbacService) DeleteClusterRoleBinding(ctx context.Context, req model.DeleteClusterRoleBindingRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, r.k8sClient, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API删除ClusterRoleBinding
	err = clientset.RbacV1().ClusterRoleBindings().Delete(ctx, req.Name, metav1.DeleteOptions{})
	if err != nil {
		r.logger.Error("删除 ClusterRoleBinding 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("删除 ClusterRoleBinding 失败: %w", err)
	}

	r.logger.Info("ClusterRoleBinding 删除成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("name", req.Name))
	return nil
}

// ========== 辅助转换函数 ==========

// convertRole 将Kubernetes原生Role对象转换为内部模型
func (r *rbacService) convertRole(role *rbacv1.Role) model.K8sRole {
	return model.K8sRole{
		Name:        role.Name,
		Namespace:   role.Namespace,
		UID:         string(role.UID),
		Labels:      r.convertMapToStringList(role.Labels),
		Annotations: r.convertMapToStringList(role.Annotations),
		Rules:       r.convertK8sPolicyRules(role.Rules),
		CreatedAt:   role.CreationTimestamp.Time,
	}
}

// convertClusterRole 将Kubernetes原生ClusterRole对象转换为内部模型
func (r *rbacService) convertClusterRole(clusterRole *rbacv1.ClusterRole) model.K8sClusterRole {
	return model.K8sClusterRole{
		Name:        clusterRole.Name,
		UID:         string(clusterRole.UID),
		Labels:      r.convertMapToStringList(clusterRole.Labels),
		Annotations: r.convertMapToStringList(clusterRole.Annotations),
		Rules:       r.convertK8sPolicyRules(clusterRole.Rules),
		CreatedAt:   clusterRole.CreationTimestamp.Time,
	}
}

// convertRoleBinding 将Kubernetes原生RoleBinding对象转换为内部模型
func (r *rbacService) convertRoleBinding(roleBinding *rbacv1.RoleBinding) model.K8sRoleBinding {
	return model.K8sRoleBinding{
		Name:        roleBinding.Name,
		Namespace:   roleBinding.Namespace,
		UID:         string(roleBinding.UID),
		Labels:      r.convertMapToStringList(roleBinding.Labels),
		Annotations: r.convertMapToStringList(roleBinding.Annotations),
		Subjects:    r.convertK8sSubjects(roleBinding.Subjects),
		RoleRef:     r.convertK8sRoleRef(roleBinding.RoleRef),
		CreatedAt:   roleBinding.CreationTimestamp.Time,
	}
}

// convertClusterRoleBinding 将Kubernetes原生ClusterRoleBinding对象转换为内部模型
func (r *rbacService) convertClusterRoleBinding(clusterRoleBinding *rbacv1.ClusterRoleBinding) model.K8sClusterRoleBinding {
	return model.K8sClusterRoleBinding{
		Name:        clusterRoleBinding.Name,
		UID:         string(clusterRoleBinding.UID),
		Labels:      r.convertMapToStringList(clusterRoleBinding.Labels),
		Annotations: r.convertMapToStringList(clusterRoleBinding.Annotations),
		Subjects:    r.convertK8sSubjects(clusterRoleBinding.Subjects),
		RoleRef:     r.convertK8sRoleRef(clusterRoleBinding.RoleRef),
		CreatedAt:   clusterRoleBinding.CreationTimestamp.Time,
	}
}

// convertPolicyRules 将内部模型策略规则转换为Kubernetes原生策略规则
func (r *rbacService) convertPolicyRules(rules []model.PolicyRule) []rbacv1.PolicyRule {
	var result []rbacv1.PolicyRule
	for _, rule := range rules {
		result = append(result, rbacv1.PolicyRule{
			Verbs:           rule.Verbs,           // 操作动词：get, list, create, update, delete等
			APIGroups:       rule.APIGroups,       // API组：如"", "apps", "extensions"
			Resources:       rule.Resources,       // 资源类型：如pods, services, deployments
			ResourceNames:   rule.ResourceNames,   // 特定资源名称
			NonResourceURLs: rule.NonResourceURLs, // 非资源URL：如"/healthz"
		})
	}
	return result
}

// convertK8sPolicyRules 将Kubernetes原生策略规则转换为内部模型策略规则
func (r *rbacService) convertK8sPolicyRules(rules []rbacv1.PolicyRule) []model.PolicyRule {
	var result []model.PolicyRule
	for _, rule := range rules {
		result = append(result, model.PolicyRule{
			Verbs:           rule.Verbs,           // 操作动词
			APIGroups:       rule.APIGroups,       // API组
			Resources:       rule.Resources,       // 资源类型
			ResourceNames:   rule.ResourceNames,   // 特定资源名称
			NonResourceURLs: rule.NonResourceURLs, // 非资源URL
		})
	}
	return result
}

// convertSubjects 将内部模型主体转换为Kubernetes原生主体
func (r *rbacService) convertSubjects(subjects []model.Subject) []rbacv1.Subject {
	var result []rbacv1.Subject
	for _, subject := range subjects {
		result = append(result, rbacv1.Subject{
			Kind:      subject.Kind,      // 主体类型：User, Group, ServiceAccount
			APIGroup:  subject.APIGroup,  // API组
			Name:      subject.Name,      // 主体名称
			Namespace: subject.Namespace, // 命名空间（仅ServiceAccount需要）
		})
	}
	return result
}

// convertK8sSubjects 将Kubernetes原生主体转换为内部模型主体
func (r *rbacService) convertK8sSubjects(subjects []rbacv1.Subject) []model.Subject {
	var result []model.Subject
	for _, subject := range subjects {
		result = append(result, model.Subject{
			Kind:      subject.Kind,      // 主体类型
			APIGroup:  subject.APIGroup,  // API组
			Name:      subject.Name,      // 主体名称
			Namespace: subject.Namespace, // 命名空间
		})
	}
	return result
}

// convertRoleRef 将内部模型角色引用转换为Kubernetes原生角色引用
func (r *rbacService) convertRoleRef(roleRef model.RoleRef) rbacv1.RoleRef {
	return rbacv1.RoleRef{
		APIGroup: roleRef.APIGroup, // API组：通常为"rbac.authorization.k8s.io"
		Kind:     roleRef.Kind,     // 角色类型：Role或ClusterRole
		Name:     roleRef.Name,     // 角色名称
	}
}

// convertK8sRoleRef 将Kubernetes原生角色引用转换为内部模型角色引用
func (r *rbacService) convertK8sRoleRef(roleRef rbacv1.RoleRef) model.RoleRef {
	return model.RoleRef{
		APIGroup: roleRef.APIGroup, // API组
		Kind:     roleRef.Kind,     // 角色类型
		Name:     roleRef.Name,     // 角色名称
	}
}

// convertStringListToMap 将字符串列表转换为键值对映射
// 支持两种格式：
// 1. "key=value" 格式：解析为 key -> value
// 2. "key" 格式：解析为 key -> ""
func (r *rbacService) convertStringListToMap(stringList model.StringList) map[string]string {
	result := make(map[string]string)
	for _, item := range stringList {
		if len(item) > 0 {
			// 查找等号分隔符
			if idx := strings.Index(item, "="); idx != -1 {
				// "key=value" 格式
				key := item[:idx]
				value := item[idx+1:]
				result[key] = value
			} else {
				// "key" 格式，值为空字符串
				result[item] = ""
			}
		}
	}
	return result
}

// convertMapToStringList 将键值对映射转换为字符串列表
// 输出格式：
// - 如果值不为空：输出 "key=value"
// - 如果值为空：输出 "key"
func (r *rbacService) convertMapToStringList(m map[string]string) model.StringList {
	var result model.StringList
	for key, value := range m {
		if value != "" {
			// 有值的情况：输出 "key=value"
			result = append(result, fmt.Sprintf("%s=%s", key, value))
		} else {
			// 值为空的情况：只输出 "key"
			result = append(result, key)
		}
	}
	return result
}

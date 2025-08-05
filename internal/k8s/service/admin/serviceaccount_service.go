/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceAccountService ServiceAccount（服务账户）管理服务接口
// 提供对 Kubernetes ServiceAccount 资源的完整管理功能，包括创建、更新、删除和权限绑定
type ServiceAccountService interface {
	// ServiceAccount 基本管理接口

	// GetServiceAccountsByNamespace 获取指定命名空间下的所有 ServiceAccount
	// @param ctx 上下文
	// @param clusterID 集群ID
	// @param namespace 命名空间名称
	// @return []model.K8sServiceAccount ServiceAccount列表
	// @return error 错误信息
	GetServiceAccountsByNamespace(ctx context.Context, clusterID int, namespace string) ([]model.K8sServiceAccount, error)

	// GetServiceAccount 获取指定的 ServiceAccount 详细信息
	// @param ctx 上下文
	// @param clusterID 集群ID
	// @param namespace 命名空间名称
	// @param name ServiceAccount名称
	// @return *model.K8sServiceAccount ServiceAccount详细信息
	// @return error 错误信息
	GetServiceAccount(ctx context.Context, clusterID int, namespace, name string) (*model.K8sServiceAccount, error)

	// CreateServiceAccount 创建新的 ServiceAccount
	// @param ctx 上下文
	// @param req 创建请求参数
	// @return error 错误信息
	CreateServiceAccount(ctx context.Context, req model.CreateServiceAccountRequest) error

	// UpdateServiceAccount 更新现有的 ServiceAccount
	// @param ctx 上下文
	// @param req 更新请求参数
	// @return error 错误信息
	UpdateServiceAccount(ctx context.Context, req model.UpdateServiceAccountRequest) error

	// DeleteServiceAccount 删除指定的 ServiceAccount
	// @param ctx 上下文
	// @param req 删除请求参数
	// @return error 错误信息
	DeleteServiceAccount(ctx context.Context, req model.DeleteServiceAccountRequest) error

	// ServiceAccount Token 管理接口

	// CreateServiceAccountToken 为 ServiceAccount 创建访问令牌
	// @param ctx 上下文
	// @param req 创建Token请求参数
	// @return *model.ServiceAccountToken Token信息
	// @return error 错误信息
	CreateServiceAccountToken(ctx context.Context, req model.ServiceAccountTokenRequest) (*model.ServiceAccountToken, error)

	// ServiceAccount 权限管理接口

	// GetServiceAccountPermissions 获取 ServiceAccount 的权限信息
	// 包括绑定的Role、ClusterRole以及相关的RoleBinding、ClusterRoleBinding
	// @param ctx 上下文
	// @param clusterID 集群ID
	// @param namespace 命名空间名称
	// @param serviceAccountName ServiceAccount名称
	// @return *model.ServiceAccountPermissions 权限信息
	// @return error 错误信息
	GetServiceAccountPermissions(ctx context.Context, clusterID int, namespace, serviceAccountName string) (*model.ServiceAccountPermissions, error)

	// BindRoleToServiceAccount 将Role绑定到ServiceAccount
	// 通过创建RoleBinding实现权限绑定
	// @param ctx 上下文
	// @param req 绑定请求参数
	// @return error 错误信息
	BindRoleToServiceAccount(ctx context.Context, req model.BindRoleToServiceAccountRequest) error

	// BindClusterRoleToServiceAccount 将ClusterRole绑定到ServiceAccount
	// 通过创建ClusterRoleBinding实现权限绑定
	// @param ctx 上下文
	// @param req 绑定请求参数
	// @return error 错误信息
	BindClusterRoleToServiceAccount(ctx context.Context, req model.BindClusterRoleToServiceAccountRequest) error

	// UnbindRoleFromServiceAccount 解绑ServiceAccount的Role权限
	// 通过删除RoleBinding实现权限解绑
	// @param ctx 上下文
	// @param req 解绑请求参数
	// @return error 错误信息
	UnbindRoleFromServiceAccount(ctx context.Context, req model.UnbindRoleFromServiceAccountRequest) error

	// UnbindClusterRoleFromServiceAccount 解绑ServiceAccount的ClusterRole权限
	// 通过删除ClusterRoleBinding实现权限解绑
	// @param ctx 上下文
	// @param req 解绑请求参数
	// @return error 错误信息
	UnbindClusterRoleFromServiceAccount(ctx context.Context, req model.UnbindClusterRoleFromServiceAccountRequest) error
}

// serviceAccountService ServiceAccount服务实现结构体
type serviceAccountService struct {
	logger     *zap.Logger      // 日志记录器
	k8sClient  client.K8sClient // Kubernetes客户端
	clusterDao admin.ClusterDAO // 集群数据访问对象
}

// NewServiceAccountService 创建新的ServiceAccount服务实例
// 参数:
//
//	logger: 日志记录器
//	k8sClient: Kubernetes客户端
//	clusterDao: 集群数据访问对象
//
// 返回: ServiceAccountService ServiceAccount服务接口实例
func NewServiceAccountService(logger *zap.Logger, k8sClient client.K8sClient, clusterDao admin.ClusterDAO) ServiceAccountService {
	return &serviceAccountService{
		logger:     logger,
		k8sClient:  k8sClient,
		clusterDao: clusterDao,
	}
}

// ========== ServiceAccount 基本管理实现 ==========

// GetServiceAccountsByNamespace 获取指定命名空间下的所有ServiceAccount
func (s *serviceAccountService) GetServiceAccountsByNamespace(ctx context.Context, clusterID int, namespace string) ([]model.K8sServiceAccount, error) {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(clusterID, s.k8sClient, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API获取ServiceAccount列表
	serviceAccounts, err := clientset.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		s.logger.Error("获取 ServiceAccount 列表失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID),
			zap.String("namespace", namespace))
		return nil, fmt.Errorf("获取 ServiceAccount 列表失败: %w", err)
	}

	// 转换Kubernetes原生ServiceAccount对象为内部模型
	var result []model.K8sServiceAccount
	for _, sa := range serviceAccounts.Items {
		result = append(result, s.convertServiceAccount(&sa))
	}

	return result, nil
}

// GetServiceAccount 获取指定的ServiceAccount详细信息
func (s *serviceAccountService) GetServiceAccount(ctx context.Context, clusterID int, namespace, name string) (*model.K8sServiceAccount, error) {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(clusterID, s.k8sClient, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API获取指定ServiceAccount
	serviceAccount, err := clientset.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取 ServiceAccount 失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name))
		return nil, fmt.Errorf("获取 ServiceAccount 失败: %w", err)
	}

	// 转换为内部模型并返回
	result := s.convertServiceAccount(serviceAccount)
	return &result, nil
}

// CreateServiceAccount 创建新的ServiceAccount
func (s *serviceAccountService) CreateServiceAccount(ctx context.Context, req model.CreateServiceAccountRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, s.k8sClient, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 构建Kubernetes ServiceAccount对象
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      s.convertStringListToMap(req.Labels),      // 转换标签
			Annotations: s.convertStringListToMap(req.Annotations), // 转换注解
		},
		Secrets:                      s.convertLocalObjectReferences(req.Secrets),     // 转换关联Secret
		ImagePullSecrets:             s.convertImagePullSecrets(req.ImagePullSecrets), // 转换镜像拉取Secret
		AutomountServiceAccountToken: req.AutomountServiceAccountToken,                // 自动挂载令牌设置
	}

	// 调用Kubernetes API创建ServiceAccount
	_, err = clientset.CoreV1().ServiceAccounts(req.Namespace).Create(ctx, serviceAccount, metav1.CreateOptions{})
	if err != nil {
		s.logger.Error("创建 ServiceAccount 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建 ServiceAccount 失败: %w", err)
	}

	s.logger.Info("ServiceAccount 创建成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// UpdateServiceAccount 更新现有的ServiceAccount
func (s *serviceAccountService) UpdateServiceAccount(ctx context.Context, req model.UpdateServiceAccountRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, s.k8sClient, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 首先获取现有的ServiceAccount
	existingSA, err := clientset.CoreV1().ServiceAccounts(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取现有 ServiceAccount 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("获取现有 ServiceAccount 失败: %w", err)
	}

	// 更新ServiceAccount的字段
	existingSA.Labels = s.convertStringListToMap(req.Labels)
	existingSA.Annotations = s.convertStringListToMap(req.Annotations)
	existingSA.Secrets = s.convertLocalObjectReferences(req.Secrets)
	existingSA.ImagePullSecrets = s.convertImagePullSecrets(req.ImagePullSecrets)
	existingSA.AutomountServiceAccountToken = req.AutomountServiceAccountToken

	// 调用Kubernetes API更新ServiceAccount
	_, err = clientset.CoreV1().ServiceAccounts(req.Namespace).Update(ctx, existingSA, metav1.UpdateOptions{})
	if err != nil {
		s.logger.Error("更新 ServiceAccount 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新 ServiceAccount 失败: %w", err)
	}

	s.logger.Info("ServiceAccount 更新成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// DeleteServiceAccount 删除指定的ServiceAccount
func (s *serviceAccountService) DeleteServiceAccount(ctx context.Context, req model.DeleteServiceAccountRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, s.k8sClient, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API删除ServiceAccount
	err = clientset.CoreV1().ServiceAccounts(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{})
	if err != nil {
		s.logger.Error("删除 ServiceAccount 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除 ServiceAccount 失败: %w", err)
	}

	s.logger.Info("ServiceAccount 删除成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// ========== ServiceAccount Token 管理实现 ==========

// CreateServiceAccountToken 为ServiceAccount创建访问令牌
func (s *serviceAccountService) CreateServiceAccountToken(ctx context.Context, req model.ServiceAccountTokenRequest) (*model.ServiceAccountToken, error) {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, s.k8sClient, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 构建Token请求对象
	tokenRequest := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			ExpirationSeconds: req.ExpirationSeconds, // 设置过期时间
		},
	}

	// 调用Kubernetes API创建Token
	tokenResponse, err := clientset.CoreV1().ServiceAccounts(req.Namespace).CreateToken(
		ctx, req.ServiceAccountName, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		s.logger.Error("创建 ServiceAccount Token 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("service_account", req.ServiceAccountName))
		return nil, fmt.Errorf("创建 ServiceAccount Token 失败: %w", err)
	}

	s.logger.Info("ServiceAccount Token 创建成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("service_account", req.ServiceAccountName))

	return &model.ServiceAccountToken{
		Token: tokenResponse.Status.Token,
	}, nil
}

// ========== ServiceAccount 权限管理实现 ==========

// GetServiceAccountPermissions 获取ServiceAccount的权限信息
func (s *serviceAccountService) GetServiceAccountPermissions(ctx context.Context, clusterID int, namespace, serviceAccountName string) (*model.ServiceAccountPermissions, error) {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(clusterID, s.k8sClient, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	result := &model.ServiceAccountPermissions{
		ServiceAccountName: serviceAccountName,
		Namespace:          namespace,
	}

	// 获取所有RoleBinding，查找绑定到该ServiceAccount的
	roleBindings, err := clientset.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		s.logger.Error("获取 RoleBinding 列表失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取 RoleBinding 列表失败: %w", err)
	}

	// 处理RoleBinding
	for _, rb := range roleBindings.Items {
		// 检查是否绑定到目标ServiceAccount
		if s.isServiceAccountInSubjects(rb.Subjects, serviceAccountName, namespace) {
			// 添加到结果中
			result.RoleBindings = append(result.RoleBindings, s.convertRoleBinding(&rb))

			// 如果引用的是Role，获取Role详情
			if rb.RoleRef.Kind == "Role" {
				role, err := clientset.RbacV1().Roles(namespace).Get(ctx, rb.RoleRef.Name, metav1.GetOptions{})
				if err != nil {
					s.logger.Warn("获取 Role 失败",
						zap.Error(err),
						zap.String("role_name", rb.RoleRef.Name))
					continue
				}
				result.Roles = append(result.Roles, s.convertRole(role))
			}
		}
	}

	// 获取所有ClusterRoleBinding，查找绑定到该ServiceAccount的
	clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		s.logger.Error("获取 ClusterRoleBinding 列表失败",
			zap.Error(err),
			zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("获取 ClusterRoleBinding 列表失败: %w", err)
	}

	// 处理ClusterRoleBinding
	for _, crb := range clusterRoleBindings.Items {
		// 检查是否绑定到目标ServiceAccount
		if s.isServiceAccountInSubjects(crb.Subjects, serviceAccountName, namespace) {
			// 添加到结果中
			result.ClusterRoleBindings = append(result.ClusterRoleBindings, s.convertClusterRoleBinding(&crb))

			// 获取ClusterRole详情
			clusterRole, err := clientset.RbacV1().ClusterRoles().Get(ctx, crb.RoleRef.Name, metav1.GetOptions{})
			if err != nil {
				s.logger.Warn("获取 ClusterRole 失败",
					zap.Error(err),
					zap.String("cluster_role_name", crb.RoleRef.Name))
				continue
			}
			result.ClusterRoles = append(result.ClusterRoles, s.convertClusterRole(clusterRole))
		}
	}

	return result, nil
}

// BindRoleToServiceAccount 将Role绑定到ServiceAccount
func (s *serviceAccountService) BindRoleToServiceAccount(ctx context.Context, req model.BindRoleToServiceAccountRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, s.k8sClient, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 设置默认的RoleBinding名称
	roleBindingName := req.RoleBindingName
	if roleBindingName == "" {
		roleBindingName = fmt.Sprintf("%s-%s-binding", req.ServiceAccountName, req.RoleName)
	}

	// 构建RoleBinding对象
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: req.Namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      req.ServiceAccountName,
				Namespace: req.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     req.RoleName,
		},
	}

	// 调用Kubernetes API创建RoleBinding
	_, err = clientset.RbacV1().RoleBindings(req.Namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		s.logger.Error("绑定 Role 到 ServiceAccount 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("service_account", req.ServiceAccountName),
			zap.String("role", req.RoleName))
		return fmt.Errorf("绑定 Role 到 ServiceAccount 失败: %w", err)
	}

	s.logger.Info("Role 绑定到 ServiceAccount 成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("service_account", req.ServiceAccountName),
		zap.String("role", req.RoleName),
		zap.String("role_binding", roleBindingName))
	return nil
}

// BindClusterRoleToServiceAccount 将ClusterRole绑定到ServiceAccount
func (s *serviceAccountService) BindClusterRoleToServiceAccount(ctx context.Context, req model.BindClusterRoleToServiceAccountRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, s.k8sClient, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 设置默认的ClusterRoleBinding名称
	clusterRoleBindingName := req.ClusterRoleBindingName
	if clusterRoleBindingName == "" {
		clusterRoleBindingName = fmt.Sprintf("%s-%s-cluster-binding", req.ServiceAccountName, req.ClusterRoleName)
	}

	// 构建ClusterRoleBinding对象
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      req.ServiceAccountName,
				Namespace: req.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     req.ClusterRoleName,
		},
	}

	// 调用Kubernetes API创建ClusterRoleBinding
	_, err = clientset.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
	if err != nil {
		s.logger.Error("绑定 ClusterRole 到 ServiceAccount 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("service_account", req.ServiceAccountName),
			zap.String("cluster_role", req.ClusterRoleName))
		return fmt.Errorf("绑定 ClusterRole 到 ServiceAccount 失败: %w", err)
	}

	s.logger.Info("ClusterRole 绑定到 ServiceAccount 成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("service_account", req.ServiceAccountName),
		zap.String("cluster_role", req.ClusterRoleName),
		zap.String("cluster_role_binding", clusterRoleBindingName))
	return nil
}

// UnbindRoleFromServiceAccount 解绑ServiceAccount的Role权限
func (s *serviceAccountService) UnbindRoleFromServiceAccount(ctx context.Context, req model.UnbindRoleFromServiceAccountRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, s.k8sClient, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API删除RoleBinding
	err = clientset.RbacV1().RoleBindings(req.Namespace).Delete(ctx, req.RoleBindingName, metav1.DeleteOptions{})
	if err != nil {
		s.logger.Error("解绑 Role 从 ServiceAccount 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("service_account", req.ServiceAccountName),
			zap.String("role_binding", req.RoleBindingName))
		return fmt.Errorf("解绑 Role 从 ServiceAccount 失败: %w", err)
	}

	s.logger.Info("Role 从 ServiceAccount 解绑成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("service_account", req.ServiceAccountName),
		zap.String("role_binding", req.RoleBindingName))
	return nil
}

// UnbindClusterRoleFromServiceAccount 解绑ServiceAccount的ClusterRole权限
func (s *serviceAccountService) UnbindClusterRoleFromServiceAccount(ctx context.Context, req model.UnbindClusterRoleFromServiceAccountRequest) error {
	// 获取指定集群的Kubernetes客户端
	clientset, err := utils.GetKubeClient(req.ClusterID, s.k8sClient, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 调用Kubernetes API删除ClusterRoleBinding
	err = clientset.RbacV1().ClusterRoleBindings().Delete(ctx, req.ClusterRoleBindingName, metav1.DeleteOptions{})
	if err != nil {
		s.logger.Error("解绑 ClusterRole 从 ServiceAccount 失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("service_account", req.ServiceAccountName),
			zap.String("cluster_role_binding", req.ClusterRoleBindingName))
		return fmt.Errorf("解绑 ClusterRole 从 ServiceAccount 失败: %w", err)
	}

	s.logger.Info("ClusterRole 从 ServiceAccount 解绑成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("service_account", req.ServiceAccountName),
		zap.String("cluster_role_binding", req.ClusterRoleBindingName))
	return nil
}

// ========== 辅助转换函数 ==========

// convertServiceAccount 将Kubernetes原生ServiceAccount对象转换为内部模型
func (s *serviceAccountService) convertServiceAccount(sa *corev1.ServiceAccount) model.K8sServiceAccount {
	return model.K8sServiceAccount{
		Name:                         sa.Name,
		Namespace:                    sa.Namespace,
		UID:                          string(sa.UID),
		Labels:                       s.convertMapToStringList(sa.Labels),
		Annotations:                  s.convertMapToStringList(sa.Annotations),
		Secrets:                      s.convertK8sObjectReferences(sa.Secrets),
		ImagePullSecrets:             s.convertK8sLocalObjectReferences(sa.ImagePullSecrets),
		AutomountServiceAccountToken: sa.AutomountServiceAccountToken,
		CreatedAt:                    sa.CreationTimestamp.Time,
	}
}

// convertRole 将Kubernetes原生Role对象转换为内部模型
func (s *serviceAccountService) convertRole(role *rbacv1.Role) model.K8sRole {
	return model.K8sRole{
		Name:        role.Name,
		Namespace:   role.Namespace,
		UID:         string(role.UID),
		Labels:      s.convertMapToStringList(role.Labels),
		Annotations: s.convertMapToStringList(role.Annotations),
		Rules:       s.convertK8sPolicyRules(role.Rules),
		CreatedAt:   role.CreationTimestamp.Time,
	}
}

// convertClusterRole 将Kubernetes原生ClusterRole对象转换为内部模型
func (s *serviceAccountService) convertClusterRole(clusterRole *rbacv1.ClusterRole) model.K8sClusterRole {
	return model.K8sClusterRole{
		Name:        clusterRole.Name,
		UID:         string(clusterRole.UID),
		Labels:      s.convertMapToStringList(clusterRole.Labels),
		Annotations: s.convertMapToStringList(clusterRole.Annotations),
		Rules:       s.convertK8sPolicyRules(clusterRole.Rules),
		CreatedAt:   clusterRole.CreationTimestamp.Time,
	}
}

// convertRoleBinding 将Kubernetes原生RoleBinding对象转换为内部模型
func (s *serviceAccountService) convertRoleBinding(roleBinding *rbacv1.RoleBinding) model.K8sRoleBinding {
	return model.K8sRoleBinding{
		Name:        roleBinding.Name,
		Namespace:   roleBinding.Namespace,
		UID:         string(roleBinding.UID),
		Labels:      s.convertMapToStringList(roleBinding.Labels),
		Annotations: s.convertMapToStringList(roleBinding.Annotations),
		Subjects:    s.convertK8sSubjects(roleBinding.Subjects),
		RoleRef:     s.convertK8sRoleRef(roleBinding.RoleRef),
		CreatedAt:   roleBinding.CreationTimestamp.Time,
	}
}

// convertClusterRoleBinding 将Kubernetes原生ClusterRoleBinding对象转换为内部模型
func (s *serviceAccountService) convertClusterRoleBinding(clusterRoleBinding *rbacv1.ClusterRoleBinding) model.K8sClusterRoleBinding {
	return model.K8sClusterRoleBinding{
		Name:        clusterRoleBinding.Name,
		UID:         string(clusterRoleBinding.UID),
		Labels:      s.convertMapToStringList(clusterRoleBinding.Labels),
		Annotations: s.convertMapToStringList(clusterRoleBinding.Annotations),
		Subjects:    s.convertK8sSubjects(clusterRoleBinding.Subjects),
		RoleRef:     s.convertK8sRoleRef(clusterRoleBinding.RoleRef),
		CreatedAt:   clusterRoleBinding.CreationTimestamp.Time,
	}
}

// convertK8sPolicyRules 将Kubernetes原生策略规则转换为内部模型策略规则
func (s *serviceAccountService) convertK8sPolicyRules(rules []rbacv1.PolicyRule) []model.PolicyRule {
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

// convertK8sSubjects 将Kubernetes原生主体转换为内部模型主体
func (s *serviceAccountService) convertK8sSubjects(subjects []rbacv1.Subject) []model.Subject {
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

// convertK8sRoleRef 将Kubernetes原生角色引用转换为内部模型角色引用
func (s *serviceAccountService) convertK8sRoleRef(roleRef rbacv1.RoleRef) model.RoleRef {
	return model.RoleRef{
		APIGroup: roleRef.APIGroup, // API组
		Kind:     roleRef.Kind,     // 角色类型
		Name:     roleRef.Name,     // 角色名称
	}
}

// convertLocalObjectReferences 将内部模型本地对象引用转换为Kubernetes原生对象引用
func (s *serviceAccountService) convertLocalObjectReferences(refs []model.LocalObjectReference) []corev1.ObjectReference {
	var result []corev1.ObjectReference
	for _, ref := range refs {
		result = append(result, corev1.ObjectReference{
			Name: ref.Name,
		})
	}
	return result
}

// convertImagePullSecrets 将内部模型本地对象引用转换为Kubernetes原生本地对象引用
func (s *serviceAccountService) convertImagePullSecrets(refs []model.LocalObjectReference) []corev1.LocalObjectReference {
	var result []corev1.LocalObjectReference
	for _, ref := range refs {
		result = append(result, corev1.LocalObjectReference{
			Name: ref.Name,
		})
	}
	return result
}

// convertK8sLocalObjectReferences 将Kubernetes原生本地对象引用转换为内部模型本地对象引用
func (s *serviceAccountService) convertK8sLocalObjectReferences(refs []corev1.LocalObjectReference) []model.LocalObjectReference {
	var result []model.LocalObjectReference
	for _, ref := range refs {
		result = append(result, model.LocalObjectReference{
			Name: ref.Name,
		})
	}
	return result
}

// convertK8sObjectReferences 将Kubernetes原生对象引用转换为内部模型本地对象引用
func (s *serviceAccountService) convertK8sObjectReferences(refs []corev1.ObjectReference) []model.LocalObjectReference {
	var result []model.LocalObjectReference
	for _, ref := range refs {
		result = append(result, model.LocalObjectReference{
			Name: ref.Name,
		})
	}
	return result
}

// convertStringListToMap 将字符串列表转换为键值对映射
// 支持两种格式：
// 1. "key=value" 格式：解析为 key -> value
// 2. "key" 格式：解析为 key -> ""
func (s *serviceAccountService) convertStringListToMap(stringList model.StringList) map[string]string {
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
func (s *serviceAccountService) convertMapToStringList(m map[string]string) model.StringList {
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

// isServiceAccountInSubjects 检查ServiceAccount是否在主体列表中
// 参数:
//   - subjects: 主体列表
//   - serviceAccountName: ServiceAccount名称
//   - namespace: 命名空间
// 返回: bool 是否包含指定的ServiceAccount
func (s *serviceAccountService) isServiceAccountInSubjects(subjects []rbacv1.Subject, serviceAccountName, namespace string) bool {
	for _, subject := range subjects {
		if subject.Kind == "ServiceAccount" &&
			subject.Name == serviceAccountName &&
			subject.Namespace == namespace {
			return true
		}
	}
	return false
}

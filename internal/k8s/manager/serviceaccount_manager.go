package manager

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"go.uber.org/zap"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ServiceAccountManager ServiceAccount 资源管理器
type ServiceAccountManager interface {
	// 基础 CRUD 操作
	CreateServiceAccount(ctx context.Context, clusterID int, namespace string, sa *corev1.ServiceAccount) error
	GetServiceAccount(ctx context.Context, clusterID int, namespace, name string) (*corev1.ServiceAccount, error)
	GetServiceAccountList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*corev1.ServiceAccountList, error)
	UpdateServiceAccount(ctx context.Context, clusterID int, namespace string, sa *corev1.ServiceAccount) error
	DeleteServiceAccount(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error

	// 批量操作
	BatchDeleteServiceAccounts(ctx context.Context, clusterID int, namespace string, serviceAccountNames []string) error

	// 高级功能
	PatchServiceAccount(ctx context.Context, clusterID int, namespace, name string, data []byte, patchType string) (*corev1.ServiceAccount, error)

	// ServiceAccount 特定操作
	GetServiceAccountSecrets(ctx context.Context, clusterID int, namespace, name string) ([]corev1.Secret, error)
	GetServiceAccountTokens(ctx context.Context, clusterID int, namespace, name string) ([]string, error)
	CreateServiceAccountToken(ctx context.Context, clusterID int, namespace, name string, tokenRequest *authv1.TokenRequest) (*authv1.TokenRequest, error)
	BindServiceAccountToRole(ctx context.Context, clusterID int, namespace, saName, roleName string) error
	BindServiceAccountToClusterRole(ctx context.Context, clusterID int, namespace, saName, clusterRoleName string) error
}

type serviceAccountManager struct {
	logger *zap.Logger
	client client.K8sClient
}

// NewServiceAccountManager 创建新的 ServiceAccountManager 实例
func NewServiceAccountManager(logger *zap.Logger, client client.K8sClient) ServiceAccountManager {
	return &serviceAccountManager{
		logger: logger,
		client: client,
	}
}

// CreateServiceAccount 创建ServiceAccount
func (s *serviceAccountManager) CreateServiceAccount(ctx context.Context, clusterID int, namespace string, sa *corev1.ServiceAccount) error {
	kubeClient, err := s.client.GetKubeClient(clusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil {
		s.logger.Error("创建ServiceAccount失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", sa.Name),
			zap.Error(err))
		return err
	}

	s.logger.Info("成功创建ServiceAccount",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", sa.Name))

	return nil
}

// GetServiceAccount 获取指定ServiceAccount
func (s *serviceAccountManager) GetServiceAccount(ctx context.Context, clusterID int, namespace, name string) (*corev1.ServiceAccount, error) {
	kubeClient, err := s.client.GetKubeClient(clusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	sa, err := kubeClient.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取ServiceAccount失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, err
	}

	return sa, nil
}

// GetServiceAccountList 获取ServiceAccount列表
func (s *serviceAccountManager) GetServiceAccountList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*corev1.ServiceAccountList, error) {
	kubeClient, err := s.client.GetKubeClient(clusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	saList, err := kubeClient.CoreV1().ServiceAccounts(namespace).List(ctx, listOptions)
	if err != nil {
		s.logger.Error("获取ServiceAccount列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, err
	}

	s.logger.Debug("成功获取ServiceAccount列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(saList.Items)))

	return saList, nil
}

// UpdateServiceAccount 更新ServiceAccount
func (s *serviceAccountManager) UpdateServiceAccount(ctx context.Context, clusterID int, namespace string, sa *corev1.ServiceAccount) error {
	kubeClient, err := s.client.GetKubeClient(clusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.CoreV1().ServiceAccounts(namespace).Update(ctx, sa, metav1.UpdateOptions{})
	if err != nil {
		s.logger.Error("更新ServiceAccount失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", sa.Name),
			zap.Error(err))
		return err
	}

	s.logger.Info("成功更新ServiceAccount",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", sa.Name))

	return nil
}

// DeleteServiceAccount 删除ServiceAccount
func (s *serviceAccountManager) DeleteServiceAccount(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := s.client.GetKubeClient(clusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	err = kubeClient.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		s.logger.Error("删除ServiceAccount失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return err
	}

	s.logger.Info("成功删除ServiceAccount",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	return nil
}

// BatchDeleteServiceAccounts 批量删除ServiceAccount
func (s *serviceAccountManager) BatchDeleteServiceAccounts(ctx context.Context, clusterID int, namespace string, serviceAccountNames []string) error {
	kubeClient, err := s.client.GetKubeClient(clusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	deleteOptions := metav1.DeleteOptions{}
	var failedDeletions []string

	for _, name := range serviceAccountNames {
		err := kubeClient.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, deleteOptions)
		if err != nil {
			s.logger.Error("删除ServiceAccount失败",
				zap.Int("clusterID", clusterID),
				zap.String("namespace", namespace),
				zap.String("name", name),
				zap.Error(err))
			failedDeletions = append(failedDeletions, name)
		} else {
			s.logger.Info("成功删除ServiceAccount",
				zap.Int("clusterID", clusterID),
				zap.String("namespace", namespace),
				zap.String("name", name))
		}
	}

	if len(failedDeletions) > 0 {
		s.logger.Warn("部分ServiceAccount删除失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Strings("failedDeletions", failedDeletions))
		return err // 返回最后一个错误
	}

	s.logger.Info("批量删除ServiceAccount完成",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(serviceAccountNames)))

	return nil
}

// PatchServiceAccount 部分更新ServiceAccount
func (s *serviceAccountManager) PatchServiceAccount(ctx context.Context, clusterID int, namespace, name string, data []byte, patchType string) (*corev1.ServiceAccount, error) {
	kubeClient, err := s.client.GetKubeClient(clusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	// 转换 patch 类型
	pt := types.PatchType(patchType)
	sa, err := kubeClient.CoreV1().ServiceAccounts(namespace).Patch(ctx, name, pt, data, metav1.PatchOptions{})
	if err != nil {
		s.logger.Error("Patch ServiceAccount失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.String("patchType", patchType),
			zap.Error(err))
		return nil, err
	}

	s.logger.Info("成功Patch ServiceAccount",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	return sa, nil
}

// GetServiceAccountSecrets 获取ServiceAccount关联的Secrets
func (s *serviceAccountManager) GetServiceAccountSecrets(ctx context.Context, clusterID int, namespace, name string) ([]corev1.Secret, error) {
	sa, err := s.GetServiceAccount(ctx, clusterID, namespace, name)
	if err != nil {
		return nil, err
	}

	kubeClient, err := s.client.GetKubeClient(clusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	var secrets []corev1.Secret
	for _, secretRef := range sa.Secrets {
		secret, err := kubeClient.CoreV1().Secrets(namespace).Get(ctx, secretRef.Name, metav1.GetOptions{})
		if err != nil {
			s.logger.Warn("获取ServiceAccount关联的Secret失败",
				zap.Int("clusterID", clusterID),
				zap.String("namespace", namespace),
				zap.String("secretName", secretRef.Name),
				zap.Error(err))
			continue
		}
		secrets = append(secrets, *secret)
	}

	s.logger.Debug("获取ServiceAccount关联的Secrets",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("serviceAccountName", name),
		zap.Int("secretCount", len(secrets)))

	return secrets, nil
}

// GetServiceAccountTokens 获取ServiceAccount的Token
func (s *serviceAccountManager) GetServiceAccountTokens(ctx context.Context, clusterID int, namespace, name string) ([]string, error) {
	secrets, err := s.GetServiceAccountSecrets(ctx, clusterID, namespace, name)
	if err != nil {
		return nil, err
	}

	var tokens []string
	for _, secret := range secrets {
		if secret.Type == corev1.SecretTypeServiceAccountToken {
			if token, exists := secret.Data["token"]; exists {
				tokens = append(tokens, string(token))
			}
		}
	}

	s.logger.Debug("获取ServiceAccount的Token",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("serviceAccountName", name),
		zap.Int("tokenCount", len(tokens)))

	return tokens, nil
}

// CreateServiceAccountToken 为ServiceAccount创建Token
func (s *serviceAccountManager) CreateServiceAccountToken(ctx context.Context, clusterID int, namespace, name string, tokenRequest *authv1.TokenRequest) (*authv1.TokenRequest, error) {
	kubeClient, err := s.client.GetKubeClient(clusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	token, err := kubeClient.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		s.logger.Error("创建ServiceAccount Token失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, err
	}

	s.logger.Info("成功创建ServiceAccount Token",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	return token, nil
}

// BindServiceAccountToRole 绑定ServiceAccount到Role
func (s *serviceAccountManager) BindServiceAccountToRole(ctx context.Context, clusterID int, namespace, saName, roleName string) error {
	kubeClient, err := s.client.GetKubeClient(clusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	// 创建RoleBinding
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName + "-" + roleName + "-binding",
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
	}

	_, err = kubeClient.RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		s.logger.Error("绑定ServiceAccount到Role失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("serviceAccount", saName),
			zap.String("role", roleName),
			zap.Error(err))
		return err
	}

	s.logger.Info("成功绑定ServiceAccount到Role",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("serviceAccount", saName),
		zap.String("role", roleName))

	return nil
}

// BindServiceAccountToClusterRole 绑定ServiceAccount到ClusterRole
func (s *serviceAccountManager) BindServiceAccountToClusterRole(ctx context.Context, clusterID int, namespace, saName, clusterRoleName string) error {
	kubeClient, err := s.client.GetKubeClient(clusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	// 创建ClusterRoleBinding
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: saName + "-" + clusterRoleName + "-binding",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
	}

	_, err = kubeClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
	if err != nil {
		s.logger.Error("绑定ServiceAccount到ClusterRole失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("serviceAccount", saName),
			zap.String("clusterRole", clusterRoleName),
			zap.Error(err))
		return err
	}

	s.logger.Info("成功绑定ServiceAccount到ClusterRole",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("serviceAccount", saName),
		zap.String("clusterRole", clusterRoleName))

	return nil
}

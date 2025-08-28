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

package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type ServiceAccountService interface {
	// 获取ServiceAccount列表
	GetServiceAccountList(ctx context.Context, req *model.ServiceAccountListReq) ([]*model.K8sServiceAccountResponse, error)

	// 获取ServiceAccount详情
	GetServiceAccountDetails(ctx context.Context, clusterID int, namespace, name string) (*model.K8sServiceAccountResponse, error)

	// 创建ServiceAccount
	CreateServiceAccount(ctx context.Context, req *model.ServiceAccountCreateReq) error

	// 更新ServiceAccount
	UpdateServiceAccount(ctx context.Context, req *model.ServiceAccountUpdateReq) error

	// 删除ServiceAccount
	DeleteServiceAccount(ctx context.Context, req *model.ServiceAccountDeleteReq) error

	// 批量删除ServiceAccount
	BatchDeleteServiceAccount(ctx context.Context, req *model.ServiceAccountBatchDeleteReq) error

	// 获取ServiceAccount统计信息
	GetServiceAccountStatistics(ctx context.Context, req *model.ServiceAccountStatisticsReq) (*model.ServiceAccountStatisticsResp, error)

	// 获取ServiceAccount令牌
	GetServiceAccountToken(ctx context.Context, req *model.ServiceAccountTokenReq) (*model.ServiceAccountTokenResp, error)

	// 获取ServiceAccount YAML
	GetServiceAccountYaml(ctx context.Context, req *model.ServiceAccountYamlReq) (*model.ServiceAccountYamlResp, error)

	// 更新ServiceAccount YAML
	UpdateServiceAccountYaml(ctx context.Context, req *model.ServiceAccountUpdateYamlReq) error
}

type serviceAccountService struct {
	dao    dao.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

func NewServiceAccountService(dao dao.ClusterDAO, client client.K8sClient, logger *zap.Logger) ServiceAccountService {
	return &serviceAccountService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetServiceAccountList 获取ServiceAccount列表
func (s *serviceAccountService) GetServiceAccountList(ctx context.Context, req *model.ServiceAccountListReq) ([]*model.K8sServiceAccountResponse, error) {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, utils.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	listOptions := metav1.ListOptions{}
	if req.LabelSelector != "" {
		listOptions.LabelSelector = req.LabelSelector
	}
	if req.FieldSelector != "" {
		listOptions.FieldSelector = req.FieldSelector
	}

	saList, err := kubeClient.CoreV1().ServiceAccounts(req.Namespace).List(ctx, listOptions)
	if err != nil {
		s.logger.Error("获取ServiceAccount列表失败",
			zap.String("Namespace", req.Namespace),
			zap.Error(err))
		return nil, utils.NewBusinessError(constants.ErrK8sResourceList, "获取ServiceAccount列表失败")
	}

	serviceAccounts := make([]*model.K8sServiceAccountResponse, 0, len(saList.Items))
	for _, sa := range saList.Items {
		// TODO: 实现BuildServiceAccountResponse函数
		// saResponse := utils.BuildServiceAccountResponse(&sa, req.ClusterID)
		saResponse := &model.K8sServiceAccountResponse{
			Name:      sa.Name,
			Namespace: sa.Namespace,
			UID:       string(sa.UID),
			// 其他字段暂时设为默认值
		}
		serviceAccounts = append(serviceAccounts, saResponse)
	}

	// 处理分页
	if req.Page > 0 && req.PageSize > 0 {
		start := (req.Page - 1) * req.PageSize
		end := start + req.PageSize
		if start > len(serviceAccounts) {
			return []*model.K8sServiceAccountResponse{}, nil
		}
		if end > len(serviceAccounts) {
			end = len(serviceAccounts)
		}
		serviceAccounts = serviceAccounts[start:end]
	}

	return serviceAccounts, nil
}

// GetServiceAccountDetails 获取ServiceAccount详情
func (s *serviceAccountService) GetServiceAccountDetails(ctx context.Context, clusterID int, namespace, name string) (*model.K8sServiceAccountResponse, error) {
	kubeClient, err := s.client.GetKubeClient(clusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, fmt.Errorf("无法连接到Kubernetes集群: %w", err)
	}

	sa, err := kubeClient.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取ServiceAccount详情失败",
			zap.String("Namespace", namespace),
			zap.String("Name", name),
			zap.Error(err))
		return nil, fmt.Errorf("获取ServiceAccount详情失败: %w", err)
	}

	// TODO: 实现BuildServiceAccountResponse函数
	// response := utils.BuildServiceAccountResponse(sa, clusterID)
	response := &model.K8sServiceAccountResponse{
		Name:      sa.Name,
		Namespace: sa.Namespace,
		UID:       string(sa.UID),
		// 其他字段暂时设为默认值
	}

	// 获取ServiceAccount的Token（如果有的话）
	if len(sa.Secrets) > 0 {
		for _, secretRef := range sa.Secrets {
			secret, err := kubeClient.CoreV1().Secrets(namespace).Get(ctx, secretRef.Name, metav1.GetOptions{})
			if err != nil {
				s.logger.Warn("获取ServiceAccount关联的Secret失败",
					zap.String("SecretName", secretRef.Name),
					zap.Error(err))
				continue
			}

			if secret.Type == corev1.SecretTypeServiceAccountToken {
				if token, exists := secret.Data["token"]; exists {
					response.Token = string(token)
				}
				if caCert, exists := secret.Data["ca.crt"]; exists {
					response.CACert = string(caCert)
				}
				break
			}
		}
	}

	return response, nil
}

// CreateServiceAccount 创建ServiceAccount
func (s *serviceAccountService) CreateServiceAccount(ctx context.Context, req *model.ServiceAccountCreateReq) error {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return fmt.Errorf("无法连接到Kubernetes集群: %w", err)
	}

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		AutomountServiceAccountToken: req.AutomountServiceAccountToken,
	}

	// 添加ImagePullSecrets
	if len(req.ImagePullSecrets) > 0 {
		sa.ImagePullSecrets = make([]corev1.LocalObjectReference, 0, len(req.ImagePullSecrets))
		for _, secretName := range req.ImagePullSecrets {
			sa.ImagePullSecrets = append(sa.ImagePullSecrets, corev1.LocalObjectReference{
				Name: secretName,
			})
		}
	}

	_, err = kubeClient.CoreV1().ServiceAccounts(req.Namespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil {
		s.logger.Error("创建ServiceAccount失败",
			zap.String("Namespace", req.Namespace),
			zap.String("Name", req.Name),
			zap.Error(err))
		return utils.NewBusinessError(constants.ErrK8sResourceOperation, "创建ServiceAccount失败")
	}

	s.logger.Info("成功创建ServiceAccount",
		zap.String("Namespace", req.Namespace),
		zap.String("Name", req.Name))
	return nil
}

// UpdateServiceAccount 更新ServiceAccount
func (s *serviceAccountService) UpdateServiceAccount(ctx context.Context, req *model.ServiceAccountUpdateReq) error {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return utils.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	// 先获取现有的ServiceAccount
	existingSA, err := kubeClient.CoreV1().ServiceAccounts(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取ServiceAccount失败",
			zap.String("Namespace", req.Namespace),
			zap.String("Name", req.Name),
			zap.Error(err))
		return utils.NewBusinessError(constants.ErrK8sResourceGet, "获取ServiceAccount失败")
	}

	// 更新字段
	existingSA.Labels = req.Labels
	existingSA.Annotations = req.Annotations
	existingSA.AutomountServiceAccountToken = req.AutomountServiceAccountToken

	// 更新ImagePullSecrets
	if len(req.ImagePullSecrets) > 0 {
		existingSA.ImagePullSecrets = make([]corev1.LocalObjectReference, 0, len(req.ImagePullSecrets))
		for _, secretName := range req.ImagePullSecrets {
			existingSA.ImagePullSecrets = append(existingSA.ImagePullSecrets, corev1.LocalObjectReference{
				Name: secretName,
			})
		}
	} else {
		existingSA.ImagePullSecrets = nil
	}

	_, err = kubeClient.CoreV1().ServiceAccounts(req.Namespace).Update(ctx, existingSA, metav1.UpdateOptions{})
	if err != nil {
		s.logger.Error("更新ServiceAccount失败",
			zap.String("Namespace", req.Namespace),
			zap.String("Name", req.Name),
			zap.Error(err))
		return utils.NewBusinessError(constants.ErrK8sResourceOperation, "更新ServiceAccount失败")
	}

	s.logger.Info("成功更新ServiceAccount",
		zap.String("Namespace", req.Namespace),
		zap.String("Name", req.Name))
	return nil
}

// DeleteServiceAccount 删除ServiceAccount
func (s *serviceAccountService) DeleteServiceAccount(ctx context.Context, req *model.ServiceAccountDeleteReq) error {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return utils.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	deleteOptions := metav1.DeleteOptions{}
	if req.GracePeriodSeconds != nil {
		deleteOptions.GracePeriodSeconds = req.GracePeriodSeconds
	}

	if req.Force {
		// 强制删除需要设置GracePeriodSeconds为0
		zero := int64(0)
		deleteOptions.GracePeriodSeconds = &zero
	}

	err = kubeClient.CoreV1().ServiceAccounts(req.Namespace).Delete(ctx, req.Name, deleteOptions)
	if err != nil {
		s.logger.Error("删除ServiceAccount失败",
			zap.String("Namespace", req.Namespace),
			zap.String("Name", req.Name),
			zap.Error(err))
		return utils.NewBusinessError(constants.ErrK8sResourceDelete, "删除ServiceAccount失败")
	}

	s.logger.Info("成功删除ServiceAccount",
		zap.String("Namespace", req.Namespace),
		zap.String("Name", req.Name))
	return nil
}

// BatchDeleteServiceAccount 批量删除ServiceAccount
func (s *serviceAccountService) BatchDeleteServiceAccount(ctx context.Context, req *model.ServiceAccountBatchDeleteReq) error {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return utils.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	var errors []string
	deleteOptions := metav1.DeleteOptions{}
	if req.GracePeriodSeconds != nil {
		deleteOptions.GracePeriodSeconds = req.GracePeriodSeconds
	}

	if req.Force {
		zero := int64(0)
		deleteOptions.GracePeriodSeconds = &zero
	}

	for _, name := range req.Names {
		err := kubeClient.CoreV1().ServiceAccounts(req.Namespace).Delete(ctx, name, deleteOptions)
		if err != nil {
			errorMsg := fmt.Sprintf("删除ServiceAccount %s 失败: %v", name, err)
			errors = append(errors, errorMsg)
			s.logger.Error("批量删除ServiceAccount中的单个ServiceAccount失败",
				zap.String("Name", name),
				zap.Error(err))
		}
	}

	if len(errors) > 0 {
		return utils.NewBusinessError(constants.ErrK8sResourceDelete,
			fmt.Sprintf("批量删除失败，详情: %s", strings.Join(errors, "; ")))
	}

	s.logger.Info("成功批量删除ServiceAccount",
		zap.String("Namespace", req.Namespace),
		zap.Int("Count", len(req.Names)))
	return nil
}

// GetServiceAccountStatistics 获取ServiceAccount统计信息
func (s *serviceAccountService) GetServiceAccountStatistics(ctx context.Context, req *model.ServiceAccountStatisticsReq) (*model.ServiceAccountStatisticsResp, error) {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, utils.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	saList, err := kubeClient.CoreV1().ServiceAccounts(req.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		s.logger.Error("获取ServiceAccount列表失败",
			zap.String("Namespace", req.Namespace),
			zap.Error(err))
		return nil, utils.NewBusinessError(constants.ErrK8sResourceList, "获取ServiceAccount列表失败")
	}

	stats := &model.ServiceAccountStatisticsResp{
		TotalCount:  len(saList.Items),
		ActiveCount: len(saList.Items), // ServiceAccount一般都是活跃的
	}

	for _, sa := range saList.Items {
		if len(sa.Secrets) > 0 {
			stats.WithSecretsCount++
		}
		if len(sa.ImagePullSecrets) > 0 {
			stats.WithImagePullSecretsCount++
		}
		if sa.AutomountServiceAccountToken == nil || *sa.AutomountServiceAccountToken {
			stats.AutoMountEnabledCount++
		}
	}

	return stats, nil
}

// GetServiceAccountToken 获取ServiceAccount令牌
func (s *serviceAccountService) GetServiceAccountToken(ctx context.Context, req *model.ServiceAccountTokenReq) (*model.ServiceAccountTokenResp, error) {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, utils.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	// 创建TokenRequest
	tokenRequest := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			Audiences: []string{"https://kubernetes.default.svc.cluster.local"},
		},
	}

	if req.ExpirationSeconds != nil {
		tokenRequest.Spec.ExpirationSeconds = req.ExpirationSeconds
	}

	// 请求Token
	tokenResponse, err := kubeClient.CoreV1().ServiceAccounts(req.Namespace).CreateToken(
		ctx, req.Name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		s.logger.Error("获取ServiceAccount Token失败",
			zap.String("Namespace", req.Namespace),
			zap.String("Name", req.Name),
			zap.Error(err))
		return nil, utils.NewBusinessError(constants.ErrK8sResourceOperation, "获取ServiceAccount Token失败")
	}

	resp := &model.ServiceAccountTokenResp{
		Token: tokenResponse.Status.Token,
	}

	if !tokenResponse.Status.ExpirationTimestamp.IsZero() {
		resp.ExpirationTimestamp = &tokenResponse.Status.ExpirationTimestamp.Time
	}

	return resp, nil
}

// GetServiceAccountYaml 获取ServiceAccount YAML
func (s *serviceAccountService) GetServiceAccountYaml(ctx context.Context, req *model.ServiceAccountYamlReq) (*model.ServiceAccountYamlResp, error) {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, utils.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	sa, err := kubeClient.CoreV1().ServiceAccounts(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取ServiceAccount失败",
			zap.String("Namespace", req.Namespace),
			zap.String("Name", req.Name),
			zap.Error(err))
		return nil, utils.NewBusinessError(constants.ErrK8sResourceGet, "获取ServiceAccount失败")
	}

	// 清理不需要的字段
	sa.ManagedFields = nil

	// 转换为YAML
	yamlBytes, err := yaml.Marshal(sa)
	if err != nil {
		s.logger.Error("转换ServiceAccount为YAML失败", zap.Error(err))
		return nil, utils.NewBusinessError(constants.ErrK8sResourceOperation, "转换YAML失败")
	}

	return &model.ServiceAccountYamlResp{
		YAML: string(yamlBytes),
	}, nil
}

// UpdateServiceAccountYaml 更新ServiceAccount YAML
func (s *serviceAccountService) UpdateServiceAccountYaml(ctx context.Context, req *model.ServiceAccountUpdateYamlReq) error {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return utils.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	// 解析YAML
	var sa corev1.ServiceAccount
	if err := yaml.Unmarshal([]byte(req.YAML), &sa); err != nil {
		s.logger.Error("解析ServiceAccount YAML失败", zap.Error(err))
		return utils.NewBusinessError(constants.ErrInvalidParam, "YAML格式错误")
	}

	// 验证名称和命名空间是否匹配
	if sa.Name != req.Name {
		return utils.NewBusinessError(constants.ErrInvalidParam, "YAML中的名称与请求参数不匹配")
	}
	if sa.Namespace != req.Namespace {
		return utils.NewBusinessError(constants.ErrInvalidParam, "YAML中的命名空间与请求参数不匹配")
	}

	// 获取现有的ServiceAccount以保留ResourceVersion
	existingSA, err := kubeClient.CoreV1().ServiceAccounts(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取现有ServiceAccount失败",
			zap.String("Namespace", req.Namespace),
			zap.String("Name", req.Name),
			zap.Error(err))
		return utils.NewBusinessError(constants.ErrK8sResourceGet, "获取现有ServiceAccount失败")
	}

	// 保留必要的元数据
	sa.ResourceVersion = existingSA.ResourceVersion
	sa.UID = existingSA.UID

	// 更新ServiceAccount
	_, err = kubeClient.CoreV1().ServiceAccounts(req.Namespace).Update(ctx, &sa, metav1.UpdateOptions{})
	if err != nil {
		s.logger.Error("更新ServiceAccount YAML失败",
			zap.String("Namespace", req.Namespace),
			zap.String("Name", req.Name),
			zap.Error(err))
		return utils.NewBusinessError(constants.ErrK8sResourceOperation, "更新ServiceAccount失败")
	}

	s.logger.Info("成功更新ServiceAccount YAML",
		zap.String("Namespace", req.Namespace),
		zap.String("Name", req.Name))
	return nil
}

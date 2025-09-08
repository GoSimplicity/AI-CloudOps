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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceAccountService interface {
	// 基础 CRUD 操作
	GetServiceAccountList(ctx context.Context, req *model.GetServiceAccountListReq) (model.ListResp[*model.K8sServiceAccount], error)
	GetServiceAccountDetails(ctx context.Context, req *model.GetServiceAccountDetailsReq) (*model.K8sServiceAccount, error)
	CreateServiceAccount(ctx context.Context, req *model.CreateServiceAccountReq) error
	UpdateServiceAccount(ctx context.Context, req *model.UpdateServiceAccountReq) error
	DeleteServiceAccount(ctx context.Context, req *model.DeleteServiceAccountReq) error

	// YAML 操作
	GetServiceAccountYaml(ctx context.Context, req *model.GetServiceAccountYamlReq) (*model.K8sYaml, error)
	UpdateServiceAccountYaml(ctx context.Context, req *model.UpdateServiceAccountYamlReq) error

	// 扩展功能
	GetServiceAccountEvents(ctx context.Context, req *model.GetServiceAccountEventsReq) (model.ListResp[*model.K8sServiceAccountEvent], error)
	GetServiceAccountUsage(ctx context.Context, req *model.GetServiceAccountUsageReq) (*model.K8sServiceAccountUsage, error)

	GetServiceAccountToken(ctx context.Context, req *model.GetServiceAccountTokenReq) (*model.K8sServiceAccountToken, error)
	CreateServiceAccountToken(ctx context.Context, req *model.CreateServiceAccountTokenReq) (*model.K8sServiceAccountToken, error)
}

type serviceAccountService struct {
	rbacManager manager.RBACManager
}

func NewServiceAccountService(rbacManager manager.RBACManager) ServiceAccountService {
	return &serviceAccountService{
		rbacManager: rbacManager,
	}
}

// ======================== 基础 CRUD 操作 ========================

func (s *serviceAccountService) GetServiceAccountList(ctx context.Context, req *model.GetServiceAccountListReq) (model.ListResp[*model.K8sServiceAccount], error) {
	// 构建查询选项
	options := k8sutils.BuildServiceAccountListOptions(req)

	// 从 Manager 获取原始 ServiceAccount 列表（model 格式）
	serviceAccounts, err := s.rbacManager.GetServiceAccountList(ctx, req.ClusterID, req.Namespace, options)
	if err != nil {
		return model.ListResp[*model.K8sServiceAccount]{}, err
	}

	// 分页处理（在调用方处理，或保持原样返回并由上层分页）
	resp := model.ListResp[*model.K8sServiceAccount]{
		Items: serviceAccounts,
		Total: int64(len(serviceAccounts)),
	}
	return resp, nil
}

func (s *serviceAccountService) GetServiceAccountDetails(ctx context.Context, req *model.GetServiceAccountDetailsReq) (*model.K8sServiceAccount, error) {
	serviceAccount, err := s.rbacManager.GetServiceAccount(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	return k8sutils.BuildServiceAccountResponse(serviceAccount, req.ClusterID), nil
}

func (s *serviceAccountService) CreateServiceAccount(ctx context.Context, req *model.CreateServiceAccountReq) error {
	sa := k8sutils.ConvertToK8sServiceAccount(req)
	return s.rbacManager.CreateServiceAccount(ctx, req.ClusterID, req.Namespace, sa)
}

func (s *serviceAccountService) UpdateServiceAccount(ctx context.Context, req *model.UpdateServiceAccountReq) error {
	createReq := &model.CreateServiceAccountReq{
		ClusterID:                    req.ClusterID,
		Namespace:                    req.Namespace,
		Name:                         req.Name,
		Labels:                       req.Labels,
		Annotations:                  req.Annotations,
		AutomountServiceAccountToken: req.AutomountServiceAccountToken,
	}
	sa := k8sutils.ConvertToK8sServiceAccount(createReq)
	return s.rbacManager.UpdateServiceAccount(ctx, req.ClusterID, req.Namespace, sa)
}

func (s *serviceAccountService) DeleteServiceAccount(ctx context.Context, req *model.DeleteServiceAccountReq) error {
	return s.rbacManager.DeleteServiceAccount(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
}

// ======================== YAML 操作 ========================

func (s *serviceAccountService) GetServiceAccountYaml(ctx context.Context, req *model.GetServiceAccountYamlReq) (*model.K8sYaml, error) {
	serviceAccount, err := s.rbacManager.GetServiceAccount(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	yamlContent, err := k8sutils.ServiceAccountToYAML(serviceAccount)
	if err != nil {
		return nil, err
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

func (s *serviceAccountService) UpdateServiceAccountYaml(ctx context.Context, req *model.UpdateServiceAccountYamlReq) error {
	serviceAccount, err := k8sutils.YAMLToServiceAccount(req.Yaml)
	if err != nil {
		return err
	}

	return s.rbacManager.UpdateServiceAccount(ctx, req.ClusterID, req.Namespace, serviceAccount)
}

// ======================== 扩展功能 ========================

func (s *serviceAccountService) GetServiceAccountEvents(ctx context.Context, req *model.GetServiceAccountEventsReq) (model.ListResp[*model.K8sServiceAccountEvent], error) {
	// 暂未在 RBACManager 暴露，后续如需可在 Manager 层实现相同签名方法
	return model.ListResp[*model.K8sServiceAccountEvent]{}, nil
}

func (s *serviceAccountService) GetServiceAccountUsage(ctx context.Context, req *model.GetServiceAccountUsageReq) (*model.K8sServiceAccountUsage, error) {
	// 暂未在 RBACManager 暴露，后续如需可在 Manager 层实现相同签名方法
	return nil, nil
}

func (s *serviceAccountService) GetServiceAccountToken(ctx context.Context, req *model.GetServiceAccountTokenReq) (*model.K8sServiceAccountToken, error) {
	token, err := s.rbacManager.GetServiceAccountToken(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (s *serviceAccountService) CreateServiceAccountToken(ctx context.Context, req *model.CreateServiceAccountTokenReq) (*model.K8sServiceAccountToken, error) {
	token, err := s.rbacManager.CreateServiceAccountToken(ctx, req.ClusterID, req.Namespace, req.Name, req.ExpiryTime)
	if err != nil {
		return nil, err
	}
	return token, nil
}

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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceAccountService interface {
	GetServiceAccountList(ctx context.Context, req *model.GetServiceAccountListReq) (model.ListResp[*model.K8sServiceAccount], error)
	GetServiceAccountDetails(ctx context.Context, req *model.GetServiceAccountDetailsReq) (*model.K8sServiceAccount, error)
	CreateServiceAccount(ctx context.Context, req *model.CreateServiceAccountReq) error
	UpdateServiceAccount(ctx context.Context, req *model.UpdateServiceAccountReq) error
	DeleteServiceAccount(ctx context.Context, req *model.DeleteServiceAccountReq) error
	CreateServiceAccountByYaml(ctx context.Context, req *model.CreateServiceAccountByYamlReq) error

	GetServiceAccountYaml(ctx context.Context, req *model.GetServiceAccountYamlReq) (*model.K8sYaml, error)
	UpdateServiceAccountYaml(ctx context.Context, req *model.UpdateServiceAccountByYamlReq) error

	GetServiceAccountToken(ctx context.Context, req *model.GetServiceAccountTokenReq) (*model.ServiceAccountTokenInfo, error)
	CreateServiceAccountToken(ctx context.Context, req *model.CreateServiceAccountTokenReq) (*model.ServiceAccountTokenInfo, error)
}

type serviceAccountService struct {
	serviceAccountManager manager.ServiceAccountManager
	logger                *zap.Logger
}

func NewServiceAccountService(serviceAccountManager manager.ServiceAccountManager, logger *zap.Logger) ServiceAccountService {
	return &serviceAccountService{
		serviceAccountManager: serviceAccountManager,
		logger:                logger,
	}
}

func (s *serviceAccountService) GetServiceAccountList(ctx context.Context, req *model.GetServiceAccountListReq) (model.ListResp[*model.K8sServiceAccount], error) {

	options := k8sutils.BuildServiceAccountListOptions(req)

	serviceAccountList, err := s.serviceAccountManager.GetServiceAccountList(ctx, req.ClusterID, req.Namespace, options)
	if err != nil {
		return model.ListResp[*model.K8sServiceAccount]{}, err
	}

	var filtered []*model.K8sServiceAccount
	keyword := strings.TrimSpace(req.Keyword)
	for _, sa := range serviceAccountList.Items {
		entity := k8sutils.BuildServiceAccountResponse(&sa, req.ClusterID)
		if entity == nil {
			continue
		}
		if keyword != "" {
			// 名称、标签、注解关键字匹配
			match := strings.Contains(entity.Name, keyword)
			if !match {
				for k, v := range entity.Labels {
					if strings.Contains(k, keyword) || strings.Contains(v, keyword) {
						match = true
						break
					}
				}
			}
			if !match {
				for k, v := range entity.Annotations {
					if strings.Contains(k, keyword) || strings.Contains(v, keyword) {
						match = true
						break
					}
				}
			}
			if !match {
				continue
			}
		}
		filtered = append(filtered, entity)
	}

	// 分页
	page := req.Page
	pageSize := req.Size
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	return model.ListResp[*model.K8sServiceAccount]{
		Items: filtered[start:end],
		Total: int64(len(filtered)),
	}, nil
}

func (s *serviceAccountService) GetServiceAccountDetails(ctx context.Context, req *model.GetServiceAccountDetailsReq) (*model.K8sServiceAccount, error) {
	serviceAccount, err := s.serviceAccountManager.GetServiceAccount(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	return k8sutils.BuildServiceAccountResponse(serviceAccount, req.ClusterID), nil
}

func (s *serviceAccountService) CreateServiceAccount(ctx context.Context, req *model.CreateServiceAccountReq) error {
	sa := k8sutils.ConvertToK8sServiceAccount(req)
	return s.serviceAccountManager.CreateServiceAccount(ctx, req.ClusterID, req.Namespace, sa)
}

func (s *serviceAccountService) UpdateServiceAccount(ctx context.Context, req *model.UpdateServiceAccountReq) error {
	createReq := &model.CreateServiceAccountReq{
		ClusterID:                    req.ClusterID,
		Namespace:                    req.Namespace,
		Name:                         req.Name,
		Labels:                       req.Labels,
		Annotations:                  req.Annotations,
		AutomountServiceAccountToken: req.AutomountServiceAccountToken,
		ImagePullSecrets:             req.ImagePullSecrets,
		Secrets:                      req.Secrets,
	}
	sa := k8sutils.ConvertToK8sServiceAccount(createReq)
	return s.serviceAccountManager.UpdateServiceAccount(ctx, req.ClusterID, req.Namespace, sa)
}

func (s *serviceAccountService) DeleteServiceAccount(ctx context.Context, req *model.DeleteServiceAccountReq) error {
	return s.serviceAccountManager.DeleteServiceAccount(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
}

// ======================== YAML 操作 ========================

func (s *serviceAccountService) GetServiceAccountYaml(ctx context.Context, req *model.GetServiceAccountYamlReq) (*model.K8sYaml, error) {
	serviceAccount, err := s.serviceAccountManager.GetServiceAccount(ctx, req.ClusterID, req.Namespace, req.Name)
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

func (s *serviceAccountService) UpdateServiceAccountYaml(ctx context.Context, req *model.UpdateServiceAccountByYamlReq) error {
	serviceAccount, err := k8sutils.YAMLToServiceAccount(req.YamlContent)
	if err != nil {
		return err
	}

	return s.serviceAccountManager.UpdateServiceAccount(ctx, req.ClusterID, req.Namespace, serviceAccount)
}

func (s *serviceAccountService) CreateServiceAccountByYaml(ctx context.Context, req *model.CreateServiceAccountByYamlReq) error {
	serviceAccount, err := k8sutils.YAMLToServiceAccount(req.YamlContent)
	if err != nil {
		return err
	}

	// 如果YAML中没有指定namespace，使用default命名空间
	if serviceAccount.Namespace == "" {
		serviceAccount.Namespace = "default"
		s.logger.Info("YAML中未指定namespace，使用default命名空间",
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", serviceAccount.Name))
	}

	return s.serviceAccountManager.CreateServiceAccount(ctx, req.ClusterID, serviceAccount.Namespace, serviceAccount)
}

func (s *serviceAccountService) GetServiceAccountToken(ctx context.Context, req *model.GetServiceAccountTokenReq) (*model.ServiceAccountTokenInfo, error) {
	// Get existing tokens - this is a simplified implementation
	tokens, err := s.serviceAccountManager.GetServiceAccountTokens(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	if len(tokens) == 0 {
		return nil, fmt.Errorf("no tokens found for service account %s", req.Name)
	}

	// Return the first token (simplified)
	return &model.ServiceAccountTokenInfo{
		Token: tokens[0],
	}, nil
}

func (s *serviceAccountService) CreateServiceAccountToken(ctx context.Context, req *model.CreateServiceAccountTokenReq) (*model.ServiceAccountTokenInfo, error) {
	// Create token request
	tokenRequest := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			ExpirationSeconds: req.ExpirationSeconds,
		},
	}

	tokenResp, err := s.serviceAccountManager.CreateServiceAccountToken(ctx, req.ClusterID, req.Namespace, req.ServiceAccountName, tokenRequest)
	if err != nil {
		return nil, err
	}

	return &model.ServiceAccountTokenInfo{
		Token:             tokenResp.Status.Token,
		ExpirationSeconds: req.ExpirationSeconds,
		ExpirationTime:    tokenResp.Status.ExpirationTimestamp.Time.Format("2006-01-02T15:04:05Z"),
	}, nil
}

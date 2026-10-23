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
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	systemService "github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"go.uber.org/zap"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

	// GetServiceAccountToken 返回属于该 SA 的 token Secret 中的 token；每次调用同步写审计（不含 token），审计失败不返回 token
	GetServiceAccountToken(ctx context.Context, req *model.GetServiceAccountTokenReq, actor model.SecretRevealActor) (*model.ServiceAccountTokenInfo, error)
	// CreateServiceAccountToken 通过 TokenRequest 签发 token；审计规则同 GetServiceAccountToken
	CreateServiceAccountToken(ctx context.Context, req *model.CreateServiceAccountTokenReq, actor model.SecretRevealActor) (*model.ServiceAccountTokenInfo, error)
}

type serviceAccountService struct {
	serviceAccountManager manager.ServiceAccountManager
	auditService          systemService.AuditService
	logger                *zap.Logger
}

func NewServiceAccountService(serviceAccountManager manager.ServiceAccountManager, auditService systemService.AuditService, logger *zap.Logger) ServiceAccountService {
	return &serviceAccountService{
		serviceAccountManager: serviceAccountManager,
		auditService:          auditService,
		logger:                logger,
	}
}

func (s *serviceAccountService) GetServiceAccountList(ctx context.Context, req *model.GetServiceAccountListReq) (model.ListResp[*model.K8sServiceAccount], error) {

	options := k8sutils.BuildServiceAccountListOptions(req)

	serviceAccountList, err := s.serviceAccountManager.GetServiceAccountList(ctx, req.ClusterID, req.Namespace, options)
	if err != nil {
		return model.ListResp[*model.K8sServiceAccount]{}, err
	}

	// 应用过滤条件
	var filtered []*model.K8sServiceAccount
	for _, sa := range serviceAccountList.Items {
		entity := k8sutils.BuildServiceAccountResponse(&sa, req.ClusterID)
		if entity == nil {
			continue
		}
		// 名称过滤（使用通用的Search字段，支持不区分大小写）
		if !k8sutils.FilterByName(entity.Name, req.Search) {
			continue
		}
		filtered = append(filtered, entity)
	}

	// 按创建时间排序（最新的在前）
	k8sutils.SortByCreationTime(filtered, func(sa *model.K8sServiceAccount) time.Time {
		t, _ := time.Parse(time.RFC3339, sa.CreatedAt)
		return t
	})

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
	// SEC-29：secrets 引用由集群（token controller）维护，用户提交的引用一律忽略，防止借挂其他 SA 的 token Secret
	s.dropUserSecretRefs(sa, req.ClusterID, "create")
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
	}
	sa := k8sutils.ConvertToK8sServiceAccount(createReq)
	// SEC-29：忽略请求中的 secrets，保留集群中现有的引用（Update 为整体替换，不保留会清空集群维护的引用）
	if err := s.keepExistingSecretRefs(ctx, req.ClusterID, req.Namespace, req.Name, sa, req.Secrets, "update"); err != nil {
		return err
	}
	return s.serviceAccountManager.UpdateServiceAccount(ctx, req.ClusterID, req.Namespace, sa)
}

// dropUserSecretRefs 创建时丢弃用户提交的 secrets 引用并记录告警日志（只记录名称）
func (s *serviceAccountService) dropUserSecretRefs(sa *corev1.ServiceAccount, clusterID int, op string) {
	if len(sa.Secrets) == 0 {
		return
	}
	s.logger.Warn("忽略请求中的ServiceAccount secrets引用（SEC-29：该字段由集群维护）",
		zap.String("operation", op),
		zap.Int("cluster_id", clusterID),
		zap.String("namespace", sa.Namespace),
		zap.String("name", sa.Name),
		zap.Strings("ignored_secrets", k8sutils.ServiceAccountSecretRefNames(sa.Secrets)))
	sa.Secrets = nil
}

// keepExistingSecretRefs 更新时用集群中现有的 secrets 引用覆盖请求值；请求值与现有值不同则记录告警日志
func (s *serviceAccountService) keepExistingSecretRefs(ctx context.Context, clusterID int, namespace, name string, sa *corev1.ServiceAccount, requested []string, op string) error {
	existing, err := s.serviceAccountManager.GetServiceAccount(ctx, clusterID, namespace, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("ServiceAccount不存在: %s/%s", namespace, name)
		}
		return fmt.Errorf("获取现有ServiceAccount失败: %w", err)
	}
	existingNames := k8sutils.ServiceAccountSecretRefNames(existing.Secrets)
	if requested != nil && !slices.Equal(requested, existingNames) {
		s.logger.Warn("忽略请求中的ServiceAccount secrets引用（SEC-29：该字段由集群维护），保留集群现有引用",
			zap.String("operation", op),
			zap.Int("cluster_id", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Strings("ignored_secrets", requested),
			zap.Strings("kept_secrets", existingNames))
	}
	sa.Secrets = existing.Secrets
	return nil
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

	// 以URL参数为准，避免 YAML 中的 name/namespace 指向另一个 SA
	serviceAccount.Namespace = req.Namespace
	serviceAccount.Name = req.Name

	var requested []string
	if serviceAccount.Secrets != nil {
		requested = k8sutils.ServiceAccountSecretRefNames(serviceAccount.Secrets)
	}
	// SEC-29：YAML 中的 secrets 同样忽略，保留集群现有引用（从详情 YAML 原样回写时不产生变化）
	if err := s.keepExistingSecretRefs(ctx, req.ClusterID, req.Namespace, req.Name, serviceAccount, requested, "update_yaml"); err != nil {
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

	// SEC-29：YAML 中的 secrets 引用同样忽略
	s.dropUserSecretRefs(serviceAccount, req.ClusterID, "create_yaml")

	return s.serviceAccountManager.CreateServiceAccount(ctx, req.ClusterID, serviceAccount.Namespace, serviceAccount)
}

// GetServiceAccountToken 只返回注解 kubernetes.io/service-account.name（及存在时的 .uid）指向该 SA 的 token Secret。
// 每次调用同步写审计（只含元数据，绝不含 token）；成功读取后审计写入失败则不返回 token（fail closed）。
func (s *serviceAccountService) GetServiceAccountToken(ctx context.Context, req *model.GetServiceAccountTokenReq, actor model.SecretRevealActor) (*model.ServiceAccountTokenInfo, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}
	start := time.Now()
	record := model.ServiceAccountTokenAuditRecord{
		Action:         model.ServiceAccountTokenRevealAuditAction,
		ClusterID:      req.ClusterID,
		Namespace:      req.Namespace,
		ServiceAccount: req.Name,
	}
	fail := func(status int, err error) error {
		_ = s.writeTokenAudit(ctx, actor, record, model.ServiceAccountTokenRevealAuditOperation, "GET", start, status, err)
		return err
	}

	sa, err := s.serviceAccountManager.GetServiceAccount(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return nil, fail(k8sErrorStatusCode(err), serviceAccountGetError(req.Namespace, req.Name, err))
	}
	record.ServiceAccountUID = string(sa.UID)

	secrets, err := s.serviceAccountManager.GetServiceAccountSecrets(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return nil, fail(k8sErrorStatusCode(err), fmt.Errorf("获取ServiceAccount关联的Secret失败: %w", err))
	}

	selected, rejected := k8sutils.SelectServiceAccountTokenSecret(sa, secrets)
	if len(rejected) > 0 {
		s.logger.Warn("忽略不属于该ServiceAccount的token Secret（注解 service-account.name/uid 不匹配）",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Strings("rejected_secrets", rejected))
	}
	if selected == nil {
		return nil, fail(http.StatusNotFound, fmt.Errorf("ServiceAccount %s/%s 没有属于自己的token Secret", req.Namespace, req.Name))
	}
	record.SecretName = selected.Name

	if err := s.writeTokenAudit(ctx, actor, record, model.ServiceAccountTokenRevealAuditOperation, "GET", start, http.StatusOK, nil); err != nil {
		s.logger.Error("写入ServiceAccount token查看审计失败，拒绝返回token", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("%w，已拒绝返回ServiceAccount token", ErrAuditWriteFailed)
	}

	s.logger.Info("ServiceAccount token已查看",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name),
		zap.String("secret_name", selected.Name),
		zap.Int("user_id", actor.UserID))

	return &model.ServiceAccountTokenInfo{
		Token:     string(selected.Data[corev1.ServiceAccountTokenKey]),
		CreatedAt: selected.CreationTimestamp.UTC().Format(time.RFC3339),
	}, nil
}

// CreateServiceAccountToken 通过 TokenRequest 为 SA 签发 token。审计与 fail closed 规则同 GetServiceAccountToken：
// 签发成功但审计写入失败时不返回 token（已签发的 token 无法撤销，只能等待过期，见发布文档）。
func (s *serviceAccountService) CreateServiceAccountToken(ctx context.Context, req *model.CreateServiceAccountTokenReq, actor model.SecretRevealActor) (*model.ServiceAccountTokenInfo, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}
	start := time.Now()
	record := model.ServiceAccountTokenAuditRecord{
		Action:            model.ServiceAccountTokenCreateAuditAction,
		ClusterID:         req.ClusterID,
		Namespace:         req.Namespace,
		ServiceAccount:    req.ServiceAccountName,
		ExpirationSeconds: req.ExpirationSeconds,
	}

	tokenRequest := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			ExpirationSeconds: req.ExpirationSeconds,
		},
	}

	tokenResp, err := s.serviceAccountManager.CreateServiceAccountToken(ctx, req.ClusterID, req.Namespace, req.ServiceAccountName, tokenRequest)
	if err != nil {
		userErr := fmt.Errorf("创建ServiceAccount token失败: %w", err)
		_ = s.writeTokenAudit(ctx, actor, record, model.ServiceAccountTokenCreateAuditOperation, "POST", start, k8sErrorStatusCode(err), userErr)
		return nil, userErr
	}

	if err := s.writeTokenAudit(ctx, actor, record, model.ServiceAccountTokenCreateAuditOperation, "POST", start, http.StatusOK, nil); err != nil {
		s.logger.Error("写入ServiceAccount token签发审计失败，拒绝返回token", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.ServiceAccountName))
		return nil, fmt.Errorf("%w，已拒绝返回ServiceAccount token", ErrAuditWriteFailed)
	}

	s.logger.Info("ServiceAccount token已签发",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.ServiceAccountName),
		zap.Int("user_id", actor.UserID))

	expiration := req.ExpirationSeconds
	if tokenResp.Status.ExpirationTimestamp.IsZero() {
		return &model.ServiceAccountTokenInfo{Token: tokenResp.Status.Token, ExpirationSeconds: expiration}, nil
	}
	return &model.ServiceAccountTokenInfo{
		Token:             tokenResp.Status.Token,
		ExpirationSeconds: expiration,
		ExpirationTime:    tokenResp.Status.ExpirationTimestamp.Time.Format("2006-01-02T15:04:05Z"),
	}, nil
}

func serviceAccountGetError(namespace, name string, err error) error {
	if errors.IsNotFound(err) {
		return fmt.Errorf("ServiceAccount不存在: %s/%s", namespace, name)
	}
	return fmt.Errorf("获取ServiceAccount失败: %w", err)
}

// writeTokenAudit 同步写入 token 接口审计；RequestBody 只含元数据，绝不含 token。
func (s *serviceAccountService) writeTokenAudit(ctx context.Context, actor model.SecretRevealActor, record model.ServiceAccountTokenAuditRecord, operation, method string, start time.Time, statusCode int, opErr error) error {
	if s.auditService == nil {
		return fmt.Errorf("审计服务未初始化")
	}
	record.UserID = actor.UserID
	record.Username = actor.Username
	record.Success = opErr == nil
	record.IssuedAt = start.UTC().Format(time.RFC3339)
	body, err := json.Marshal(record)
	if err != nil {
		return err
	}
	errMsg := ""
	if opErr != nil {
		errMsg = opErr.Error()
	}
	return s.auditService.CreateAuditLog(ctx, &model.CreateAuditLogRequest{
		UserID:        actor.UserID,
		TraceID:       actor.TraceID,
		IPAddress:     actor.IP,
		UserAgent:     actor.UserAgent,
		HttpMethod:    method,
		Endpoint:      actor.Endpoint,
		OperationType: operation,
		TargetType:    model.ServiceAccountTokenAuditTargetType,
		TargetID:      fmt.Sprintf("%d/%s/%s", record.ClusterID, record.Namespace, record.ServiceAccount),
		StatusCode:    statusCode,
		RequestBody:   body,
		Duration:      time.Since(start).Microseconds(),
		ErrorMsg:      errMsg,
	})
}

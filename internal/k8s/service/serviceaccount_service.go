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

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	systemService "github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
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
	desired := k8sutils.ConvertToK8sServiceAccount(createReq)
	return s.updateWithRetry(ctx, req.ClusterID, req.Namespace, req.Name, desired, req.Secrets, "update")
}

// updateWithRetry 用 RetryOnConflict 更新 SA：每次重试先读取最新版本，只把允许修改的字段
// （labels/annotations/automountServiceAccountToken/imagePullSecrets）应用上去，
// SEC-29：请求中的 secrets 一律忽略，保留集群现有引用；name/namespace 以 URL 为准。
func (s *serviceAccountService) updateWithRetry(ctx context.Context, clusterID int, namespace, name string, desired *corev1.ServiceAccount, requestedSecrets []string, op string) error {
	err := s.serviceAccountManager.UpdateServiceAccountWithRetry(ctx, clusterID, namespace, name, func(latest *corev1.ServiceAccount) error {
		s.warnIgnoredSecretRefs(latest, clusterID, requestedSecrets, op)
		k8sutils.ApplyServiceAccountEditableFields(latest, desired)
		return nil
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("ServiceAccount不存在: %s/%s", namespace, name)
		}
		return fmt.Errorf("更新ServiceAccount失败: %w", err)
	}
	return nil
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

// warnIgnoredSecretRefs 请求中的 secrets 与集群现有引用不同时记录告警日志（只记录名称）；引用本身不会被修改
func (s *serviceAccountService) warnIgnoredSecretRefs(latest *corev1.ServiceAccount, clusterID int, requested []string, op string) {
	if requested == nil {
		return
	}
	existingNames := k8sutils.ServiceAccountSecretRefNames(latest.Secrets)
	if slices.Equal(requested, existingNames) {
		return
	}
	s.logger.Warn("忽略请求中的ServiceAccount secrets引用（SEC-29：该字段由集群维护），保留集群现有引用",
		zap.String("operation", op),
		zap.Int("cluster_id", clusterID),
		zap.String("namespace", latest.Namespace),
		zap.String("name", latest.Name),
		zap.Strings("ignored_secrets", requested),
		zap.Strings("kept_secrets", existingNames))
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

	// name/namespace 以 URL 为准（YAML 中的 name/namespace 不生效）；
	// SEC-29：YAML 中的 secrets 同样忽略，保留集群现有引用（从详情 YAML 原样回写时不产生变化）
	var requested []string
	if serviceAccount.Secrets != nil {
		requested = k8sutils.ServiceAccountSecretRefNames(serviceAccount.Secrets)
	}
	if rv := serviceAccount.ResourceVersion; rv != "" {
		// YAML 带了 metadata.resourceVersion：严格乐观锁，只 Update 一次，版本过期返回 409，不重试
		return s.updateWithResourceVersion(ctx, req.ClusterID, req.Namespace, req.Name, rv, serviceAccount, requested, "update_yaml")
	}
	// 未带 resourceVersion：RetryOnConflict，基于最新版本应用可编辑字段
	return s.updateWithRetry(ctx, req.ClusterID, req.Namespace, req.Name, serviceAccount, requested, "update_yaml")
}

// updateWithResourceVersion 以调用方提交的 resourceVersion 做一次乐观锁更新；可编辑字段与 secrets 规则同 updateWithRetry。
// 版本冲突返回 BusinessError{constants.ErrK8sResourceConflict}，handler 映射为 HTTP 409。
func (s *serviceAccountService) updateWithResourceVersion(ctx context.Context, clusterID int, namespace, name, resourceVersion string, desired *corev1.ServiceAccount, requestedSecrets []string, op string) error {
	err := s.serviceAccountManager.UpdateServiceAccountWithResourceVersion(ctx, clusterID, namespace, name, resourceVersion, func(latest *corev1.ServiceAccount) error {
		s.warnIgnoredSecretRefs(latest, clusterID, requestedSecrets, op)
		k8sutils.ApplyServiceAccountEditableFields(latest, desired)
		return nil
	})
	if err != nil {
		switch {
		case errors.IsNotFound(err):
			return fmt.Errorf("ServiceAccount不存在: %s/%s", namespace, name)
		case errors.IsConflict(err):
			return base.NewBusinessError(constants.ErrK8sResourceConflict, fmt.Sprintf(
				"ServiceAccount %s/%s 已被修改（提交的 resourceVersion=%s 已过期），请重新获取 YAML 后再提交", namespace, name, resourceVersion))
		}
		return fmt.Errorf("更新ServiceAccount失败: %w", err)
	}
	return nil
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

	// 复用上面读到的 SA（归属校验用的也是同一个对象），不再重复 Get
	secrets, err := s.serviceAccountManager.GetServiceAccountSecrets(ctx, req.ClusterID, sa)
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

	// 有效期校验必须在调用 K8s 之前完成：越界直接拒绝（400），不签发 token，但仍写审计
	expiration, err := normalizeTokenExpiration(req.ExpirationSeconds)
	if err != nil {
		_ = s.writeTokenAudit(ctx, actor, record, model.ServiceAccountTokenCreateAuditOperation, "POST", start, http.StatusBadRequest, err)
		return nil, err
	}
	record.ExpirationSeconds = &expiration

	tokenRequest := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			ExpirationSeconds: &expiration,
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

	if tokenResp.Spec.ExpirationSeconds != nil {
		expiration = *tokenResp.Spec.ExpirationSeconds
	}
	if tokenResp.Status.ExpirationTimestamp.IsZero() {
		return &model.ServiceAccountTokenInfo{Token: tokenResp.Status.Token, ExpirationSeconds: &expiration}, nil
	}
	return &model.ServiceAccountTokenInfo{
		Token:             tokenResp.Status.Token,
		ExpirationSeconds: &expiration,
		ExpirationTime:    tokenResp.Status.ExpirationTimestamp.Time.UTC().Format("2006-01-02T15:04:05Z"),
	}, nil
}

// normalizeTokenExpiration 校验 TokenRequest 有效期：nil 或 0 取默认值 3600；
// 小于 600 或大于 3600（含负数）返回参数错误（BusinessError{ErrInvalidParam}，handler 映射为 HTTP 400）。
func normalizeTokenExpiration(requested *int64) (int64, error) {
	if requested == nil || *requested == 0 {
		return model.ServiceAccountTokenDefaultExpirationSeconds, nil
	}
	v := *requested
	if v < model.ServiceAccountTokenMinExpirationSeconds || v > model.ServiceAccountTokenMaxExpirationSeconds {
		return 0, base.NewBusinessError(constants.ErrInvalidParam, fmt.Sprintf("expiration_seconds 取值范围为 %d~%d 秒（不传或传 0 默认 %d），当前为 %d",
			model.ServiceAccountTokenMinExpirationSeconds, model.ServiceAccountTokenMaxExpirationSeconds,
			model.ServiceAccountTokenDefaultExpirationSeconds, v))
	}
	return v, nil
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

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
	stderrors "errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	systemService "github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SecretService interface {
	GetSecretList(ctx context.Context, req *model.GetSecretListReq) (model.ListResp[*model.K8sSecret], error)
	GetSecret(ctx context.Context, req *model.GetSecretDetailsReq) (*model.K8sSecret, error)
	CreateSecret(ctx context.Context, req *model.CreateSecretReq) error
	UpdateSecret(ctx context.Context, req *model.UpdateSecretReq) error
	// YAML相关方法
	CreateSecretByYaml(ctx context.Context, req *model.CreateSecretByYamlReq) error
	UpdateSecretByYaml(ctx context.Context, req *model.UpdateSecretByYamlReq) error
	DeleteSecret(ctx context.Context, req *model.DeleteSecretReq) error
	GetSecretYAML(ctx context.Context, req *model.GetSecretYamlReq) (*model.K8sYaml, error)
	// RevealSecret 返回 Secret 明文；每次调用都会写一条审计（只含元数据与键名，不含值）
	RevealSecret(ctx context.Context, req *model.RevealSecretReq, actor model.SecretRevealActor) (*model.K8sSecretReveal, error)
}

// secretService Secret服务实现
type secretService struct {
	secretManager manager.SecretManager
	auditService  systemService.AuditService
	logger        *zap.Logger
}

func NewSecretService(secretManager manager.SecretManager, auditService systemService.AuditService, logger *zap.Logger) SecretService {
	return &secretService{
		secretManager: secretManager,
		auditService:  auditService,
		logger:        logger,
	}
}

func (s *secretService) GetSecretList(ctx context.Context, req *model.GetSecretListReq) (model.ListResp[*model.K8sSecret], error) {
	if req == nil {
		return model.ListResp[*model.K8sSecret]{}, fmt.Errorf("请求参数不能为空")
	}

	var list *corev1.SecretList
	var err error

	labelSelector := ""
	if len(req.Labels) > 0 {
		var labels []string
		for k, v := range req.Labels {
			labels = append(labels, fmt.Sprintf("%s=%s", k, v))
		}
		labelSelector = strings.Join(labels, ",")
	}

	if labelSelector != "" {
		list, err = s.secretManager.ListSecretsBySelectors(ctx, req.ClusterID, req.Namespace, labelSelector, "")
	} else {
		list, err = s.secretManager.ListSecrets(ctx, req.ClusterID, req.Namespace)
	}
	if err != nil {
		s.logger.Error("获取Secret列表失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID), zap.String("namespace", req.Namespace))
		return model.ListResp[*model.K8sSecret]{}, fmt.Errorf("获取Secret列表失败: %w", err)
	}

	// 应用过滤条件
	entities := make([]*model.K8sSecret, 0, len(list.Items))
	for _, item := range list.Items {
		entity := s.convertToK8sSecret(&item, req.ClusterID)
		// 类型过滤
		if req.Type != "" && string(entity.Type) != string(req.Type) {
			continue
		}
		// 名称过滤（使用通用的Search字段，支持不区分大小写）
		if !utils.FilterByName(entity.Name, req.Search) {
			continue
		}
		entities = append(entities, entity)
	}

	// 按创建时间排序（最新的在前）
	utils.SortByCreationTime(entities, func(secret *model.K8sSecret) time.Time {
		return secret.CreatedAt
	})

	page := req.Page
	size := req.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	start := (page - 1) * size
	end := start + size
	total := int64(len(entities))
	if start > len(entities) {
		start = len(entities)
	}
	if end > len(entities) {
		end = len(entities)
	}

	s.logger.Info("成功获取Secret列表",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.Int("count", len(entities)))

	return model.ListResp[*model.K8sSecret]{Items: entities[start:end], Total: total}, nil
}

func (s *secretService) GetSecret(ctx context.Context, req *model.GetSecretDetailsReq) (*model.K8sSecret, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}

	secret, err := s.secretManager.GetSecret(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("Secret不存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("获取Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Secret失败: %w", err)
	}

	result := s.convertToK8sSecret(secret, req.ClusterID)

	s.logger.Info("成功获取Secret详情",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return result, nil
}

// convertToK8sSecret 将Kubernetes Secret转换为模型对象
func (s *secretService) convertToK8sSecret(secret *corev1.Secret, clusterID int) *model.K8sSecret {
	var totalSize int64
	for _, v := range secret.Data {
		totalSize += int64(len(v))
	}
	for _, v := range secret.StringData {
		totalSize += int64(len(v))
	}

	// 格式化大小
	size := k8sutils.FormatBytes(totalSize)

	dataCount := len(secret.Data) + len(secret.StringData)

	// 判断是否不可变
	immutable := false
	if secret.Immutable != nil {
		immutable = *secret.Immutable
	}

	return &model.K8sSecret{
		Name:      secret.Name,
		Namespace: secret.Namespace,
		ClusterID: clusterID,
		UID:       string(secret.UID),
		Type:      model.K8sSecretType(secret.Type),
		// SEC-25：列表/详情默认脱敏，只返回键与值长度；明文走 reveal 接口
		Data:           k8sutils.RedactSecretData(secret.Data),
		StringData:     k8sutils.RedactSecretStringData(secret.StringData), // 通常为空，因为K8s会转换为Data
		DataKeys:       k8sutils.SecretDataKeyInfos(secret.Data),
		StringDataKeys: k8sutils.SecretStringDataKeyInfos(secret.StringData),
		Redacted:       true,
		Labels:         secret.Labels,
		Annotations:    k8sutils.RedactSecretAnnotations(secret.Annotations),
		Immutable:      immutable,
		DataCount:      dataCount,
		Size:           size,
		Age:            time.Since(secret.CreationTimestamp.Time).String(),
		CreatedAt:      secret.CreationTimestamp.Time,
		RawSecret:      secret,
	}
}

func (s *secretService) CreateSecret(ctx context.Context, req *model.CreateSecretReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}

	// 构造Secret对象
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Type:       corev1.SecretType(req.Type),
		Data:       map[string][]byte(req.Data),
		StringData: req.StringData,
	}

	// 如果没有指定类型，默认为 Opaque
	if secret.Type == "" {
		secret.Type = corev1.SecretTypeOpaque
	}

	if req.Immutable {
		secret.Immutable = &req.Immutable
	}

	if err := k8sutils.ValidateSecretData(secret.Type, map[string][]byte(req.Data), secret.StringData); err != nil {
		return err
	}
	_, err := s.secretManager.CreateSecret(ctx, req.ClusterID, secret)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return fmt.Errorf("Secret已存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("创建Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建Secret失败: %w", err)
	}
	s.logger.Info("成功创建Secret",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

func (s *secretService) UpdateSecret(ctx context.Context, req *model.UpdateSecretReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}

	// 获取现有的Secret以获取ResourceVersion（用于乐观锁）
	existingSecret, err := s.secretManager.GetSecret(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("Secret不存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("获取Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("获取Secret失败: %w", err)
	}

	updatedSecret := existingSecret.DeepCopy()

	updatedSecret.Data = preserveRedactedData(map[string][]byte(req.Data), existingSecret.Data)
	updatedSecret.Data, updatedSecret.StringData = preserveRedactedStringData(req.StringData, updatedSecret.Data, existingSecret)
	updatedSecret.Labels = req.Labels
	updatedSecret.Annotations = preserveLastApplied(req.Annotations, existingSecret.Annotations)

	// Type和Immutable字段在创建后通常不能修改，保持原值
	// 如果需要修改，K8s会返回错误

	_, err = s.secretManager.UpdateSecret(ctx, req.ClusterID, updatedSecret)
	if err != nil {
		s.logger.Error("更新Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新Secret失败: %w", err)
	}

	s.logger.Info("成功更新Secret (完全覆盖)",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

func (s *secretService) DeleteSecret(ctx context.Context, req *model.DeleteSecretReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}

	err := s.secretManager.DeleteSecret(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("Secret不存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("删除Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除Secret失败: %w", err)
	}

	s.logger.Info("成功删除Secret",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

func (s *secretService) GetSecretYAML(ctx context.Context, req *model.GetSecretYamlReq) (*model.K8sYaml, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}

	secret, err := s.secretManager.GetSecret(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("Secret不存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("获取Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Secret失败: %w", err)
	}

	yamlStr, err := k8sutils.SecretToYAML(secret)
	if err != nil {
		s.logger.Error("转换Secret为YAML失败", zap.Error(err))
		return nil, fmt.Errorf("转换Secret为YAML失败: %w", err)
	}

	s.logger.Info("成功获取Secret YAML",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return &model.K8sYaml{
		YAML: yamlStr,
	}, nil
}

func (s *secretService) CreateSecretByYaml(ctx context.Context, req *model.CreateSecretByYamlReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}

	// SEC-25：脱敏占位符不是真实值，禁止用于创建
	if hasPlaceholder, err := k8sutils.SecretYAMLHasRedactedPlaceholder(req.YAML); err != nil {
		return err
	} else if hasPlaceholder {
		return fmt.Errorf("YAML 中包含脱敏占位符 %s，请填写真实值后再创建", k8sutils.SecretRedactedPlaceholder)
	}

	sec, err := k8sutils.YAMLToSecret(req.YAML)
	if err != nil {
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 如果YAML中没有指定namespace，使用default命名空间
	if sec.Namespace == "" {
		sec.Namespace = "default"
		s.logger.Info("YAML中未指定namespace，使用default命名空间",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("name", sec.Name))
	}

	_, err = s.secretManager.CreateSecret(ctx, req.ClusterID, sec)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return fmt.Errorf("Secret已存在: %s/%s", sec.Namespace, sec.Name)
		}
		s.logger.Error("通过YAML创建Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID), zap.String("namespace", sec.Namespace), zap.String("name", sec.Name))
		return fmt.Errorf("创建Secret %s/%s 失败: %w", sec.Namespace, sec.Name, err)
	}

	s.logger.Info("成功通过YAML创建Secret",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", sec.Namespace),
		zap.String("name", sec.Name))

	return nil
}

func (s *secretService) UpdateSecretByYaml(ctx context.Context, req *model.UpdateSecretByYamlReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}

	s.logger.Info("开始通过YAML更新Secret",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	// 获取现有资源以获取ResourceVersion（用于乐观锁，避免并发冲突），并用于恢复脱敏占位符
	existing, err := s.secretManager.GetSecret(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("Secret不存在: %s/%s", req.Namespace, req.Name)
		}
		s.logger.Error("获取现有Secret失败", zap.Error(err))
		return fmt.Errorf("获取现有Secret失败: %w", err)
	}

	// SEC-25：详情 YAML 已脱敏，仍为占位符的键保持集群中的原值，避免占位符被写回
	restoredYAML, err := k8sutils.RestoreRedactedSecretYAML(req.YAML, existing)
	if err != nil {
		return err
	}

	// 解析YAML
	sec, err := k8sutils.YAMLToSecret(restoredYAML)
	if err != nil {
		s.logger.Error("解析YAML失败", zap.Error(err))
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 强制设置命名空间和名称（以URL参数为准）
	sec.Namespace = req.Namespace
	sec.Name = req.Name

	// 保留ResourceVersion和UID等关键元数据
	sec.ResourceVersion = existing.ResourceVersion
	sec.UID = existing.UID

	_, err = s.secretManager.UpdateSecret(ctx, req.ClusterID, sec)
	if err != nil {
		s.logger.Error("通过YAML更新Secret失败", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("通过YAML更新Secret失败: %w", err)
	}

	s.logger.Info("成功通过YAML更新Secret (完全覆盖)",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

// preserveRedactedData 兼容脱敏后的详情回填：请求中值为空、而集群中该键已有非空值时保留原值。
// 详情接口只返回空值，前端若直接回填提交，不能因此把真实值清空。
// 如需删除某个键，请在请求中去掉该键。
func preserveRedactedData(requested, existing map[string][]byte) map[string][]byte {
	if requested == nil {
		return nil
	}
	out := make(map[string][]byte, len(requested))
	for k, v := range requested {
		if len(v) == 0 {
			if old, ok := existing[k]; ok && len(old) > 0 {
				out[k] = old
				continue
			}
		}
		out[k] = v
	}
	return out
}

// preserveRedactedStringData 与 data 相同按“空值保留原值”处理 stringData。
// K8s 写入时 stringData 会覆盖 data 中的同名键，若把脱敏回填的空字符串原样提交，会把 data 中的真实值清空。
// 因此：stringData 中值为空、而集群中该键已有非空值（data 优先，其次 stringData）时，
// 从 stringData 中移除该键，并在 data 中保留原值（请求 data 对同名键给了非空新值时以请求为准）。
// 集群中不存在或原值为空的键按原样写入。返回处理后的 data 与 stringData。
func preserveRedactedStringData(requested map[string]string, data map[string][]byte, existing *corev1.Secret) (map[string][]byte, map[string]string) {
	if requested == nil {
		return data, nil
	}
	var existingData map[string][]byte
	var existingStringData map[string]string
	if existing != nil {
		existingData, existingStringData = existing.Data, existing.StringData
	}
	out := make(map[string]string, len(requested))
	for k, v := range requested {
		if v == "" {
			old, ok := existingData[k]
			if !ok || len(old) == 0 {
				if s, sok := existingStringData[k]; sok && s != "" {
					old, ok = []byte(s), true
				}
			}
			if ok && len(old) > 0 {
				if data == nil {
					data = map[string][]byte{}
				}
				if cur, set := data[k]; !set || len(cur) == 0 {
					data[k] = old
				}
				continue
			}
		}
		out[k] = v
	}
	return data, out
}

// preserveLastApplied 列表/详情不再返回 last-applied-configuration，更新时若请求未携带则保留集群中的原注解
func preserveLastApplied(requested, existing map[string]string) map[string]string {
	old, ok := existing[k8sutils.LastAppliedConfigAnnotation]
	if !ok {
		return requested
	}
	if _, provided := requested[k8sutils.LastAppliedConfigAnnotation]; provided {
		return requested
	}
	out := make(map[string]string, len(requested)+1)
	for k, v := range requested {
		out[k] = v
	}
	out[k8sutils.LastAppliedConfigAnnotation] = old
	return out
}

// RevealSecret 返回 Secret 明文。每次调用都写审计，审计只含操作者、集群、命名空间、名称与键名，绝不含值。
// 成功读取后若审计写入失败，则不返回明文（fail closed）。
func (s *secretService) RevealSecret(ctx context.Context, req *model.RevealSecretReq, actor model.SecretRevealActor) (*model.K8sSecretReveal, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}
	start := time.Now()

	secret, err := s.secretManager.GetSecret(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		var userErr error
		if errors.IsNotFound(err) {
			userErr = fmt.Errorf("Secret不存在: %s/%s", req.Namespace, req.Name)
		} else {
			s.logger.Error("获取Secret失败", zap.Error(err),
				zap.Int("cluster_id", req.ClusterID),
				zap.String("namespace", req.Namespace),
				zap.String("name", req.Name))
			userErr = fmt.Errorf("获取Secret失败: %w", err)
		}
		_ = s.writeRevealAudit(ctx, req, actor, normalizeKeys(req.Keys), start, k8sErrorStatusCode(err), userErr)
		return nil, userErr
	}

	result := &model.K8sSecretReveal{
		ClusterID: req.ClusterID,
		Namespace: secret.Namespace,
		Name:      secret.Name,
		Type:      model.K8sSecretType(secret.Type),
	}

	wanted := normalizeKeys(req.Keys)
	if len(wanted) == 0 {
		result.Data = model.BinaryDataMap(secret.Data)
		result.StringData = secret.StringData
	} else {
		result.Data = model.BinaryDataMap{}
		for _, k := range wanted {
			found := false
			if v, ok := secret.Data[k]; ok {
				result.Data[k] = v
				found = true
			}
			// data 与 stringData 存在同名键时两者都返回，与不指定 keys 时的输出保持一致
			if v, ok := secret.StringData[k]; ok {
				if result.StringData == nil {
					result.StringData = map[string]string{}
				}
				result.StringData[k] = v
				found = true
			}
			if found {
				continue
			}
			keyErr := fmt.Errorf("Secret %s/%s 中不存在键 %s", req.Namespace, req.Name, k)
			_ = s.writeRevealAudit(ctx, req, actor, wanted, start, http.StatusBadRequest, keyErr)
			return nil, keyErr
		}
	}

	keySet := make(map[string]struct{}, len(result.Data)+len(result.StringData))
	for k := range result.Data {
		keySet[k] = struct{}{}
	}
	for k := range result.StringData {
		keySet[k] = struct{}{}
	}
	result.Keys = make([]string, 0, len(keySet))
	for k := range keySet {
		result.Keys = append(result.Keys, k)
	}
	sort.Strings(result.Keys)

	if err := s.writeRevealAudit(ctx, req, actor, result.Keys, start, http.StatusOK, nil); err != nil {
		s.logger.Error("写入Secret明文查看审计失败，拒绝返回明文", zap.Error(err),
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("%w，已拒绝返回Secret明文", ErrAuditWriteFailed)
	}

	s.logger.Info("Secret明文已查看",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name),
		zap.Int("user_id", actor.UserID),
		zap.Strings("keys", result.Keys))

	return result, nil
}

// normalizeKeys 去重、去空并排序请求中的键，同时支持逗号分隔
func normalizeKeys(keys []string) []string {
	seen := make(map[string]struct{}, len(keys))
	out := make([]string, 0, len(keys))
	for _, raw := range keys {
		for _, k := range strings.Split(raw, ",") {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// ErrAuditWriteFailed 敏感读取接口（Secret reveal 等）同步审计写入失败。
// handler 据此返回 HTTP 500，且不返回任何明文。
var ErrAuditWriteFailed = stderrors.New("写入审计失败")

// k8sErrorStatusCode 将读取 K8s 资源的错误映射为审计记录中的 HTTP 状态码：
// 优先使用 K8s API 返回的状态码（404/403 等），无法识别时记为 500。
func k8sErrorStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}
	var status errors.APIStatus
	if stderrors.As(err, &status) {
		if code := int(status.Status().Code); code >= 400 && code <= 599 {
			return code
		}
	}
	if errors.IsNotFound(err) {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

// writeRevealAudit 同步写入 reveal 审计；内容只含元数据与键名。statusCode 为本次操作实际的结果码（成功 200，失败为 4xx/5xx）
func (s *secretService) writeRevealAudit(ctx context.Context, req *model.RevealSecretReq, actor model.SecretRevealActor, keys []string, start time.Time, statusCode int, opErr error) error {
	if s.auditService == nil {
		return fmt.Errorf("审计服务未初始化")
	}
	if keys == nil {
		keys = []string{}
	}
	body, err := json.Marshal(model.SecretRevealAuditRecord{
		Action:     model.SecretRevealAuditAction,
		UserID:     actor.UserID,
		Username:   actor.Username,
		ClusterID:  req.ClusterID,
		Namespace:  req.Namespace,
		Name:       req.Name,
		Keys:       keys,
		Success:    opErr == nil,
		RevealedAt: start.UTC().Format(time.RFC3339),
	})
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
		HttpMethod:    "GET",
		Endpoint:      actor.Endpoint,
		OperationType: model.SecretRevealAuditOperation,
		TargetType:    model.SecretRevealAuditTargetType,
		TargetID:      fmt.Sprintf("%d/%s/%s", req.ClusterID, req.Namespace, req.Name),
		StatusCode:    statusCode,
		RequestBody:   body,
		Duration:      time.Since(start).Microseconds(),
		ErrorMsg:      errMsg,
	})
}

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
	"strconv"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// TolerationService Kubernetes容忍度管理服务接口
// 提供Kubernetes容忍度的完整管理功能，包括：
// - 基本的增删改查操作，支持Pod、Deployment、StatefulSet、DaemonSet等资源类型
// - 容忍度配置验证和节点兼容性分析
// - 容忍度时间参数配置和验证
// - 批量操作支持，提高多资源管理效率
// - 容忍度模板管理，支持配置复用
type TolerationService interface {
	// AddTolerations 为指定的K8s资源添加容忍度配置
	// 支持为Pod、Deployment、StatefulSet、DaemonSet等资源添加新的容忍度，不会覆盖现有配置
	AddTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) (*model.K8sTaintTolerationResponse, error)
	
	// UpdateTolerations 更新指定K8s资源的容忍度配置
	// 完全替换现有的容忍度配置，而不是增量更新
	UpdateTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) (*model.K8sTaintTolerationResponse, error)
	
	// DeleteTolerations 从指定K8s资源中删除特定的容忍度配置
	// 根据请求中的容忍度列表，精确匹配并删除对应的配置项
	DeleteTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) error
	
	// ValidateTolerations 验证容忍度配置的有效性和节点兼容性
	// 检查容忍度配置是否符合K8s规范，并分析与集群节点的兼容性
	ValidateTolerations(ctx context.Context, req *model.K8sTaintTolerationValidationRequest) (*model.K8sTaintTolerationValidationResponse, error)
	
	// ListTolerations 获取指定K8s资源的当前容忍度配置列表
	// 查询并返回目标资源当前配置的所有容忍度信息
	ListTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) (*model.K8sTaintTolerationResponse, error)
	
	// ConfigTolerationTime 配置容忍度的时间参数
	// 设置容忍度的超时时间、默认时间和条件化超时配置，支持全局和资源级别的配置
	ConfigTolerationTime(ctx context.Context, req *model.K8sTolerationTimeRequest) (*model.K8sTolerationTimeResponse, error)
	
	// ValidateTolerationTime 验证容忍度时间配置的有效性
	// 检查时间配置参数是否合理，包括最大值、最小值和条件化超时的有效性
	ValidateTolerationTime(ctx context.Context, req *model.K8sTolerationTimeRequest) (*model.K8sTolerationTimeResponse, error)
	
	// BatchUpdateTolerations 批量更新多个资源的容忍度配置
	// 同时对指定命名空间下的所有同类型资源进行容忍度更新，支持并发处理提高效率
	BatchUpdateTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) (*model.K8sTaintTolerationResponse, error)
	
	// CreateTolerationTemplate 创建容忍度模板
	// 保存常用的容忍度配置为模板，方便后续快速应用到多个资源
	CreateTolerationTemplate(ctx context.Context, req *model.K8sTolerationConfigRequest) (*model.K8sTolerationTemplate, error)
	
	// GetTolerationTemplate 根据名称获取容忍度模板
	// 查询并返回指定名称的容忍度模板配置信息
	GetTolerationTemplate(ctx context.Context, req *model.K8sTolerationConfigRequest) (*model.K8sTolerationTemplate, error)
	
	// DeleteTolerationTemplate 根据名称删除容忍度模板
	// 永久删除指定名称的容忍度模板，不影响已经应用了该模板的资源
	DeleteTolerationTemplate(ctx context.Context, req *model.K8sTolerationConfigRequest) error
}

// tolerationService 容忍度管理服务的具体实现
// 实现了TolerationService接口的所有方法，提供完整的Kubernetes容忍度管理功能
type tolerationService struct {
	dao    admin.ClusterDAO    // 集群数据访问对象，用于集群相关的数据库操作
	client client.K8sClient   // Kubernetes客户端接口，提供与K8s集群的连接能力
	logger *zap.Logger        // 结构化日志记录器，用于记录服务操作日志
	config *config.K8sConfig  // K8s配置对象，包含污点和容忍度的默认配置参数
}

// NewTolerationService 创建新的容忍度管理服务实例
// 初始化所有必要的依赖项并返回服务接口实现
// 参数:
//   - dao: 集群数据访问对象，用于数据库操作
//   - client: K8s客户端接口，用于与Kubernetes集群通信
//   - logger: 日志记录器，用于记录服务操作和错误信息
// 返回: TolerationService接口实现
func NewTolerationService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) TolerationService {
	return &tolerationService{
		dao:    dao,
		client: client,
		logger: logger,
		config: config.GetK8sConfig(), // 获取全局K8s配置，包含污点默认值等配置
	}
}

// ===============================
// 容忍度管理服务的核心业务方法实现
// ===============================

// AddTolerations 为指定的K8s资源添加容忍度配置
// 这是一个增量操作，新的容忍度配置将添加到现有配置中，不会覆盖原有配置
// 支持Pod、Deployment、StatefulSet、DaemonSet等主要工作负载资源类型
// 
// 实现逻辑:
// 1. 获取目标集群的Kubernetes客户端连接
// 2. 根据资源类型调用对应的添加方法
// 3. 检查并去重，避免添加重复的容忍度配置
// 4. 查找与新容忍度配置兼容的节点列表
// 5. 返回操作结果和兼容节点信息
//
// 参数:
//   - ctx: 上下文对象，用于控制请求生命周期和取消操作
//   - req: 包含集群ID、资源信息和要添加的容忍度配置的请求对象
// 返回:
//   - *model.K8sTaintTolerationResponse: 包含操作结果和兼容节点信息的响应对象
//   - error: 操作过程中发生的错误信息
func (t *tolerationService) AddTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) (*model.K8sTaintTolerationResponse, error) {
	// 根据集群ID获取对应的Kubernetes客户端实例
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, t.client, t.logger)
	if err != nil {
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	// 构建响应对象，包含请求的基本信息和当前时间戳
	response := &model.K8sTaintTolerationResponse{
		ResourceType:      req.ResourceType,
		ResourceName:      req.ResourceName,
		Namespace:         req.Namespace,
		Tolerations:       req.Tolerations,
		CreationTimestamp: time.Now(),
	}

	// 根据不同的资源类型执行相应的添加操作
	switch req.ResourceType {
	case "Pod":
		err = t.addPodTolerations(ctx, kubeClient, req)
	case "Deployment":
		err = t.addDeploymentTolerations(ctx, kubeClient, req)
	case "StatefulSet":
		err = t.addStatefulSetTolerations(ctx, kubeClient, req)
	case "DaemonSet":
		err = t.addDaemonSetTolerations(ctx, kubeClient, req)
	default:
		return nil, fmt.Errorf("不支持的资源类型: %s", req.ResourceType)
	}

	if err != nil {
		return nil, err
	}

	// 查找与新容忍度配置兼容的集群节点，用于调度分析
	response.CompatibleNodes, err = t.findCompatibleNodes(ctx, kubeClient, req.Tolerations)
	if err != nil {
		// 兼容节点查找失败不应该阻止主要操作，只记录警告日志
		t.logger.Warn("查找兼容节点失败", zap.Error(err))
	}

	return response, nil
}

func (t *tolerationService) UpdateTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) (*model.K8sTaintTolerationResponse, error) {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, t.client, t.logger)
	if err != nil {
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	response := &model.K8sTaintTolerationResponse{
		ResourceType:      req.ResourceType,
		ResourceName:      req.ResourceName,
		Namespace:         req.Namespace,
		Tolerations:       req.Tolerations,
		CreationTimestamp: time.Now(),
	}

	switch req.ResourceType {
	case "Pod":
		err = t.updatePodTolerations(ctx, kubeClient, req)
	case "Deployment":
		err = t.updateDeploymentTolerations(ctx, kubeClient, req)
	case "StatefulSet":
		err = t.updateStatefulSetTolerations(ctx, kubeClient, req)
	case "DaemonSet":
		err = t.updateDaemonSetTolerations(ctx, kubeClient, req)
	default:
		return nil, fmt.Errorf("不支持的资源类型: %s", req.ResourceType)
	}

	if err != nil {
		return nil, err
	}

	response.CompatibleNodes, err = t.findCompatibleNodes(ctx, kubeClient, req.Tolerations)
	if err != nil {
		t.logger.Warn("查找兼容节点失败", zap.Error(err))
	}

	return response, nil
}

func (t *tolerationService) DeleteTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, t.client, t.logger)
	if err != nil {
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	switch req.ResourceType {
	case "Pod":
		return t.deletePodTolerations(ctx, kubeClient, req)
	case "Deployment":
		return t.deleteDeploymentTolerations(ctx, kubeClient, req)
	case "StatefulSet":
		return t.deleteStatefulSetTolerations(ctx, kubeClient, req)
	case "DaemonSet":
		return t.deleteDaemonSetTolerations(ctx, kubeClient, req)
	default:
		return fmt.Errorf("不支持的资源类型: %s", req.ResourceType)
	}
}

func (t *tolerationService) ValidateTolerations(ctx context.Context, req *model.K8sTaintTolerationValidationRequest) (*model.K8sTaintTolerationValidationResponse, error) {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, t.client, t.logger)
	if err != nil {
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	response := &model.K8sTaintTolerationValidationResponse{
		ValidationTime: time.Now(),
	}

	var validationErrors []string

	for _, toleration := range req.Tolerations {
		if err := t.validateToleration(toleration); err != nil {
			validationErrors = append(validationErrors, err.Error())
		}
	}

	if len(validationErrors) > 0 {
		response.Valid = false
		response.ValidationErrors = validationErrors
	} else {
		response.Valid = true
	}

	compatibleNodes, incompatibleNodes, err := t.checkNodeCompatibility(ctx, kubeClient, req.Tolerations)
	if err != nil {
		return nil, err
	}

	response.CompatibleNodes = compatibleNodes
	response.IncompatibleNodes = incompatibleNodes

	if req.SimulateScheduling {
		response.SchedulingResult = t.simulateScheduling(compatibleNodes, incompatibleNodes)
	}

	response.Suggestions = t.generateSuggestions(compatibleNodes, incompatibleNodes, req.Tolerations)

	return response, nil
}

func (t *tolerationService) ListTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) (*model.K8sTaintTolerationResponse, error) {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, t.client, t.logger)
	if err != nil {
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	response := &model.K8sTaintTolerationResponse{
		ResourceType:      req.ResourceType,
		ResourceName:      req.ResourceName,
		Namespace:         req.Namespace,
		CreationTimestamp: time.Now(),
	}

	switch req.ResourceType {
	case "Pod":
		tolerations, err := t.getPodTolerations(ctx, kubeClient, req)
		if err != nil {
			return nil, err
		}
		response.Tolerations = tolerations
	case "Deployment":
		tolerations, err := t.getDeploymentTolerations(ctx, kubeClient, req)
		if err != nil {
			return nil, err
		}
		response.Tolerations = tolerations
	case "StatefulSet":
		tolerations, err := t.getStatefulSetTolerations(ctx, kubeClient, req)
		if err != nil {
			return nil, err
		}
		response.Tolerations = tolerations
	case "DaemonSet":
		tolerations, err := t.getDaemonSetTolerations(ctx, kubeClient, req)
		if err != nil {
			return nil, err
		}
		response.Tolerations = tolerations
	default:
		return nil, fmt.Errorf("不支持的资源类型: %s", req.ResourceType)
	}

	return response, nil
}

func (t *tolerationService) ConfigTolerationTime(ctx context.Context, req *model.K8sTolerationTimeRequest) (*model.K8sTolerationTimeResponse, error) {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, t.client, t.logger)
	if err != nil {
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	response := &model.K8sTolerationTimeResponse{
		ResourceType:      req.ResourceType,
		ResourceName:      req.ResourceName,
		Namespace:         req.Namespace,
		CreationTimestamp: time.Now(),
		Status:            "configured",
	}

	if req.GlobalSettings {
		err = t.applyGlobalTolerationTimeSettings(ctx, kubeClient, req)
	} else {
		err = t.applyResourceTolerationTimeSettings(ctx, kubeClient, req)
	}

	if err != nil {
		response.Status = "failed"
		return response, err
	}

	response.AppliedTimeouts = t.buildAppliedTimeouts(req.TimeConfig)
	response.ValidationResults = t.validateTimeConfiguration(req.TimeConfig)

	return response, nil
}

func (t *tolerationService) ValidateTolerationTime(ctx context.Context, req *model.K8sTolerationTimeRequest) (*model.K8sTolerationTimeResponse, error) {
	response := &model.K8sTolerationTimeResponse{
		ResourceType:      req.ResourceType,
		ResourceName:      req.ResourceName,
		Namespace:         req.Namespace,
		CreationTimestamp: time.Now(),
		Status:            "validated",
	}

	response.ValidationResults = t.validateTimeConfiguration(req.TimeConfig)

	hasErrors := false
	for _, result := range response.ValidationResults {
		if !result.IsValid {
			hasErrors = true
			break
		}
	}

	if hasErrors {
		response.Status = "validation_failed"
	}

	return response, nil
}

func (t *tolerationService) BatchUpdateTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) (*model.K8sTaintTolerationResponse, error) {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, t.client, t.logger)
	if err != nil {
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	response := &model.K8sTaintTolerationResponse{
		ResourceType:      req.ResourceType,
		Namespace:         req.Namespace,
		Tolerations:       req.Tolerations,
		CreationTimestamp: time.Now(),
	}

	g, ctx := errgroup.WithContext(ctx)

	var resources []string
	switch req.ResourceType {
	case "Deployment":
		deployments, err := kubeClient.AppsV1().Deployments(req.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, deployment := range deployments.Items {
			resources = append(resources, deployment.Name)
		}
	case "StatefulSet":
		statefulSets, err := kubeClient.AppsV1().StatefulSets(req.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, statefulSet := range statefulSets.Items {
			resources = append(resources, statefulSet.Name)
		}
	case "DaemonSet":
		daemonSets, err := kubeClient.AppsV1().DaemonSets(req.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, daemonSet := range daemonSets.Items {
			resources = append(resources, daemonSet.Name)
		}
	default:
		return nil, fmt.Errorf("批量更新不支持的资源类型: %s", req.ResourceType)
	}

	for _, resourceName := range resources {
		resourceName := resourceName
		g.Go(func() error {
			resourceReq := *req
			resourceReq.ResourceName = resourceName
			_, err := t.UpdateTolerations(ctx, &resourceReq)
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("批量更新容忍度失败: %w", err)
	}

	return response, nil
}

func (t *tolerationService) CreateTolerationTemplate(ctx context.Context, req *model.K8sTolerationConfigRequest) (*model.K8sTolerationTemplate, error) {
	template := &req.TolerationTemplate
	template.Tags = make(map[string]string)
	template.Tags["created_at"] = time.Now().Format(time.RFC3339)
	template.Tags["cluster_id"] = strconv.Itoa(req.ClusterID)

	if req.ApplyToExisting {
		kubeClient, err := pkg.GetKubeClient(req.ClusterID, t.client, t.logger)
		if err != nil {
			return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
		}

		err = t.applyTemplateToExistingResources(ctx, kubeClient, req)
		if err != nil {
			return nil, err
		}
	}

	return template, nil
}

func (t *tolerationService) GetTolerationTemplate(ctx context.Context, req *model.K8sTolerationConfigRequest) (*model.K8sTolerationTemplate, error) {
	template := &req.TolerationTemplate
	return template, nil
}

func (t *tolerationService) DeleteTolerationTemplate(ctx context.Context, req *model.K8sTolerationConfigRequest) error {
	return nil
}

// Helper methods for toleration operations

func (t *tolerationService) addPodTolerations(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintTolerationRequest) error {
	pod, err := kubeClient.CoreV1().Pods(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取Pod失败: %w", err)
	}

	for _, newToleration := range req.Tolerations {
		tolerationExists := false
		for _, existingToleration := range pod.Spec.Tolerations {
			if t.tolerationsEqual(existingToleration, newToleration) {
				tolerationExists = true
				break
			}
		}
		if !tolerationExists {
			pod.Spec.Tolerations = append(pod.Spec.Tolerations, corev1.Toleration{
				Key:               newToleration.Key,
				Operator:          corev1.TolerationOperator(newToleration.Operator),
				Value:             newToleration.Value,
				Effect:            corev1.TaintEffect(newToleration.Effect),
				TolerationSeconds: newToleration.TolerationSeconds,
			})
		}
	}

	_, err = kubeClient.CoreV1().Pods(req.Namespace).Update(ctx, pod, metav1.UpdateOptions{})
	return err
}

func (t *tolerationService) addDeploymentTolerations(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintTolerationRequest) error {
	deployment, err := kubeClient.AppsV1().Deployments(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取Deployment失败: %w", err)
	}

	for _, newToleration := range req.Tolerations {
		tolerationExists := false
		for _, existingToleration := range deployment.Spec.Template.Spec.Tolerations {
			if t.tolerationsEqual(existingToleration, newToleration) {
				tolerationExists = true
				break
			}
		}
		if !tolerationExists {
			deployment.Spec.Template.Spec.Tolerations = append(deployment.Spec.Template.Spec.Tolerations, corev1.Toleration{
				Key:               newToleration.Key,
				Operator:          corev1.TolerationOperator(newToleration.Operator),
				Value:             newToleration.Value,
				Effect:            corev1.TaintEffect(newToleration.Effect),
				TolerationSeconds: newToleration.TolerationSeconds,
			})
		}
	}

	_, err = kubeClient.AppsV1().Deployments(req.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

func (t *tolerationService) addStatefulSetTolerations(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintTolerationRequest) error {
	statefulSet, err := kubeClient.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取StatefulSet失败: %w", err)
	}

	for _, newToleration := range req.Tolerations {
		tolerationExists := false
		for _, existingToleration := range statefulSet.Spec.Template.Spec.Tolerations {
			if t.tolerationsEqual(existingToleration, newToleration) {
				tolerationExists = true
				break
			}
		}
		if !tolerationExists {
			statefulSet.Spec.Template.Spec.Tolerations = append(statefulSet.Spec.Template.Spec.Tolerations, corev1.Toleration{
				Key:               newToleration.Key,
				Operator:          corev1.TolerationOperator(newToleration.Operator),
				Value:             newToleration.Value,
				Effect:            corev1.TaintEffect(newToleration.Effect),
				TolerationSeconds: newToleration.TolerationSeconds,
			})
		}
	}

	_, err = kubeClient.AppsV1().StatefulSets(req.Namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
	return err
}

func (t *tolerationService) addDaemonSetTolerations(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintTolerationRequest) error {
	daemonSet, err := kubeClient.AppsV1().DaemonSets(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取DaemonSet失败: %w", err)
	}

	for _, newToleration := range req.Tolerations {
		tolerationExists := false
		for _, existingToleration := range daemonSet.Spec.Template.Spec.Tolerations {
			if t.tolerationsEqual(existingToleration, newToleration) {
				tolerationExists = true
				break
			}
		}
		if !tolerationExists {
			daemonSet.Spec.Template.Spec.Tolerations = append(daemonSet.Spec.Template.Spec.Tolerations, corev1.Toleration{
				Key:               newToleration.Key,
				Operator:          corev1.TolerationOperator(newToleration.Operator),
				Value:             newToleration.Value,
				Effect:            corev1.TaintEffect(newToleration.Effect),
				TolerationSeconds: newToleration.TolerationSeconds,
			})
		}
	}

	_, err = kubeClient.AppsV1().DaemonSets(req.Namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
	return err
}

func (t *tolerationService) updatePodTolerations(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintTolerationRequest) error {
	pod, err := kubeClient.CoreV1().Pods(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取Pod失败: %w", err)
	}

	var newTolerations []corev1.Toleration
	for _, newToleration := range req.Tolerations {
		newTolerations = append(newTolerations, corev1.Toleration{
			Key:               newToleration.Key,
			Operator:          corev1.TolerationOperator(newToleration.Operator),
			Value:             newToleration.Value,
			Effect:            corev1.TaintEffect(newToleration.Effect),
			TolerationSeconds: newToleration.TolerationSeconds,
		})
	}

	pod.Spec.Tolerations = newTolerations

	_, err = kubeClient.CoreV1().Pods(req.Namespace).Update(ctx, pod, metav1.UpdateOptions{})
	return err
}

func (t *tolerationService) updateDeploymentTolerations(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintTolerationRequest) error {
	deployment, err := kubeClient.AppsV1().Deployments(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取Deployment失败: %w", err)
	}

	var newTolerations []corev1.Toleration
	for _, newToleration := range req.Tolerations {
		newTolerations = append(newTolerations, corev1.Toleration{
			Key:               newToleration.Key,
			Operator:          corev1.TolerationOperator(newToleration.Operator),
			Value:             newToleration.Value,
			Effect:            corev1.TaintEffect(newToleration.Effect),
			TolerationSeconds: newToleration.TolerationSeconds,
		})
	}

	deployment.Spec.Template.Spec.Tolerations = newTolerations

	_, err = kubeClient.AppsV1().Deployments(req.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

func (t *tolerationService) updateStatefulSetTolerations(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintTolerationRequest) error {
	statefulSet, err := kubeClient.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取StatefulSet失败: %w", err)
	}

	var newTolerations []corev1.Toleration
	for _, newToleration := range req.Tolerations {
		newTolerations = append(newTolerations, corev1.Toleration{
			Key:               newToleration.Key,
			Operator:          corev1.TolerationOperator(newToleration.Operator),
			Value:             newToleration.Value,
			Effect:            corev1.TaintEffect(newToleration.Effect),
			TolerationSeconds: newToleration.TolerationSeconds,
		})
	}

	statefulSet.Spec.Template.Spec.Tolerations = newTolerations

	_, err = kubeClient.AppsV1().StatefulSets(req.Namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
	return err
}

func (t *tolerationService) updateDaemonSetTolerations(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintTolerationRequest) error {
	daemonSet, err := kubeClient.AppsV1().DaemonSets(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取DaemonSet失败: %w", err)
	}

	var newTolerations []corev1.Toleration
	for _, newToleration := range req.Tolerations {
		newTolerations = append(newTolerations, corev1.Toleration{
			Key:               newToleration.Key,
			Operator:          corev1.TolerationOperator(newToleration.Operator),
			Value:             newToleration.Value,
			Effect:            corev1.TaintEffect(newToleration.Effect),
			TolerationSeconds: newToleration.TolerationSeconds,
		})
	}

	daemonSet.Spec.Template.Spec.Tolerations = newTolerations

	_, err = kubeClient.AppsV1().DaemonSets(req.Namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
	return err
}

func (t *tolerationService) deletePodTolerations(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintTolerationRequest) error {
	pod, err := kubeClient.CoreV1().Pods(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取Pod失败: %w", err)
	}

	var filteredTolerations []corev1.Toleration
	for _, existingToleration := range pod.Spec.Tolerations {
		shouldRemove := false
		for _, tolerationToRemove := range req.Tolerations {
			if t.tolerationsEqual(existingToleration, tolerationToRemove) {
				shouldRemove = true
				break
			}
		}
		if !shouldRemove {
			filteredTolerations = append(filteredTolerations, existingToleration)
		}
	}

	pod.Spec.Tolerations = filteredTolerations

	_, err = kubeClient.CoreV1().Pods(req.Namespace).Update(ctx, pod, metav1.UpdateOptions{})
	return err
}

func (t *tolerationService) deleteDeploymentTolerations(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintTolerationRequest) error {
	deployment, err := kubeClient.AppsV1().Deployments(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取Deployment失败: %w", err)
	}

	var filteredTolerations []corev1.Toleration
	for _, existingToleration := range deployment.Spec.Template.Spec.Tolerations {
		shouldRemove := false
		for _, tolerationToRemove := range req.Tolerations {
			if t.tolerationsEqual(existingToleration, tolerationToRemove) {
				shouldRemove = true
				break
			}
		}
		if !shouldRemove {
			filteredTolerations = append(filteredTolerations, existingToleration)
		}
	}

	deployment.Spec.Template.Spec.Tolerations = filteredTolerations

	_, err = kubeClient.AppsV1().Deployments(req.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

func (t *tolerationService) deleteStatefulSetTolerations(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintTolerationRequest) error {
	statefulSet, err := kubeClient.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取StatefulSet失败: %w", err)
	}

	var filteredTolerations []corev1.Toleration
	for _, existingToleration := range statefulSet.Spec.Template.Spec.Tolerations {
		shouldRemove := false
		for _, tolerationToRemove := range req.Tolerations {
			if t.tolerationsEqual(existingToleration, tolerationToRemove) {
				shouldRemove = true
				break
			}
		}
		if !shouldRemove {
			filteredTolerations = append(filteredTolerations, existingToleration)
		}
	}

	statefulSet.Spec.Template.Spec.Tolerations = filteredTolerations

	_, err = kubeClient.AppsV1().StatefulSets(req.Namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
	return err
}

func (t *tolerationService) deleteDaemonSetTolerations(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintTolerationRequest) error {
	daemonSet, err := kubeClient.AppsV1().DaemonSets(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取DaemonSet失败: %w", err)
	}

	var filteredTolerations []corev1.Toleration
	for _, existingToleration := range daemonSet.Spec.Template.Spec.Tolerations {
		shouldRemove := false
		for _, tolerationToRemove := range req.Tolerations {
			if t.tolerationsEqual(existingToleration, tolerationToRemove) {
				shouldRemove = true
				break
			}
		}
		if !shouldRemove {
			filteredTolerations = append(filteredTolerations, existingToleration)
		}
	}

	daemonSet.Spec.Template.Spec.Tolerations = filteredTolerations

	_, err = kubeClient.AppsV1().DaemonSets(req.Namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
	return err
}

func (t *tolerationService) getPodTolerations(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintTolerationRequest) ([]model.K8sToleration, error) {
	pod, err := kubeClient.CoreV1().Pods(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取Pod失败: %w", err)
	}

	var tolerations []model.K8sToleration
	for _, toleration := range pod.Spec.Tolerations {
		tolerations = append(tolerations, model.K8sToleration{
			Key:               toleration.Key,
			Operator:          string(toleration.Operator),
			Value:             toleration.Value,
			Effect:            string(toleration.Effect),
			TolerationSeconds: toleration.TolerationSeconds,
		})
	}

	return tolerations, nil
}

func (t *tolerationService) getDeploymentTolerations(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintTolerationRequest) ([]model.K8sToleration, error) {
	deployment, err := kubeClient.AppsV1().Deployments(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取Deployment失败: %w", err)
	}

	var tolerations []model.K8sToleration
	for _, toleration := range deployment.Spec.Template.Spec.Tolerations {
		tolerations = append(tolerations, model.K8sToleration{
			Key:               toleration.Key,
			Operator:          string(toleration.Operator),
			Value:             toleration.Value,
			Effect:            string(toleration.Effect),
			TolerationSeconds: toleration.TolerationSeconds,
		})
	}

	return tolerations, nil
}

func (t *tolerationService) getStatefulSetTolerations(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintTolerationRequest) ([]model.K8sToleration, error) {
	statefulSet, err := kubeClient.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取StatefulSet失败: %w", err)
	}

	var tolerations []model.K8sToleration
	for _, toleration := range statefulSet.Spec.Template.Spec.Tolerations {
		tolerations = append(tolerations, model.K8sToleration{
			Key:               toleration.Key,
			Operator:          string(toleration.Operator),
			Value:             toleration.Value,
			Effect:            string(toleration.Effect),
			TolerationSeconds: toleration.TolerationSeconds,
		})
	}

	return tolerations, nil
}

func (t *tolerationService) getDaemonSetTolerations(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintTolerationRequest) ([]model.K8sToleration, error) {
	daemonSet, err := kubeClient.AppsV1().DaemonSets(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取DaemonSet失败: %w", err)
	}

	var tolerations []model.K8sToleration
	for _, toleration := range daemonSet.Spec.Template.Spec.Tolerations {
		tolerations = append(tolerations, model.K8sToleration{
			Key:               toleration.Key,
			Operator:          string(toleration.Operator),
			Value:             toleration.Value,
			Effect:            string(toleration.Effect),
			TolerationSeconds: toleration.TolerationSeconds,
		})
	}

	return tolerations, nil
}

func (t *tolerationService) findCompatibleNodes(ctx context.Context, kubeClient kubernetes.Interface, tolerations []model.K8sToleration) ([]string, error) {
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点列表失败: %w", err)
	}

	var compatibleNodes []string
	for _, node := range nodes.Items {
		if t.isNodeCompatible(node, tolerations) {
			compatibleNodes = append(compatibleNodes, node.Name)
		}
	}

	return compatibleNodes, nil
}

func (t *tolerationService) isNodeCompatible(node corev1.Node, tolerations []model.K8sToleration) bool {
	for _, taint := range node.Spec.Taints {
		tolerationFound := false
		for _, toleration := range tolerations {
			if t.tolerationMatchesTaint(toleration, taint) {
				tolerationFound = true
				break
			}
		}
		if !tolerationFound {
			return false
		}
	}
	return true
}

func (t *tolerationService) tolerationMatchesTaint(toleration model.K8sToleration, taint corev1.Taint) bool {
	if toleration.Key != taint.Key {
		return false
	}

	if toleration.Operator == "Equal" && toleration.Value != taint.Value {
		return false
	}

	if toleration.Effect != "" && toleration.Effect != string(taint.Effect) {
		return false
	}

	return true
}

func (t *tolerationService) tolerationsEqual(existing corev1.Toleration, new model.K8sToleration) bool {
	return existing.Key == new.Key &&
		string(existing.Operator) == new.Operator &&
		existing.Value == new.Value &&
		string(existing.Effect) == new.Effect
}

func (t *tolerationService) validateToleration(toleration model.K8sToleration) error {
	if toleration.Key == "" {
		return fmt.Errorf("容忍度键不能为空")
	}

	if toleration.Operator != "Equal" && toleration.Operator != "Exists" {
		return fmt.Errorf("容忍度操作符必须为 Equal 或 Exists")
	}

	if toleration.Operator == "Equal" && toleration.Value == "" {
		return fmt.Errorf("当操作符为 Equal 时，值不能为空")
	}

	if toleration.Effect != "" && toleration.Effect != "NoSchedule" && toleration.Effect != "PreferNoSchedule" && toleration.Effect != "NoExecute" {
		return fmt.Errorf("容忍度效果必须为 NoSchedule, PreferNoSchedule 或 NoExecute")
	}

	if toleration.TolerationSeconds != nil && *toleration.TolerationSeconds < 0 {
		return fmt.Errorf("容忍度时间不能为负数")
	}

	return nil
}

func (t *tolerationService) checkNodeCompatibility(ctx context.Context, kubeClient kubernetes.Interface, tolerations []model.K8sToleration) ([]string, []string, error) {
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("获取节点列表失败: %w", err)
	}

	var compatibleNodes []string
	var incompatibleNodes []string

	for _, node := range nodes.Items {
		if t.isNodeCompatible(node, tolerations) {
			compatibleNodes = append(compatibleNodes, node.Name)
		} else {
			incompatibleNodes = append(incompatibleNodes, node.Name)
		}
	}

	return compatibleNodes, incompatibleNodes, nil
}

func (t *tolerationService) simulateScheduling(compatibleNodes []string, incompatibleNodes []string) string {
	if len(compatibleNodes) == 0 {
		return "调度失败：没有兼容的节点"
	}

	if len(incompatibleNodes) == 0 {
		return "调度成功：所有节点都兼容"
	}

	return fmt.Sprintf("调度成功：%d个兼容节点，%d个不兼容节点", len(compatibleNodes), len(incompatibleNodes))
}

func (t *tolerationService) generateSuggestions(compatibleNodes []string, incompatibleNodes []string, tolerations []model.K8sToleration) []string {
	var suggestions []string

	if len(compatibleNodes) == 0 {
		suggestions = append(suggestions, "建议检查容忍度配置是否正确")
		suggestions = append(suggestions, "建议检查节点污点是否与容忍度匹配")
	}

	if len(incompatibleNodes) > 0 {
		suggestions = append(suggestions, "建议为不兼容的节点添加相应的容忍度")
	}

	for _, toleration := range tolerations {
		if toleration.Effect == "NoExecute" && toleration.TolerationSeconds == nil {
			suggestions = append(suggestions, "建议为 NoExecute 效果的容忍度设置超时时间")
		}
	}

	return suggestions
}

func (t *tolerationService) applyGlobalTolerationTimeSettings(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTolerationTimeRequest) error {
	namespaces, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("获取命名空间列表失败: %w", err)
	}

	for _, namespace := range namespaces.Items {
		err := t.applyNamespaceTolerationTimeSettings(ctx, kubeClient, namespace.Name, req)
		if err != nil {
			t.logger.Warn("应用命名空间容忍度时间设置失败", zap.String("namespace", namespace.Name), zap.Error(err))
		}
	}

	return nil
}

func (t *tolerationService) applyResourceTolerationTimeSettings(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTolerationTimeRequest) error {
	switch req.ResourceType {
	case "Pod":
		return t.applyPodTolerationTimeSettings(ctx, kubeClient, req)
	case "Deployment":
		return t.applyDeploymentTolerationTimeSettings(ctx, kubeClient, req)
	case "StatefulSet":
		return t.applyStatefulSetTolerationTimeSettings(ctx, kubeClient, req)
	case "DaemonSet":
		return t.applyDaemonSetTolerationTimeSettings(ctx, kubeClient, req)
	default:
		return fmt.Errorf("不支持的资源类型: %s", req.ResourceType)
	}
}

func (t *tolerationService) applyNamespaceTolerationTimeSettings(ctx context.Context, kubeClient kubernetes.Interface, namespace string, req *model.K8sTolerationTimeRequest) error {
	deployments, err := kubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, deployment := range deployments.Items {
		t.updateDeploymentTolerationTimes(&deployment, req.TimeConfig)
		_, err := kubeClient.AppsV1().Deployments(namespace).Update(ctx, &deployment, metav1.UpdateOptions{})
		if err != nil {
			t.logger.Warn("更新Deployment容忍度时间失败", zap.String("deployment", deployment.Name), zap.Error(err))
		}
	}

	statefulSets, err := kubeClient.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, statefulSet := range statefulSets.Items {
		t.updateStatefulSetTolerationTimes(&statefulSet, req.TimeConfig)
		_, err := kubeClient.AppsV1().StatefulSets(namespace).Update(ctx, &statefulSet, metav1.UpdateOptions{})
		if err != nil {
			t.logger.Warn("更新StatefulSet容忍度时间失败", zap.String("statefulset", statefulSet.Name), zap.Error(err))
		}
	}

	daemonSets, err := kubeClient.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, daemonSet := range daemonSets.Items {
		t.updateDaemonSetTolerationTimes(&daemonSet, req.TimeConfig)
		_, err := kubeClient.AppsV1().DaemonSets(namespace).Update(ctx, &daemonSet, metav1.UpdateOptions{})
		if err != nil {
			t.logger.Warn("更新DaemonSet容忍度时间失败", zap.String("daemonset", daemonSet.Name), zap.Error(err))
		}
	}

	return nil
}

func (t *tolerationService) applyPodTolerationTimeSettings(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTolerationTimeRequest) error {
	pod, err := kubeClient.CoreV1().Pods(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取Pod失败: %w", err)
	}

	t.updatePodTolerationTimes(pod, req.TimeConfig)

	_, err = kubeClient.CoreV1().Pods(req.Namespace).Update(ctx, pod, metav1.UpdateOptions{})
	return err
}

func (t *tolerationService) applyDeploymentTolerationTimeSettings(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTolerationTimeRequest) error {
	deployment, err := kubeClient.AppsV1().Deployments(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取Deployment失败: %w", err)
	}

	t.updateDeploymentTolerationTimes(deployment, req.TimeConfig)

	_, err = kubeClient.AppsV1().Deployments(req.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

func (t *tolerationService) applyStatefulSetTolerationTimeSettings(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTolerationTimeRequest) error {
	statefulSet, err := kubeClient.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取StatefulSet失败: %w", err)
	}

	t.updateStatefulSetTolerationTimes(statefulSet, req.TimeConfig)

	_, err = kubeClient.AppsV1().StatefulSets(req.Namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
	return err
}

func (t *tolerationService) applyDaemonSetTolerationTimeSettings(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTolerationTimeRequest) error {
	daemonSet, err := kubeClient.AppsV1().DaemonSets(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取DaemonSet失败: %w", err)
	}

	t.updateDaemonSetTolerationTimes(daemonSet, req.TimeConfig)

	_, err = kubeClient.AppsV1().DaemonSets(req.Namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
	return err
}

func (t *tolerationService) updatePodTolerationTimes(pod *corev1.Pod, config model.TolerationTimeConfig) {
	for i := range pod.Spec.Tolerations {
		toleration := &pod.Spec.Tolerations[i]
		if toleration.Effect == corev1.TaintEffectNoExecute {
			toleration.TolerationSeconds = t.calculateTolerationTime(config, string(toleration.Key))
		}
	}
}

func (t *tolerationService) updateDeploymentTolerationTimes(deployment *appsv1.Deployment, config model.TolerationTimeConfig) {
	for i := range deployment.Spec.Template.Spec.Tolerations {
		toleration := &deployment.Spec.Template.Spec.Tolerations[i]
		if toleration.Effect == corev1.TaintEffectNoExecute {
			toleration.TolerationSeconds = t.calculateTolerationTime(config, string(toleration.Key))
		}
	}
}

func (t *tolerationService) updateStatefulSetTolerationTimes(statefulSet *appsv1.StatefulSet, config model.TolerationTimeConfig) {
	for i := range statefulSet.Spec.Template.Spec.Tolerations {
		toleration := &statefulSet.Spec.Template.Spec.Tolerations[i]
		if toleration.Effect == corev1.TaintEffectNoExecute {
			toleration.TolerationSeconds = t.calculateTolerationTime(config, string(toleration.Key))
		}
	}
}

func (t *tolerationService) updateDaemonSetTolerationTimes(daemonSet *appsv1.DaemonSet, config model.TolerationTimeConfig) {
	for i := range daemonSet.Spec.Template.Spec.Tolerations {
		toleration := &daemonSet.Spec.Template.Spec.Tolerations[i]
		if toleration.Effect == corev1.TaintEffectNoExecute {
			toleration.TolerationSeconds = t.calculateTolerationTime(config, string(toleration.Key))
		}
	}
}

func (t *tolerationService) calculateTolerationTime(config model.TolerationTimeConfig, taintKey string) *int64 {
	for _, conditional := range config.ConditionalTimeouts {
		if strings.Contains(conditional.Condition, taintKey) {
			return conditional.TimeoutValue
		}
	}

	baseTime := config.DefaultTolerationTime
	if baseTime == nil {
		defaultTime := t.config.TaintDefaults.DefaultTolerationTime
		baseTime = &defaultTime
	}

	switch config.TimeScalingPolicy.PolicyType {
	case "linear":
		scaledTime := int64(float64(*baseTime) * config.TimeScalingPolicy.ScalingFactor)
		if config.TimeScalingPolicy.MaxScaledTime != nil && scaledTime > *config.TimeScalingPolicy.MaxScaledTime {
			scaledTime = *config.TimeScalingPolicy.MaxScaledTime
		}
		return &scaledTime
	case "exponential":
		scaledTime := int64(float64(*baseTime) * config.TimeScalingPolicy.ScalingFactor * config.TimeScalingPolicy.ScalingFactor)
		if config.TimeScalingPolicy.MaxScaledTime != nil && scaledTime > *config.TimeScalingPolicy.MaxScaledTime {
			scaledTime = *config.TimeScalingPolicy.MaxScaledTime
		}
		return &scaledTime
	default:
		return baseTime
	}
}

func (t *tolerationService) buildAppliedTimeouts(config model.TolerationTimeConfig) []model.AppliedTimeout {
	var appliedTimeouts []model.AppliedTimeout

	for _, conditional := range config.ConditionalTimeouts {
		appliedTimeouts = append(appliedTimeouts, model.AppliedTimeout{
			TaintKey:         conditional.Condition,
			TimeoutValue:     conditional.TimeoutValue,
			AppliedCondition: conditional.Condition,
			Source:           "conditional",
		})
	}

	if config.DefaultTolerationTime != nil {
		appliedTimeouts = append(appliedTimeouts, model.AppliedTimeout{
			TaintKey:         "default",
			TimeoutValue:     config.DefaultTolerationTime,
			AppliedCondition: "default",
			Source:           "default",
		})
	}

	return appliedTimeouts
}

func (t *tolerationService) validateTimeConfiguration(config model.TolerationTimeConfig) []model.TimeValidationResult {
	var results []model.TimeValidationResult

	if config.DefaultTolerationTime != nil {
		result := model.TimeValidationResult{
			TaintKey:       "default",
			IsValid:        *config.DefaultTolerationTime >= 0,
			ValidationTime: time.Now(),
		}
		if !result.IsValid {
			result.ValidationMessage = "默认容忍时间不能为负数"
		} else {
			result.ValidationMessage = "默认容忍时间配置有效"
		}
		results = append(results, result)
	}

	if config.MaxTolerationTime != nil && config.MinTolerationTime != nil {
		if *config.MaxTolerationTime < *config.MinTolerationTime {
			results = append(results, model.TimeValidationResult{
				TaintKey:          "time_range",
				IsValid:           false,
				ValidationMessage: "最大容忍时间不能小于最小容忍时间",
				ValidationTime:    time.Now(),
			})
		}
	}

	for _, conditional := range config.ConditionalTimeouts {
		result := model.TimeValidationResult{
			TaintKey:       conditional.Condition,
			IsValid:        conditional.TimeoutValue != nil && *conditional.TimeoutValue >= 0,
			ValidationTime: time.Now(),
		}
		if !result.IsValid {
			result.ValidationMessage = "条件超时时间不能为负数"
		} else {
			result.ValidationMessage = "条件超时时间配置有效"
		}
		results = append(results, result)
	}

	return results
}

func (t *tolerationService) applyTemplateToExistingResources(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTolerationConfigRequest) error {
	if req.Namespace != "" {
		return t.applyTemplateToNamespace(ctx, kubeClient, req.Namespace, req)
	}

	if req.ResourceType != "" && req.ResourceName != "" {
		return t.applyTemplateToResource(ctx, kubeClient, req)
	}

	return nil
}

func (t *tolerationService) applyTemplateToNamespace(ctx context.Context, kubeClient kubernetes.Interface, namespace string, req *model.K8sTolerationConfigRequest) error {
	deployments, err := kubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, deployment := range deployments.Items {
		t.applyTemplateToDeployment(&deployment, req.TolerationTemplate)
		_, err := kubeClient.AppsV1().Deployments(namespace).Update(ctx, &deployment, metav1.UpdateOptions{})
		if err != nil {
			t.logger.Warn("应用模板到Deployment失败", zap.String("deployment", deployment.Name), zap.Error(err))
		}
	}

	return nil
}

func (t *tolerationService) applyTemplateToResource(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTolerationConfigRequest) error {
	switch req.ResourceType {
	case "Deployment":
		deployment, err := kubeClient.AppsV1().Deployments(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		t.applyTemplateToDeployment(deployment, req.TolerationTemplate)
		_, err = kubeClient.AppsV1().Deployments(req.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
		return err
	case "StatefulSet":
		statefulSet, err := kubeClient.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		t.applyTemplateToStatefulSet(statefulSet, req.TolerationTemplate)
		_, err = kubeClient.AppsV1().StatefulSets(req.Namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
		return err
	case "DaemonSet":
		daemonSet, err := kubeClient.AppsV1().DaemonSets(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		t.applyTemplateToDaemonSet(daemonSet, req.TolerationTemplate)
		_, err = kubeClient.AppsV1().DaemonSets(req.Namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
		return err
	default:
		return fmt.Errorf("不支持的资源类型: %s", req.ResourceType)
	}
}

func (t *tolerationService) applyTemplateToDeployment(deployment *appsv1.Deployment, template model.K8sTolerationTemplate) {
	var newTolerations []corev1.Toleration
	for _, tolerationSpec := range template.Tolerations {
		newTolerations = append(newTolerations, corev1.Toleration{
			Key:               tolerationSpec.Key,
			Operator:          corev1.TolerationOperator(tolerationSpec.Operator),
			Value:             tolerationSpec.Value,
			Effect:            corev1.TaintEffect(tolerationSpec.Effect),
			TolerationSeconds: tolerationSpec.TolerationSeconds,
		})
	}
	deployment.Spec.Template.Spec.Tolerations = newTolerations
}

func (t *tolerationService) applyTemplateToStatefulSet(statefulSet *appsv1.StatefulSet, template model.K8sTolerationTemplate) {
	var newTolerations []corev1.Toleration
	for _, tolerationSpec := range template.Tolerations {
		newTolerations = append(newTolerations, corev1.Toleration{
			Key:               tolerationSpec.Key,
			Operator:          corev1.TolerationOperator(tolerationSpec.Operator),
			Value:             tolerationSpec.Value,
			Effect:            corev1.TaintEffect(tolerationSpec.Effect),
			TolerationSeconds: tolerationSpec.TolerationSeconds,
		})
	}
	statefulSet.Spec.Template.Spec.Tolerations = newTolerations
}

func (t *tolerationService) applyTemplateToDaemonSet(daemonSet *appsv1.DaemonSet, template model.K8sTolerationTemplate) {
	var newTolerations []corev1.Toleration
	for _, tolerationSpec := range template.Tolerations {
		newTolerations = append(newTolerations, corev1.Toleration{
			Key:               tolerationSpec.Key,
			Operator:          corev1.TolerationOperator(tolerationSpec.Operator),
			Value:             tolerationSpec.Value,
			Effect:            corev1.TaintEffect(tolerationSpec.Effect),
			TolerationSeconds: tolerationSpec.TolerationSeconds,
		})
	}
	daemonSet.Spec.Template.Spec.Tolerations = newTolerations
}
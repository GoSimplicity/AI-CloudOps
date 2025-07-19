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

// Package admin 提供Kubernetes污点容忍管理服务
// 该包实现了污点容忍的完整管理功能，包括容忍度的应用、验证、节点污点管理等
// 支持多种资源类型（Pod、Deployment、StatefulSet、DaemonSet）的容忍度配置
package admin

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/validator"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// TaintTolerationService 污点容忍服务
// 提供Kubernetes污点容忍的完整管理功能，包括：
// - 容忍度的应用和验证（支持多种资源类型）
// - 节点污点的管理和操作
// - 容忍度时间参数的配置
// - 节点兼容性分析和调度模拟
// - 批量操作和模板管理
type TaintTolerationService struct {
	k8sClient client.K8sClient               // Kubernetes客户端接口，用于与K8s集群通信
	logger    *zap.Logger                    // 结构化日志记录器，用于记录服务操作日志
	validator *validator.TolerationValidator // 容忍度验证器，用于验证配置的有效性
	config    *config.K8sConfig              // K8s配置对象，包含污点和容忍度的默认配置
}

// NewTaintTolerationService 创建污点容忍服务实例
// 初始化所有必要的依赖项并返回服务实例
// 参数:
//   - k8sClient: K8s客户端接口，用于与Kubernetes集群通信
//   - logger: 日志记录器，用于记录服务操作和错误信息
//
// 返回: 配置好的TaintTolerationService实例
func NewTaintTolerationService(k8sClient client.K8sClient, logger *zap.Logger) *TaintTolerationService {
	return &TaintTolerationService{
		k8sClient: k8sClient,
		logger:    logger,
		validator: validator.NewTolerationValidator(), // 创建容忍度验证器
		config:    config.GetK8sConfig(),              // 获取全局K8s配置
	}
}

// ApplyTolerations 应用容忍度到资源
// 为指定的Kubernetes资源（Pod、Deployment、StatefulSet、DaemonSet）应用容忍度配置
// 主要功能：
// - 验证容忍度配置的有效性
// - 根据资源类型应用容忍度
// - 查找兼容的节点列表
// - 返回操作结果和兼容性信息
//
// 实现逻辑:
// 1. 验证请求参数和容忍度配置
// 2. 获取目标集群的Kubernetes客户端连接
// 3. 处理容忍度配置（应用默认值、优化等）
// 4. 根据资源类型调用对应的应用方法
// 5. 查找与容忍度配置兼容的节点
// 6. 返回操作结果和兼容节点信息
//
// 参数:
//   - ctx: 上下文对象，用于控制请求生命周期和取消操作
//   - req: 包含集群ID、资源信息和容忍度配置的请求对象
//
// 返回:
//   - *model.K8sTaintTolerationResponse: 包含操作结果和兼容节点信息的响应对象
//   - error: 操作过程中发生的错误信息
func (s *TaintTolerationService) ApplyTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) (*model.K8sTaintTolerationResponse, error) {
	s.logger.Info("开始应用容忍度", zap.Int("cluster_id", req.ClusterID), zap.String("resource", req.ResourceName))

	// 验证请求
	validationResult := s.validator.ValidateTolerationsRequest(req)
	if !validationResult.Valid {
		return nil, fmt.Errorf("验证失败: %v", validationResult.Errors)
	}

	// 获取Kubernetes客户端
	kubeClient, err := utils.GetKubeClient(req.ClusterID, s.k8sClient, s.logger)
	if err != nil {
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 处理容忍度
	processedTolerations := s.processTolerations(req.Tolerations)

	// 根据资源类型应用容忍度
	switch req.ResourceType {
	case "Deployment":
		err = s.applyTolerationsToDeployment(ctx, kubeClient, req.Namespace, req.ResourceName, processedTolerations)
	case "StatefulSet":
		err = s.applyTolerationsToStatefulSet(ctx, kubeClient, req.Namespace, req.ResourceName, processedTolerations)
	case "DaemonSet":
		err = s.applyTolerationsToDaemonSet(ctx, kubeClient, req.Namespace, req.ResourceName, processedTolerations)
	case "Pod":
		err = s.applyTolerationsToPod(ctx, kubeClient, req.Namespace, req.ResourceName, processedTolerations)
	default:
		return nil, fmt.Errorf("不支持的资源类型: %s", req.ResourceType)
	}

	if err != nil {
		return nil, fmt.Errorf("应用容忍度失败: %w", err)
	}

	// 获取兼容的节点列表
	compatibleNodes, err := s.getCompatibleNodes(ctx, kubeClient, processedTolerations)
	if err != nil {
		s.logger.Warn("获取兼容节点列表失败", zap.Error(err))
		compatibleNodes = []string{}
	}

	response := &model.K8sTaintTolerationResponse{
		ResourceType:      req.ResourceType,
		ResourceName:      req.ResourceName,
		Namespace:         req.Namespace,
		Tolerations:       processedTolerations,
		CompatibleNodes:   compatibleNodes,
		CreationTimestamp: time.Now(),
	}

	s.logger.Info("容忍度应用成功", zap.String("resource", req.ResourceName), zap.Int("compatible_nodes", len(compatibleNodes)))
	return response, nil
}

// ValidateTolerations 验证容忍度配置
// 验证容忍度配置的有效性和节点兼容性，提供详细的验证结果和建议
// 主要功能：
// - 验证容忍度配置的语法和语义正确性
// - 检查与集群节点的兼容性
// - 模拟调度结果
// - 提供优化建议
//
// 参数:
//   - ctx: 上下文对象
//   - req: 容忍度验证请求，包含要验证的容忍度配置
//
// 返回:
//   - *model.K8sTaintTolerationValidationResponse: 验证结果和建议
//   - error: 验证过程中的错误
func (s *TaintTolerationService) ValidateTolerations(ctx context.Context, req *model.K8sTaintTolerationValidationRequest) (*model.K8sTaintTolerationValidationResponse, error) {
	s.logger.Info("开始验证容忍度配置", zap.Int("cluster_id", req.ClusterID))

	// 获取Kubernetes客户端
	kubeClient, err := utils.GetKubeClient(req.ClusterID, s.k8sClient, s.logger)
	if err != nil {
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	var compatibleNodes []string
	var incompatibleNodes []string
	var validationErrors []string
	var suggestions []string

	// 验证每个容忍度
	for _, toleration := range req.Tolerations {
		result := s.validator.ValidateToleration(&toleration)
		if !result.Valid {
			validationErrors = append(validationErrors, result.Errors...)
		}
		suggestions = append(suggestions, result.Suggestions...)
	}

	// 获取节点列表并验证兼容性
	if req.CheckAllNodes {
		// 检查所有节点的兼容性
		nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取节点列表失败: %w", err)
		}

		for _, node := range nodes.Items {
			if s.isNodeCompatible(&node, req.Tolerations) {
				compatibleNodes = append(compatibleNodes, node.Name)
			} else {
				incompatibleNodes = append(incompatibleNodes, node.Name)
			}
		}
	} else if req.NodeName != "" {
		// 检查指定节点的兼容性
		node, err := kubeClient.CoreV1().Nodes().Get(ctx, req.NodeName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取节点失败: %w", err)
		}

		if s.isNodeCompatible(node, req.Tolerations) {
			compatibleNodes = append(compatibleNodes, node.Name)
		} else {
			incompatibleNodes = append(incompatibleNodes, node.Name)
		}
	}

	// 模拟调度
	var schedulingResult string
	if req.SimulateScheduling {
		schedulingResult = s.simulateScheduling(compatibleNodes, incompatibleNodes, req.Tolerations)
	}

	response := &model.K8sTaintTolerationValidationResponse{
		Valid:             len(validationErrors) == 0,
		CompatibleNodes:   compatibleNodes,
		IncompatibleNodes: incompatibleNodes,
		ValidationErrors:  validationErrors,
		Suggestions:       suggestions,
		SchedulingResult:  schedulingResult,
		ValidationTime:    time.Now(),
	}

	return response, nil
}

// ManageNodeTaints 管理节点污点
// 对指定节点进行污点的添加、更新或删除操作
// 主要功能：
// - 验证污点操作请求
// - 执行污点操作（add、update、delete）
// - 分析受影响的Pod
// - 返回操作结果和影响范围
//
// 参数:
//   - ctx: 上下文对象
//   - req: 节点污点管理请求
//
// 返回:
//   - *model.K8sNodeTaintResponse: 污点管理结果
//   - error: 操作过程中的错误
func (s *TaintTolerationService) ManageNodeTaints(ctx context.Context, req *model.K8sNodeTaintRequest) (*model.K8sNodeTaintResponse, error) {
	s.logger.Info("开始管理节点污点", zap.Int("cluster_id", req.ClusterID), zap.String("node", req.NodeName))

	// 验证请求
	validationResult := s.validator.ValidateTaintRequest(req)
	if !validationResult.Valid {
		return nil, fmt.Errorf("验证失败: %v", validationResult.Errors)
	}

	// 获取Kubernetes客户端
	kubeClient, err := utils.GetKubeClient(req.ClusterID, s.k8sClient, s.logger)
	if err != nil {
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 获取节点
	node, err := kubeClient.CoreV1().Nodes().Get(ctx, req.NodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点失败: %w", err)
	}

	// 获取受影响的Pod列表
	affectedPods, err := s.getAffectedPods(ctx, kubeClient, req.NodeName, req.Taints)
	if err != nil {
		s.logger.Warn("获取受影响Pod列表失败", zap.Error(err))
	}

	// 执行污点操作
	var updatedTaints []corev1.Taint
	switch req.Operation {
	case "add":
		updatedTaints = s.addTaints(node.Spec.Taints, req.Taints)
	case "update":
		updatedTaints = s.updateTaints(node.Spec.Taints, req.Taints)
	case "delete":
		updatedTaints = s.deleteTaints(node.Spec.Taints, req.Taints)
	default:
		return nil, fmt.Errorf("不支持的操作: %s", req.Operation)
	}

	// 更新节点
	node.Spec.Taints = updatedTaints
	_, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("更新节点污点失败: %w", err)
	}

	// 转换污点格式
	responseTaints := make([]model.K8sTaint, len(updatedTaints))
	for i, taint := range updatedTaints {
		responseTaints[i] = model.K8sTaint{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: string(taint.Effect),
		}
	}

	response := &model.K8sNodeTaintResponse{
		NodeName:      req.NodeName,
		Taints:        responseTaints,
		AffectedPods:  affectedPods,
		Operation:     req.Operation,
		OperationTime: time.Now(),
	}

	s.logger.Info("节点污点管理成功", zap.String("node", req.NodeName), zap.String("operation", req.Operation))
	return response, nil
}

// ConfigureTolerationTime 配置容忍时间
// 为容忍度配置时间参数，包括超时时间、默认时间和条件化超时
// 主要功能：
// - 验证时间配置参数
// - 计算应用的超时值
// - 验证超时配置的有效性
// - 返回配置结果和验证信息
//
// 参数:
//   - ctx: 上下文对象
//   - req: 容忍时间配置请求
//
// 返回:
//   - *model.K8sTolerationTimeResponse: 时间配置结果
//   - error: 配置过程中的错误
func (s *TaintTolerationService) ConfigureTolerationTime(ctx context.Context, req *model.K8sTolerationTimeRequest) (*model.K8sTolerationTimeResponse, error) {
	s.logger.Info("开始配置容忍时间", zap.Int("cluster_id", req.ClusterID))

	// 验证时间配置
	validationResult := s.validator.ValidateTolerationTimeConfig(&req.TimeConfig)
	if !validationResult.Valid {
		return nil, fmt.Errorf("时间配置验证失败: %v", validationResult.Errors)
	}

	// 计算应用的超时
	appliedTimeouts := s.calculateAppliedTimeouts(&req.TimeConfig)

	// 验证每个超时配置
	var validationResults []model.TimeValidationResult
	for _, timeout := range appliedTimeouts {
		validationResult := s.validateAppliedTimeout(&timeout)
		validationResults = append(validationResults, validationResult)
	}

	response := &model.K8sTolerationTimeResponse{
		ResourceType:      req.ResourceType,
		ResourceName:      req.ResourceName,
		Namespace:         req.Namespace,
		AppliedTimeouts:   appliedTimeouts,
		ValidationResults: validationResults,
		CreationTimestamp: time.Now(),
		Status:            "success",
	}

	return response, nil
}

// processTolerations 处理容忍度，应用默认值和优化
// 对容忍度配置进行预处理，包括：
// - 应用默认操作符
// - 设置默认容忍时间（针对NoExecute效果）
// - 按效果优先级排序
//
// 参数:
//   - tolerations: 原始容忍度配置列表
//
// 返回:
//   - []model.K8sToleration: 处理后的容忍度配置列表
func (s *TaintTolerationService) processTolerations(tolerations []model.K8sToleration) []model.K8sToleration {
	processed := make([]model.K8sToleration, len(tolerations))

	for i, toleration := range tolerations {
		processed[i] = toleration

		// 应用默认操作符
		if processed[i].Operator == "" {
			processed[i].Operator = s.config.TaintDefaults.DefaultOperator
		}

		// 应用默认容忍时间（仅对NoExecute效果）
		if processed[i].Effect == "NoExecute" && processed[i].TolerationSeconds == nil {
			defaultTime := s.config.TaintDefaults.DefaultTolerationTime
			processed[i].TolerationSeconds = &defaultTime
		}
	}

	// 按效果优先级排序
	sort.Slice(processed, func(i, j int) bool {
		return s.getEffectPriority(processed[i].Effect) < s.getEffectPriority(processed[j].Effect)
	})

	return processed
}

// getEffectPriority 获取效果优先级
// 根据配置的效果优先级列表确定效果的优先级顺序
//
// 参数:
//   - effect: 污点效果字符串
//
// 返回:
//   - int: 效果优先级（数值越小优先级越高）
func (s *TaintTolerationService) getEffectPriority(effect string) int {
	for i, priorityEffect := range s.config.TaintDefaults.EffectPriority {
		if effect == priorityEffect {
			return i
		}
	}
	return len(s.config.TaintDefaults.EffectPriority)
}

// applyTolerationsToDeployment 应用容忍度到Deployment
// 为Deployment资源应用容忍度配置，更新Pod模板中的容忍度设置
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - namespace: 命名空间
//   - name: Deployment名称
//   - tolerations: 要应用的容忍度配置
//
// 返回:
//   - error: 操作过程中的错误
func (s *TaintTolerationService) applyTolerationsToDeployment(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, name string, tolerations []model.K8sToleration) error {
	deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取Deployment失败: %w", err)
	}

	// 转换容忍度格式
	k8sTolerations := s.convertToK8sTolerations(tolerations)
	deployment.Spec.Template.Spec.Tolerations = k8sTolerations

	_, err = kubeClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("更新Deployment失败: %w", err)
	}

	return nil
}

// applyTolerationsToStatefulSet 应用容忍度到StatefulSet
// 为StatefulSet资源应用容忍度配置，更新Pod模板中的容忍度设置
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - namespace: 命名空间
//   - name: StatefulSet名称
//   - tolerations: 要应用的容忍度配置
//
// 返回:
//   - error: 操作过程中的错误
func (s *TaintTolerationService) applyTolerationsToStatefulSet(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, name string, tolerations []model.K8sToleration) error {
	statefulSet, err := kubeClient.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取StatefulSet失败: %w", err)
	}

	// 转换容忍度格式
	k8sTolerations := s.convertToK8sTolerations(tolerations)
	statefulSet.Spec.Template.Spec.Tolerations = k8sTolerations

	_, err = kubeClient.AppsV1().StatefulSets(namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("更新StatefulSet失败: %w", err)
	}

	return nil
}

// applyTolerationsToDaemonSet 应用容忍度到DaemonSet
// 为DaemonSet资源应用容忍度配置，更新Pod模板中的容忍度设置
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - namespace: 命名空间
//   - name: DaemonSet名称
//   - tolerations: 要应用的容忍度配置
//
// 返回:
//   - error: 操作过程中的错误
func (s *TaintTolerationService) applyTolerationsToDaemonSet(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, name string, tolerations []model.K8sToleration) error {
	daemonSet, err := kubeClient.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取DaemonSet失败: %w", err)
	}

	// 转换容忍度格式
	k8sTolerations := s.convertToK8sTolerations(tolerations)
	daemonSet.Spec.Template.Spec.Tolerations = k8sTolerations

	_, err = kubeClient.AppsV1().DaemonSets(namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("更新DaemonSet失败: %w", err)
	}

	return nil
}

// applyTolerationsToPod 应用容忍度到Pod
// 为Pod资源应用容忍度配置，直接更新Pod的容忍度设置
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - namespace: 命名空间
//   - name: Pod名称
//   - tolerations: 要应用的容忍度配置
//
// 返回:
//   - error: 操作过程中的错误
func (s *TaintTolerationService) applyTolerationsToPod(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, name string, tolerations []model.K8sToleration) error {
	pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取Pod失败: %w", err)
	}

	// 转换容忍度格式
	k8sTolerations := s.convertToK8sTolerations(tolerations)
	pod.Spec.Tolerations = k8sTolerations

	_, err = kubeClient.CoreV1().Pods(namespace).Update(ctx, pod, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("更新Pod失败: %w", err)
	}

	return nil
}

// convertToK8sTolerations 转换为Kubernetes容忍度格式
// 将内部容忍度模型转换为Kubernetes API的容忍度格式
//
// 参数:
//   - tolerations: 内部容忍度配置列表
//
// 返回:
//   - []corev1.Toleration: Kubernetes容忍度格式列表
func (s *TaintTolerationService) convertToK8sTolerations(tolerations []model.K8sToleration) []corev1.Toleration {
	k8sTolerations := make([]corev1.Toleration, len(tolerations))

	for i, toleration := range tolerations {
		k8sTolerations[i] = corev1.Toleration{
			Key:               toleration.Key,
			Operator:          corev1.TolerationOperator(toleration.Operator),
			Value:             toleration.Value,
			Effect:            corev1.TaintEffect(toleration.Effect),
			TolerationSeconds: toleration.TolerationSeconds,
		}
	}

	return k8sTolerations
}

// getCompatibleNodes 获取兼容的节点列表
// 查找与指定容忍度配置兼容的集群节点
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - tolerations: 容忍度配置列表
//
// 返回:
//   - []string: 兼容节点的名称列表
//   - error: 查找过程中的错误
func (s *TaintTolerationService) getCompatibleNodes(ctx context.Context, kubeClient *kubernetes.Clientset, tolerations []model.K8sToleration) ([]string, error) {
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var compatibleNodes []string
	for _, node := range nodes.Items {
		if s.isNodeCompatible(&node, tolerations) {
			compatibleNodes = append(compatibleNodes, node.Name)
		}
	}

	return compatibleNodes, nil
}

// isNodeCompatible 检查节点是否与容忍度兼容
// 检查节点的污点是否能被指定的容忍度配置容忍
//
// 参数:
//   - node: 要检查的节点
//   - tolerations: 容忍度配置列表
//
// 返回:
//   - bool: true表示节点兼容，false表示不兼容
func (s *TaintTolerationService) isNodeCompatible(node *corev1.Node, tolerations []model.K8sToleration) bool {
	// 将容忍度转换为map便于查找
	tolerationMap := make(map[string]model.K8sToleration)
	for _, toleration := range tolerations {
		key := fmt.Sprintf("%s:%s:%s", toleration.Key, toleration.Value, toleration.Effect)
		tolerationMap[key] = toleration
	}

	// 检查每个污点是否有对应的容忍度
	for _, taint := range node.Spec.Taints {
		if !s.isTaintTolerated(taint, tolerationMap) {
			return false
		}
	}

	return true
}

// isTaintTolerated 检查污点是否被容忍
// 检查指定的污点是否能被容忍度配置容忍
//
// 参数:
//   - taint: 要检查的污点
//   - tolerationMap: 容忍度配置映射
//
// 返回:
//   - bool: true表示污点被容忍，false表示不被容忍
func (s *TaintTolerationService) isTaintTolerated(taint corev1.Taint, tolerationMap map[string]model.K8sToleration) bool {
	// 精确匹配
	exactKey := fmt.Sprintf("%s:%s:%s", taint.Key, taint.Value, string(taint.Effect))
	if _, exists := tolerationMap[exactKey]; exists {
		return true
	}

	// 检查Exists操作符的容忍度
	for _, toleration := range tolerationMap {
		if toleration.Operator == "Exists" {
			if toleration.Key == taint.Key && (toleration.Effect == "" || toleration.Effect == string(taint.Effect)) {
				return true
			}
		}
	}

	return false
}

// getAffectedPods 获取受污点影响的Pod列表
// 查找指定节点上可能受污点影响的Pod
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - nodeName: 节点名称
//   - taints: 污点配置列表
//
// 返回:
//   - []string: 受影响Pod的名称列表
//   - error: 查找过程中的错误
func (s *TaintTolerationService) getAffectedPods(ctx context.Context, kubeClient *kubernetes.Clientset, nodeName string, taints []model.K8sTaint) ([]string, error) {
	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		return nil, err
	}

	var affectedPods []string
	for _, pod := range pods.Items {
		if s.isPodAffectedByTaints(pod, taints) {
			affectedPods = append(affectedPods, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
		}
	}

	return affectedPods, nil
}

// isPodAffectedByTaints 检查Pod是否受污点影响
// 检查Pod的容忍度配置是否能容忍指定的污点
//
// 参数:
//   - pod: 要检查的Pod
//   - taints: 污点配置列表
//
// 返回:
//   - bool: true表示Pod受影响，false表示不受影响
func (s *TaintTolerationService) isPodAffectedByTaints(pod corev1.Pod, taints []model.K8sTaint) bool {
	// 将Pod的容忍度转换为map
	tolerationMap := make(map[string]corev1.Toleration)
	for _, toleration := range pod.Spec.Tolerations {
		key := fmt.Sprintf("%s:%s:%s", toleration.Key, toleration.Value, string(toleration.Effect))
		tolerationMap[key] = toleration
	}

	// 检查是否有污点无法被容忍
	for _, taint := range taints {
		k8sTaint := corev1.Taint{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: corev1.TaintEffect(taint.Effect),
		}

		if !s.isTaintToleratedByPod(k8sTaint, tolerationMap) {
			return true
		}
	}

	return false
}

// isTaintToleratedByPod 检查污点是否被Pod容忍
// 检查Pod的容忍度配置是否能容忍指定的污点
//
// 参数:
//   - taint: 要检查的污点
//   - tolerationMap: Pod的容忍度配置映射
//
// 返回:
//   - bool: true表示污点被Pod容忍，false表示不被容忍
func (s *TaintTolerationService) isTaintToleratedByPod(taint corev1.Taint, tolerationMap map[string]corev1.Toleration) bool {
	// 精确匹配
	exactKey := fmt.Sprintf("%s:%s:%s", taint.Key, taint.Value, string(taint.Effect))
	if _, exists := tolerationMap[exactKey]; exists {
		return true
	}

	// 检查Exists操作符的容忍度
	for _, toleration := range tolerationMap {
		if toleration.Operator == corev1.TolerationOpExists {
			if toleration.Key == taint.Key && (toleration.Effect == "" || toleration.Effect == taint.Effect) {
				return true
			}
		}
	}

	return false
}

// addTaints 添加污点
// 将新的污点添加到现有污点列表中，避免重复
//
// 参数:
//   - existingTaints: 现有的污点列表
//   - newTaints: 要添加的新污点列表
//
// 返回:
//   - []corev1.Taint: 合并后的污点列表
func (s *TaintTolerationService) addTaints(existingTaints []corev1.Taint, newTaints []model.K8sTaint) []corev1.Taint {
	// 转换新污点为Kubernetes格式
	k8sNewTaints := make([]corev1.Taint, len(newTaints))
	for i, taint := range newTaints {
		k8sNewTaints[i] = corev1.Taint{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: corev1.TaintEffect(taint.Effect),
		}
	}

	// 使用工具函数合并污点
	return utils.MergeTaints(existingTaints, k8sNewTaints)
}

// updateTaints 更新污点
// 更新现有污点的配置，先删除再添加
//
// 参数:
//   - existingTaints: 现有的污点列表
//   - updateTaints: 要更新的污点列表
//
// 返回:
//   - []corev1.Taint: 更新后的污点列表
func (s *TaintTolerationService) updateTaints(existingTaints []corev1.Taint, updateTaints []model.K8sTaint) []corev1.Taint {
	// 先删除现有的同名污点，再添加新污点
	k8sUpdateTaints := make([]corev1.Taint, len(updateTaints))
	for i, taint := range updateTaints {
		k8sUpdateTaints[i] = corev1.Taint{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: corev1.TaintEffect(taint.Effect),
		}
	}

	// 删除要更新的污点
	updatedTaints := utils.RemoveTaints(existingTaints, k8sUpdateTaints)
	// 添加新的污点
	return utils.MergeTaints(updatedTaints, k8sUpdateTaints)
}

// deleteTaints 删除污点
// 从现有污点列表中删除指定的污点
//
// 参数:
//   - existingTaints: 现有的污点列表
//   - deleteTaints: 要删除的污点列表
//
// 返回:
//   - []corev1.Taint: 删除后的污点列表
func (s *TaintTolerationService) deleteTaints(existingTaints []corev1.Taint, deleteTaints []model.K8sTaint) []corev1.Taint {
	// 转换删除污点为Kubernetes格式
	k8sDeleteTaints := make([]corev1.Taint, len(deleteTaints))
	for i, taint := range deleteTaints {
		k8sDeleteTaints[i] = corev1.Taint{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: corev1.TaintEffect(taint.Effect),
		}
	}

	// 使用工具函数删除污点
	return utils.RemoveTaints(existingTaints, k8sDeleteTaints)
}

// simulateScheduling 模拟调度
// 根据兼容节点和不兼容节点的情况模拟调度结果
//
// 参数:
//   - compatibleNodes: 兼容节点列表
//   - incompatibleNodes: 不兼容节点列表
//   - tolerations: 容忍度配置列表
//
// 返回:
//   - string: 调度结果描述
func (s *TaintTolerationService) simulateScheduling(compatibleNodes, incompatibleNodes []string, tolerations []model.K8sToleration) string {
	if len(compatibleNodes) == 0 {
		return "调度失败：没有兼容的节点"
	}

	if len(incompatibleNodes) == 0 {
		return fmt.Sprintf("调度成功：所有节点都兼容，可选择节点: %v", compatibleNodes)
	}

	return fmt.Sprintf("调度成功：找到%d个兼容节点，%d个不兼容节点。兼容节点: %v",
		len(compatibleNodes), len(incompatibleNodes), compatibleNodes)
}

// calculateAppliedTimeouts 计算应用的超时
// 根据时间配置计算实际应用的超时值
//
// 参数:
//   - timeConfig: 时间配置对象
//
// 返回:
//   - []model.AppliedTimeout: 应用的超时配置列表
func (s *TaintTolerationService) calculateAppliedTimeouts(timeConfig *model.TolerationTimeConfig) []model.AppliedTimeout {
	var appliedTimeouts []model.AppliedTimeout

	// 应用默认超时
	if timeConfig.DefaultTolerationTime != nil {
		for _, effect := range s.config.TaintDefaults.EffectPriority {
			if effect == "NoExecute" {
				appliedTimeouts = append(appliedTimeouts, model.AppliedTimeout{
					TaintKey:         "*",
					Effect:           effect,
					TimeoutValue:     timeConfig.DefaultTolerationTime,
					AppliedCondition: "default",
					Source:           "time_config",
				})
			}
		}
	}

	// 应用条件超时
	for _, conditionalTimeout := range timeConfig.ConditionalTimeouts {
		for _, effect := range conditionalTimeout.ApplyToEffects {
			appliedTimeouts = append(appliedTimeouts, model.AppliedTimeout{
				TaintKey:         fmt.Sprintf("condition:%s", conditionalTimeout.Condition),
				Effect:           effect,
				TimeoutValue:     conditionalTimeout.TimeoutValue,
				AppliedCondition: conditionalTimeout.Condition,
				Source:           "conditional_timeout",
			})
		}
	}

	return appliedTimeouts
}

// validateAppliedTimeout 验证应用的超时
// 验证超时配置的有效性，包括范围检查和合理性验证
//
// 参数:
//   - timeout: 要验证的超时配置
//
// 返回:
//   - model.TimeValidationResult: 验证结果
func (s *TaintTolerationService) validateAppliedTimeout(timeout *model.AppliedTimeout) model.TimeValidationResult {
	result := model.TimeValidationResult{
		TaintKey:       timeout.TaintKey,
		IsValid:        true,
		ValidationTime: time.Now(),
	}

	if timeout.TimeoutValue == nil || *timeout.TimeoutValue <= 0 {
		result.IsValid = false
		result.ValidationMessage = "超时值必须大于0"
		recommendedTimeout := s.config.TaintDefaults.DefaultTolerationTime
		result.RecommendedTimeout = &recommendedTimeout
		return result
	}

	if *timeout.TimeoutValue > s.config.ValidationRules.MaxTolerationTimeSeconds {
		result.IsValid = false
		result.ValidationMessage = fmt.Sprintf("超时值超过最大限制 %d 秒", s.config.ValidationRules.MaxTolerationTimeSeconds)
		result.RecommendedTimeout = &s.config.ValidationRules.MaxTolerationTimeSeconds
		return result
	}

	result.ValidationMessage = "超时配置有效"
	return result
}

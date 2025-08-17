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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// TaintEffectService 污点效果管理服务
// 提供Kubernetes节点污点效果的完整管理功能，包括：
// - 污点效果的应用和管理（NoSchedule、PreferNoSchedule、NoExecute）
// - Pod驱逐策略的执行和监控
// - 批量节点操作支持
// - 效果转换和条件化应用
// - 兼容性检查和状态监控
type TaintEffectService struct {
	k8sClient client.K8sClient  // Kubernetes客户端接口，用于与K8s集群通信
	logger    *zap.Logger       // 结构化日志记录器，用于记录服务操作日志
	config    *config.K8sConfig // K8s配置对象，包含污点效果管理的默认配置
}

// NewTaintEffectService 创建污点效果管理服务实例
// 初始化所有必要的依赖项并返回服务实例
// 参数:
//   - k8sClient: K8s客户端接口，用于与Kubernetes集群通信
//   - logger: 日志记录器，用于记录服务操作和错误信息
//
// 返回: 配置好的TaintEffectService实例
func NewTaintEffectService(k8sClient client.K8sClient, logger *zap.Logger) *TaintEffectService {
	return &TaintEffectService{
		k8sClient: k8sClient,
		logger:    logger,
		config:    config.GetK8sConfig(), // 获取全局K8s配置
	}
}

// ManageTaintEffects 管理污点效果
// 这是污点效果管理的核心方法，支持多种操作模式：
// - 单个节点操作：直接指定节点名称
// - 批量操作：使用节点选择器或处理所有节点
// - 条件化操作：根据节点状态和配置条件应用效果
//
// 实现逻辑:
// 1. 获取目标集群的Kubernetes客户端连接
// 2. 根据请求参数确定目标节点（单个、选择器匹配或全部）
// 3. 根据操作模式选择处理策略（批量或单个）
// 4. 对每个目标节点应用相应的污点效果配置
// 5. 收集操作结果和警告信息
// 6. 返回合并后的操作响应
//
// 参数:
//   - ctx: 上下文对象，用于控制请求生命周期和取消操作
//   - req: 包含集群ID、节点信息和污点效果配置的请求对象
//
// 返回:
//   - *model.K8sTaintEffectManagementResponse: 包含操作结果和受影响Pod信息的响应对象
//   - error: 操作过程中发生的错误信息
func (s *TaintEffectService) ManageTaintEffects(ctx context.Context, req *model.K8sTaintEffectManagementRequest) (*model.K8sTaintEffectManagementResponse, error) {
	s.logger.Info("开始管理污点效果", zap.Int("cluster_id", req.ClusterID))

	// 获取Kubernetes客户端
	kubeClient, err := utils.GetKubeClient(req.ClusterID, s.k8sClient, s.logger)
	if err != nil {
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	var responses []*model.K8sTaintEffectManagementResponse
	var warnings []string

	// 处理节点列表
	nodes, err := s.getTargetNodes(ctx, kubeClient, req)
	if err != nil {
		return nil, fmt.Errorf("获取目标节点失败: %w", err)
	}

	// 批量处理或单个处理
	if req.BatchOperation {
		// 批量操作模式：处理多个节点
		responses, warnings = s.processBatchNodes(ctx, kubeClient, nodes, req)
	} else if req.NodeName != "" {
		// 单个节点操作模式：处理指定节点
		response, err := s.processSingleNode(ctx, kubeClient, req.NodeName, req)
		if err != nil {
			return nil, err
		}
		responses = append(responses, response)
	} else {
		return nil, fmt.Errorf("必须指定节点名称或启用批量操作")
	}

	// 合并响应结果
	if len(responses) == 0 {
		return nil, fmt.Errorf("没有处理任何节点")
	}

	// 返回第一个响应作为主要结果，将其他结果合并到警告中
	mainResponse := responses[0]
	if len(responses) > 1 {
		mainResponse.Warnings = append(mainResponse.Warnings, fmt.Sprintf("批量处理了%d个节点", len(responses)))
	}
	mainResponse.Warnings = append(mainResponse.Warnings, warnings...)

	return mainResponse, nil
}

// getTargetNodes 获取目标节点列表
// 根据请求参数确定需要处理的节点：
// - 如果指定了节点名称，返回单个节点
// - 如果指定了节点选择器，返回匹配的节点
// - 如果都没有指定，返回所有节点
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - req: 包含节点选择条件的请求对象
//
// 返回:
//   - []corev1.Node: 目标节点列表
//   - error: 获取节点过程中的错误
func (s *TaintEffectService) getTargetNodes(ctx context.Context, kubeClient *kubernetes.Clientset, req *model.K8sTaintEffectManagementRequest) ([]corev1.Node, error) {
	var nodes []corev1.Node

	if req.NodeName != "" {
		// 获取单个节点
		node, err := kubeClient.CoreV1().Nodes().Get(ctx, req.NodeName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取节点失败: %w", err)
		}
		nodes = append(nodes, *node)
	} else if len(req.NodeSelector) > 0 {
		// 根据选择器获取节点
		labelSelector := s.buildLabelSelector(req.NodeSelector)
		nodeList, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return nil, fmt.Errorf("根据选择器获取节点失败: %w", err)
		}
		nodes = nodeList.Items
	} else {
		// 获取所有节点
		nodeList, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取所有节点失败: %w", err)
		}
		nodes = nodeList.Items
	}

	return nodes, nil
}

// processBatchNodes 批量处理节点
// 对多个节点并行应用污点效果配置，收集所有操作结果和警告信息
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - nodes: 要处理的节点列表
//   - req: 污点效果管理请求
//
// 返回:
//   - []*model.K8sTaintEffectManagementResponse: 每个节点的处理结果
//   - []string: 处理过程中的警告信息
func (s *TaintEffectService) processBatchNodes(ctx context.Context, kubeClient *kubernetes.Clientset, nodes []corev1.Node, req *model.K8sTaintEffectManagementRequest) ([]*model.K8sTaintEffectManagementResponse, []string) {
	var responses []*model.K8sTaintEffectManagementResponse
	var warnings []string

	// 遍历处理每个节点
	for _, node := range nodes {
		response, err := s.processSingleNode(ctx, kubeClient, node.Name, req)
		if err != nil {
			// 单个节点处理失败不影响其他节点，记录警告并继续
			warnings = append(warnings, fmt.Sprintf("处理节点 %s 失败: %v", node.Name, err))
			continue
		}
		responses = append(responses, response)
	}

	return responses, warnings
}

// processSingleNode 处理单个节点
// 对指定节点应用污点效果配置，包括：
// - 分析节点当前的污点状态
// - 应用配置的污点效果（NoSchedule、PreferNoSchedule、NoExecute）
// - 处理受影响的Pod（驱逐、重新调度等）
// - 收集操作统计信息
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - nodeName: 要处理的节点名称
//   - req: 污点效果管理请求
//
// 返回:
//   - *model.K8sTaintEffectManagementResponse: 节点处理结果
//   - error: 处理过程中的错误
func (s *TaintEffectService) processSingleNode(ctx context.Context, kubeClient *kubernetes.Clientset, nodeName string, req *model.K8sTaintEffectManagementRequest) (*model.K8sTaintEffectManagementResponse, error) {
	// 获取节点
	node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点失败: %w", err)
	}

	// 获取节点上的Pod
	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		return nil, fmt.Errorf("获取节点Pod失败: %w", err)
	}

	var affectedPods []model.PodEvictionInfo
	var effectChanges []model.EffectChange
	var evictionSummary model.EvictionSummary

	// 处理NoSchedule效果
	if req.TaintEffectConfig.NoScheduleConfig.Enabled {
		changes, affected := s.handleNoScheduleEffect(ctx, kubeClient, node, pods.Items, &req.TaintEffectConfig.NoScheduleConfig)
		effectChanges = append(effectChanges, changes...)
		affectedPods = append(affectedPods, affected...)
	}

	// 处理PreferNoSchedule效果
	if req.TaintEffectConfig.PreferNoScheduleConfig.Enabled {
		changes, affected := s.handlePreferNoScheduleEffect(ctx, kubeClient, node, pods.Items, &req.TaintEffectConfig.PreferNoScheduleConfig)
		effectChanges = append(effectChanges, changes...)
		affectedPods = append(affectedPods, affected...)
	}

	// 处理NoExecute效果
	if req.TaintEffectConfig.NoExecuteConfig.Enabled {
		changes, affected, summary := s.handleNoExecuteEffect(ctx, kubeClient, node, pods.Items, &req.TaintEffectConfig.NoExecuteConfig, req.GracePeriod, req.ForceEviction)
		effectChanges = append(effectChanges, changes...)
		affectedPods = append(affectedPods, affected...)
		evictionSummary = summary
	}

	response := &model.K8sTaintEffectManagementResponse{
		NodeName:        nodeName,
		AffectedPods:    affectedPods,
		EffectChanges:   effectChanges,
		EvictionSummary: evictionSummary,
		OperationTime:   time.Now(),
		Status:          "success",
		Warnings:        []string{},
	}

	return response, nil
}

// handleNoScheduleEffect 处理NoSchedule效果
// NoSchedule效果阻止新Pod调度到节点上，但不影响现有Pod
// 主要功能：
// - 记录NoSchedule污点的应用
// - 检查例外Pod配置
// - 分析对现有Pod的影响
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - node: 目标节点
//   - pods: 节点上的Pod列表
//   - config: NoSchedule配置
//
// 返回:
//   - []model.EffectChange: 效果变化记录
//   - []model.PodEvictionInfo: 受影响的Pod信息
func (s *TaintEffectService) handleNoScheduleEffect(ctx context.Context, kubeClient *kubernetes.Clientset, node *corev1.Node, pods []corev1.Pod, config *model.NoScheduleConfig) ([]model.EffectChange, []model.PodEvictionInfo) {
	var effectChanges []model.EffectChange
	var affectedPods []model.PodEvictionInfo

	s.logger.Info("处理NoSchedule效果", zap.String("node", node.Name))

	// NoSchedule效果主要是阻止新Pod调度，对现有Pod无影响
	// 这里主要记录效果变化
	for _, taint := range node.Spec.Taints {
		if taint.Effect == corev1.TaintEffectNoSchedule {
			effectChange := model.EffectChange{
				TaintKey:     taint.Key,
				OldEffect:    "None",
				NewEffect:    "NoSchedule",
				ChangeReason: "NoSchedule effect applied",
				ChangeTime:   time.Now(),
			}
			effectChanges = append(effectChanges, effectChange)
		}
	}

	// 检查例外Pod
	if len(config.ExceptionPods) > 0 {
		for _, pod := range pods {
			podName := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
			for _, exceptionPod := range config.ExceptionPods {
				if strings.Contains(podName, exceptionPod) {
					affectedPod := model.PodEvictionInfo{
						PodName:            pod.Name,
						Namespace:          pod.Namespace,
						EvictionReason:     "Exception pod - NoSchedule bypassed",
						Status:             "running",
						RescheduleAttempts: 0,
					}
					affectedPods = append(affectedPods, affectedPod)
				}
			}
		}
	}

	return effectChanges, affectedPods
}

// handlePreferNoScheduleEffect 处理PreferNoSchedule效果
// PreferNoSchedule是软约束，调度器会尽量避免将Pod调度到有该污点的节点
// 主要功能：
// - 记录PreferNoSchedule污点的应用
// - 设置调度偏好权重
// - 分析对现有Pod的影响（通常无直接影响）
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - node: 目标节点
//   - pods: 节点上的Pod列表
//   - config: PreferNoSchedule配置
//
// 返回:
//   - []model.EffectChange: 效果变化记录
//   - []model.PodEvictionInfo: 受影响的Pod信息
func (s *TaintEffectService) handlePreferNoScheduleEffect(ctx context.Context, kubeClient *kubernetes.Clientset, node *corev1.Node, pods []corev1.Pod, config *model.PreferNoScheduleConfig) ([]model.EffectChange, []model.PodEvictionInfo) {
	var effectChanges []model.EffectChange
	var affectedPods []model.PodEvictionInfo

	s.logger.Info("处理PreferNoSchedule效果", zap.String("node", node.Name))

	// PreferNoSchedule是软约束，不会驱逐现有Pod
	// 记录效果变化和权重设置
	for _, taint := range node.Spec.Taints {
		if taint.Effect == corev1.TaintEffectPreferNoSchedule {
			effectChange := model.EffectChange{
				TaintKey:     taint.Key,
				OldEffect:    "None",
				NewEffect:    fmt.Sprintf("PreferNoSchedule(weight:%d)", config.PreferenceWeight),
				ChangeReason: "PreferNoSchedule effect applied with weight",
				ChangeTime:   time.Now(),
			}
			effectChanges = append(effectChanges, effectChange)
		}
	}

	// PreferNoSchedule对现有Pod无直接影响，只影响调度偏好
	for _, pod := range pods {
		affectedPod := model.PodEvictionInfo{
			PodName:            pod.Name,
			Namespace:          pod.Namespace,
			EvictionReason:     "PreferNoSchedule - no eviction needed",
			Status:             "running",
			RescheduleAttempts: 0,
		}
		affectedPods = append(affectedPods, affectedPod)
	}

	return effectChanges, affectedPods
}

// handleNoExecuteEffect 处理NoExecute效果
// NoExecute效果会驱逐不能容忍该污点的Pod，这是最严格的污点效果
// 主要功能：
// - 检查Pod是否容忍NoExecute污点
// - 执行Pod驱逐策略（立即、优雅、延迟）
// - 统计驱逐结果和性能指标
// - 处理PodDisruptionBudget限制
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - node: 目标节点
//   - pods: 节点上的Pod列表
//   - config: NoExecute配置
//   - gracePeriod: 优雅期时间
//   - forceEviction: 是否强制驱逐
//
// 返回:
//   - []model.EffectChange: 效果变化记录
//   - []model.PodEvictionInfo: 受影响的Pod信息
//   - model.EvictionSummary: 驱逐统计信息
func (s *TaintEffectService) handleNoExecuteEffect(ctx context.Context, kubeClient *kubernetes.Clientset, node *corev1.Node, pods []corev1.Pod, config *model.NoExecuteConfig, gracePeriod *int64, forceEviction bool) ([]model.EffectChange, []model.PodEvictionInfo, model.EvictionSummary) {
	var effectChanges []model.EffectChange
	var affectedPods []model.PodEvictionInfo
	var evictionSummary model.EvictionSummary

	s.logger.Info("处理NoExecute效果", zap.String("node", node.Name))

	// 统计信息初始化
	evictionSummary.TotalPods = len(pods)
	startTime := time.Now()

	// 记录效果变化
	for _, taint := range node.Spec.Taints {
		if taint.Effect == corev1.TaintEffectNoExecute {
			effectChange := model.EffectChange{
				TaintKey:     taint.Key,
				OldEffect:    "None",
				NewEffect:    "NoExecute",
				ChangeReason: "NoExecute effect applied - will evict intolerant pods",
				ChangeTime:   time.Now(),
			}
			effectChanges = append(effectChanges, effectChange)
		}
	}

	// 处理每个Pod
	for _, pod := range pods {
		// 检查Pod是否容忍NoExecute污点
		toleratesNoExecute := s.checkPodToleratesNoExecute(pod, node.Spec.Taints)

		if !toleratesNoExecute {
			// 需要驱逐的Pod
			affectedPod := s.processPodEviction(ctx, kubeClient, pod, config, gracePeriod, forceEviction)
			affectedPods = append(affectedPods, affectedPod)

			// 更新统计
			switch affectedPod.Status {
			case "evicted":
				evictionSummary.EvictedPods++
			case "failed":
				evictionSummary.FailedEvictions++
			case "pending":
				evictionSummary.PendingEvictions++
			case "rescheduled":
				evictionSummary.RescheduledPods++
			}
		} else {
			// 容忍NoExecute的Pod
			affectedPod := model.PodEvictionInfo{
				PodName:            pod.Name,
				Namespace:          pod.Namespace,
				EvictionReason:     "Pod tolerates NoExecute - no eviction needed",
				Status:             "running",
				RescheduleAttempts: 0,
			}
			affectedPods = append(affectedPods, affectedPod)
		}
	}

	// 计算平均驱逐时间
	if evictionSummary.EvictedPods > 0 {
		totalTime := time.Since(startTime).Seconds()
		evictionSummary.AverageEvictionTime = totalTime / float64(evictionSummary.EvictedPods)
	}

	return effectChanges, affectedPods, evictionSummary
}

// checkPodToleratesNoExecute 检查Pod是否容忍NoExecute效果
// 遍历Pod的所有容忍度，检查是否能容忍节点上的NoExecute污点
//
// 参数:
//   - pod: 要检查的Pod
//   - nodeTaints: 节点上的污点列表
//
// 返回:
//   - bool: true表示Pod容忍所有NoExecute污点，false表示不能容忍
func (s *TaintEffectService) checkPodToleratesNoExecute(pod corev1.Pod, nodeTaints []corev1.Taint) bool {
	for _, taint := range nodeTaints {
		if taint.Effect != corev1.TaintEffectNoExecute {
			continue
		}

		// 检查Pod是否有对应的容忍度
		tolerates := false
		for _, toleration := range pod.Spec.Tolerations {
			if s.tolerationMatches(toleration, taint) {
				tolerates = true
				break
			}
		}

		if !tolerates {
			return false
		}
	}

	return true
}

// tolerationMatches 检查容忍度是否匹配污点
// 根据Kubernetes的容忍度匹配规则进行匹配：
// - 检查Effect是否匹配
// - 检查Key和Value是否匹配
// - 支持Exists操作符的模糊匹配
//
// 参数:
//   - toleration: Pod的容忍度配置
//   - taint: 节点的污点配置
//
// 返回:
//   - bool: true表示匹配，false表示不匹配
func (s *TaintEffectService) tolerationMatches(toleration corev1.Toleration, taint corev1.Taint) bool {
	// 检查Effect
	if toleration.Effect != "" && toleration.Effect != taint.Effect {
		return false
	}

	// 检查Key和Value
	if toleration.Operator == corev1.TolerationOpExists {
		return toleration.Key == taint.Key || toleration.Key == ""
	} else {
		return toleration.Key == taint.Key && toleration.Value == taint.Value
	}
}

// processPodEviction 处理Pod驱逐
// 根据配置的驱逐策略执行Pod驱逐操作：
// - immediate: 立即驱逐
// - graceful: 优雅驱逐，考虑PodDisruptionBudget
// - delayed: 延迟驱逐，标记为待处理
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - pod: 要驱逐的Pod
//   - config: NoExecute配置
//   - gracePeriod: 优雅期时间
//   - forceEviction: 是否强制驱逐
//
// 返回:
//   - model.PodEvictionInfo: Pod驱逐信息
func (s *TaintEffectService) processPodEviction(ctx context.Context, kubeClient *kubernetes.Clientset, pod corev1.Pod, config *model.NoExecuteConfig, gracePeriod *int64, forceEviction bool) model.PodEvictionInfo {
	affectedPod := model.PodEvictionInfo{
		PodName:            pod.Name,
		Namespace:          pod.Namespace,
		EvictionReason:     "Pod does not tolerate NoExecute taint",
		RescheduleAttempts: 0,
		Status:             "pending",
	}

	// 确定驱逐策略
	strategy := config.EvictionPolicy.Strategy
	if strategy == "" {
		strategy = s.config.EffectManagement.NoExecute.MaxEvictionRate
	}

	// 根据策略处理驱逐
	switch strategy {
	case "immediate":
		if s.evictPodImmediately(ctx, kubeClient, pod, gracePeriod) {
			affectedPod.Status = "evicted"
			now := time.Now()
			affectedPod.EvictionTime = &now
		} else {
			affectedPod.Status = "failed"
		}

	case "graceful":
		if config.GracefulEviction {
			if s.evictPodGracefully(ctx, kubeClient, pod, config.EvictionTimeout) {
				affectedPod.Status = "evicted"
				now := time.Now()
				affectedPod.EvictionTime = &now
			} else {
				affectedPod.Status = "failed"
			}
		}

	case "delayed":
		// 延迟驱逐，标记为待处理
		affectedPod.Status = "pending"
		affectedPod.EvictionReason = "Delayed eviction scheduled"

	default:
		// 默认优雅驱逐
		if s.evictPodGracefully(ctx, kubeClient, pod, config.EvictionTimeout) {
			affectedPod.Status = "evicted"
			now := time.Now()
			affectedPod.EvictionTime = &now
		} else {
			affectedPod.Status = "failed"
		}
	}

	return affectedPod
}

// evictPodImmediately 立即驱逐Pod
// 使用Kubernetes的Eviction API立即驱逐Pod，不考虑PodDisruptionBudget
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - pod: 要驱逐的Pod
//   - gracePeriod: 优雅期时间
//
// 返回:
//   - bool: true表示驱逐成功，false表示失败
func (s *TaintEffectService) evictPodImmediately(ctx context.Context, kubeClient *kubernetes.Clientset, pod corev1.Pod, gracePeriod *int64) bool {
	s.logger.Info("立即驱逐Pod", zap.String("pod", pod.Name), zap.String("namespace", pod.Namespace))

	eviction := &policyv1.Eviction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
	}

	if gracePeriod != nil {
		eviction.DeleteOptions = &metav1.DeleteOptions{
			GracePeriodSeconds: gracePeriod,
		}
	}

	err := kubeClient.CoreV1().Pods(pod.Namespace).EvictV1(ctx, eviction)
	if err != nil {
		s.logger.Error("驱逐Pod失败", zap.Error(err))
		return false
	}

	return true
}

// evictPodGracefully 优雅驱逐Pod
// 在驱逐Pod之前检查PodDisruptionBudget，确保不会违反可用性要求
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - pod: 要驱逐的Pod
//   - timeout: 驱逐超时时间
//
// 返回:
//   - bool: true表示驱逐成功，false表示失败
func (s *TaintEffectService) evictPodGracefully(ctx context.Context, kubeClient *kubernetes.Clientset, pod corev1.Pod, timeout *int64) bool {
	s.logger.Info("优雅驱逐Pod", zap.String("pod", pod.Name), zap.String("namespace", pod.Namespace))

	// 使用PodDisruptionBudget检查
	if s.checkPodDisruptionBudget(ctx, kubeClient, pod) {
		return s.evictPodImmediately(ctx, kubeClient, pod, timeout)
	}

	s.logger.Warn("Pod驱逐被PodDisruptionBudget阻止", zap.String("pod", pod.Name))
	return false
}

// checkPodDisruptionBudget 检查PodDisruptionBudget
// 检查Pod是否受到PodDisruptionBudget保护，确保驱逐不会违反可用性要求
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - pod: 要检查的Pod
//
// 返回:
//   - bool: true表示可以驱逐，false表示不能驱逐
func (s *TaintEffectService) checkPodDisruptionBudget(ctx context.Context, kubeClient *kubernetes.Clientset, pod corev1.Pod) bool {
	// 获取相关的PodDisruptionBudget
	pdbList, err := kubeClient.PolicyV1().PodDisruptionBudgets(pod.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		s.logger.Warn("获取PodDisruptionBudget失败", zap.Error(err))
		return true // 如果无法获取PDB，允许驱逐
	}

	for _, pdb := range pdbList.Items {
		if s.pdbAppliesToPod(pdb, pod) {
			// 检查是否有足够的可用副本
			if pdb.Status.DisruptionsAllowed <= 0 {
				return false
			}
		}
	}

	return true
}

// pdbAppliesToPod 检查PDB是否适用于Pod
// 检查PodDisruptionBudget的选择器是否匹配目标Pod
//
// 参数:
//   - pdb: PodDisruptionBudget对象
//   - pod: 目标Pod
//
// 返回:
//   - bool: true表示PDB适用于该Pod，false表示不适用
func (s *TaintEffectService) pdbAppliesToPod(pdb policyv1.PodDisruptionBudget, pod corev1.Pod) bool {
	if pdb.Spec.Selector == nil {
		return false
	}

	selector, err := metav1.LabelSelectorAsSelector(pdb.Spec.Selector)
	if err != nil {
		return false
	}

	return selector.Matches(labels.Set(pod.Labels))
}

// buildLabelSelector 构建标签选择器
// 将map格式的节点选择器转换为Kubernetes标签选择器字符串
//
// 参数:
//   - nodeSelector: 节点选择器map
//
// 返回:
//   - string: 格式化的标签选择器字符串
func (s *TaintEffectService) buildLabelSelector(nodeSelector map[string]string) string {
	var parts []string
	for key, value := range nodeSelector {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(parts, ",")
}

// ConvertTaintEffect 转换污点效果
// 支持污点效果之间的转换，如从NoSchedule转换为PreferNoSchedule
// 主要功能：
// - 检查效果转换是否启用
// - 应用转换规则
// - 评估转换条件
// - 更新节点污点配置
//
// 参数:
//   - ctx: 上下文对象
//   - req: 污点效果转换请求
//
// 返回:
//   - *model.K8sTaintEffectManagementResponse: 转换结果
//   - error: 转换过程中的错误
func (s *TaintEffectService) ConvertTaintEffect(ctx context.Context, req *model.K8sTaintEffectManagementRequest) (*model.K8sTaintEffectManagementResponse, error) {
	s.logger.Info("开始转换污点效果", zap.Int("cluster_id", req.ClusterID))

	// 检查是否允许效果转换
	if !req.TaintEffectConfig.EffectTransition.AllowTransition {
		return nil, fmt.Errorf("效果转换未启用")
	}

	// 获取Kubernetes客户端
	kubeClient, err := utils.GetKubeClient(req.ClusterID, s.k8sClient, s.logger)
	if err != nil {
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	var effectChanges []model.EffectChange

	// 获取目标节点
	nodes, err := s.getTargetNodes(ctx, kubeClient, req)
	if err != nil {
		return nil, fmt.Errorf("获取目标节点失败: %w", err)
	}

	// 处理每个节点的效果转换
	for _, node := range nodes {
		changes := s.processEffectTransition(ctx, kubeClient, &node, &req.TaintEffectConfig.EffectTransition)
		effectChanges = append(effectChanges, changes...)
	}

	response := &model.K8sTaintEffectManagementResponse{
		NodeName:      req.NodeName,
		EffectChanges: effectChanges,
		OperationTime: time.Now(),
		Status:        "success",
	}

	return response, nil
}

// processEffectTransition 处理效果转换
// 根据转换规则自动转换污点效果，支持条件化转换
//
// 参数:
//   - ctx: 上下文对象
//   - kubeClient: Kubernetes客户端
//   - node: 目标节点
//   - transition: 效果转换配置
//
// 返回:
//   - []model.EffectChange: 效果变化记录
func (s *TaintEffectService) processEffectTransition(ctx context.Context, kubeClient *kubernetes.Clientset, node *corev1.Node, transition *model.EffectTransition) []model.EffectChange {
	var effectChanges []model.EffectChange

	// 应用转换规则
	for _, rule := range transition.TransitionRules {
		if rule.AutoApply {
			for i, taint := range node.Spec.Taints {
				if string(taint.Effect) == rule.FromEffect {
					// 检查转换条件
					if s.evaluateTransitionCondition(rule.Condition, node) {
						oldEffect := string(taint.Effect)
						node.Spec.Taints[i].Effect = corev1.TaintEffect(rule.ToEffect)

						effectChange := model.EffectChange{
							TaintKey:     taint.Key,
							OldEffect:    oldEffect,
							NewEffect:    rule.ToEffect,
							ChangeReason: fmt.Sprintf("Auto transition: %s", rule.Condition),
							ChangeTime:   time.Now(),
						}
						effectChanges = append(effectChanges, effectChange)
					}
				}
			}
		}
	}

	// 如果有变化，更新节点
	if len(effectChanges) > 0 {
		_, err := kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err != nil {
			s.logger.Error("更新节点污点效果失败", zap.Error(err))
		}
	}

	return effectChanges
}

// evaluateTransitionCondition 评估转换条件
// 根据节点状态评估是否满足转换条件，支持多种条件类型：
// - node_ready: 节点就绪状态
// - node_not_ready: 节点未就绪状态
// - memory_pressure: 内存压力
// - disk_pressure: 磁盘压力
//
// 参数:
//   - condition: 转换条件字符串
//   - node: 目标节点
//
// 返回:
//   - bool: true表示满足条件，false表示不满足
func (s *TaintEffectService) evaluateTransitionCondition(condition string, node *corev1.Node) bool {
	// 简单的条件评估逻辑，可以根据需要扩展
	switch condition {
	case "node_ready":
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				return true
			}
		}
		return false

	case "node_not_ready":
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status != corev1.ConditionTrue {
				return true
			}
		}
		return false

	case "memory_pressure":
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeMemoryPressure && cond.Status == corev1.ConditionTrue {
				return true
			}
		}
		return false

	case "disk_pressure":
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeDiskPressure && cond.Status == corev1.ConditionTrue {
				return true
			}
		}
		return false

	default:
		// 自定义条件可以在这里扩展
		return false
	}
}

// GetEffectManagementStatus 获取效果管理状态
// 获取指定节点的当前污点效果管理状态，包括：
// - 当前活跃的污点效果
// - 受影响的Pod列表
// - Pod的容忍度状态
// - 驱逐风险分析
//
// 参数:
//   - ctx: 上下文对象
//   - clusterID: 集群ID
//   - nodeName: 节点名称
//
// 返回:
//   - *model.K8sTaintEffectManagementResponse: 效果管理状态
//   - error: 获取状态过程中的错误
func (s *TaintEffectService) GetEffectManagementStatus(ctx context.Context, clusterID int, nodeName string) (*model.K8sTaintEffectManagementResponse, error) {
	s.logger.Info("获取效果管理状态", zap.Int("cluster_id", clusterID), zap.String("node", nodeName))

	// 获取Kubernetes客户端
	kubeClient, err := utils.GetKubeClient(clusterID, s.k8sClient, s.logger)
	if err != nil {
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 获取节点
	node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点失败: %w", err)
	}

	// 获取节点上的Pod
	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		return nil, fmt.Errorf("获取节点Pod失败: %w", err)
	}

	// 分析当前效果状态
	var effectChanges []model.EffectChange
	var affectedPods []model.PodEvictionInfo

	for _, taint := range node.Spec.Taints {
		effectChange := model.EffectChange{
			TaintKey:     taint.Key,
			OldEffect:    "N/A",
			NewEffect:    string(taint.Effect),
			ChangeReason: "Current active taint effect",
			ChangeTime:   time.Now(),
		}
		effectChanges = append(effectChanges, effectChange)
	}

	// 分析受影响的Pod
	for _, pod := range pods.Items {
		toleratesAllTaints := true
		for _, taint := range node.Spec.Taints {
			if !s.checkPodToleratesTaint(pod, taint) {
				toleratesAllTaints = false
				break
			}
		}

		status := "running"
		evictionReason := "Pod tolerates all taints"
		if !toleratesAllTaints {
			status = "at_risk"
			evictionReason = "Pod may be evicted due to intolerant taints"
		}

		affectedPod := model.PodEvictionInfo{
			PodName:        pod.Name,
			Namespace:      pod.Namespace,
			EvictionReason: evictionReason,
			Status:         status,
		}
		affectedPods = append(affectedPods, affectedPod)
	}

	response := &model.K8sTaintEffectManagementResponse{
		NodeName:      nodeName,
		AffectedPods:  affectedPods,
		EffectChanges: effectChanges,
		OperationTime: time.Now(),
		Status:        "active",
	}

	return response, nil
}

// checkPodToleratesTaint 检查Pod是否容忍特定污点
// 检查Pod的容忍度配置是否能容忍指定的污点
//
// 参数:
//   - pod: 目标Pod
//   - taint: 要检查的污点
//
// 返回:
//   - bool: true表示Pod容忍该污点，false表示不能容忍
func (s *TaintEffectService) checkPodToleratesTaint(pod corev1.Pod, taint corev1.Taint) bool {
	for _, toleration := range pod.Spec.Tolerations {
		if s.tolerationMatches(toleration, taint) {
			return true
		}
	}
	return false
}

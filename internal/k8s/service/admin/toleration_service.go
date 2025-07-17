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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type TolerationService interface {
	AddTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) (*model.K8sTaintTolerationResponse, error)
	UpdateTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) (*model.K8sTaintTolerationResponse, error)
	DeleteTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) error
	ValidateTolerations(ctx context.Context, req *model.K8sTaintTolerationValidationRequest) (*model.K8sTaintTolerationValidationResponse, error)
	ListTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) (*model.K8sTaintTolerationResponse, error)
	ConfigTolerationTime(ctx context.Context, req *model.K8sTolerationTimeRequest) (*model.K8sTolerationTimeResponse, error)
	ValidateTolerationTime(ctx context.Context, req *model.K8sTolerationTimeRequest) (*model.K8sTolerationTimeResponse, error)
	BatchUpdateTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) (*model.K8sTaintTolerationResponse, error)
	CreateTolerationTemplate(ctx context.Context, req *model.K8sTolerationConfigRequest) (*model.K8sTolerationTemplate, error)
	GetTolerationTemplate(ctx context.Context, req *model.K8sTolerationConfigRequest) (*model.K8sTolerationTemplate, error)
	DeleteTolerationTemplate(ctx context.Context, req *model.K8sTolerationConfigRequest) error
	ManageTaintEffects(ctx context.Context, req *model.K8sTaintEffectManagementRequest) (*model.K8sTaintEffectManagementResponse, error)
	TransitionTaintEffect(ctx context.Context, req *model.K8sTaintEffectManagementRequest) (*model.K8sTaintEffectManagementResponse, error)
	ValidateTaintEffects(ctx context.Context, req *model.K8sTaintEffectManagementRequest) (*model.K8sTaintEffectManagementResponse, error)
	GetTaintEffectStatus(ctx context.Context, req *model.K8sTaintEffectManagementRequest) (*model.K8sTaintEffectManagementResponse, error)
	BatchManageTaintEffects(ctx context.Context, req *model.K8sTaintEffectManagementRequest) (*model.K8sTaintEffectManagementResponse, error)
}

type tolerationService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

func NewTolerationService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) TolerationService {
	return &tolerationService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

func (t *tolerationService) AddTolerations(ctx context.Context, req *model.K8sTaintTolerationRequest) (*model.K8sTaintTolerationResponse, error) {
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

	response.CompatibleNodes, err = t.findCompatibleNodes(ctx, kubeClient, req.Tolerations)
	if err != nil {
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

func (t *tolerationService) ManageTaintEffects(ctx context.Context, req *model.K8sTaintEffectManagementRequest) (*model.K8sTaintEffectManagementResponse, error) {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, t.client, t.logger)
	if err != nil {
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	response := &model.K8sTaintEffectManagementResponse{
		OperationTime: time.Now(),
		Status:        "processing",
	}

	if req.BatchOperation {
		return t.batchManageTaintEffects(ctx, kubeClient, req)
	}

	node, err := kubeClient.CoreV1().Nodes().Get(ctx, req.NodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点失败: %w", err)
	}

	response.NodeName = req.NodeName

	oldTaints := make([]corev1.Taint, len(node.Spec.Taints))
	copy(oldTaints, node.Spec.Taints)

	affectedPods, err := t.getAffectedPods(ctx, kubeClient, req.NodeName, req.TaintEffectConfig)
	if err != nil {
		return nil, err
	}

	response.AffectedPods = affectedPods

	newTaints := t.manageTaints(oldTaints, req.TaintEffectConfig)
	node.Spec.Taints = newTaints

	_, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("更新节点污点失败: %w", err)
	}

	response.EffectChanges = t.buildEffectChanges(oldTaints, newTaints)

	if req.TaintEffectConfig.NoExecuteConfig.Enabled {
		evictionSummary, err := t.handleNoExecuteEviction(ctx, kubeClient, req)
		if err != nil {
			response.Warnings = append(response.Warnings, fmt.Sprintf("处理NoExecute驱逐失败: %v", err))
		} else {
			response.EvictionSummary = *evictionSummary
		}
	}

	response.Status = "completed"
	return response, nil
}

func (t *tolerationService) TransitionTaintEffect(ctx context.Context, req *model.K8sTaintEffectManagementRequest) (*model.K8sTaintEffectManagementResponse, error) {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, t.client, t.logger)
	if err != nil {
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	response := &model.K8sTaintEffectManagementResponse{
		NodeName:      req.NodeName,
		OperationTime: time.Now(),
		Status:        "processing",
	}

	node, err := kubeClient.CoreV1().Nodes().Get(ctx, req.NodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点失败: %w", err)
	}

	oldTaints := make([]corev1.Taint, len(node.Spec.Taints))
	copy(oldTaints, node.Spec.Taints)

	newTaints := t.applyEffectTransitions(oldTaints, req.TaintEffectConfig.EffectTransition)

	if req.TaintEffectConfig.EffectTransition.TransitionDelay != nil {
		time.Sleep(time.Duration(*req.TaintEffectConfig.EffectTransition.TransitionDelay) * time.Second)
	}

	node.Spec.Taints = newTaints

	_, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("更新节点污点失败: %w", err)
	}

	response.EffectChanges = t.buildEffectChanges(oldTaints, newTaints)
	response.Status = "completed"

	return response, nil
}

func (t *tolerationService) ValidateTaintEffects(ctx context.Context, req *model.K8sTaintEffectManagementRequest) (*model.K8sTaintEffectManagementResponse, error) {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, t.client, t.logger)
	if err != nil {
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	response := &model.K8sTaintEffectManagementResponse{
		NodeName:      req.NodeName,
		OperationTime: time.Now(),
		Status:        "validated",
	}

	warnings := t.validateTaintEffectConfig(req.TaintEffectConfig)
	response.Warnings = warnings

	affectedPods, err := t.getAffectedPods(ctx, kubeClient, req.NodeName, req.TaintEffectConfig)
	if err != nil {
		return nil, err
	}

	response.AffectedPods = affectedPods

	response.EvictionSummary = model.EvictionSummary{
		TotalPods:        len(affectedPods),
		PendingEvictions: len(affectedPods),
	}

	if len(warnings) > 0 {
		response.Status = "validation_warnings"
	}

	return response, nil
}

func (t *tolerationService) GetTaintEffectStatus(ctx context.Context, req *model.K8sTaintEffectManagementRequest) (*model.K8sTaintEffectManagementResponse, error) {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, t.client, t.logger)
	if err != nil {
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	response := &model.K8sTaintEffectManagementResponse{
		NodeName:      req.NodeName,
		OperationTime: time.Now(),
		Status:        "active",
	}

	node, err := kubeClient.CoreV1().Nodes().Get(ctx, req.NodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点失败: %w", err)
	}

	response.EffectChanges = t.analyzeCurrentTaintEffects(node.Spec.Taints)

	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + req.NodeName,
	})
	if err != nil {
		return nil, fmt.Errorf("获取节点上的Pod失败: %w", err)
	}

	var affectedPods []model.PodEvictionInfo
	for _, pod := range pods.Items {
		if t.isPodAffectedByTaints(pod, node.Spec.Taints) {
			affectedPods = append(affectedPods, model.PodEvictionInfo{
				PodName:   pod.Name,
				Namespace: pod.Namespace,
				Status:    string(pod.Status.Phase),
			})
		}
	}

	response.AffectedPods = affectedPods
	response.EvictionSummary = model.EvictionSummary{
		TotalPods: len(affectedPods),
	}

	return response, nil
}

func (t *tolerationService) BatchManageTaintEffects(ctx context.Context, req *model.K8sTaintEffectManagementRequest) (*model.K8sTaintEffectManagementResponse, error) {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, t.client, t.logger)
	if err != nil {
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	return t.batchManageTaintEffects(ctx, kubeClient, req)
}

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
		defaultTime := int64(300)
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

func (t *tolerationService) getAffectedPods(ctx context.Context, kubeClient kubernetes.Interface, nodeName string, config model.K8sTaintEffectConfig) ([]model.PodEvictionInfo, error) {
	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	if err != nil {
		return nil, fmt.Errorf("获取节点上的Pod失败: %w", err)
	}

	var affectedPods []model.PodEvictionInfo
	for _, pod := range pods.Items {
		if t.isPodAffectedByTaintConfig(pod, config) {
			evictionInfo := model.PodEvictionInfo{
				PodName:        pod.Name,
				Namespace:      pod.Namespace,
				EvictionReason: t.getEvictionReason(pod, config),
				Status:         string(pod.Status.Phase),
			}
			affectedPods = append(affectedPods, evictionInfo)
		}
	}

	return affectedPods, nil
}

func (t *tolerationService) isPodAffectedByTaintConfig(pod corev1.Pod, config model.K8sTaintEffectConfig) bool {
	if config.NoExecuteConfig.Enabled {
		return true
	}

	if config.NoScheduleConfig.Enabled {
		for _, exceptionPod := range config.NoScheduleConfig.ExceptionPods {
			if pod.Name == exceptionPod {
				return false
			}
		}
		return true
	}

	return false
}

func (t *tolerationService) isPodAffectedByTaints(pod corev1.Pod, taints []corev1.Taint) bool {
	for _, taint := range taints {
		if taint.Effect == corev1.TaintEffectNoExecute {
			tolerationFound := false
			for _, toleration := range pod.Spec.Tolerations {
				if toleration.Key == taint.Key && string(toleration.Effect) == string(taint.Effect) {
					tolerationFound = true
					break
				}
			}
			if !tolerationFound {
				return true
			}
		}
	}
	return false
}

func (t *tolerationService) getEvictionReason(pod corev1.Pod, config model.K8sTaintEffectConfig) string {
	if config.NoExecuteConfig.Enabled {
		return "NoExecute污点效果"
	}
	if config.NoScheduleConfig.Enabled {
		return "NoSchedule污点效果"
	}
	return "未知原因"
}

func (t *tolerationService) manageTaints(oldTaints []corev1.Taint, config model.K8sTaintEffectConfig) []corev1.Taint {
	var newTaints []corev1.Taint

	for _, taint := range oldTaints {
		modifiedTaint := taint

		if config.NoScheduleConfig.Enabled && taint.Effect == corev1.TaintEffectNoSchedule {
			continue
		}

		if config.PreferNoScheduleConfig.Enabled && taint.Effect == corev1.TaintEffectPreferNoSchedule {
			continue
		}

		if config.NoExecuteConfig.Enabled && taint.Effect == corev1.TaintEffectNoExecute {
			continue
		}

		newTaints = append(newTaints, modifiedTaint)
	}

	if config.NoScheduleConfig.Enabled {
		newTaints = append(newTaints, corev1.Taint{
			Key:    "node.kubernetes.io/no-schedule",
			Value:  "true",
			Effect: corev1.TaintEffectNoSchedule,
		})
	}

	if config.PreferNoScheduleConfig.Enabled {
		newTaints = append(newTaints, corev1.Taint{
			Key:    "node.kubernetes.io/prefer-no-schedule",
			Value:  "true",
			Effect: corev1.TaintEffectPreferNoSchedule,
		})
	}

	if config.NoExecuteConfig.Enabled {
		newTaints = append(newTaints, corev1.Taint{
			Key:    "node.kubernetes.io/no-execute",
			Value:  "true",
			Effect: corev1.TaintEffectNoExecute,
		})
	}

	return newTaints
}

func (t *tolerationService) buildEffectChanges(oldTaints, newTaints []corev1.Taint) []model.EffectChange {
	var changes []model.EffectChange
	oldTaintMap := make(map[string]corev1.Taint)
	newTaintMap := make(map[string]corev1.Taint)

	for _, taint := range oldTaints {
		oldTaintMap[taint.Key] = taint
	}

	for _, taint := range newTaints {
		newTaintMap[taint.Key] = taint
	}

	for key, oldTaint := range oldTaintMap {
		if newTaint, exists := newTaintMap[key]; exists {
			if oldTaint.Effect != newTaint.Effect {
				changes = append(changes, model.EffectChange{
					TaintKey:     key,
					OldEffect:    string(oldTaint.Effect),
					NewEffect:    string(newTaint.Effect),
					ChangeReason: "效果转换",
					ChangeTime:   time.Now(),
				})
			}
		} else {
			changes = append(changes, model.EffectChange{
				TaintKey:     key,
				OldEffect:    string(oldTaint.Effect),
				NewEffect:    "",
				ChangeReason: "污点删除",
				ChangeTime:   time.Now(),
			})
		}
	}

	for key, newTaint := range newTaintMap {
		if _, exists := oldTaintMap[key]; !exists {
			changes = append(changes, model.EffectChange{
				TaintKey:     key,
				OldEffect:    "",
				NewEffect:    string(newTaint.Effect),
				ChangeReason: "污点添加",
				ChangeTime:   time.Now(),
			})
		}
	}

	return changes
}

func (t *tolerationService) handleNoExecuteEviction(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintEffectManagementRequest) (*model.EvictionSummary, error) {
	affectedPods, err := t.getAffectedPods(ctx, kubeClient, req.NodeName, req.TaintEffectConfig)
	if err != nil {
		return nil, err
	}

	evictionSummary := &model.EvictionSummary{
		TotalPods: len(affectedPods),
	}

	if !req.TaintEffectConfig.NoExecuteConfig.GracefulEviction {
		return evictionSummary, nil
	}

	g, ctx := errgroup.WithContext(ctx)

	for _, podInfo := range affectedPods {
		podInfo := podInfo
		g.Go(func() error {
			return t.evictPod(ctx, kubeClient, podInfo, req.TaintEffectConfig.NoExecuteConfig)
		})
	}

	if err := g.Wait(); err != nil {
		evictionSummary.FailedEvictions = len(affectedPods)
		return evictionSummary, err
	}

	evictionSummary.EvictedPods = len(affectedPods)
	return evictionSummary, nil
}

func (t *tolerationService) evictPod(ctx context.Context, kubeClient kubernetes.Interface, podInfo model.PodEvictionInfo, evictionConfig model.NoExecuteConfig) error {
	eviction := &policyv1.Eviction{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1",
			Kind:       "Eviction",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podInfo.PodName,
			Namespace: podInfo.Namespace,
		},
		DeleteOptions: &metav1.DeleteOptions{},
	}

	if evictionConfig.EvictionTimeout != nil {
		eviction.DeleteOptions.GracePeriodSeconds = evictionConfig.EvictionTimeout
	}

	return kubeClient.PolicyV1().Evictions(podInfo.Namespace).Evict(ctx, eviction)
}

func (t *tolerationService) applyEffectTransitions(oldTaints []corev1.Taint, transition model.EffectTransition) []corev1.Taint {
	if !transition.AllowTransition {
		return oldTaints
	}

	var newTaints []corev1.Taint

	for _, taint := range oldTaints {
		modifiedTaint := taint

		for _, rule := range transition.TransitionRules {
			if string(taint.Effect) == rule.FromEffect && rule.AutoApply {
				modifiedTaint.Effect = corev1.TaintEffect(rule.ToEffect)
				break
			}
		}

		newTaints = append(newTaints, modifiedTaint)
	}

	return newTaints
}

func (t *tolerationService) validateTaintEffectConfig(config model.K8sTaintEffectConfig) []string {
	var warnings []string

	if config.NoExecuteConfig.Enabled && config.NoExecuteConfig.EvictionTimeout == nil {
		warnings = append(warnings, "NoExecute配置已启用但未设置驱逐超时时间")
	}

	if config.PreferNoScheduleConfig.Enabled && config.PreferNoScheduleConfig.PreferenceWeight <= 0 {
		warnings = append(warnings, "PreferNoSchedule配置已启用但偏好权重无效")
	}

	if config.EffectTransition.AllowTransition && len(config.EffectTransition.TransitionRules) == 0 {
		warnings = append(warnings, "效果转换已启用但未定义转换规则")
	}

	return warnings
}

func (t *tolerationService) analyzeCurrentTaintEffects(taints []corev1.Taint) []model.EffectChange {
	var changes []model.EffectChange

	for _, taint := range taints {
		changes = append(changes, model.EffectChange{
			TaintKey:     taint.Key,
			NewEffect:    string(taint.Effect),
			ChangeReason: "当前状态",
			ChangeTime:   time.Now(),
		})
	}

	return changes
}

func (t *tolerationService) batchManageTaintEffects(ctx context.Context, kubeClient kubernetes.Interface, req *model.K8sTaintEffectManagementRequest) (*model.K8sTaintEffectManagementResponse, error) {
	var nodes []corev1.Node

	if req.NodeSelector != nil && len(req.NodeSelector) > 0 {
		labelSelector := labels.SelectorFromSet(req.NodeSelector)
		nodeList, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector.String(),
		})
		if err != nil {
			return nil, fmt.Errorf("获取节点列表失败: %w", err)
		}
		nodes = nodeList.Items
	} else {
		nodeList, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取节点列表失败: %w", err)
		}
		nodes = nodeList.Items
	}

	response := &model.K8sTaintEffectManagementResponse{
		OperationTime: time.Now(),
		Status:        "processing",
	}

	var totalAffectedPods []model.PodEvictionInfo
	var totalEffectChanges []model.EffectChange

	g, ctx := errgroup.WithContext(ctx)

	for _, node := range nodes {
		node := node
		g.Go(func() error {
			nodeReq := *req
			nodeReq.NodeName = node.Name
			nodeReq.BatchOperation = false

			nodeResponse, err := t.ManageTaintEffects(ctx, &nodeReq)
			if err != nil {
				return err
			}

			totalAffectedPods = append(totalAffectedPods, nodeResponse.AffectedPods...)
			totalEffectChanges = append(totalEffectChanges, nodeResponse.EffectChanges...)

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		response.Status = "failed"
		response.Warnings = append(response.Warnings, fmt.Sprintf("批量操作失败: %v", err))
		return response, err
	}

	response.AffectedPods = totalAffectedPods
	response.EffectChanges = totalEffectChanges
	response.EvictionSummary = model.EvictionSummary{
		TotalPods:   len(totalAffectedPods),
		EvictedPods: len(totalAffectedPods),
	}
	response.Status = "completed"

	return response, nil
}

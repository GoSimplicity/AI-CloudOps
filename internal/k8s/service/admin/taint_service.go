package admin

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

import (
	"context"
	"errors"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type TaintService interface {
	// CheckTaintYaml 检查 Taint YAML 配置是否合法
	CheckTaintYaml(ctx context.Context, taint *model.TaintK8sNodesRequest) error
	// BatchEnableSwitchNodes 批量启用或禁用节点
	BatchEnableSwitchNodes(ctx context.Context, req *model.ScheduleK8sNodesRequest) error
	// AddOrUpdateNodeTaint 添加或更新节点的 Taint
	AddOrUpdateNodeTaint(ctx context.Context, taint *model.TaintK8sNodesRequest) error
	// DrainPods 驱逐 Pod
	DrainPods(ctx context.Context, req *model.K8sClusterNodesRequest) error
}

type taintService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	l      *zap.Logger
}

func NewTaintService(dao admin.ClusterDAO, client client.K8sClient, l *zap.Logger) TaintService {
	return &taintService{
		dao:    dao,
		client: client,
		l:      l,
	}
}

// CheckTaintYaml 检查 Taint YAML 配置是否合法
func (t *taintService) CheckTaintYaml(ctx context.Context, req *model.TaintK8sNodesRequest) error {
	var taintsToProcess []corev1.Taint
	if err := yaml.UnmarshalStrict([]byte(req.TaintYaml), &taintsToProcess); err != nil {
		t.l.Error("解析 Taint YAML 配置失败", zap.Error(err))
		return err
	}

	// 检查重复 Taint 键
	taintsKey := make(map[string]struct{})
	for _, taint := range taintsToProcess {
		if _, exists := taintsKey[taint.Key]; exists {
			return constants.ErrorTaintsKeyDuplicate
		}
		taintsKey[taint.Key] = struct{}{}
	}

	cluster, err := t.dao.GetClusterByID(ctx, req.ClusterId)
	if err != nil {
		return err
	}

	kubeClient, err := t.client.GetKubeClient(cluster.ID)
	if err != nil {
		return err
	}

	var errs []error
	for _, nodeName := range req.NodeNames {
		if _, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{}); err != nil {
			t.l.Error("获取节点信息失败", zap.Error(err))
			errs = append(errs, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("在处理节点 Taints 时遇到以下错误: %v", errs)
	}

	return nil
}

// BatchEnableSwitchNodes 批量启用或禁用节点
func (t *taintService) BatchEnableSwitchNodes(ctx context.Context, req *model.ScheduleK8sNodesRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterId, t.client, t.l)
	if err != nil {
		t.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	var errs []error
	for _, nodeName := range req.NodeNames {
		node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			errs = append(errs, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err))
			t.l.Error("获取节点信息失败", zap.Error(err))
			continue
		}

		node.Spec.Unschedulable = !req.ScheduleEnable
		if _, err := kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
			t.l.Error("更新节点信息失败", zap.Error(err))
			errs = append(errs, fmt.Errorf("更新节点 %s 信息失败: %w", nodeName, err))
			continue
		}

		t.l.Info("更新节点调度状态成功", zap.String("nodeName", nodeName))
	}

	if len(errs) > 0 {
		return fmt.Errorf("在处理节点调度状态时遇到以下错误: %v", errs)
	}

	return nil
}

// AddOrUpdateNodeTaint 更新节点的 Taint
func (t *taintService) AddOrUpdateNodeTaint(ctx context.Context, req *model.TaintK8sNodesRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterId, t.client, t.l)
	if err != nil {
		t.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	var taintsToProcess []corev1.Taint
	if err := yaml.UnmarshalStrict([]byte(req.TaintYaml), &taintsToProcess); err != nil {
		t.l.Error("解析 Taint YAML 配置失败", zap.Error(err))
		return err
	}

	var errs []error
	for _, nodeName := range req.NodeNames {
		node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			errs = append(errs, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err))
			t.l.Error("获取节点信息失败", zap.Error(err))
			continue
		}

		switch req.ModType {
		case "add":
			node.Spec.Taints = pkg.MergeTaints(node.Spec.Taints, taintsToProcess)
		case "del":
			node.Spec.Taints = pkg.RemoveTaints(node.Spec.Taints, taintsToProcess)
		default:
			errMsg := fmt.Sprintf("未知的修改类型: %s", req.ModType)
			t.l.Error(errMsg)
			errs = append(errs, errors.New(errMsg))
			continue
		}

		if _, err := kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
			t.l.Error("更新节点信息失败", zap.Error(err))
			errs = append(errs, fmt.Errorf("更新节点 %s 信息失败: %w", nodeName, err))
			continue
		}

		t.l.Info("更新节点Taint成功", zap.String("nodeName", nodeName))
	}

	if len(errs) > 0 {
		return fmt.Errorf("在处理节点 Taints 时遇到以下错误: %v", errs)
	}

	return nil
}

// DrainPods 并发驱逐 Pods
func (t *taintService) DrainPods(ctx context.Context, req *model.K8sClusterNodesRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterId, t.client, t.l)
	if err != nil {
		t.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 获取指定节点的 Pod 列表
	pods, err := pkg.GetPodsByNodeName(ctx, kubeClient, req.NodeNames[0])
	if err != nil {
		t.l.Error("获取 Pod 列表失败", zap.Error(err))
		return err
	}

	// 设置驱逐参数
	evictionTemplate := &policyv1.Eviction{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1",
			Kind:       "Eviction",
		},
		DeleteOptions: &metav1.DeleteOptions{
			GracePeriodSeconds: new(int64),
		},
	}

	// 使用 errgroup 控制并发驱逐 Pods
	var errs []error

	g, ctx := errgroup.WithContext(ctx)
	for _, pod := range pods.Items {
		pod := pod // 避免闭包引用问题
		g.Go(func() error {
			eviction := evictionTemplate.DeepCopy()
			eviction.Name = pod.Name
			eviction.Namespace = pod.Namespace

			if err := kubeClient.PolicyV1().Evictions(eviction.Namespace).Evict(ctx, eviction); err != nil {
				t.l.Error("驱逐 Pod 失败", zap.Error(err), zap.String("podName", pod.Name))
				return fmt.Errorf("驱逐 Pod %s 失败: %w", pod.Name, err)
			}

			t.l.Debug("驱逐 Pod 成功", zap.String("podName", pod.Name))
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("在驱逐 Pod 时遇到以下错误: %v", errs)
	}

	return nil
}

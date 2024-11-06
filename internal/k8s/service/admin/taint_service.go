package admin

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
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type TaintService interface {
	// CheckTaintYaml 检查 Taint Yaml 是否合法
	CheckTaintYaml(ctx context.Context, taint *model.TaintK8sNodesRequest) error
	// BatchEnableSwitchNodes 批量启用或切换 Kubernetes 节点调度
	BatchEnableSwitchNodes(ctx context.Context, req *model.ScheduleK8sNodesRequest) error
	// UpdateNodeTaint 添加或者删除指定节点 Taint
	UpdateNodeTaint(ctx context.Context, taint *model.TaintK8sNodesRequest) error
	// DrainPods 删除指定 Node 上的 Pod
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

func (t *taintService) CheckTaintYaml(ctx context.Context, taint *model.TaintK8sNodesRequest) error {
	var taintsToProcess []corev1.Taint

	// 检查 YAML 配置是否合法
	if err := yaml.UnmarshalStrict([]byte(taint.TaintYaml), &taintsToProcess); err != nil {
		t.l.Error("解析 Taint YAML 配置失败", zap.Error(err))
		return err
	}

	// 检查 Taint key 是否重复
	taintsKey := make(map[string]struct{})
	for _, taint := range taintsToProcess {
		if _, ok := taintsKey[taint.Key]; ok {
			return constants.ErrorTaintsKeyDuplicate
		}
		taintsKey[taint.Key] = struct{}{}
	}

	// 获取集群信息
	cluster, err := t.dao.GetClusterByName(ctx, taint.ClusterName)
	if err != nil {
		return err
	}

	kubeClient, err := t.client.GetKubeClient(cluster.ID)
	if err != nil {
		return err
	}

	// 检查节点信息
	var errs []error
	for _, nodeName := range taint.NodeNames {
		if _, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{}); err != nil {
			t.l.Error("获取节点信息失败", zap.Error(err))
			errs = append(errs, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err))
		}
	}

	// 如果有错误，则返回
	if len(errs) > 0 {
		return fmt.Errorf("在处理节点 Taints 时遇到以下错误: %v", errs)
	}

	return nil
}

func (t *taintService) BatchEnableSwitchNodes(ctx context.Context, req *model.ScheduleK8sNodesRequest) error {
	kubeClient, err := pkg.GetKubeClient(ctx, req.ClusterName, t.dao, t.client, t.l)
	if err != nil {
		t.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 遍历每个节点并更新调度状态
	var errs []error
	for _, nodeName := range req.NodeNames {
		node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			errs = append(errs, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err))
			t.l.Error("获取节点信息失败", zap.Error(err))
			continue
		}

		// 更新节点调度状态
		node.Spec.Unschedulable = !req.ScheduleEnable

		// 更新节点信息
		if _, err := kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
			t.l.Error("更新节点信息失败", zap.Error(err))
			errs = append(errs, fmt.Errorf("更新节点 %s 信息失败: %w", nodeName, err))
			continue
		}

		t.l.Info("更新节点调度状态成功", zap.String("nodeName", nodeName))
	}

	// 返回遇到的所有错误
	if len(errs) > 0 {
		return fmt.Errorf("在处理节点调度状态时遇到以下错误: %v", errs)
	}

	return nil
}

func (t *taintService) UpdateNodeTaint(ctx context.Context, taintResource *model.TaintK8sNodesRequest) error {
	kubeClient, err := pkg.GetKubeClient(ctx, taintResource.ClusterName, t.dao, t.client, t.l)
	if err != nil {
		t.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 解析 YAML 配置中的 Taints
	var taintsToProcess []corev1.Taint
	if err := yaml.UnmarshalStrict([]byte(taintResource.TaintYaml), &taintsToProcess); err != nil {
		t.l.Error("解析 Taint YAML 配置失败", zap.Error(err))
		return err
	}

	// 遍历每个节点进行处理
	var errs []error
	for _, nodeName := range taintResource.NodeNames {
		node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			errs = append(errs, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err))
			t.l.Error("获取节点信息失败", zap.Error(err))
			continue
		}

		switch taintResource.ModType {
		case "add":
			// 添加新的 Taints
			node.Spec.Taints = pkg.MergeTaints(node.Spec.Taints, taintsToProcess)
		case "del":
			// 删除指定的 Taints
			node.Spec.Taints = pkg.RemoveTaints(node.Spec.Taints, taintsToProcess)
		default:
			// 处理未知的修改类型
			errMsg := fmt.Sprintf("未知的修改类型: %s", taintResource.ModType)
			t.l.Error(errMsg)
			errs = append(errs, errors.New(errMsg))
			continue
		}

		// 更新节点信息
		if _, err := kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
			t.l.Error("更新节点信息失败", zap.Error(err))
			errs = append(errs, fmt.Errorf("更新节点 %s 信息失败: %w", nodeName, err))
			continue
		}

		t.l.Info("更新节点Taint成功", zap.String("nodeName", nodeName))
	}

	// 返回遇到的所有错误
	if len(errs) > 0 {
		return fmt.Errorf("在处理节点 Taints 时遇到以下错误: %v", errs)
	}

	return nil
}

func (t *taintService) DrainPods(ctx context.Context, req *model.K8sClusterNodesRequest) error {
	kubeClient, err := pkg.GetKubeClient(ctx, req.ClusterName, t.dao, t.client, t.l)
	if err != nil {
		t.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 获取指定节点的 pods
	pods, err := pkg.GetPodsByNodeName(ctx, kubeClient, req.NodeNames[0])
	if err != nil {
		t.l.Error("获取 Pod 列表失败", zap.Error(err))
		return err
	}

	// 创建 Eviction 对象
	eviction := &policyv1.Eviction{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1",
			Kind:       "Eviction",
		},
		DeleteOptions: &metav1.DeleteOptions{
			GracePeriodSeconds: new(int64),
		},
	}

	// 遍历每个 Pod 并驱逐
	var errs []error
	for _, pod := range pods.Items {
		eviction.Name = pod.Name
		eviction.Namespace = pod.Namespace

		// 驱逐 Pod
		if err := kubeClient.PolicyV1().Evictions(eviction.Namespace).Evict(ctx, eviction); err != nil {
			t.l.Error("驱逐 Pod 失败", zap.Error(err), zap.String("podName", pod.Name))
			errs = append(errs, fmt.Errorf("驱逐 Pod %s 失败: %w", pod.Name, err))
			continue
		}

		t.l.Info("驱逐 Pod 成功", zap.String("podName", pod.Name))
	}

	// 返回遇到的所有错误
	if len(errs) > 0 {
		return fmt.Errorf("在驱逐 Pod 时遇到以下错误: %v", errs)
	}

	return nil
}

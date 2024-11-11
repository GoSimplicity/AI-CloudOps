package admin

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type TaintService interface {
	CheckTaintYaml(ctx context.Context, taint *model.TaintK8sNodesRequest) error
	BatchEnableSwitchNodes(ctx context.Context, req *model.ScheduleK8sNodesRequest) error
	UpdateNodeTaint(ctx context.Context, taint *model.TaintK8sNodesRequest) error
	DrainPods(ctx context.Context, req *model.K8sClusterNodesRequest) error
}

type taintService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

func NewTaintService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) TaintService {
	return &taintService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// CheckTaintYaml 检查 Taint YAML 配置是否合法
func (t *taintService) CheckTaintYaml(ctx context.Context, taint *model.TaintK8sNodesRequest) error {
	//var taintsToProcess []corev1.Taint
	//
	//if err := yaml.UnmarshalStrict([]byte(taint.TaintYaml), &taintsToProcess); err != nil {
	//	t.logger.Error("解析 Taint YAML 配置失败", zap.Error(err))
	//	return err
	//}
	//
	//taintsKey := make(map[string]struct{})
	//for _, taint := range taintsToProcess {
	//	if _, exists := taintsKey[taint.Key]; exists {
	//		return constants.ErrorTaintsKeyDuplicate
	//	}
	//	taintsKey[taint.Key] = struct{}{}
	//}
	//
	//cluster, err := t.dao.GetClusterByName(ctx, taint.ClusterName)
	//if err != nil {
	//	return err
	//}
	//
	//kubeClient, err := t.client.GetKubeClient(cluster.ID)
	//if err != nil {
	//	return err
	//}
	//
	//var errs []error
	//for _, nodeName := range taint.NodeNames {
	//	if _, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{}); err != nil {
	//		t.logger.Error("获取节点信息失败", zap.Error(err))
	//		errs = append(errs, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err))
	//	}
	//}
	//
	//if len(errs) > 0 {
	//	return fmt.Errorf("在处理节点 Taints 时遇到以下错误: %v", errs)
	//}
	//
	//return nil
	return nil
}

// BatchEnableSwitchNodes 批量启用或禁用节点
func (t *taintService) BatchEnableSwitchNodes(ctx context.Context, req *model.ScheduleK8sNodesRequest) error {
	//kubeClient, err := pkg.GetKubeClient(ctx, req.ClusterName, t.dao, t.client, t.logger)
	//if err != nil {
	//	t.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
	//	return err
	//}
	//
	//var errs []error
	//for _, nodeName := range req.NodeNames {
	//	node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	//	if err != nil {
	//		errs = append(errs, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err))
	//		t.logger.Error("获取节点信息失败", zap.Error(err))
	//		continue
	//	}
	//
	//	node.Spec.Unschedulable = !req.ScheduleEnable
	//
	//	if _, err := kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
	//		t.logger.Error("更新节点信息失败", zap.Error(err))
	//		errs = append(errs, fmt.Errorf("更新节点 %s 信息失败: %w", nodeName, err))
	//		continue
	//	}
	//
	//	t.logger.Info("更新节点调度状态成功", zap.String("nodeName", nodeName))
	//}
	//
	//if len(errs) > 0 {
	//	return fmt.Errorf("在处理节点调度状态时遇到以下错误: %v", errs)
	//}
	//
	//return nil
	return nil
}

// UpdateNodeTaint 更新节点的 Taint
func (t *taintService) UpdateNodeTaint(ctx context.Context, taintResource *model.TaintK8sNodesRequest) error {
	//kubeClient, err := pkg.GetKubeClient(ctx, taintResource.ClusterName, t.dao, t.client, t.logger)
	//if err != nil {
	//	t.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
	//	return err
	//}
	//
	//var taintsToProcess []corev1.Taint
	//if err := yaml.UnmarshalStrict([]byte(taintResource.TaintYaml), &taintsToProcess); err != nil {
	//	t.logger.Error("解析 Taint YAML 配置失败", zap.Error(err))
	//	return err
	//}
	//
	//var errs []error
	//for _, nodeName := range taintResource.NodeNames {
	//	node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	//	if err != nil {
	//		errs = append(errs, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err))
	//		t.logger.Error("获取节点信息失败", zap.Error(err))
	//		continue
	//	}
	//
	//	switch taintResource.ModType {
	//	case "add":
	//		node.Spec.Taints = pkg.MergeTaints(node.Spec.Taints, taintsToProcess)
	//	case "del":
	//		node.Spec.Taints = pkg.RemoveTaints(node.Spec.Taints, taintsToProcess)
	//	default:
	//		errMsg := fmt.Sprintf("未知的修改类型: %s", taintResource.ModType)
	//		t.logger.Error(errMsg)
	//		errs = append(errs, errors.New(errMsg))
	//		continue
	//	}
	//
	//	if _, err := kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
	//		t.logger.Error("更新节点信息失败", zap.Error(err))
	//		errs = append(errs, fmt.Errorf("更新节点 %s 信息失败: %w", nodeName, err))
	//		continue
	//	}
	//
	//	t.logger.Info("更新节点Taint成功", zap.String("nodeName", nodeName))
	//}
	//
	//if len(errs) > 0 {
	//	return fmt.Errorf("在处理节点 Taints 时遇到以下错误: %v", errs)
	//}
	//
	//return nil
	return nil
}

// DrainPods 驱逐 Pod
func (t *taintService) DrainPods(ctx context.Context, req *model.K8sClusterNodesRequest) error {
	//kubeClient, err := pkg.GetKubeClient(ctx, req.ClusterName, t.dao, t.client, t.logger)
	//if err != nil {
	//	t.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
	//	return err
	//}
	//
	//pods, err := pkg.GetPodsByNodeName(ctx, kubeClient, req.NodeNames[0])
	//if err != nil {
	//	t.logger.Error("获取 Pod 列表失败", zap.Error(err))
	//	return err
	//}
	//
	//eviction := &policyv1.Eviction{
	//	TypeMeta: metav1.TypeMeta{
	//		APIVersion: "policy/v1",
	//		Kind:       "Eviction",
	//	},
	//	DeleteOptions: &metav1.DeleteOptions{
	//		GracePeriodSeconds: new(int64),
	//	},
	//}
	//
	//var errs []error
	//for _, pod := range pods.Items {
	//	eviction.Name = pod.Name
	//	eviction.Namespace = pod.Namespace
	//
	//	if err := kubeClient.PolicyV1().Evictions(eviction.Namespace).Evict(ctx, eviction); err != nil {
	//		t.logger.Error("驱逐 Pod 失败", zap.Error(err), zap.String("podName", pod.Name))
	//		errs = append(errs, fmt.Errorf("驱逐 Pod %s 失败: %w", pod.Name, err))
	//		continue
	//	}
	//
	//	t.logger.Info("驱逐 Pod 成功", zap.String("podName", pod.Name))
	//}
	//
	//if len(errs) > 0 {
	//	return fmt.Errorf("在驱逐 Pod 时遇到以下错误: %v", errs)
	//}
	//
	//return nil
	return nil
}

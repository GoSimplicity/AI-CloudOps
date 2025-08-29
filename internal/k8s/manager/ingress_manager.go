package manager

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"go.uber.org/zap"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// IngressManager Ingress 资源管理器
type IngressManager interface {
	// 基础 CRUD 操作
	CreateIngress(ctx context.Context, clusterID int, namespace string, ingress *networkingv1.Ingress) error
	GetIngress(ctx context.Context, clusterID int, namespace, name string) (*networkingv1.Ingress, error)
	GetIngressList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*networkingv1.IngressList, error)
	UpdateIngress(ctx context.Context, clusterID int, namespace string, ingress *networkingv1.Ingress) error
	DeleteIngress(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error

	// 批量操作
	BatchDeleteIngresses(ctx context.Context, clusterID int, namespace string, ingressNames []string) error

	// 高级功能
	PatchIngress(ctx context.Context, clusterID int, namespace, name string, data []byte, patchType string) (*networkingv1.Ingress, error)
	UpdateIngressStatus(ctx context.Context, clusterID int, namespace string, ingress *networkingv1.Ingress) error
}

type ingressManager struct {
	logger *zap.Logger
	client client.K8sClient
}

// NewIngressManager 创建新的 IngressManager 实例
func NewIngressManager(logger *zap.Logger, client client.K8sClient) IngressManager {
	return &ingressManager{
		logger: logger,
		client: client,
	}
}

// CreateIngress 创建Ingress
func (i *ingressManager) CreateIngress(ctx context.Context, clusterID int, namespace string, ingress *networkingv1.Ingress) error {
	kubeClient, err := i.client.GetKubeClient(clusterID)
	if err != nil {
		i.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.NetworkingV1().Ingresses(namespace).Create(ctx, ingress, metav1.CreateOptions{})
	if err != nil {
		i.logger.Error("创建Ingress失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", ingress.Name),
			zap.Error(err))
		return err
	}

	i.logger.Info("成功创建Ingress",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", ingress.Name))

	return nil
}

// GetIngress 获取指定Ingress
func (i *ingressManager) GetIngress(ctx context.Context, clusterID int, namespace, name string) (*networkingv1.Ingress, error) {
	kubeClient, err := i.client.GetKubeClient(clusterID)
	if err != nil {
		i.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	ingress, err := kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		i.logger.Error("获取Ingress失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, err
	}

	return ingress, nil
}

// GetIngressList 获取Ingress列表
func (i *ingressManager) GetIngressList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*networkingv1.IngressList, error) {
	kubeClient, err := i.client.GetKubeClient(clusterID)
	if err != nil {
		i.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	ingressList, err := kubeClient.NetworkingV1().Ingresses(namespace).List(ctx, listOptions)
	if err != nil {
		i.logger.Error("获取Ingress列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, err
	}

	i.logger.Debug("成功获取Ingress列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(ingressList.Items)))

	return ingressList, nil
}

// UpdateIngress 更新Ingress
func (i *ingressManager) UpdateIngress(ctx context.Context, clusterID int, namespace string, ingress *networkingv1.Ingress) error {
	kubeClient, err := i.client.GetKubeClient(clusterID)
	if err != nil {
		i.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.NetworkingV1().Ingresses(namespace).Update(ctx, ingress, metav1.UpdateOptions{})
	if err != nil {
		i.logger.Error("更新Ingress失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", ingress.Name),
			zap.Error(err))
		return err
	}

	i.logger.Info("成功更新Ingress",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", ingress.Name))

	return nil
}

// DeleteIngress 删除Ingress
func (i *ingressManager) DeleteIngress(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := i.client.GetKubeClient(clusterID)
	if err != nil {
		i.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	err = kubeClient.NetworkingV1().Ingresses(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		i.logger.Error("删除Ingress失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return err
	}

	i.logger.Info("成功删除Ingress",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	return nil
}

// BatchDeleteIngresses 批量删除Ingress
func (i *ingressManager) BatchDeleteIngresses(ctx context.Context, clusterID int, namespace string, ingressNames []string) error {
	kubeClient, err := i.client.GetKubeClient(clusterID)
	if err != nil {
		i.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	deleteOptions := metav1.DeleteOptions{}
	var failedDeletions []string

	for _, name := range ingressNames {
		err := kubeClient.NetworkingV1().Ingresses(namespace).Delete(ctx, name, deleteOptions)
		if err != nil {
			i.logger.Error("删除Ingress失败",
				zap.Int("clusterID", clusterID),
				zap.String("namespace", namespace),
				zap.String("name", name),
				zap.Error(err))
			failedDeletions = append(failedDeletions, name)
		} else {
			i.logger.Info("成功删除Ingress",
				zap.Int("clusterID", clusterID),
				zap.String("namespace", namespace),
				zap.String("name", name))
		}
	}

	if len(failedDeletions) > 0 {
		i.logger.Warn("部分Ingress删除失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Strings("failedDeletions", failedDeletions))
		return err // 返回最后一个错误
	}

	i.logger.Info("批量删除Ingress完成",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(ingressNames)))

	return nil
}

// PatchIngress 部分更新Ingress
func (i *ingressManager) PatchIngress(ctx context.Context, clusterID int, namespace, name string, data []byte, patchType string) (*networkingv1.Ingress, error) {
	kubeClient, err := i.client.GetKubeClient(clusterID)
	if err != nil {
		i.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	// 转换 patch 类型
	pt := types.PatchType(patchType)
	ingress, err := kubeClient.NetworkingV1().Ingresses(namespace).Patch(ctx, name, pt, data, metav1.PatchOptions{})
	if err != nil {
		i.logger.Error("Patch Ingress失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.String("patchType", patchType),
			zap.Error(err))
		return nil, err
	}

	i.logger.Info("成功Patch Ingress",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	return ingress, nil
}

// UpdateIngressStatus 更新Ingress状态
func (i *ingressManager) UpdateIngressStatus(ctx context.Context, clusterID int, namespace string, ingress *networkingv1.Ingress) error {
	kubeClient, err := i.client.GetKubeClient(clusterID)
	if err != nil {
		i.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.NetworkingV1().Ingresses(namespace).UpdateStatus(ctx, ingress, metav1.UpdateOptions{})
	if err != nil {
		i.logger.Error("更新Ingress状态失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", ingress.Name),
			zap.Error(err))
		return err
	}

	i.logger.Info("成功更新Ingress状态",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", ingress.Name))

	return nil
}

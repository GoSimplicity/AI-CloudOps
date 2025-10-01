package manager

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// PVCManager PersistentVolumeClaim 资源管理器
type PVCManager interface {
	// 基础 CRUD 操作
	CreatePVC(ctx context.Context, clusterID int, namespace string, pvc *corev1.PersistentVolumeClaim) error
	GetPVC(ctx context.Context, clusterID int, namespace, name string) (*corev1.PersistentVolumeClaim, error)
	GetPVCList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*corev1.PersistentVolumeClaimList, error)
	UpdatePVC(ctx context.Context, clusterID int, namespace string, pvc *corev1.PersistentVolumeClaim) error
	DeletePVC(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error

	// 批量操作
	BatchDeletePVCs(ctx context.Context, clusterID int, namespace string, pvcNames []string) error

	// 高级功能
	PatchPVC(ctx context.Context, clusterID int, namespace, name string, data []byte, patchType string) (*corev1.PersistentVolumeClaim, error)
	UpdatePVCStatus(ctx context.Context, clusterID int, namespace string, pvc *corev1.PersistentVolumeClaim) error

	// PVC 特定操作
	GetPVCsByStorageClass(ctx context.Context, clusterID int, namespace, storageClass string) (*corev1.PersistentVolumeClaimList, error)
	GetPendingPVCs(ctx context.Context, clusterID int, namespace string) (*corev1.PersistentVolumeClaimList, error)
	GetBoundPVCs(ctx context.Context, clusterID int, namespace string) (*corev1.PersistentVolumeClaimList, error)
	ExpandPVC(ctx context.Context, clusterID int, namespace, name string, newSize string) error
}

type pvcManager struct {
	logger *zap.Logger
	client client.K8sClient
}

// NewPVCManager 创建新的 PVCManager 实例
func NewPVCManager(logger *zap.Logger, client client.K8sClient) PVCManager {
	return &pvcManager{
		logger: logger,
		client: client,
	}
}

// CreatePVC 创建PersistentVolumeClaim
func (m *pvcManager) CreatePVC(ctx context.Context, clusterID int, namespace string, pvc *corev1.PersistentVolumeClaim) error {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建PersistentVolumeClaim失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", pvc.Name),
			zap.Error(err))
		return err
	}

	m.logger.Info("成功创建PersistentVolumeClaim",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", pvc.Name))

	return nil
}

// GetPVC 获取指定PersistentVolumeClaim
func (m *pvcManager) GetPVC(ctx context.Context, clusterID int, namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取PersistentVolumeClaim失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, err
	}

	return pvc, nil
}

// GetPVCList 获取PersistentVolumeClaim列表
func (m *pvcManager) GetPVCList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*corev1.PersistentVolumeClaimList, error) {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	pvcList, err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取PersistentVolumeClaim列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, err
	}

	m.logger.Debug("成功获取PersistentVolumeClaim列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(pvcList.Items)))

	return pvcList, nil
}

// UpdatePVC 更新PersistentVolumeClaim
func (m *pvcManager) UpdatePVC(ctx context.Context, clusterID int, namespace string, pvc *corev1.PersistentVolumeClaim) error {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.CoreV1().PersistentVolumeClaims(namespace).Update(ctx, pvc, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新PersistentVolumeClaim失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", pvc.Name),
			zap.Error(err))
		return err
	}

	m.logger.Info("成功更新PersistentVolumeClaim",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", pvc.Name))

	return nil
}

// DeletePVC 删除PersistentVolumeClaim
func (m *pvcManager) DeletePVC(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	err = kubeClient.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		m.logger.Error("删除PersistentVolumeClaim失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return err
	}

	m.logger.Info("成功删除PersistentVolumeClaim",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	return nil
}

// BatchDeletePVCs 批量删除PersistentVolumeClaim
func (m *pvcManager) BatchDeletePVCs(ctx context.Context, clusterID int, namespace string, pvcNames []string) error {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	deleteOptions := metav1.DeleteOptions{}
	var failedDeletions []string

	for _, name := range pvcNames {
		err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, deleteOptions)
		if err != nil {
			m.logger.Error("删除PersistentVolumeClaim失败",
				zap.Int("clusterID", clusterID),
				zap.String("namespace", namespace),
				zap.String("name", name),
				zap.Error(err))
			failedDeletions = append(failedDeletions, name)
		} else {
			m.logger.Info("成功删除PersistentVolumeClaim",
				zap.Int("clusterID", clusterID),
				zap.String("namespace", namespace),
				zap.String("name", name))
		}
	}

	if len(failedDeletions) > 0 {
		m.logger.Warn("部分PersistentVolumeClaim删除失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Strings("failedDeletions", failedDeletions))
		return err // 返回最后一个错误
	}

	m.logger.Info("批量删除PersistentVolumeClaim完成",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(pvcNames)))

	return nil
}

// PatchPVC 部分更新PersistentVolumeClaim
func (m *pvcManager) PatchPVC(ctx context.Context, clusterID int, namespace, name string, data []byte, patchType string) (*corev1.PersistentVolumeClaim, error) {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	// 转换 patch 类型
	pt := types.PatchType(patchType)
	pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).Patch(ctx, name, pt, data, metav1.PatchOptions{})
	if err != nil {
		m.logger.Error("Patch PersistentVolumeClaim失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.String("patchType", patchType),
			zap.Error(err))
		return nil, err
	}

	m.logger.Info("成功Patch PersistentVolumeClaim",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	return pvc, nil
}

// UpdatePVCStatus 更新PersistentVolumeClaim状态
func (m *pvcManager) UpdatePVCStatus(ctx context.Context, clusterID int, namespace string, pvc *corev1.PersistentVolumeClaim) error {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.CoreV1().PersistentVolumeClaims(namespace).UpdateStatus(ctx, pvc, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新PersistentVolumeClaim状态失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", pvc.Name),
			zap.Error(err))
		return err
	}

	m.logger.Info("成功更新PersistentVolumeClaim状态",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", pvc.Name))

	return nil
}

// GetPVCsByStorageClass 根据存储类获取PersistentVolumeClaim
func (m *pvcManager) GetPVCsByStorageClass(ctx context.Context, clusterID int, namespace, storageClass string) (*corev1.PersistentVolumeClaimList, error) {
	pvcList, err := m.GetPVCList(ctx, clusterID, namespace, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// 过滤指定存储类的PVC
	var filteredPVCs []corev1.PersistentVolumeClaim
	for _, pvc := range pvcList.Items {
		if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName == storageClass {
			filteredPVCs = append(filteredPVCs, pvc)
		}
	}

	filteredList := &corev1.PersistentVolumeClaimList{
		TypeMeta: pvcList.TypeMeta,
		ListMeta: pvcList.ListMeta,
		Items:    filteredPVCs,
	}

	m.logger.Debug("根据存储类过滤PVC",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("storageClass", storageClass),
		zap.Int("filteredCount", len(filteredPVCs)))

	return filteredList, nil
}

// GetPendingPVCs 获取Pending状态的PersistentVolumeClaim
func (m *pvcManager) GetPendingPVCs(ctx context.Context, clusterID int, namespace string) (*corev1.PersistentVolumeClaimList, error) {
	listOptions := metav1.ListOptions{
		FieldSelector: "status.phase=Pending",
	}

	return m.GetPVCList(ctx, clusterID, namespace, listOptions)
}

// GetBoundPVCs 获取Bound状态的PersistentVolumeClaim
func (m *pvcManager) GetBoundPVCs(ctx context.Context, clusterID int, namespace string) (*corev1.PersistentVolumeClaimList, error) {
	listOptions := metav1.ListOptions{
		FieldSelector: "status.phase=Bound",
	}

	return m.GetPVCList(ctx, clusterID, namespace, listOptions)
}

// ExpandPVC 扩容PersistentVolumeClaim
func (m *pvcManager) ExpandPVC(ctx context.Context, clusterID int, namespace, name, newSize string) error {
	// 获取PVC
	pvc, err := m.GetPVC(ctx, clusterID, namespace, name)
	if err != nil {
		return err
	}

	// 解析新的存储大小
	newQuantity, err := resource.ParseQuantity(newSize)
	if err != nil {
		m.logger.Error("解析存储大小失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.String("newSize", newSize),
			zap.Error(err))
		return err
	}

	// 更新PVC的存储请求
	if pvc.Spec.Resources.Requests == nil {
		pvc.Spec.Resources.Requests = make(corev1.ResourceList)
	}
	pvc.Spec.Resources.Requests[corev1.ResourceStorage] = newQuantity

	// 更新PVC
	err = m.UpdatePVC(ctx, clusterID, namespace, pvc)
	if err != nil {
		m.logger.Error("扩容PersistentVolumeClaim失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.String("newSize", newSize),
			zap.Error(err))
		return err
	}

	m.logger.Info("成功扩容PersistentVolumeClaim",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name),
		zap.String("newSize", newSize))

	return nil
}

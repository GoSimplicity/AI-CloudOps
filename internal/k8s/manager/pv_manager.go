package manager

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// PVManager PersistentVolume 资源管理器
type PVManager interface {
	// 基础 CRUD 操作
	CreatePV(ctx context.Context, clusterID int, pv *corev1.PersistentVolume) error
	GetPV(ctx context.Context, clusterID int, name string) (*corev1.PersistentVolume, error)
	GetPVList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) (*corev1.PersistentVolumeList, error)
	UpdatePV(ctx context.Context, clusterID int, pv *corev1.PersistentVolume) error
	DeletePV(ctx context.Context, clusterID int, name string, deleteOptions metav1.DeleteOptions) error

	// 批量操作
	BatchDeletePVs(ctx context.Context, clusterID int, pvNames []string) error

	// 高级功能
	PatchPV(ctx context.Context, clusterID int, name string, data []byte, patchType string) (*corev1.PersistentVolume, error)
	UpdatePVStatus(ctx context.Context, clusterID int, pv *corev1.PersistentVolume) error

	// PV 特定操作
	GetAvailablePVs(ctx context.Context, clusterID int) (*corev1.PersistentVolumeList, error)
	GetPVByStorageClass(ctx context.Context, clusterID int, storageClass string) (*corev1.PersistentVolumeList, error)
	ReclaimPV(ctx context.Context, clusterID int, name string) error
}

type pvManager struct {
	logger *zap.Logger
	client client.K8sClient
}

// NewPVManager 创建新的 PVManager 实例
func NewPVManager(logger *zap.Logger, client client.K8sClient) PVManager {
	return &pvManager{
		logger: logger,
		client: client,
	}
}

// CreatePV 创建PersistentVolume
func (m *pvManager) CreatePV(ctx context.Context, clusterID int, pv *corev1.PersistentVolume) error {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.CoreV1().PersistentVolumes().Create(ctx, pv, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建PersistentVolume失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", pv.Name),
			zap.Error(err))
		return err
	}

	m.logger.Info("成功创建PersistentVolume",
		zap.Int("clusterID", clusterID),
		zap.String("name", pv.Name))

	return nil
}

// GetPV 获取指定PersistentVolume
func (m *pvManager) GetPV(ctx context.Context, clusterID int, name string) (*corev1.PersistentVolume, error) {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	pv, err := kubeClient.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取PersistentVolume失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", name),
			zap.Error(err))
		return nil, err
	}

	return pv, nil
}

// GetPVList 获取PersistentVolume列表
func (m *pvManager) GetPVList(ctx context.Context, clusterID int, listOptions metav1.ListOptions) (*corev1.PersistentVolumeList, error) {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	pvList, err := kubeClient.CoreV1().PersistentVolumes().List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取PersistentVolume列表失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	m.logger.Debug("成功获取PersistentVolume列表",
		zap.Int("clusterID", clusterID),
		zap.Int("count", len(pvList.Items)))

	return pvList, nil
}

// UpdatePV 更新PersistentVolume
func (m *pvManager) UpdatePV(ctx context.Context, clusterID int, pv *corev1.PersistentVolume) error {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.CoreV1().PersistentVolumes().Update(ctx, pv, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新PersistentVolume失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", pv.Name),
			zap.Error(err))
		return err
	}

	m.logger.Info("成功更新PersistentVolume",
		zap.Int("clusterID", clusterID),
		zap.String("name", pv.Name))

	return nil
}

// DeletePV 删除PersistentVolume
func (m *pvManager) DeletePV(ctx context.Context, clusterID int, name string, deleteOptions metav1.DeleteOptions) error {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	err = kubeClient.CoreV1().PersistentVolumes().Delete(ctx, name, deleteOptions)
	if err != nil {
		m.logger.Error("删除PersistentVolume失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", name),
			zap.Error(err))
		return err
	}

	m.logger.Info("成功删除PersistentVolume",
		zap.Int("clusterID", clusterID),
		zap.String("name", name))

	return nil
}

// BatchDeletePVs 批量删除PersistentVolume
func (m *pvManager) BatchDeletePVs(ctx context.Context, clusterID int, pvNames []string) error {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	deleteOptions := metav1.DeleteOptions{}
	var failedDeletions []string

	for _, name := range pvNames {
		err := kubeClient.CoreV1().PersistentVolumes().Delete(ctx, name, deleteOptions)
		if err != nil {
			m.logger.Error("删除PersistentVolume失败",
				zap.Int("clusterID", clusterID),
				zap.String("name", name),
				zap.Error(err))
			failedDeletions = append(failedDeletions, name)
		} else {
			m.logger.Info("成功删除PersistentVolume",
				zap.Int("clusterID", clusterID),
				zap.String("name", name))
		}
	}

	if len(failedDeletions) > 0 {
		m.logger.Warn("部分PersistentVolume删除失败",
			zap.Int("clusterID", clusterID),
			zap.Strings("failedDeletions", failedDeletions))
		return err // 返回最后一个错误
	}

	m.logger.Info("批量删除PersistentVolume完成",
		zap.Int("clusterID", clusterID),
		zap.Int("count", len(pvNames)))

	return nil
}

// PatchPV 部分更新PersistentVolume
func (m *pvManager) PatchPV(ctx context.Context, clusterID int, name string, data []byte, patchType string) (*corev1.PersistentVolume, error) {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	// 转换 patch 类型
	pt := types.PatchType(patchType)
	pv, err := kubeClient.CoreV1().PersistentVolumes().Patch(ctx, name, pt, data, metav1.PatchOptions{})
	if err != nil {
		m.logger.Error("Patch PersistentVolume失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", name),
			zap.String("patchType", patchType),
			zap.Error(err))
		return nil, err
	}

	m.logger.Info("成功Patch PersistentVolume",
		zap.Int("clusterID", clusterID),
		zap.String("name", name))

	return pv, nil
}

// UpdatePVStatus 更新PersistentVolume状态
func (m *pvManager) UpdatePVStatus(ctx context.Context, clusterID int, pv *corev1.PersistentVolume) error {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	_, err = kubeClient.CoreV1().PersistentVolumes().UpdateStatus(ctx, pv, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新PersistentVolume状态失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", pv.Name),
			zap.Error(err))
		return err
	}

	m.logger.Info("成功更新PersistentVolume状态",
		zap.Int("clusterID", clusterID),
		zap.String("name", pv.Name))

	return nil
}

// GetAvailablePVs 获取可用的PersistentVolume
func (m *pvManager) GetAvailablePVs(ctx context.Context, clusterID int) (*corev1.PersistentVolumeList, error) {
	listOptions := metav1.ListOptions{
		FieldSelector: "status.phase=Available",
	}

	return m.GetPVList(ctx, clusterID, listOptions)
}

// GetPVByStorageClass 根据存储类获取PersistentVolume
func (m *pvManager) GetPVByStorageClass(ctx context.Context, clusterID int, storageClass string) (*corev1.PersistentVolumeList, error) {
	pvList, err := m.GetPVList(ctx, clusterID, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// 过滤指定存储类的PV
	var filteredPVs []corev1.PersistentVolume
	for _, pv := range pvList.Items {
		if pv.Spec.StorageClassName == storageClass {
			filteredPVs = append(filteredPVs, pv)
		}
	}

	filteredList := &corev1.PersistentVolumeList{
		TypeMeta: pvList.TypeMeta,
		ListMeta: pvList.ListMeta,
		Items:    filteredPVs,
	}

	m.logger.Debug("根据存储类过滤PV",
		zap.Int("clusterID", clusterID),
		zap.String("storageClass", storageClass),
		zap.Int("filteredCount", len(filteredPVs)))

	return filteredList, nil
}

// ReclaimPV 回收PersistentVolume
func (m *pvManager) ReclaimPV(ctx context.Context, clusterID int, name string) error {
	// 获取PV
	pv, err := m.GetPV(ctx, clusterID, name)
	if err != nil {
		return err
	}

	// 清空ClaimRef以回收PV
	pv.Spec.ClaimRef = nil

	// 更新PV
	err = m.UpdatePV(ctx, clusterID, pv)
	if err != nil {
		m.logger.Error("回收PersistentVolume失败",
			zap.Int("clusterID", clusterID),
			zap.String("name", name),
			zap.Error(err))
		return err
	}

	m.logger.Info("成功回收PersistentVolume",
		zap.Int("clusterID", clusterID),
		zap.String("name", name))

	return nil
}

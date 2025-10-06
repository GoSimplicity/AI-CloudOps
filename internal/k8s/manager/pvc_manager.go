package manager

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// PVCManager PersistentVolumeClaim 资源管理器
type PVCManager interface {
	CreatePVC(ctx context.Context, clusterID int, namespace string, pvc *corev1.PersistentVolumeClaim) error
	GetPVC(ctx context.Context, clusterID int, namespace, name string) (*corev1.PersistentVolumeClaim, error)
	GetPVCList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) (*corev1.PersistentVolumeClaimList, error)
	UpdatePVC(ctx context.Context, clusterID int, namespace string, pvc *corev1.PersistentVolumeClaim) error
	DeletePVC(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error

	BatchDeletePVCs(ctx context.Context, clusterID int, namespace string, pvcNames []string) error

	// 高级功能
	PatchPVC(ctx context.Context, clusterID int, namespace, name string, data []byte, patchType string) (*corev1.PersistentVolumeClaim, error)
	UpdatePVCStatus(ctx context.Context, clusterID int, namespace string, pvc *corev1.PersistentVolumeClaim) error

	GetPVCsByStorageClass(ctx context.Context, clusterID int, namespace, storageClass string) (*corev1.PersistentVolumeClaimList, error)
	GetPendingPVCs(ctx context.Context, clusterID int, namespace string) (*corev1.PersistentVolumeClaimList, error)
	GetBoundPVCs(ctx context.Context, clusterID int, namespace string) (*corev1.PersistentVolumeClaimList, error)
	ExpandPVC(ctx context.Context, clusterID int, namespace, name string, newSize string) error
	GetPVCPods(ctx context.Context, clusterID int, namespace, pvcName string) ([]corev1.Pod, error)
}

type pvcManager struct {
	logger *zap.Logger
	client client.K8sClient
}

func NewPVCManager(logger *zap.Logger, client client.K8sClient) PVCManager {
	return &pvcManager{
		logger: logger,
		client: client,
	}
}

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

func (m *pvcManager) GetPendingPVCs(ctx context.Context, clusterID int, namespace string) (*corev1.PersistentVolumeClaimList, error) {
	listOptions := metav1.ListOptions{
		FieldSelector: "status.phase=Pending",
	}

	return m.GetPVCList(ctx, clusterID, namespace, listOptions)
}

func (m *pvcManager) GetBoundPVCs(ctx context.Context, clusterID int, namespace string) (*corev1.PersistentVolumeClaimList, error) {
	listOptions := metav1.ListOptions{
		FieldSelector: "status.phase=Bound",
	}

	return m.GetPVCList(ctx, clusterID, namespace, listOptions)
}

// ExpandPVC 扩容PersistentVolumeClaim
func (m *pvcManager) ExpandPVC(ctx context.Context, clusterID int, namespace, name, newSize string) error {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return err
	}

	// 获取PVC
	pvc, err := m.GetPVC(ctx, clusterID, namespace, name)
	if err != nil {
		return err
	}

	// 检查PVC是否有StorageClass（动态配置的PVC才能扩容）
	if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName == "" {
		errMsg := fmt.Sprintf("PVC '%s' 未使用StorageClass，无法进行扩容。只有通过StorageClass动态配置的PVC才支持扩容功能。建议：创建一个新的使用StorageClass的PVC，然后迁移数据", name)
		m.logger.Error("PVC未使用StorageClass",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name))
		return fmt.Errorf("%s", errMsg)
	}

	// 获取StorageClass并检查是否支持卷扩容
	storageClass, err := kubeClient.StorageV1().StorageClasses().Get(ctx, *pvc.Spec.StorageClassName, metav1.GetOptions{})
	if err != nil {
		errMsg := fmt.Sprintf("PVC '%s' 使用的StorageClass '%s' 在集群中不存在。这可能是因为：1) StorageClass已被删除；2) PVC配置错误。请检查集群中可用的StorageClass列表（kubectl get storageclass），并确保PVC配置正确",
			name, *pvc.Spec.StorageClassName)
		m.logger.Error("获取StorageClass失败",
			zap.Int("clusterID", clusterID),
			zap.String("storageClass", *pvc.Spec.StorageClassName),
			zap.Error(err))
		return fmt.Errorf("%s", errMsg)
	}

	// 检查StorageClass是否支持卷扩容
	if storageClass.AllowVolumeExpansion == nil || !*storageClass.AllowVolumeExpansion {
		errMsg := fmt.Sprintf("StorageClass '%s' 不支持卷扩容功能。解决方案：\n1. 如果您是集群管理员，可以修改StorageClass，设置 allowVolumeExpansion: true（注意：某些存储驱动可能不支持此功能）\n2. 或者创建一个支持扩容的新StorageClass，然后创建新PVC并迁移数据\n3. 使用命令查看StorageClass详情：kubectl get storageclass %s -o yaml",
			*pvc.Spec.StorageClassName, *pvc.Spec.StorageClassName)
		m.logger.Error("StorageClass不支持卷扩容",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.String("storageClass", *pvc.Spec.StorageClassName))
		return fmt.Errorf("%s", errMsg)
	}

	// 解析新的存储大小
	newQuantity, err := resource.ParseQuantity(newSize)
	if err != nil {
		errMsg := fmt.Sprintf("存储容量格式不正确：'%s'。请使用正确的格式，例如：1Gi, 2Gi, 500Mi, 1Ti 等", newSize)
		m.logger.Error("解析存储大小失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.String("newSize", newSize),
			zap.Error(err))
		return fmt.Errorf("%s", errMsg)
	}

	// 获取当前容量
	currentQuantity := pvc.Spec.Resources.Requests[corev1.ResourceStorage]

	// 验证新容量是否大于当前容量
	if newQuantity.Cmp(currentQuantity) <= 0 {
		errMsg := fmt.Sprintf("扩容失败：新容量 %s 必须大于当前容量 %s",
			newSize, currentQuantity.String())
		m.logger.Error("新容量必须大于当前容量",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.String("currentSize", currentQuantity.String()),
			zap.String("newSize", newSize))
		return fmt.Errorf("%s", errMsg)
	}

	// 更新PVC的存储请求大小
	if pvc.Spec.Resources.Requests == nil {
		pvc.Spec.Resources.Requests = make(corev1.ResourceList)
	}
	pvc.Spec.Resources.Requests[corev1.ResourceStorage] = newQuantity

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
		zap.String("oldSize", currentQuantity.String()),
		zap.String("newSize", newSize))

	return nil
}

// GetPVCPods 获取使用指定PVC的所有Pod
func (m *pvcManager) GetPVCPods(ctx context.Context, clusterID int, namespace, pvcName string) ([]corev1.Pod, error) {
	kubeClient, err := m.client.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	// 获取命名空间下的所有Pod
	podList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		m.logger.Error("获取Pod列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, err
	}

	// 过滤使用指定PVC的Pod
	var podsUsingPVC []corev1.Pod
	for _, pod := range podList.Items {
		// 检查Pod的所有卷
		for _, volume := range pod.Spec.Volumes {
			// 检查是否是PersistentVolumeClaim类型的卷，且ClaimName匹配
			if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == pvcName {
				podsUsingPVC = append(podsUsingPVC, pod)
				m.logger.Debug("找到使用PVC的Pod",
					zap.Int("clusterID", clusterID),
					zap.String("namespace", namespace),
					zap.String("pvcName", pvcName),
					zap.String("podName", pod.Name))
				break // 找到匹配的卷就跳出内层循环，继续下一个Pod
			}
		}
	}

	m.logger.Info("成功获取PVC关联的Pod",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("pvcName", pvcName),
		zap.Int("count", len(podsUsingPVC)))

	return podsUsingPVC, nil
}

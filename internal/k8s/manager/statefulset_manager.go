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

package manager

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type StatefulSetManager interface {
	CreateStatefulSet(ctx context.Context, clusterID int, namespace string, statefulSet *appsv1.StatefulSet) error
	GetStatefulSet(ctx context.Context, clusterID int, namespace, name string) (*appsv1.StatefulSet, error)
	GetStatefulSetList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sStatefulSet, error)
	UpdateStatefulSet(ctx context.Context, clusterID int, namespace string, statefulSet *appsv1.StatefulSet) error
	DeleteStatefulSet(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error
	RestartStatefulSet(ctx context.Context, clusterID int, namespace, name string) error
	ScaleStatefulSet(ctx context.Context, clusterID int, namespace, name string, replicas int32) error
	BatchDeleteStatefulSets(ctx context.Context, clusterID int, namespace string, statefulSetNames []string) error
	BatchRestartStatefulSets(ctx context.Context, clusterID int, namespace string, statefulSetNames []string) error
	GetStatefulSetHistory(ctx context.Context, clusterID int, namespace, statefulSetName string) ([]*model.K8sStatefulSetHistory, int64, error)
	GetStatefulSetPods(ctx context.Context, clusterID int, namespace, statefulSetName string) ([]*model.K8sPod, int64, error)
	RollbackStatefulSet(ctx context.Context, clusterID int, namespace, name string, revision int64) error
}

type statefulSetManager struct {
	clientFactory client.K8sClient
	logger        *zap.Logger
}

func NewStatefulSetManager(clientFactory client.K8sClient, logger *zap.Logger) StatefulSetManager {
	return &statefulSetManager{
		clientFactory: clientFactory,
		logger:        logger,
	}
}

// getKubeClient 获取 Kubernetes 客户端
func (m *statefulSetManager) getKubeClient(clusterID int) (*kubernetes.Clientset, error) {
	kubeClient, err := m.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		m.logger.Error("获取 Kubernetes 客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}
	return kubeClient, nil
}

func (m *statefulSetManager) CreateStatefulSet(ctx context.Context, clusterID int, namespace string, statefulSet *appsv1.StatefulSet) error {
	if statefulSet == nil {
		return fmt.Errorf("statefulSet 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = kubeClient.AppsV1().StatefulSets(namespace).Create(ctx, statefulSet, metav1.CreateOptions{})
	if err != nil {
		m.logger.Error("创建 StatefulSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", statefulSet.Name),
			zap.Error(err))
		return fmt.Errorf("创建 StatefulSet 失败: %w", err)
	}

	m.logger.Info("成功创建 StatefulSet",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", statefulSet.Name))

	return nil
}

func (m *statefulSetManager) GetStatefulSet(ctx context.Context, clusterID int, namespace, name string) (*appsv1.StatefulSet, error) {
	if name == "" {
		return nil, fmt.Errorf("StatefulSet name 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	statefulSet, err := kubeClient.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取 StatefulSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, fmt.Errorf("获取 StatefulSet 失败: %w", err)
	}

	return statefulSet, nil
}

func (m *statefulSetManager) GetStatefulSetList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sStatefulSet, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	statefulSetList, err := kubeClient.AppsV1().StatefulSets(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取 StatefulSet 列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, fmt.Errorf("获取 StatefulSet 列表失败: %w", err)
	}

	var k8sStatefulSets []*model.K8sStatefulSet
	for _, statefulSet := range statefulSetList.Items {
		k8sStatefulSet, err := utils.BuildK8sStatefulSet(ctx, clusterID, statefulSet)
		if err != nil {
			m.logger.Warn("构建 K8sStatefulSet 失败",
				zap.String("statefulSetName", statefulSet.Name),
				zap.Error(err))
			continue
		}
		k8sStatefulSets = append(k8sStatefulSets, k8sStatefulSet)
	}

	return k8sStatefulSets, nil
}

func (m *statefulSetManager) UpdateStatefulSet(ctx context.Context, clusterID int, namespace string, statefulSet *appsv1.StatefulSet) error {
	if statefulSet == nil {
		return fmt.Errorf("statefulSet 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = kubeClient.AppsV1().StatefulSets(namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("更新 StatefulSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", statefulSet.Name),
			zap.Error(err))
		return fmt.Errorf("更新 StatefulSet 失败: %w", err)
	}

	m.logger.Info("成功更新 StatefulSet",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", statefulSet.Name))

	return nil
}

func (m *statefulSetManager) DeleteStatefulSet(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	if name == "" {
		return fmt.Errorf("StatefulSet name 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = kubeClient.AppsV1().StatefulSets(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		m.logger.Error("删除 StatefulSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("删除 StatefulSet 失败: %w", err)
	}

	m.logger.Info("成功删除 StatefulSet",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	return nil
}

func (m *statefulSetManager) RestartStatefulSet(ctx context.Context, clusterID int, namespace, name string) error {
	if name == "" {
		return fmt.Errorf("StatefulSet name 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 通过添加注解来触发 StatefulSet 重启
	patchData := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))

	_, err = kubeClient.AppsV1().StatefulSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
	if err != nil {
		m.logger.Error("重启 StatefulSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("重启 StatefulSet 失败: %w", err)
	}

	m.logger.Info("成功重启 StatefulSet",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	return nil
}

func (m *statefulSetManager) ScaleStatefulSet(ctx context.Context, clusterID int, namespace, name string, replicas int32) error {
	if name == "" {
		return fmt.Errorf("StatefulSet name 不能为空")
	}
	if replicas < 0 {
		return fmt.Errorf("replicas 不能为负数")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	scale, err := kubeClient.AppsV1().StatefulSets(namespace).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取 StatefulSet Scale 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("获取 StatefulSet Scale 失败: %w", err)
	}

	// 更新副本数
	scale.Spec.Replicas = replicas

	// 执行扩缩容
	_, err = kubeClient.AppsV1().StatefulSets(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("扩缩容 StatefulSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Int32("replicas", replicas),
			zap.Error(err))
		return fmt.Errorf("扩缩容 StatefulSet 失败: %w", err)
	}

	m.logger.Info("成功扩缩容 StatefulSet",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name),
		zap.Int32("replicas", replicas))

	return nil
}

// BatchDeleteStatefulSets 批量删除 StatefulSets
func (m *statefulSetManager) BatchDeleteStatefulSets(ctx context.Context, clusterID int, namespace string, statefulSetNames []string) error {
	if len(statefulSetNames) == 0 {
		return fmt.Errorf("StatefulSet names 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []string

	deleteOptions := metav1.DeleteOptions{}

	for _, name := range statefulSetNames {
		wg.Add(1)
		go func(statefulSetName string) {
			defer wg.Done()

			err := kubeClient.AppsV1().StatefulSets(namespace).Delete(ctx, statefulSetName, deleteOptions)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Sprintf("删除 StatefulSet %s 失败: %v", statefulSetName, err))
				mu.Unlock()
				m.logger.Error("批量删除 StatefulSet 失败",
					zap.Int("clusterID", clusterID),
					zap.String("namespace", namespace),
					zap.String("name", statefulSetName),
					zap.Error(err))
			} else {
				m.logger.Info("成功删除 StatefulSet",
					zap.Int("clusterID", clusterID),
					zap.String("namespace", namespace),
					zap.String("name", statefulSetName))
			}
		}(name)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("批量删除 StatefulSets 部分失败: %s", strings.Join(errors, "; "))
	}

	return nil
}

// BatchRestartStatefulSets 批量重启 StatefulSets
func (m *statefulSetManager) BatchRestartStatefulSets(ctx context.Context, clusterID int, namespace string, statefulSetNames []string) error {
	if len(statefulSetNames) == 0 {
		return fmt.Errorf("StatefulSet names 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []string

	restartTime := time.Now().Format(time.RFC3339)
	patchData := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, restartTime)

	for _, name := range statefulSetNames {
		wg.Add(1)
		go func(statefulSetName string) {
			defer wg.Done()

			_, err := kubeClient.AppsV1().StatefulSets(namespace).Patch(ctx, statefulSetName, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Sprintf("重启 StatefulSet %s 失败: %v", statefulSetName, err))
				mu.Unlock()
				m.logger.Error("批量重启 StatefulSet 失败",
					zap.Int("clusterID", clusterID),
					zap.String("namespace", namespace),
					zap.String("name", statefulSetName),
					zap.Error(err))
			} else {
				m.logger.Info("成功重启 StatefulSet",
					zap.Int("clusterID", clusterID),
					zap.String("namespace", namespace),
					zap.String("name", statefulSetName))
			}
		}(name)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("批量重启 StatefulSets 部分失败: %s", strings.Join(errors, "; "))
	}

	return nil
}

func (m *statefulSetManager) GetStatefulSetHistory(ctx context.Context, clusterID int, namespace, statefulSetName string) ([]*model.K8sStatefulSetHistory, int64, error) {
	if statefulSetName == "" {
		return nil, 0, fmt.Errorf("StatefulSet name 不能为空")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, 0, err
	}

	statefulSet, err := kubeClient.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取 StatefulSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", statefulSetName),
			zap.Error(err))
		return nil, 0, fmt.Errorf("获取 StatefulSet 失败: %w", err)
	}

	// 获取所有 ControllerRevision
	listOptions := metav1.ListOptions{}
	revisionList, err := kubeClient.AppsV1().ControllerRevisions(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取 StatefulSet 历史版本失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("statefulSetName", statefulSetName),
			zap.Error(err))
		return nil, 0, fmt.Errorf("获取 StatefulSet 历史版本失败: %w", err)
	}

	var history []*model.K8sStatefulSetHistory
	for _, revision := range revisionList.Items {

		belongsToStatefulSet := false
		if revision.OwnerReferences != nil {
			for _, owner := range revision.OwnerReferences {
				if owner.Kind == "StatefulSet" && owner.Name == statefulSetName && owner.UID == statefulSet.UID {
					belongsToStatefulSet = true
					break
				}
			}
		}

		if belongsToStatefulSet {
			k8sHistory, err := utils.BuildK8sStatefulSetHistory(revision)
			if err != nil {
				m.logger.Warn("构建 K8sStatefulSetHistory 失败",
					zap.String("revisionName", revision.Name),
					zap.Error(err))
				continue
			}
			history = append(history, k8sHistory)
		}
	}

	// 按版本号排序（从新到旧）
	sort.Slice(history, func(i, j int) bool {
		return history[i].Revision > history[j].Revision
	})

	return history, int64(len(history)), nil
}

func (m *statefulSetManager) GetStatefulSetPods(ctx context.Context, clusterID int, namespace, statefulSetName string) ([]*model.K8sPod, int64, error) {
	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return nil, 0, err
	}

	pods, total, err := utils.GetStatefulSetPods(ctx, kubeClient, namespace, statefulSetName)
	if err != nil {
		m.logger.Error("获取 StatefulSet Pods 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("statefulSetName", statefulSetName),
			zap.Error(err))
		return nil, 0, fmt.Errorf("获取 StatefulSet Pods 失败: %w", err)
	}

	return pods, total, nil
}

func (m *statefulSetManager) RollbackStatefulSet(ctx context.Context, clusterID int, namespace, name string, revision int64) error {
	if name == "" {
		return fmt.Errorf("StatefulSet name 不能为空")
	}
	if revision <= 0 {
		return fmt.Errorf("revision 必须大于 0")
	}

	kubeClient, err := m.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 获取当前 StatefulSet
	currentStatefulSet, err := kubeClient.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		m.logger.Error("获取当前 StatefulSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("获取当前 StatefulSet 失败: %w", err)
	}

	// 获取所有 ControllerRevision
	listOptions := metav1.ListOptions{}
	revisionList, err := kubeClient.AppsV1().ControllerRevisions(namespace).List(ctx, listOptions)
	if err != nil {
		m.logger.Error("获取 ControllerRevision 列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("statefulSetName", name),
			zap.Error(err))
		return fmt.Errorf("获取 ControllerRevision 列表失败: %w", err)
	}

	// 查找指定版本的 ControllerRevision
	var targetRevision *appsv1.ControllerRevision
	for _, rev := range revisionList.Items {

		if rev.Revision == revision {

			if rev.OwnerReferences != nil {
				for _, owner := range rev.OwnerReferences {
					if owner.Kind == "StatefulSet" && owner.Name == name && owner.UID == currentStatefulSet.UID {
						targetRevision = &rev
						break
					}
				}
			}
			if targetRevision != nil {
				break
			}
		}
	}

	if targetRevision == nil {
		m.logger.Error("找不到指定版本的 ControllerRevision",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("statefulSetName", name),
			zap.Int64("revision", revision))
		return fmt.Errorf("找不到版本 %d 的 ControllerRevision", revision)
	}

	var statefulSetTemplate appsv1.StatefulSet
	err = utils.ExtractStatefulSetFromRevision(targetRevision, &statefulSetTemplate)
	if err != nil {
		m.logger.Error("从 ControllerRevision 提取 StatefulSet 模板失败",
			zap.Int64("revision", revision),
			zap.Error(err))
		return fmt.Errorf("提取 StatefulSet 模板失败: %w", err)
	}

	// StatefulSet 只允许更新以下字段：replicas, ordinals, template, updateStrategy,
	// persistentVolumeClaimRetentionPolicy 和 minReadySeconds
	// 不能更新 selector, serviceName 等不可变字段
	currentStatefulSet.Spec.Template = statefulSetTemplate.Spec.Template
	currentStatefulSet.Spec.UpdateStrategy = statefulSetTemplate.Spec.UpdateStrategy

	if statefulSetTemplate.Spec.Replicas != nil {
		currentStatefulSet.Spec.Replicas = statefulSetTemplate.Spec.Replicas
	}
	if statefulSetTemplate.Spec.RevisionHistoryLimit != nil {
		currentStatefulSet.Spec.RevisionHistoryLimit = statefulSetTemplate.Spec.RevisionHistoryLimit
	}
	if statefulSetTemplate.Spec.MinReadySeconds != 0 {
		currentStatefulSet.Spec.MinReadySeconds = statefulSetTemplate.Spec.MinReadySeconds
	}
	if statefulSetTemplate.Spec.PersistentVolumeClaimRetentionPolicy != nil {
		currentStatefulSet.Spec.PersistentVolumeClaimRetentionPolicy = statefulSetTemplate.Spec.PersistentVolumeClaimRetentionPolicy
	}
	if statefulSetTemplate.Spec.Ordinals != nil {
		currentStatefulSet.Spec.Ordinals = statefulSetTemplate.Spec.Ordinals
	}

	// 添加回滚注解
	if currentStatefulSet.Annotations == nil {
		currentStatefulSet.Annotations = make(map[string]string)
	}
	currentStatefulSet.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
	currentStatefulSet.Annotations["rollback.statefulset.kubernetes.io/revision"] = fmt.Sprintf("%d", revision)

	// 执行更新
	_, err = kubeClient.AppsV1().StatefulSets(namespace).Update(ctx, currentStatefulSet, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error("回滚 StatefulSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Int64("revision", revision),
			zap.Error(err))
		return fmt.Errorf("回滚 StatefulSet 失败: %w", err)
	}

	m.logger.Info("成功回滚 StatefulSet",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name),
		zap.Int64("revision", revision))

	return nil
}

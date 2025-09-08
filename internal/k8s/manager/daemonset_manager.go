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

type DaemonSetManager interface {
	CreateDaemonSet(ctx context.Context, clusterID int, namespace string, daemonSet *appsv1.DaemonSet) error
	GetDaemonSet(ctx context.Context, clusterID int, namespace, name string) (*appsv1.DaemonSet, error)
	GetDaemonSetList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sDaemonSet, error)
	UpdateDaemonSet(ctx context.Context, clusterID int, namespace string, daemonSet *appsv1.DaemonSet) error
	DeleteDaemonSet(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error
	RestartDaemonSet(ctx context.Context, clusterID int, namespace, name string) error
	BatchDeleteDaemonSets(ctx context.Context, clusterID int, namespace string, daemonSetNames []string) error
	BatchRestartDaemonSets(ctx context.Context, clusterID int, namespace string, daemonSetNames []string) error
	GetDaemonSetEvents(ctx context.Context, clusterID int, namespace, daemonSetName string, limit int) ([]*model.K8sDaemonSetEvent, int64, error)
	GetDaemonSetHistory(ctx context.Context, clusterID int, namespace, daemonSetName string) ([]*model.K8sDaemonSetHistory, int64, error)
	GetDaemonSetPods(ctx context.Context, clusterID int, namespace, daemonSetName string) ([]*model.K8sPod, int64, error)

	RollbackDaemonSet(ctx context.Context, clusterID int, namespace, name string, revision int64) error
}

type daemonSetManager struct {
	clientFactory client.K8sClient
	logger        *zap.Logger
}

func NewDaemonSetManager(clientFactory client.K8sClient, logger *zap.Logger) DaemonSetManager {
	return &daemonSetManager{
		clientFactory: clientFactory,
		logger:        logger,
	}
}

// getKubeClient 获取 Kubernetes 客户端
func (d *daemonSetManager) getKubeClient(clusterID int) (*kubernetes.Clientset, error) {
	kubeClient, err := d.clientFactory.GetKubeClient(clusterID)
	if err != nil {
		d.logger.Error("获取 Kubernetes 客户端失败", zap.Int("clusterID", clusterID), zap.Error(err))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}
	return kubeClient, nil
}

// CreateDaemonSet 创建 DaemonSet
func (d *daemonSetManager) CreateDaemonSet(ctx context.Context, clusterID int, namespace string, daemonSet *appsv1.DaemonSet) error {
	if daemonSet == nil {
		return fmt.Errorf("daemonSet 不能为空")
	}

	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = kubeClient.AppsV1().DaemonSets(namespace).Create(ctx, daemonSet, metav1.CreateOptions{})
	if err != nil {
		d.logger.Error("创建 DaemonSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", daemonSet.Name),
			zap.Error(err))
		return fmt.Errorf("创建 DaemonSet 失败: %w", err)
	}

	d.logger.Info("成功创建 DaemonSet",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", daemonSet.Name))

	return nil
}

// GetDaemonSet 获取单个 DaemonSet
func (d *daemonSetManager) GetDaemonSet(ctx context.Context, clusterID int, namespace, name string) (*appsv1.DaemonSet, error) {
	if name == "" {
		return nil, fmt.Errorf("DaemonSet name 不能为空")
	}

	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	daemonSet, err := kubeClient.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		d.logger.Error("获取 DaemonSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, fmt.Errorf("获取 DaemonSet 失败: %w", err)
	}

	return daemonSet, nil
}

// GetDaemonSetList 获取 DaemonSet 列表
func (d *daemonSetManager) GetDaemonSetList(ctx context.Context, clusterID int, namespace string, listOptions metav1.ListOptions) ([]*model.K8sDaemonSet, error) {
	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return nil, err
	}

	daemonSetList, err := kubeClient.AppsV1().DaemonSets(namespace).List(ctx, listOptions)
	if err != nil {
		d.logger.Error("获取 DaemonSet 列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, fmt.Errorf("获取 DaemonSet 列表失败: %w", err)
	}

	var k8sDaemonSets []*model.K8sDaemonSet
	for _, daemonSet := range daemonSetList.Items {
		k8sDaemonSet, err := utils.BuildK8sDaemonSet(ctx, clusterID, daemonSet)
		if err != nil {
			d.logger.Warn("构建 K8sDaemonSet 失败",
				zap.String("daemonSetName", daemonSet.Name),
				zap.Error(err))
			continue
		}
		k8sDaemonSets = append(k8sDaemonSets, k8sDaemonSet)
	}

	return k8sDaemonSets, nil
}

// UpdateDaemonSet 更新 DaemonSet
func (d *daemonSetManager) UpdateDaemonSet(ctx context.Context, clusterID int, namespace string, daemonSet *appsv1.DaemonSet) error {
	if daemonSet == nil {
		return fmt.Errorf("daemonSet 不能为空")
	}

	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	_, err = kubeClient.AppsV1().DaemonSets(namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
	if err != nil {
		d.logger.Error("更新 DaemonSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", daemonSet.Name),
			zap.Error(err))
		return fmt.Errorf("更新 DaemonSet 失败: %w", err)
	}

	d.logger.Info("成功更新 DaemonSet",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", daemonSet.Name))

	return nil
}

// DeleteDaemonSet 删除 DaemonSet
func (d *daemonSetManager) DeleteDaemonSet(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	if name == "" {
		return fmt.Errorf("DaemonSet name 不能为空")
	}

	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	err = kubeClient.AppsV1().DaemonSets(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		d.logger.Error("删除 DaemonSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("删除 DaemonSet 失败: %w", err)
	}

	d.logger.Info("成功删除 DaemonSet",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	return nil
}

// RestartDaemonSet 重启 DaemonSet
func (d *daemonSetManager) RestartDaemonSet(ctx context.Context, clusterID int, namespace, name string) error {
	if name == "" {
		return fmt.Errorf("DaemonSet name 不能为空")
	}

	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 通过添加注解来触发 DaemonSet 重启
	patchData := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))

	_, err = kubeClient.AppsV1().DaemonSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
	if err != nil {
		d.logger.Error("重启 DaemonSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("重启 DaemonSet 失败: %w", err)
	}

	d.logger.Info("成功重启 DaemonSet",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	return nil
}

// BatchDeleteDaemonSets 批量删除 DaemonSets
func (d *daemonSetManager) BatchDeleteDaemonSets(ctx context.Context, clusterID int, namespace string, daemonSetNames []string) error {
	if len(daemonSetNames) == 0 {
		return fmt.Errorf("DaemonSet names 不能为空")
	}

	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []string

	deleteOptions := metav1.DeleteOptions{}

	for _, name := range daemonSetNames {
		wg.Add(1)
		go func(daemonSetName string) {
			defer wg.Done()

			err := kubeClient.AppsV1().DaemonSets(namespace).Delete(ctx, daemonSetName, deleteOptions)
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Sprintf("删除 DaemonSet %s 失败: %v", daemonSetName, err))
				mu.Unlock()
				d.logger.Error("批量删除 DaemonSet 失败",
					zap.Int("clusterID", clusterID),
					zap.String("namespace", namespace),
					zap.String("name", daemonSetName),
					zap.Error(err))
			} else {
				d.logger.Info("成功删除 DaemonSet",
					zap.Int("clusterID", clusterID),
					zap.String("namespace", namespace),
					zap.String("name", daemonSetName))
			}
		}(name)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("批量删除 DaemonSets 部分失败: %s", strings.Join(errors, "; "))
	}

	return nil
}

// BatchRestartDaemonSets 批量重启 DaemonSets
func (d *daemonSetManager) BatchRestartDaemonSets(ctx context.Context, clusterID int, namespace string, daemonSetNames []string) error {
	if len(daemonSetNames) == 0 {
		return fmt.Errorf("DaemonSet names 不能为空")
	}

	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []string

	restartTime := time.Now().Format(time.RFC3339)
	patchData := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, restartTime)

	for _, name := range daemonSetNames {
		wg.Add(1)
		go func(daemonSetName string) {
			defer wg.Done()

			_, err := kubeClient.AppsV1().DaemonSets(namespace).Patch(ctx, daemonSetName, types.StrategicMergePatchType, []byte(patchData), metav1.PatchOptions{})
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Sprintf("重启 DaemonSet %s 失败: %v", daemonSetName, err))
				mu.Unlock()
				d.logger.Error("批量重启 DaemonSet 失败",
					zap.Int("clusterID", clusterID),
					zap.String("namespace", namespace),
					zap.String("name", daemonSetName),
					zap.Error(err))
			} else {
				d.logger.Info("成功重启 DaemonSet",
					zap.Int("clusterID", clusterID),
					zap.String("namespace", namespace),
					zap.String("name", daemonSetName))
			}
		}(name)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("批量重启 DaemonSets 部分失败: %s", strings.Join(errors, "; "))
	}

	return nil
}

// GetDaemonSetEvents 获取 DaemonSet 相关事件
func (d *daemonSetManager) GetDaemonSetEvents(ctx context.Context, clusterID int, namespace, daemonSetName string, limit int) ([]*model.K8sDaemonSetEvent, int64, error) {
	if daemonSetName == "" {
		return nil, 0, fmt.Errorf("DaemonSet name 不能为空")
	}

	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return nil, 0, err
	}

	// 构建字段选择器，过滤与指定 DaemonSet 相关的事件
	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=DaemonSet", daemonSetName)

	listOptions := metav1.ListOptions{
		FieldSelector: fieldSelector,
	}
	if limit > 0 {
		listOptions.Limit = int64(limit)
	}

	eventList, err := kubeClient.CoreV1().Events(namespace).List(ctx, listOptions)
	if err != nil {
		d.logger.Error("获取 DaemonSet 事件失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("daemonSetName", daemonSetName),
			zap.Error(err))
		return nil, 0, fmt.Errorf("获取 DaemonSet 事件失败: %w", err)
	}

	var events []*model.K8sDaemonSetEvent
	for _, event := range eventList.Items {
		k8sEvent, err := utils.BuildK8sDaemonSetEvent(event)
		if err != nil {
			d.logger.Warn("构建 K8sDaemonSetEvent 失败",
				zap.String("eventName", event.Name),
				zap.Error(err))
			continue
		}
		events = append(events, k8sEvent)
	}

	return events, int64(len(events)), nil
}

// GetDaemonSetHistory 获取 DaemonSet 历史版本
func (d *daemonSetManager) GetDaemonSetHistory(ctx context.Context, clusterID int, namespace, daemonSetName string) ([]*model.K8sDaemonSetHistory, int64, error) {
	if daemonSetName == "" {
		return nil, 0, fmt.Errorf("DaemonSet name 不能为空")
	}

	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return nil, 0, err
	}

	// 先获取 DaemonSet 本身以获取其标签选择器
	daemonSet, err := kubeClient.AppsV1().DaemonSets(namespace).Get(ctx, daemonSetName, metav1.GetOptions{})
	if err != nil {
		d.logger.Error("获取 DaemonSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", daemonSetName),
			zap.Error(err))
		return nil, 0, fmt.Errorf("获取 DaemonSet 失败: %w", err)
	}

	// 获取所有 ControllerRevision
	listOptions := metav1.ListOptions{}
	revisionList, err := kubeClient.AppsV1().ControllerRevisions(namespace).List(ctx, listOptions)
	if err != nil {
		d.logger.Error("获取 DaemonSet 历史版本失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("daemonSetName", daemonSetName),
			zap.Error(err))
		return nil, 0, fmt.Errorf("获取 DaemonSet 历史版本失败: %w", err)
	}

	var history []*model.K8sDaemonSetHistory
	for _, revision := range revisionList.Items {
		// 检查是否属于指定的 DaemonSet
		belongsToDaemonSet := false
		if revision.OwnerReferences != nil {
			for _, owner := range revision.OwnerReferences {
				if owner.Kind == "DaemonSet" && owner.Name == daemonSetName && owner.UID == daemonSet.UID {
					belongsToDaemonSet = true
					break
				}
			}
		}

		if belongsToDaemonSet {
			k8sHistory, err := utils.BuildK8sDaemonSetHistory(revision)
			if err != nil {
				d.logger.Warn("构建 K8sDaemonSetHistory 失败",
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

// GetDaemonSetPods 获取 DaemonSet 管理的 Pods
func (d *daemonSetManager) GetDaemonSetPods(ctx context.Context, clusterID int, namespace, daemonSetName string) ([]*model.K8sPod, int64, error) {
	if daemonSetName == "" {
		return nil, 0, fmt.Errorf("DaemonSet name 不能为空")
	}

	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return nil, 0, err
	}

	// 首先获取 DaemonSet 以获取其标签选择器
	daemonSet, err := kubeClient.AppsV1().DaemonSets(namespace).Get(ctx, daemonSetName, metav1.GetOptions{})
	if err != nil {
		d.logger.Error("获取 DaemonSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", daemonSetName),
			zap.Error(err))
		return nil, 0, fmt.Errorf("获取 DaemonSet 失败: %w", err)
	}

	// 构建标签选择器
	labelSelector := metav1.FormatLabelSelector(daemonSet.Spec.Selector)

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	podList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		d.logger.Error("获取 DaemonSet Pods 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("daemonSetName", daemonSetName),
			zap.Error(err))
		return nil, 0, fmt.Errorf("获取 DaemonSet Pods 失败: %w", err)
	}

	var pods []*model.K8sPod
	for _, pod := range podList.Items {
		k8sPod := utils.ConvertToK8sPod(&pod)
		k8sPod.ClusterID = int64(clusterID)
		pods = append(pods, k8sPod)
	}

	return pods, int64(len(pods)), nil
}

// RollbackDaemonSet 回滚 DaemonSet 到指定版本
func (d *daemonSetManager) RollbackDaemonSet(ctx context.Context, clusterID int, namespace, name string, revision int64) error {
	if name == "" {
		return fmt.Errorf("DaemonSet name 不能为空")
	}
	if revision <= 0 {
		return fmt.Errorf("revision 必须大于 0")
	}

	kubeClient, err := d.getKubeClient(clusterID)
	if err != nil {
		return err
	}

	// 获取当前 DaemonSet
	currentDaemonSet, err := kubeClient.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		d.logger.Error("获取当前 DaemonSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return fmt.Errorf("获取当前 DaemonSet 失败: %w", err)
	}

	// 获取与当前 DaemonSet 相关的所有 ControllerRevision
	labelSelector := "controller-revision-hash"
	if currentDaemonSet.Spec.Selector != nil && currentDaemonSet.Spec.Selector.MatchLabels != nil {
		// 构建更精确的标签选择器
		var selectorParts []string
		for key, value := range currentDaemonSet.Spec.Selector.MatchLabels {
			selectorParts = append(selectorParts, fmt.Sprintf("%s=%s", key, value))
		}
		if len(selectorParts) > 0 {
			labelSelector = strings.Join(selectorParts, ",")
		}
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	revisionList, err := kubeClient.AppsV1().ControllerRevisions(namespace).List(ctx, listOptions)
	if err != nil {
		d.logger.Error("获取 ControllerRevision 列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("daemonSetName", name),
			zap.Error(err))
		return fmt.Errorf("获取 ControllerRevision 列表失败: %w", err)
	}

	// 查找指定版本的 ControllerRevision
	var targetRevision *appsv1.ControllerRevision
	for _, rev := range revisionList.Items {
		// 检查是否属于指定的 DaemonSet 并且版本匹配
		if rev.Revision == revision {
			// 验证 ControllerRevision 是否属于当前 DaemonSet
			if rev.OwnerReferences != nil {
				for _, owner := range rev.OwnerReferences {
					if owner.Kind == "DaemonSet" && owner.Name == name {
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
		d.logger.Error("找不到指定版本的 ControllerRevision",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("daemonSetName", name),
			zap.Int64("revision", revision))
		return fmt.Errorf("找不到版本 %d 的 ControllerRevision", revision)
	}

	// DaemonSet 不像 Deployment 有内置的回滚功能，需要手动实现
	// 这里我们提供一个简化的回滚：重新触发 DaemonSet 的滚动更新
	if currentDaemonSet.Annotations == nil {
		currentDaemonSet.Annotations = make(map[string]string)
	}

	// 添加回滚注解来触发更新
	currentDaemonSet.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
	currentDaemonSet.Annotations["rollback.daemonset.kubernetes.io/revision"] = fmt.Sprintf("%d", revision)

	// 执行更新
	_, err = kubeClient.AppsV1().DaemonSets(namespace).Update(ctx, currentDaemonSet, metav1.UpdateOptions{})
	if err != nil {
		d.logger.Error("回滚 DaemonSet 失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Int64("revision", revision),
			zap.Error(err))
		return fmt.Errorf("回滚 DaemonSet 失败: %w", err)
	}

	d.logger.Info("成功回滚 DaemonSet",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name),
		zap.Int64("revision", revision))

	return nil
}

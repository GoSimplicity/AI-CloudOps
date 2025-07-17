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
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobService interface {
	GetJobsByNamespace(ctx context.Context, id int, namespace string) ([]*batchv1.Job, error)
	CreateJob(ctx context.Context, req *model.K8sJobRequest) error
	BatchDeleteJob(ctx context.Context, id int, namespace string, jobNames []string) error
	DeleteJob(ctx context.Context, id int, namespace, jobName string) error
	GetJobYaml(ctx context.Context, id int, namespace, jobName string) (string, error)
	GetJobStatus(ctx context.Context, id int, namespace, jobName string) (*model.K8sJobStatus, error)
	GetJobHistory(ctx context.Context, id int, namespace string) ([]*model.K8sJobHistory, error)
	GetJobPods(ctx context.Context, id int, namespace, jobName string) ([]*corev1.Pod, error)
}

type jobService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

// NewJobService 创建新的 JobService 实例
func NewJobService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) JobService {
	return &jobService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetJobsByNamespace 获取指定命名空间下的所有 Job
func (j *jobService) GetJobsByNamespace(ctx context.Context, id int, namespace string) ([]*batchv1.Job, error) {
	kubeClient, err := pkg.GetKubeClient(id, j.client, j.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	jobs, err := kubeClient.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Job list: %w", err)
	}

	result := make([]*batchv1.Job, len(jobs.Items))
	for i := range jobs.Items {
		result[i] = &jobs.Items[i]
	}

	return result, nil
}

// CreateJob 创建 Job
func (j *jobService) CreateJob(ctx context.Context, req *model.K8sJobRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, j.client, j.logger)
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	_, err = kubeClient.BatchV1().Jobs(req.Namespace).Create(ctx, req.JobYaml, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Job: %w", err)
	}

	return nil
}

// GetJobYaml 获取指定 Job 的 YAML 定义
func (j *jobService) GetJobYaml(ctx context.Context, id int, namespace, jobName string) (string, error) {
	kubeClient, err := pkg.GetKubeClient(id, j.client, j.logger)
	if err != nil {
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	job, err := kubeClient.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get Job: %w", err)
	}

	yamlData, err := yaml.Marshal(job)
	if err != nil {
		return "", fmt.Errorf("failed to serialize Job YAML: %w", err)
	}

	return string(yamlData), nil
}

// BatchDeleteJob 批量删除 Job
func (j *jobService) BatchDeleteJob(ctx context.Context, id int, namespace string, jobNames []string) error {
	kubeClient, err := pkg.GetKubeClient(id, j.client, j.logger)
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(jobNames))

	propagationPolicy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	}

	for _, name := range jobNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.BatchV1().Jobs(namespace).Delete(ctx, name, deleteOptions); err != nil {
				errChan <- fmt.Errorf("failed to delete Job '%s': %w", name, err)
			}
		}(name)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred while deleting Jobs: %v", errs)
	}

	return nil
}

// DeleteJob 删除指定的 Job
func (j *jobService) DeleteJob(ctx context.Context, id int, namespace, jobName string) error {
	kubeClient, err := pkg.GetKubeClient(id, j.client, j.logger)
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	propagationPolicy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	}

	if err := kubeClient.BatchV1().Jobs(namespace).Delete(ctx, jobName, deleteOptions); err != nil {
		return fmt.Errorf("failed to delete Job '%s': %w", jobName, err)
	}

	return nil
}

// GetJobStatus 获取 Job 状态
func (j *jobService) GetJobStatus(ctx context.Context, id int, namespace, jobName string) (*model.K8sJobStatus, error) {
	kubeClient, err := pkg.GetKubeClient(id, j.client, j.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	job, err := kubeClient.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Job: %w", err)
	}

	status := &model.K8sJobStatus{
		Name:              job.Name,
		Namespace:         job.Namespace,
		Active:            job.Status.Active,
		Succeeded:         job.Status.Succeeded,
		Failed:            job.Status.Failed,
		Completions:       job.Spec.Completions,
		Parallelism:       job.Spec.Parallelism,
		BackoffLimit:      job.Spec.BackoffLimit,
		ActiveDeadlineSeconds: job.Spec.ActiveDeadlineSeconds,
		CreationTimestamp: job.CreationTimestamp.Time,
	}

	if job.Status.StartTime != nil {
		status.StartTime = &job.Status.StartTime.Time
	}

	if job.Status.CompletionTime != nil {
		status.CompletionTime = &job.Status.CompletionTime.Time
	}

	// 判断 Job 状态
	if job.Status.Succeeded > 0 {
		status.Phase = "Succeeded"
	} else if job.Status.Failed > 0 {
		status.Phase = "Failed"
	} else if job.Status.Active > 0 {
		status.Phase = "Running"
	} else {
		status.Phase = "Pending"
	}

	return status, nil
}

// GetJobHistory 获取 Job 执行历史
func (j *jobService) GetJobHistory(ctx context.Context, id int, namespace string) ([]*model.K8sJobHistory, error) {
	kubeClient, err := pkg.GetKubeClient(id, j.client, j.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	jobs, err := kubeClient.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Job list: %w", err)
	}

	var history []*model.K8sJobHistory
	for _, job := range jobs.Items {
		historyItem := &model.K8sJobHistory{
			Name:              job.Name,
			Namespace:         job.Namespace,
			Active:            job.Status.Active,
			Succeeded:         job.Status.Succeeded,
			Failed:            job.Status.Failed,
			CreationTimestamp: job.CreationTimestamp.Time,
		}

		if job.Status.StartTime != nil {
			historyItem.StartTime = &job.Status.StartTime.Time
		}

		if job.Status.CompletionTime != nil {
			historyItem.CompletionTime = &job.Status.CompletionTime.Time
		}

		// 计算执行时长
		if historyItem.StartTime != nil && historyItem.CompletionTime != nil {
			duration := historyItem.CompletionTime.Sub(*historyItem.StartTime)
			historyItem.Duration = duration.String()
		}

		// 判断状态
		if job.Status.Succeeded > 0 {
			historyItem.Status = "Succeeded"
		} else if job.Status.Failed > 0 {
			historyItem.Status = "Failed"
		} else if job.Status.Active > 0 {
			historyItem.Status = "Running"
		} else {
			historyItem.Status = "Pending"
		}

		history = append(history, historyItem)
	}

	return history, nil
}

// GetJobPods 获取 Job 关联的 Pod 列表
func (j *jobService) GetJobPods(ctx context.Context, id int, namespace, jobName string) ([]*corev1.Pod, error) {
	kubeClient, err := pkg.GetKubeClient(id, j.client, j.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 首先获取 Job 对象
	job, err := kubeClient.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Job: %w", err)
	}

	// 通过 Job 的 selector 查找关联的 Pod
	selector := metav1.FormatLabelSelector(job.Spec.Selector)
	pods, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get Job Pods: %w", err)
	}

	result := make([]*corev1.Pod, len(pods.Items))
	for i := range pods.Items {
		result[i] = &pods.Items[i]
	}

	return result, nil
}
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

package user

import (
	"context"
	"fmt"
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
)

type InstanceService interface {
	CreateInstance(ctx context.Context, req *model.K8sInstance) (*model.CreateK8sInstanceResp, error)
	UpdateInstance(ctx context.Context, req *model.K8sInstance) (*model.UpdateK8sInstanceResp, error)
	BatchDeleteInstance(ctx context.Context, req *model.BatchDeleteK8sInstanceReq) (*model.BatchDeleteK8sInstanceResp, error)
	BatchRestartInstance(ctx context.Context, req *model.BatchRestartK8sInstanceReq) (*model.BatchRestartK8sInstanceResp, error)
	GetInstanceByApp(ctx context.Context, req *model.GetK8sInstanceByAppReq) (*model.GetK8sInstanceByAppResp, error)
	GetInstance(ctx context.Context, req *model.GetK8sInstanceReq) (*model.GetK8sInstanceResp, error)
	GetInstanceList(ctx context.Context, req *model.GetK8sInstanceListReq) (*model.GetK8sInstanceListResp, error)
}

type instanceService struct {
	client      client.K8sClient
	logger      *zap.Logger
	clusterDAO  admin.ClusterDAO
}



func NewInstanceService(clusterDAO admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) InstanceService {
	return &instanceService{
		clusterDAO:  clusterDAO,
		client:      client,
		logger:      logger,
	}
}

// BatchDeleteInstance 批量删除实例
func (i *instanceService) BatchDeleteInstance(ctx context.Context, req *model.BatchDeleteK8sInstanceReq) (*model.BatchDeleteK8sInstanceResp, error) {
	deletedCount := 0
	
	for _, instance := range req.Instances {
		var err error
		
		// 根据每个实例的type删除对应的资源
		switch instance.Type {
		case "Deployment":
			err = i.client.DeleteDeployment(ctx, instance.Namespace, instance.Name, instance.ClusterID)
		case "StatefulSet":
			err = i.client.DeleteStatefulSet(ctx, instance.Namespace, instance.Name, instance.ClusterID)
		case "DaemonSet":
			err = i.client.DeleteDaemonSet(ctx, instance.Namespace, instance.Name, instance.ClusterID)
		case "Job":
			err = i.client.DeleteJob(ctx, instance.Namespace, instance.Name, instance.ClusterID)
		case "CronJob":
			err = i.client.DeleteCronJob(ctx, instance.Namespace, instance.Name, instance.ClusterID)
		default:
			i.logger.Error("不支持的资源类型", zap.String("type", instance.Type), zap.String("name", instance.Name))
			continue
		}
		
		if err != nil {
			i.logger.Error("删除K8s资源失败", 
				zap.Error(err), 
				zap.String("type", instance.Type),
				zap.String("namespace", instance.Namespace),
				zap.String("name", instance.Name),
				zap.Int("clusterId", instance.ClusterID))
			continue
		}
		deletedCount++
	}
	
	return &model.BatchDeleteK8sInstanceResp{
		DeletedCount: deletedCount,
	}, nil
}

// BatchRestartInstance 批量重启实例
func (i *instanceService) BatchRestartInstance(ctx context.Context, req *model.BatchRestartK8sInstanceReq) (*model.BatchRestartK8sInstanceResp, error) {
	restartedCount := 0

	for _, instance := range req.Instances {
		var err error
		
		// 根据每个实例的type重启对应的资源
		switch instance.Type {
		case "Deployment":
			err = i.client.RestartDeployment(ctx, instance.Namespace, instance.Name, instance.ClusterID)
		case "StatefulSet":
			err = i.client.RestartStatefulSet(ctx, instance.Namespace, instance.Name, instance.ClusterID)
		case "DaemonSet":
			err = i.client.RestartDaemonSet(ctx, instance.Namespace, instance.Name, instance.ClusterID)
		case "Job":
			err = i.client.RestartJob(ctx, instance.Namespace, instance.Name, instance.ClusterID)
		case "CronJob":
			err = i.client.RestartCronJob(ctx, instance.Namespace, instance.Name, instance.ClusterID)
		default:
			i.logger.Error("不支持的资源类型", zap.String("type", instance.Type), zap.String("name", instance.Name))
			continue
		}
		
		if err != nil {
			i.logger.Error("重启K8s资源失败", 
				zap.Error(err),
				zap.String("type", instance.Type),
				zap.String("namespace", instance.Namespace),
				zap.String("name", instance.Name),
				zap.Int("clusterId", instance.ClusterID))
			continue
		}
		restartedCount++
	}

	return &model.BatchRestartK8sInstanceResp{
		RestartedCount: restartedCount,
	}, nil
}

// CreateInstance 创建实例
func (i *instanceService) CreateInstance(ctx context.Context, req *model.K8sInstance) (*model.CreateK8sInstanceResp, error) {
	var err error

	switch req.Type {
	case "Deployment":
		// 构建Deployment创建配置
		deployment := utils.BuildDeploymentConfig(req)
		err = i.client.CreateDeployment(ctx, req.Namespace, req.ClusterID, deployment)
	case "StatefulSet":
		// 构建StatefulSet创建配置
		statefulset := utils.BuildStatefulSetConfig(req)
		err = i.client.CreateStatefulSet(ctx, req.Namespace, req.ClusterID, statefulset)
	case "DaemonSet":
		// 构建DaemonSet创建配置
		daemonset := utils.BuildDaemonSetConfig(req)
		err = i.client.CreateDaemonSet(ctx, req.Namespace, req.ClusterID, daemonset)
	case "Job":
		// 构建Job创建配置
		job := utils.BuildJobConfig(req)
		err = i.client.CreateJob(ctx, req.Namespace, req.ClusterID, job)
	case "CronJob":
		// 构建CronJob创建配置
		cronjob := utils.BuildCronJobConfig(req)
		err = i.client.CreateCronJob(ctx, req.Namespace, req.ClusterID, cronjob)
	default:
		return nil, fmt.Errorf("不支持的资源类型: %s", req.Type)
	}
	if err != nil {
		return nil, err
	}

	return &model.CreateK8sInstanceResp{}, nil
}

// GetInstance 获取实例
func (i *instanceService) GetInstance(ctx context.Context, req *model.GetK8sInstanceReq) (*model.GetK8sInstanceResp, error) {
	var err error
	var instance interface{}
	switch req.Type {
	case "Deployment":
		instance, err = i.client.GetDeployment(ctx, req.Namespace, req.Name, req.ClusterID)
	case "StatefulSet":
		instance, err = i.client.GetStatefulSet(ctx, req.Namespace, req.Name, req.ClusterID)
	case "DaemonSet":
		instance, err = i.client.GetDaemonSet(ctx, req.Namespace, req.Name, req.ClusterID)
	case "Job":
		instance, err = i.client.GetJob(ctx, req.Namespace, req.Name, req.ClusterID)
	case "CronJob":
		instance, err = i.client.GetCronJob(ctx, req.Namespace, req.Name, req.ClusterID)
	}
	if err != nil {
		return nil, err
	}

	return &model.GetK8sInstanceResp{
		Item: instance,
	}, nil
}

// GetInstanceByApp 根据应用获取实例
func (i *instanceService) GetInstanceByApp(ctx context.Context, req *model.GetK8sInstanceByAppReq) (*model.GetK8sInstanceByAppResp, error) {
	// 根据注解中的app_id获取实例
	instances, err := i.client.GetDeploymentList(ctx, req.Namespace, req.ClusterID)
	if err != nil {
		return nil, err
	}

	var instanceList []interface{}
	for _, instance := range instances {
		if instance.Annotations["app_id"] == strconv.Itoa(req.AppID) {
			instanceList = append(instanceList, instance)
		}
	}

	return &model.GetK8sInstanceByAppResp{
		Items: instanceList,
	}, nil
}

// GetInstanceList 获取实例列表
func (i *instanceService) GetInstanceList(ctx context.Context, req *model.GetK8sInstanceListReq) (*model.GetK8sInstanceListResp, error) {
	var result []model.K8sInstance

	switch req.Type {
	case "Deployment":
		deployments, err := i.client.GetDeploymentList(ctx, req.Namespace, req.ClusterID)
		if err != nil {
			return nil, err
		}
		for _, deployment := range deployments {
			result = append(result, model.K8sInstance{
				Name:      deployment.Name,
				Namespace: deployment.Namespace,
				Type:      "Deployment",
				Status:    string(deployment.Status.Conditions[0].Type),
			})
		}
	case "StatefulSet":
		statefulSets, err := i.client.GetStatefulSetList(ctx, req.Namespace, req.ClusterID)
		if err != nil {
			return nil, err
		}
		for _, statefulSet := range statefulSets {
			result = append(result, model.K8sInstance{
				Name:      statefulSet.Name,
				Namespace: statefulSet.Namespace,
				Type:      "StatefulSet",
				Status:    string(statefulSet.Status.Conditions[0].Type),
			})
		}
	case "DaemonSet":
		daemonSets, err := i.client.GetDaemonSetList(ctx, req.Namespace, req.ClusterID)
		if err != nil {
			return nil, err
		}
		for _, daemonSet := range daemonSets {
			result = append(result, model.K8sInstance{
				Name:      daemonSet.Name,
				Namespace: daemonSet.Namespace,
				Type:      "DaemonSet",
				Status:    string(daemonSet.Status.Conditions[0].Type),
			})
		}
	case "Job":
		jobs, err := i.client.GetJobList(ctx, req.Namespace, req.ClusterID)
		if err != nil {
			return nil, err
		}
		for _, job := range jobs {
			result = append(result, model.K8sInstance{
				Name:      job.Name,
				Namespace: job.Namespace,
				Type:      "Job",
				Status:    getJobStatus(job),
			})
		}
	case "CronJob":
		cronJobs, err := i.client.GetCronJobList(ctx, req.Namespace, req.ClusterID)
		if err != nil {
			return nil, err
		}
		for _, cronJob := range cronJobs {
			result = append(result, model.K8sInstance{
				Name:      cronJob.Name,
				Namespace: cronJob.Namespace,
				Type:      "CronJob",
				Status:    getCronJobStatus(cronJob),
			})
		}
	default:
		return nil, fmt.Errorf("不支持的资源类型: %s", req.Type)
	}

	return &model.GetK8sInstanceListResp{
		Items: result,
	}, nil
}

// UpdateInstance 更新实例
func (i *instanceService) UpdateInstance(ctx context.Context, req *model.K8sInstance) (*model.UpdateK8sInstanceResp, error) {
	var err error

	switch req.Type {
	case "Deployment":
		// 构建Deployment更新配置
		deployment := utils.BuildDeploymentConfig(req)
		err = i.client.UpdateDeployment(ctx, req.Namespace, req.ClusterID, deployment)
	case "StatefulSet":
		statefulset := utils.BuildStatefulSetConfig(req)
		err = i.client.UpdateStatefulSet(ctx, req.Namespace, req.ClusterID, statefulset)
	case "DaemonSet":
		daemonset := utils.BuildDaemonSetConfig(req)
		err = i.client.UpdateDaemonSet(ctx, req.Namespace, req.ClusterID, daemonset)
	case "Job":
		job := utils.BuildJobConfig(req)
		err = i.client.UpdateJob(ctx, req.Namespace, req.ClusterID, job)
	case "CronJob":
		cronjob := utils.BuildCronJobConfig(req)
		err = i.client.UpdateCronJob(ctx, req.Namespace, req.ClusterID, cronjob)
	default:
		return nil, fmt.Errorf("不支持的资源类型: %s", req.Type)
	}
	if err != nil {
		return nil, err
	}
	return &model.UpdateK8sInstanceResp{}, nil
}


// getJobStatus 获取Job状态
func getJobStatus(job batchv1.Job) string {
	if job.Status.Succeeded > 0 {
		return "Succeeded"
	}
	if job.Status.Failed > 0 {
		return "Failed"
	}
	if job.Status.Active > 0 {
		return "Active"
	}
	return "Unknown"
}

// getCronJobStatus 获取CronJob状态
func getCronJobStatus(cronJob batchv1.CronJob) string {
	if len(cronJob.Status.Active) > 0 {
		return "Active"
	}
	if cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend {
		return "Suspended"
	}
	return "Scheduled"
}


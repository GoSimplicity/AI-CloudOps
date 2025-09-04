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

package utils

import (
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	corev1 "k8s.io/api/core/v1"
)

// ConvertToK8sPods 将 Kubernetes Pod 列表转换为内部 Pod 模型列表
func ConvertToK8sPods(pods []corev1.Pod) []*model.K8sPod {
	if len(pods) == 0 {
		return nil
	}

	var results []*model.K8sPod
	for _, pod := range pods {
		results = append(results, ConvertToK8sPod(&pod))
	}
	return results
}

// ConvertToK8sPod 单个转换
func ConvertToK8sPod(pod *corev1.Pod) *model.K8sPod {
	if pod == nil {
		return nil
	}

	return &model.K8sPod{
		Name:           pod.Name,
		Namespace:      pod.Namespace,
		UID:            string(pod.UID),
		Status:         string(pod.Status.Phase),
		NodeName:       pod.Spec.NodeName,
		HostIP:         pod.Status.HostIP,
		PodIP:          pod.Status.PodIP,
		StartTime:      getPodStartTime(pod),
		Labels:         pod.Labels,
		Annotations:    pod.Annotations,
		Containers:     ConvertK8sContainers(pod.Spec.Containers),
		InitContainers: ConvertK8sContainers(pod.Spec.InitContainers),
		Conditions:     convertPodConditions(pod.Status.Conditions),
		CreatedAt:      pod.CreationTimestamp.Time,
		UpdatedAt:      time.Now(),
		RawPod:         pod,
	}
}

func getPodStartTime(pod *corev1.Pod) *time.Time {
	if pod.Status.StartTime != nil {
		t := pod.Status.StartTime.Time
		return &t
	}
	return nil
}

func convertPodConditions(conds []corev1.PodCondition) []*model.PodCondition {
	if len(conds) == 0 {
		return nil
	}
	var res []*model.PodCondition
	for _, c := range conds {
		res = append(res, &model.PodCondition{
			Type:               string(c.Type),
			Status:             string(c.Status),
			LastProbeTime:      c.LastProbeTime.Time,
			LastTransitionTime: c.LastTransitionTime.Time,
			Reason:             c.Reason,
			Message:            c.Message,
		})
	}
	return res
}

func ConvertK8sContainers(containers []corev1.Container) []*model.K8sPodContainer {
	if len(containers) == 0 {
		return nil
	}

	var results []*model.K8sPodContainer
	for _, c := range containers {
		results = append(results, convertK8sContainer(c))
	}
	return results
}

func convertK8sContainer(container corev1.Container) *model.K8sPodContainer {
	k8sContainer := &model.K8sPodContainer{
		Name:            container.Name,
		Image:           container.Image,
		Command:         model.StringList(container.Command),
		Args:            model.StringList(container.Args),
		ImagePullPolicy: string(container.ImagePullPolicy),
		Envs:            convertEnvVars(container.Env),
		Ports:           convertContainerPorts(container.Ports),
		VolumeMounts:    convertVolumeMounts(container.VolumeMounts),
		Resources:       convertResourceRequirements(container.Resources),
	}

	if container.LivenessProbe != nil {
		k8sContainer.LivenessProbe = convertK8sProbe(container.LivenessProbe)
	}
	if container.ReadinessProbe != nil {
		k8sContainer.ReadinessProbe = convertK8sProbe(container.ReadinessProbe)
	}
	return k8sContainer
}

func convertEnvVars(envs []corev1.EnvVar) []model.K8sEnvVar {
	if len(envs) == 0 {
		return nil
	}
	var res []model.K8sEnvVar
	for _, e := range envs {
		res = append(res, model.K8sEnvVar{
			Name:  e.Name,
			Value: e.Value,
		})
	}
	return res
}

func convertContainerPorts(ports []corev1.ContainerPort) []model.K8sContainerPort {
	if len(ports) == 0 {
		return nil
	}
	var res []model.K8sContainerPort
	for _, p := range ports {
		res = append(res, model.K8sContainerPort{
			Name:          p.Name,
			ContainerPort: int(p.ContainerPort),
			Protocol:      string(p.Protocol),
		})
	}
	return res
}

func convertVolumeMounts(mounts []corev1.VolumeMount) []model.K8sVolumeMount {
	if len(mounts) == 0 {
		return nil
	}
	var res []model.K8sVolumeMount
	for _, m := range mounts {
		res = append(res, model.K8sVolumeMount{
			Name:      m.Name,
			MountPath: m.MountPath,
			ReadOnly:  m.ReadOnly,
			SubPath:   m.SubPath,
		})
	}
	return res
}

func convertResourceRequirements(rr corev1.ResourceRequirements) model.ResourceRequirements {
	res := model.ResourceRequirements{}

	if rr.Requests != nil {
		if cpu, ok := rr.Requests[corev1.ResourceCPU]; ok {
			res.Requests.CPU = cpu.String()
		}
		if mem, ok := rr.Requests[corev1.ResourceMemory]; ok {
			res.Requests.Memory = mem.String()
		}
	}
	if rr.Limits != nil {
		if cpu, ok := rr.Limits[corev1.ResourceCPU]; ok {
			res.Limits.CPU = cpu.String()
		}
		if mem, ok := rr.Limits[corev1.ResourceMemory]; ok {
			res.Limits.Memory = mem.String()
		}
	}
	return res
}

// 转换容器探针
func convertK8sProbe(probe *corev1.Probe) *model.K8sProbe {
	if probe == nil {
		return nil
	}

	k8sProbe := &model.K8sProbe{
		InitialDelaySeconds: int(probe.InitialDelaySeconds),
		PeriodSeconds:       int(probe.PeriodSeconds),
		TimeoutSeconds:      int(probe.TimeoutSeconds),
		SuccessThreshold:    int(probe.SuccessThreshold),
		FailureThreshold:    int(probe.FailureThreshold),
	}

	if probe.HTTPGet != nil {
		k8sProbe.HTTPGet = &model.K8sHTTPGetAction{
			Path:   probe.HTTPGet.Path,
			Port:   probe.HTTPGet.Port.IntValue(),
			Scheme: string(probe.HTTPGet.Scheme),
		}
	}
	return k8sProbe
}

func PodStatus(pod *corev1.Pod) string {
	if pod == nil {
		return statusUnknown
	}
	if pod.DeletionTimestamp != nil {
		return statusTerminating
	}
	if pod.Status.Reason == "Evicted" {
		return statusEvicted
	}
	switch pod.Status.Phase {
	case corev1.PodPending:
		return statusPending
	case corev1.PodSucceeded:
		return statusSucceeded
	case corev1.PodFailed:
		return statusFailed
	case corev1.PodRunning:
		// 检查容器是否全部 Ready
		allReady := true
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				allReady = false
				break
			}
		}
		if allReady {
			return statusRunning
		}
		return statusUpdating
	default:
		return statusUnknown
	}
}

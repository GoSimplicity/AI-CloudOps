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
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	corev1 "k8s.io/api/core/v1"
)

// BuildK8sPods 构建pod列表
func BuildK8sPods(pods []corev1.Pod) []*model.K8sPod {
	var k8sPods []*model.K8sPod

	for _, pod := range pods {
		k8sPod := &model.K8sPod{
			Name:        pod.Name,
			Namespace:   pod.Namespace,
			Status:      string(pod.Status.Phase),
			NodeName:    pod.Spec.NodeName,
			Labels:      pod.Labels,
			Annotations: pod.Annotations,
			Containers:  BuildK8sContainers(pod.Spec.Containers),
		}
		k8sPods = append(k8sPods, k8sPod)
	}

	return k8sPods
}

// BuildK8sContainers 构建容器列表
func BuildK8sContainers(containers []corev1.Container) []model.K8sPodContainer {
	var k8sContainers []model.K8sPodContainer

	for _, container := range containers {
		k8sContainer := model.K8sPodContainer{
			Name:            container.Name,
			Image:           container.Image,
			Command:         model.StringList(container.Command),
			Args:            model.StringList(container.Args),
			ImagePullPolicy: string(container.ImagePullPolicy),
		}

		// 转换环境变量
		for _, env := range container.Env {
			k8sContainer.Envs = append(k8sContainer.Envs, model.K8sEnvVar{
				Name:  env.Name,
				Value: env.Value,
			})
		}

		// 转换端口配置
		for _, port := range container.Ports {
			k8sContainer.Ports = append(k8sContainer.Ports, model.K8sContainerPort{
				Name:          port.Name,
				ContainerPort: int(port.ContainerPort),
				Protocol:      string(port.Protocol),
			})
		}

		// 转换卷挂载
		for _, volumeMount := range container.VolumeMounts {
			k8sContainer.VolumeMounts = append(k8sContainer.VolumeMounts, model.K8sVolumeMount{
				Name:      volumeMount.Name,
				MountPath: volumeMount.MountPath,
				ReadOnly:  volumeMount.ReadOnly,
				SubPath:   volumeMount.SubPath,
			})
		}

		// 转换资源要求
		k8sContainer.Resources = model.ResourceRequirements{}

		if container.Resources.Requests != nil {
			if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
				k8sContainer.Resources.Requests.CPU = cpu.String()
			}
			if memory, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
				k8sContainer.Resources.Requests.Memory = memory.String()
			}
		}

		if container.Resources.Limits != nil {
			if cpu, ok := container.Resources.Limits[corev1.ResourceCPU]; ok {
				k8sContainer.Resources.Limits.CPU = cpu.String()
			}
			if memory, ok := container.Resources.Limits[corev1.ResourceMemory]; ok {
				k8sContainer.Resources.Limits.Memory = memory.String()
			}
		}

		// 转换存活探测
		if container.LivenessProbe != nil {
			k8sContainer.LivenessProbe = buildK8sProbe(container.LivenessProbe)
		}

		// 转换就绪探测
		if container.ReadinessProbe != nil {
			k8sContainer.ReadinessProbe = buildK8sProbe(container.ReadinessProbe)
		}

		k8sContainers = append(k8sContainers, k8sContainer)
	}

	return k8sContainers
}

// BuildK8sContainersWithPointer 转换容器指针列表
func BuildK8sContainersWithPointer(containers []model.K8sPodContainer) []*model.K8sPodContainer {
	var containerPtrs []*model.K8sPodContainer

	for i := range containers {
		containerPtrs = append(containerPtrs, &containers[i])
	}

	return containerPtrs
}

// buildK8sProbe 辅助函数：转换探测配置
func buildK8sProbe(probe *corev1.Probe) *model.K8sProbe {
	k8sProbe := &model.K8sProbe{
		InitialDelaySeconds: int(probe.InitialDelaySeconds),
		PeriodSeconds:       int(probe.PeriodSeconds),
		TimeoutSeconds:      int(probe.TimeoutSeconds),
		SuccessThreshold:    int(probe.SuccessThreshold),
		FailureThreshold:    int(probe.FailureThreshold),
	}

	// 转换 HTTP GET 探测
	if probe.HTTPGet != nil {
		k8sProbe.HTTPGet = &model.K8sHTTPGetAction{
			Path:   probe.HTTPGet.Path,
			Port:   probe.HTTPGet.Port.IntValue(),
			Scheme: string(probe.HTTPGet.Scheme),
		}
	}

	return k8sProbe
}

// BuildK8sPod 构建pod模型
func BuildK8sPod(ctx context.Context, clusterID int, pod corev1.Pod) (*model.K8sPod, error) {
	status := getPodStatus(pod)

	k8sPod := &model.K8sPod{
		Name:        pod.Name,
		Namespace:   pod.Namespace,
		Status:      status,
		NodeName:    pod.Spec.NodeName,
		Labels:      pod.Labels,
		Annotations: pod.Annotations,
		Containers:  BuildK8sContainers(pod.Spec.Containers),
	}

	return k8sPod, nil
}

// getPodStatus 获取Pod状态
func getPodStatus(pod corev1.Pod) string {
	switch pod.Status.Phase {
	case corev1.PodRunning:
		// 检查所有容器是否就绪
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				return "Running"
			}
		}
		return "Pending"
	case corev1.PodSucceeded:
		return "Succeeded"
	case corev1.PodFailed:
		return "Failed"
	case corev1.PodPending:
		return "Pending"
	default:
		return "Unknown"
	}
}

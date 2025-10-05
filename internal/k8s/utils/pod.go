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
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
)

func ConvertToK8sPods(pods []corev1.Pod) []*model.K8sPod {
	if len(pods) == 0 {
		return nil
	}

	results := make([]*model.K8sPod, 0, len(pods))
	for _, pod := range pods {
		results = append(results, ConvertToK8sPod(&pod))
	}
	return results
}

func ConvertToK8sPod(pod *corev1.Pod) *model.K8sPod {
	if pod == nil {
		return nil
	}

	labelsJSON, _ := json.Marshal(pod.Labels)
	annotationsJSON, _ := json.Marshal(pod.Annotations)

	containersJSON, _ := json.Marshal(convertContainersWithStatus(pod.Spec.Containers, pod.Status.ContainerStatuses))
	initContainersJSON, _ := json.Marshal(convertContainersWithStatus(pod.Spec.InitContainers, pod.Status.InitContainerStatuses))

	conditionsJSON, _ := json.Marshal(convertPodConditions(pod.Status.Conditions))

	volumesJSON, _ := json.Marshal(pod.Spec.Volumes)

	ownerRefsJSON, _ := json.Marshal(pod.OwnerReferences)

	specJSON, _ := json.Marshal(pod.Spec)

	return &model.K8sPod{
		Name:              pod.Name,
		Namespace:         pod.Namespace,
		UID:               string(pod.UID),
		Labels:            string(labelsJSON),
		Annotations:       string(annotationsJSON),
		Status:            PodStatus(pod),
		Phase:             string(pod.Status.Phase),
		NodeName:          pod.Spec.NodeName,
		PodIP:             pod.Status.PodIP,
		HostIP:            pod.Status.HostIP,
		QosClass:          string(pod.Status.QOSClass),
		RestartCount:      getTotalRestartCount(pod),
		Ready:             getReadyStatus(pod),
		ServiceAccount:    pod.Spec.ServiceAccountName,
		RestartPolicy:     string(pod.Spec.RestartPolicy),
		DNSPolicy:         string(pod.Spec.DNSPolicy),
		Conditions:        string(conditionsJSON),
		Containers:        string(containersJSON),
		InitContainers:    string(initContainersJSON),
		Volumes:           string(volumesJSON),
		CreationTimestamp: pod.CreationTimestamp.Time,
		StartTime:         getPodStartTime(pod),
		DeletionTimestamp: getPodDeletionTimestamp(pod),
		OwnerReferences:   string(ownerRefsJSON),
		ResourceVersion:   pod.ResourceVersion,
		Generation:        pod.Generation,
		Spec:              string(specJSON),
	}
}

// getPodStartTime 获取Pod启动时间
func getPodStartTime(pod *corev1.Pod) *time.Time {
	if pod.Status.StartTime != nil {
		t := pod.Status.StartTime.Time
		return &t
	}
	return nil
}

// getPodDeletionTimestamp 获取Pod删除时间戳
func getPodDeletionTimestamp(pod *corev1.Pod) *time.Time {
	if pod.DeletionTimestamp != nil {
		t := pod.DeletionTimestamp.Time
		return &t
	}
	return nil
}

// getTotalRestartCount 获取Pod总重启次数
func getTotalRestartCount(pod *corev1.Pod) int32 {
	var total int32
	for _, cs := range pod.Status.ContainerStatuses {
		total += cs.RestartCount
	}
	return total
}

// getReadyStatus 获取就绪状态
func getReadyStatus(pod *corev1.Pod) string {
	ready := 0
	total := len(pod.Status.ContainerStatuses)
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			ready++
		}
	}
	return fmt.Sprintf("%d/%d", ready, total)
}

// convertPodConditions 转换Pod条件
func convertPodConditions(conds []corev1.PodCondition) []model.PodCondition {
	if len(conds) == 0 {
		return nil
	}
	var res []model.PodCondition
	for _, c := range conds {
		res = append(res, model.PodCondition{
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

// convertContainers 转换容器列表（仅规格信息）
func convertContainers(containers []corev1.Container) []model.PodContainer {
	if len(containers) == 0 {
		return nil
	}

	var results []model.PodContainer
	for _, c := range containers {
		container := model.PodContainer{
			Name:            c.Name,
			Image:           c.Image,
			Command:         c.Command,
			Args:            c.Args,
			Envs:            convertEnvVars(c.Env),
			Ports:           convertContainerPorts(c.Ports),
			Resources:       convertResourceRequirements(c.Resources),
			VolumeMounts:    convertVolumeMounts(c.VolumeMounts),
			ImagePullPolicy: string(c.ImagePullPolicy),
		}

		if c.LivenessProbe != nil {
			container.LivenessProbe = convertProbe(c.LivenessProbe)
		}
		if c.ReadinessProbe != nil {
			container.ReadinessProbe = convertProbe(c.ReadinessProbe)
		}

		results = append(results, container)
	}
	return results
}

// convertContainersWithStatus 转换容器列表（包含运行时状态）
func convertContainersWithStatus(containers []corev1.Container, statuses []corev1.ContainerStatus) []model.PodContainer {
	if len(containers) == 0 {
		return nil
	}

	var results []model.PodContainer
	for _, c := range containers {
		container := model.PodContainer{
			Name:            c.Name,
			Image:           c.Image,
			Command:         c.Command,
			Args:            c.Args,
			Envs:            convertEnvVars(c.Env),
			Ports:           convertContainerPorts(c.Ports),
			Resources:       convertResourceRequirements(c.Resources),
			VolumeMounts:    convertVolumeMounts(c.VolumeMounts),
			ImagePullPolicy: string(c.ImagePullPolicy),
		}

		if c.LivenessProbe != nil {
			container.LivenessProbe = convertProbe(c.LivenessProbe)
		}
		if c.ReadinessProbe != nil {
			container.ReadinessProbe = convertProbe(c.ReadinessProbe)
		}

		// 通过容器名称匹配运行时状态
		for _, cs := range statuses {
			if cs.Name == c.Name {
				container.Ready = cs.Ready
				container.RestartCount = cs.RestartCount
				container.State = convertContainerState(cs.State)
				break
			}
		}

		results = append(results, container)
	}
	return results
}

// convertEnvVars 转换环境变量
func convertEnvVars(envs []corev1.EnvVar) []model.PodEnvVar {
	if len(envs) == 0 {
		return nil
	}
	var res []model.PodEnvVar
	for _, e := range envs {
		res = append(res, model.PodEnvVar{
			Name:  e.Name,
			Value: e.Value,
		})
	}
	return res
}

// convertContainerPorts 转换容器端口
func convertContainerPorts(ports []corev1.ContainerPort) []model.PodContainerPort {
	if len(ports) == 0 {
		return nil
	}
	var res []model.PodContainerPort
	for _, p := range ports {
		res = append(res, model.PodContainerPort{
			Name:          p.Name,
			ContainerPort: p.ContainerPort,
			Protocol:      string(p.Protocol),
		})
	}
	return res
}

// convertVolumeMounts 转换卷挂载
func convertVolumeMounts(mounts []corev1.VolumeMount) []model.PodVolumeMount {
	if len(mounts) == 0 {
		return nil
	}
	var res []model.PodVolumeMount
	for _, m := range mounts {
		res = append(res, model.PodVolumeMount{
			Name:      m.Name,
			MountPath: m.MountPath,
			ReadOnly:  m.ReadOnly,
			SubPath:   m.SubPath,
		})
	}
	return res
}

// convertResourceRequirements 转换资源要求
func convertResourceRequirements(rr corev1.ResourceRequirements) model.PodResourceRequirements {
	result := model.PodResourceRequirements{}

	if rr.Requests != nil {
		if cpu, ok := rr.Requests[corev1.ResourceCPU]; ok {
			result.Requests.CPU = cpu.String()
		}
		if mem, ok := rr.Requests[corev1.ResourceMemory]; ok {
			result.Requests.Memory = mem.String()
		}
	}
	if rr.Limits != nil {
		if cpu, ok := rr.Limits[corev1.ResourceCPU]; ok {
			result.Limits.CPU = cpu.String()
		}
		if mem, ok := rr.Limits[corev1.ResourceMemory]; ok {
			result.Limits.Memory = mem.String()
		}
	}
	return result
}

// convertProbe 转换探针
func convertProbe(probe *corev1.Probe) *model.PodProbe {
	if probe == nil {
		return nil
	}

	result := &model.PodProbe{
		InitialDelaySeconds: probe.InitialDelaySeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		TimeoutSeconds:      probe.TimeoutSeconds,
		SuccessThreshold:    probe.SuccessThreshold,
		FailureThreshold:    probe.FailureThreshold,
	}

	if probe.HTTPGet != nil {
		result.HTTPGet = &model.PodHTTPGetAction{
			Path:   probe.HTTPGet.Path,
			Port:   probe.HTTPGet.Port.IntVal,
			Scheme: string(probe.HTTPGet.Scheme),
		}
	}

	if probe.TCPSocket != nil {
		result.TCPSocket = &model.PodTCPSocketAction{
			Port: probe.TCPSocket.Port.IntVal,
		}
	}

	if probe.Exec != nil {
		result.Exec = &model.PodExecAction{
			Command: probe.Exec.Command,
		}
	}

	return result
}

func ConvertPodContainers(pod *corev1.Pod) []model.PodContainer {
	if pod == nil {
		return nil
	}

	var containers []model.PodContainer

	for _, container := range pod.Spec.Containers {
		podContainer := model.PodContainer{
			Name:            container.Name,
			Image:           container.Image,
			Command:         container.Command,
			Args:            container.Args,
			Envs:            convertEnvVars(container.Env),
			Ports:           convertContainerPorts(container.Ports),
			Resources:       convertResourceRequirements(container.Resources),
			VolumeMounts:    convertVolumeMounts(container.VolumeMounts),
			ImagePullPolicy: string(container.ImagePullPolicy),
		}

		if container.LivenessProbe != nil {
			podContainer.LivenessProbe = convertProbe(container.LivenessProbe)
		}
		if container.ReadinessProbe != nil {
			podContainer.ReadinessProbe = convertProbe(container.ReadinessProbe)
		}

		// 通过容器名称匹配运行时状态
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Name == container.Name {
				podContainer.Ready = cs.Ready
				podContainer.RestartCount = cs.RestartCount
				podContainer.State = convertContainerState(cs.State)
				break
			}
		}

		containers = append(containers, podContainer)
	}

	return containers
}

// convertContainerState 转换容器状态
func convertContainerState(state corev1.ContainerState) model.PodContainerState {
	result := model.PodContainerState{}

	if state.Waiting != nil {
		result.Waiting = &model.PodContainerStateWaiting{
			Reason:  state.Waiting.Reason,
			Message: state.Waiting.Message,
		}
	}

	if state.Running != nil {
		result.Running = &model.PodContainerStateRunning{
			StartedAt: state.Running.StartedAt.Time,
		}
	}

	if state.Terminated != nil {
		result.Terminated = &model.PodContainerStateTerminated{
			ExitCode:    state.Terminated.ExitCode,
			Signal:      state.Terminated.Signal,
			Reason:      state.Terminated.Reason,
			Message:     state.Terminated.Message,
			StartedAt:   state.Terminated.StartedAt.Time,
			FinishedAt:  state.Terminated.FinishedAt.Time,
			ContainerID: state.Terminated.ContainerID,
		}
	}

	return result
}

// PodStatus 获取Pod状态
func PodStatus(pod *corev1.Pod) string {
	if pod == nil {
		return StatusUnknown
	}
	if pod.DeletionTimestamp != nil {
		return StatusTerminating
	}
	if pod.Status.Reason == "Evicted" {
		return StatusEvicted
	}
	switch pod.Status.Phase {
	case corev1.PodPending:
		return StatusPending
	case corev1.PodSucceeded:
		return StatusSucceeded
	case corev1.PodFailed:
		return StatusFailed
	case corev1.PodRunning:

		allReady := true
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				allReady = false
				break
			}
		}
		if allReady {
			return StatusRunning
		}
		return StatusUpdating
	default:
		return StatusUnknown
	}
}

func ValidatePod(pod *corev1.Pod) error {
	if pod == nil {
		return fmt.Errorf("pod不能为空")
	}

	if pod.Name == "" {
		return fmt.Errorf("pod名称不能为空")
	}

	if pod.Namespace == "" {
		return fmt.Errorf("namespace不能为空")
	}

	if len(pod.Spec.Containers) == 0 {
		return fmt.Errorf("至少需要一个容器")
	}

	for i, container := range pod.Spec.Containers {
		if container.Name == "" {
			return fmt.Errorf("容器%d名称不能为空", i)
		}
		if container.Image == "" {
			return fmt.Errorf("容器%d镜像不能为空", i)
		}
	}

	return nil
}

// PodToYAML 将Pod转换为YAML
func PodToYAML(pod *corev1.Pod) (string, error) {
	if pod == nil {
		return "", fmt.Errorf("pod不能为空")
	}

	// 清理不需要的字段
	cleanPod := pod.DeepCopy()
	cleanPod.Status = corev1.PodStatus{}
	cleanPod.ManagedFields = nil
	cleanPod.ResourceVersion = ""
	cleanPod.UID = ""
	cleanPod.CreationTimestamp = metav1.Time{}
	cleanPod.Generation = 0

	yamlBytes, err := yaml.Marshal(cleanPod)
	if err != nil {
		return "", fmt.Errorf("转换为YAML失败: %w", err)
	}

	return string(yamlBytes), nil
}

// YAMLToPod 将YAML转换为Pod
func YAMLToPod(yamlContent string) (*corev1.Pod, error) {
	if yamlContent == "" {
		return nil, fmt.Errorf("YAML内容不能为空")
	}

	var pod corev1.Pod
	err := yaml.Unmarshal([]byte(yamlContent), &pod)
	if err != nil {
		return nil, fmt.Errorf("解析YAML失败: %w", err)
	}

	return &pod, nil
}

// IsPodReady 判断Pod是否就绪
func IsPodReady(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func GetPodAge(pod corev1.Pod) string {
	age := time.Since(pod.CreationTimestamp.Time)
	days := int(age.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(age.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(age.Minutes())
	return fmt.Sprintf("%dm", minutes)
}

func BuildPodFromRequest(req *model.CreatePodReq) (*corev1.Pod, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Spec: corev1.PodSpec{
			RestartPolicy:      corev1.RestartPolicy(req.RestartPolicy),
			NodeSelector:       req.NodeSelector,
			Tolerations:        req.Tolerations,
			Affinity:           req.Affinity,
			Volumes:            req.Volumes,
			HostNetwork:        req.HostNetwork,
			HostPID:            req.HostPID,
			DNSPolicy:          corev1.DNSPolicy(req.DNSPolicy),
			ServiceAccountName: req.ServiceAccount,
		},
	}

	for _, c := range req.Containers {
		container := corev1.Container{
			Name:            c.Name,
			Image:           c.Image,
			Command:         c.Command,
			Args:            c.Args,
			ImagePullPolicy: corev1.PullPolicy(c.ImagePullPolicy),
			WorkingDir:      c.WorkingDir,
			SecurityContext: c.SecurityContext,
		}

		for _, env := range c.Envs {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  env.Name,
				Value: env.Value,
			})
		}

		for _, port := range c.Ports {
			container.Ports = append(container.Ports, corev1.ContainerPort{
				Name:          port.Name,
				ContainerPort: port.ContainerPort,
				Protocol:      corev1.Protocol(port.Protocol),
			})
		}

		if c.Resources.Requests.CPU != "" || c.Resources.Requests.Memory != "" ||
			c.Resources.Limits.CPU != "" || c.Resources.Limits.Memory != "" {
			container.Resources = corev1.ResourceRequirements{
				Requests: make(corev1.ResourceList),
				Limits:   make(corev1.ResourceList),
			}

			// 解析资源请求
			if c.Resources.Requests.CPU != "" {
				container.Resources.Requests[corev1.ResourceCPU] = parseQuantity(c.Resources.Requests.CPU)
			}
			if c.Resources.Requests.Memory != "" {
				container.Resources.Requests[corev1.ResourceMemory] = parseQuantity(c.Resources.Requests.Memory)
			}

			// 解析资源限制
			if c.Resources.Limits.CPU != "" {
				container.Resources.Limits[corev1.ResourceCPU] = parseQuantity(c.Resources.Limits.CPU)
			}
			if c.Resources.Limits.Memory != "" {
				container.Resources.Limits[corev1.ResourceMemory] = parseQuantity(c.Resources.Limits.Memory)
			}
		}

		for _, vm := range c.VolumeMounts {
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      vm.Name,
				MountPath: vm.MountPath,
				ReadOnly:  vm.ReadOnly,
				SubPath:   vm.SubPath,
			})
		}

		if c.LivenessProbe != nil {
			container.LivenessProbe = convertModelProbeToK8sProbe(c.LivenessProbe)
		}

		if c.ReadinessProbe != nil {
			container.ReadinessProbe = convertModelProbeToK8sProbe(c.ReadinessProbe)
		}

		pod.Spec.Containers = append(pod.Spec.Containers, container)
	}

	for _, c := range req.InitContainers {
		container := corev1.Container{
			Name:            c.Name,
			Image:           c.Image,
			Command:         c.Command,
			Args:            c.Args,
			ImagePullPolicy: corev1.PullPolicy(c.ImagePullPolicy),
			WorkingDir:      c.WorkingDir,
			SecurityContext: c.SecurityContext,
		}

		for _, env := range c.Envs {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  env.Name,
				Value: env.Value,
			})
		}

		for _, port := range c.Ports {
			container.Ports = append(container.Ports, corev1.ContainerPort{
				Name:          port.Name,
				ContainerPort: port.ContainerPort,
				Protocol:      corev1.Protocol(port.Protocol),
			})
		}

		if c.Resources.Requests.CPU != "" || c.Resources.Requests.Memory != "" ||
			c.Resources.Limits.CPU != "" || c.Resources.Limits.Memory != "" {
			container.Resources = corev1.ResourceRequirements{
				Requests: make(corev1.ResourceList),
				Limits:   make(corev1.ResourceList),
			}

			// 解析资源请求
			if c.Resources.Requests.CPU != "" {
				container.Resources.Requests[corev1.ResourceCPU] = parseQuantity(c.Resources.Requests.CPU)
			}
			if c.Resources.Requests.Memory != "" {
				container.Resources.Requests[corev1.ResourceMemory] = parseQuantity(c.Resources.Requests.Memory)
			}

			// 解析资源限制
			if c.Resources.Limits.CPU != "" {
				container.Resources.Limits[corev1.ResourceCPU] = parseQuantity(c.Resources.Limits.CPU)
			}
			if c.Resources.Limits.Memory != "" {
				container.Resources.Limits[corev1.ResourceMemory] = parseQuantity(c.Resources.Limits.Memory)
			}
		}

		for _, vm := range c.VolumeMounts {
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      vm.Name,
				MountPath: vm.MountPath,
				ReadOnly:  vm.ReadOnly,
				SubPath:   vm.SubPath,
			})
		}

		if c.LivenessProbe != nil {
			container.LivenessProbe = convertModelProbeToK8sProbe(c.LivenessProbe)
		}

		if c.ReadinessProbe != nil {
			container.ReadinessProbe = convertModelProbeToK8sProbe(c.ReadinessProbe)
		}

		pod.Spec.InitContainers = append(pod.Spec.InitContainers, container)
	}

	return pod, nil
}

// parseQuantity 解析资源数量字符串，失败时返回零值
func parseQuantity(s string) resource.Quantity {
	if s == "" {
		return resource.Quantity{}
	}
	q, err := resource.ParseQuantity(s)
	if err != nil {
		// 如果解析失败，返回零值
		return resource.Quantity{}
	}
	return q
}

// convertModelProbeToK8sProbe 将 model.PodProbe 转换为 Kubernetes Probe
func convertModelProbeToK8sProbe(probe *model.PodProbe) *corev1.Probe {
	if probe == nil {
		return nil
	}

	k8sProbe := &corev1.Probe{
		InitialDelaySeconds: probe.InitialDelaySeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		TimeoutSeconds:      probe.TimeoutSeconds,
		SuccessThreshold:    probe.SuccessThreshold,
		FailureThreshold:    probe.FailureThreshold,
	}

	if probe.HTTPGet != nil {
		k8sProbe.ProbeHandler = corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   probe.HTTPGet.Path,
				Port:   intstr.FromInt(int(probe.HTTPGet.Port)),
				Scheme: corev1.URIScheme(probe.HTTPGet.Scheme),
			},
		}
	}

	if probe.TCPSocket != nil {
		k8sProbe.ProbeHandler = corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(int(probe.TCPSocket.Port)),
			},
		}
	}

	if probe.Exec != nil {
		k8sProbe.ProbeHandler = corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: probe.Exec.Command,
			},
		}
	}

	return k8sProbe
}

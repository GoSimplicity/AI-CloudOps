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

package model

type ContainerCore struct {
	Name       string            `json:"name,omitempty" gorm:"comment:容器名称"` // 容器名称
	CPU        string            `json:"cpu,omitempty" gorm:"comment:CPU 资源限制"`        // CPU 资源限制(如 "100m", "0.5")
	Memory     string            `json:"memory,omitempty" gorm:"comment:内存资源限制"`      // 内存资源限制(如 "512Mi", "2Gi")
	CPURequest string            `json:"cpu_request,omitempty" gorm:"comment:CPU 资源请求"` // CPU 资源请求
	MemRequest string            `json:"mem_request,omitempty" gorm:"comment:内存资源请求"` // 内存资源请求
	Command    []string          `json:"command,omitempty" gorm:"serializer:json;comment:容器启动命令"` // 容器启动命令
	Args       []string          `json:"args,omitempty" gorm:"serializer:json;comment:容器启动参数"`    // 容器启动参数
	Envs       map[string]string `json:"envs,omitempty" gorm:"serializer:json;comment:环境变量"`       // 环境变量
	PullPolicy string            `json:"pull_policy,omitempty" gorm:"comment:镜像拉取策略"` // 镜像拉取策略
	Volumes    []Volume          `json:"volumes,omitempty" gorm:"serializer:json;comment:挂载卷"`      // 挂载卷
}

type OneEvent struct {
	Type      string `json:"type"`       // 事件类型，例如 "Normal", "Warning"
	Component string `json:"component"`  // 事件的组件来源，例如 "kubelet"
	Reason    string `json:"reason"`     // 事件的原因，例如 "NodeReady"
	Message   string `json:"message"`    // 事件的详细消息
	FirstTime string `json:"first_time"` // 事件第一次发生的时间，例如 "2024-04-27T10:00:00Z"
	LastTime  string `json:"last_time"`  // 事件最近一次发生的时间，例如 "2024-04-27T12:00:00Z"
	Object    string `json:"object"`     // 事件关联的对象信息，例如 "kind:Node name:node-1"
	Count     int    `json:"count"`      // 事件发生的次数
}

type Taint struct {
	Key    string `json:"key" binding:"required"`                                                // Taint 的键
	Value  string `json:"value,omitempty"`                                                       // Taint 的值
	Effect string `json:"effect" binding:"required,oneof=NoSchedule PreferNoSchedule NoExecute"` // Taint 的效果，例如 "NoSchedule", "PreferNoSchedule", "NoExecute"
}

type ResourceRequirements struct {
	Requests K8sResourceList `json:"requests,omitempty" gorm:"type:text;serializer:json;comment:资源请求"` // 资源请求
	Limits   K8sResourceList `json:"limits,omitempty" gorm:"type:text;serializer:json;comment:资源限制"`   // 资源限制
}

type K8sResourceList struct {
	CPU    string `json:"cpu,omitempty" gorm:"size:50;comment:CPU 数量，例如 '500m', '2'"`     // CPU 数量，例如 "500m", "2"
	Memory string `json:"memory,omitempty" gorm:"size:50;comment:内存数量，例如 '1Gi', '512Mi'"` // 内存数量，例如 "1Gi", "512Mi"
}

type KeyValueItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type BatchDeleteReq struct {
	IDs []int `json:"ids" binding:"required"`
}
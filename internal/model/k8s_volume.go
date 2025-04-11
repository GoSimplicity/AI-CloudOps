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

// K8sOneVolume 单个卷的配置
type K8sOneVolume struct {
	Type         string `json:"type" gorm:"comment:卷类型，如 hostPath, configMap, emptyDir, pvc"`               // 卷类型
	Name         string `json:"name" gorm:"size:100;comment:卷名称"`                                           // 卷名称
	MountPath    string `json:"mount_path" gorm:"size:255;comment:挂载路径"`                                    // 挂载路径
	SubPath      string `json:"sub_path,omitempty" gorm:"size:255;comment:子路径"`                             // 子路径（可选）
	PvcName      string `json:"pvc_name,omitempty" gorm:"size:100;comment:PVC名称，当类型为 pvc 时使用"`              // PVC名称（可选）
	CmName       string `json:"cm_name,omitempty" gorm:"size:100;comment:ConfigMap名称，当类型为 configMap 时使用"`   // ConfigMap名称（可选）
	HostPath     string `json:"host_path,omitempty" gorm:"size:255;comment:Host路径，当类型为 hostPath 时使用"`       // Host路径（可选）
	HostPathType string `json:"host_path_type,omitempty" gorm:"size:50;comment:Host路径类型，当类型为 hostPath 时使用"` // Host路径类型（可选）
}